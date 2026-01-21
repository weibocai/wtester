package result

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/influxdata/tdigest"
	"github.com/wtester/pkg/config"
	"github.com/wtester/pkg/storge"
)

// FileProcessor 文件结果处理器
type FileProcessor struct {
	Sampling time.Duration // 采样频率

	resultFIle      *os.File // 结果记录器
	errorFile       *os.File // 异常结果记录
	statisticFile   *os.File // 统计记录器
	resultWriter    *bufio.Writer
	errorWriter     *bufio.Writer
	statisticWriter *bufio.Writer
}

func (fp *FileProcessor) Init() error {
	var err error
	// 请求结果
	fp.resultFIle, err = os.OpenFile(config.WTesterConfig.Db.GetResultPath(), os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	fp.resultWriter = bufio.NewWriterSize(fp.resultFIle, config.WTesterConfig.Db.GetBatchSize())

	// 异常结果
	fp.errorFile, err = os.OpenFile(config.WTesterConfig.Db.GetErrorPath(), os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	fp.errorWriter = bufio.NewWriterSize(fp.statisticFile, config.WTesterConfig.Db.GetBatchSize())

	// 统计结果
	fp.statisticFile, err = os.OpenFile(config.WTesterConfig.Db.GetStatisticPath(), os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	fp.statisticWriter = bufio.NewWriterSize(fp.statisticFile, config.WTesterConfig.Db.GetBatchSize())
	return nil
}

// Done 刷新结果，并关闭文件缓冲区
func (fp *FileProcessor) Done() error {
	errString := strings.Builder{}
	if fp.resultWriter != nil {
		if err := fp.resultWriter.Flush(); err != nil {
			errString.WriteString(fmt.Sprint(err))
		}
	}
	if fp.resultFIle != nil {
		if err := fp.resultFIle.Close(); err != nil {
			errString.WriteString(fmt.Sprint(err))
		}
	}
	if fp.errorWriter != nil {
		if err := fp.errorWriter.Flush(); err != nil {
			errString.WriteString(fmt.Sprint(err))
		}
	}
	if fp.errorFile != nil {
		if err := fp.errorFile.Close(); err != nil {
			errString.WriteString(fmt.Sprint(err))
		}
	}
	if fp.statisticWriter != nil {
		if err := fp.statisticWriter.Flush(); err != nil {
			errString.WriteString(fmt.Sprint(err))
		}
	}
	if fp.statisticFile != nil {
		if err := fp.statisticFile.Close(); err != nil {
			errString.WriteString(fmt.Sprint(err))
		}
	}

	if errString.Len() > 0 {
		return errors.New(errString.String())
	}
	return nil
}

// 处理统计结果
func (fp *FileProcessor) dealTd(eol []byte, statistic map[string]*storge.Statistic, td map[string]*tdigest.TDigest) {
	for k, ttd := range td {
		statistic[k].UpdateStatistic(ttd)
		// 结果写入文件
		if jsonResult, err := json.Marshal(statistic[k]); err != nil {
			config.Logger.Warn(fmt.Sprintf("结果写入异常：%v, %s", statistic[k], err))
		} else {
			if _, err = fp.statisticWriter.Write(jsonResult); err != nil {
				config.Logger.Warn(fmt.Sprintf("结果写入异常：%v, %s", statistic[k], err))
				continue
			}
			if _, err = fp.statisticWriter.Write(eol); err != nil {
				config.Logger.Warn(fmt.Sprintf("结果写入异常：%v, %s", statistic[k], err))
			}
		}
	}
}

func (fp *FileProcessor) Process(ctx context.Context, results chan *storge.Result) {
	ticker := time.NewTicker(fp.Sampling)
	defer ticker.Stop()
	eol := []byte{'\n'}
	td := map[string]*tdigest.TDigest{}
	statistic := map[string]*storge.Statistic{}
	for {
		select {
		case <-ctx.Done():
			// 接收终止信号，会丢弃终止信号之后的结果
			fp.dealTd(eol, statistic, td)
			return
		case result := <-results:
			// 如果结果为空，停止接收
			if result == nil {
				fp.dealTd(eol, statistic, td)
				return
			}
			key := result.Stage + "&" + result.Task
			if _, ok := td[key]; !ok {
				td[key] = tdigest.New()
				statistic[key] = &storge.Statistic{
					Stage: result.Stage, Task: result.Task, SuccessCount: 0, FailureCount: 0, Avg: 0, P50: 0, P99: 0, P95: 0, Datetime: time.Now(), CC: 0,
				}
			}
			// 更新统计结果
			if !result.IsSuccess {
				statistic[key].FailureCount++
			} else {
				td[key].Add(float64(result.ExecutionTime), 1)
				statistic[key].SuccessCount++
				statistic[key].Sum += result.ExecutionTime
			}
			statistic[key].CC += result.CC
			statistic[key].Count++

			if result.IsSuccess {
				// 结果写入文件
				if jsonResult, err := json.Marshal(result); err != nil {
					config.Logger.Warn(fmt.Sprintf("结果写入异常：%v, %s", result, err))
				} else {
					if _, err = fp.resultWriter.Write(jsonResult); err != nil {
						config.Logger.Warn(fmt.Sprintf("结果写入异常：%v, %s", result, err))
						continue
					}
					if _, err = fp.resultWriter.Write(eol); err != nil {
						config.Logger.Warn(fmt.Sprintf("结果写入异常：%v, %s", result, err))
					}
				}
			} else {
				// 结果写入文件
				errResult := result.ToError()
				if jsonResult, err := json.Marshal(errResult); err != nil {
					config.Logger.Warn(fmt.Sprintf("结果写入异常：%v, %s", result, err))
				} else {
					if _, err = fp.errorWriter.Write(jsonResult); err != nil {
						config.Logger.Warn(fmt.Sprintf("结果写入异常：%v, %s", result, err))
						continue
					}
					if _, err = fp.errorWriter.Write(eol); err != nil {
						config.Logger.Warn(fmt.Sprintf("结果写入异常：%v, %s", result, err))
					}
				}
			}
		case <-ticker.C:
			fp.dealTd(eol, statistic, td)
		}
	}
}
