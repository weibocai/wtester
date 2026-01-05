package result

import (
	"context"
	"errors"
	"math"
	"sync"
	"time"

	"github.com/influxdata/tdigest"
	"github.com/wtester/pkg/config"
	"github.com/wtester/pkg/logger"
	"gorm.io/gorm"
)

var GormDB *gorm.DB

// MysqlProcessor 结果处理器
type MysqlProcessor struct {
	Sampling time.Duration // 采样频率

	resultWriter    chan *Result
	errorWriter     chan *Error
	statisticWriter chan *Statistic
}

func (mp *MysqlProcessor) Init() error {
	if GormDB == nil {
		return errors.New("GormDB is nil")
	}
	mp.resultWriter = make(chan *Result, config.WTesterConfig.Result.DbConfig.GetBatchSize())
	mp.errorWriter = make(chan *Error, config.WTesterConfig.Result.DbConfig.GetBatchSize())
	mp.statisticWriter = make(chan *Statistic, config.WTesterConfig.Result.DbConfig.GetBatchSize())
	return nil
}
func (mp *MysqlProcessor) Done() error {
	close(mp.resultWriter)
	close(mp.errorWriter)
	close(mp.statisticWriter)
	return nil
}

// 结果写入
func (mp *MysqlProcessor) writeResult(ctx context.Context, group *sync.WaitGroup) {
	defer group.Done()
	var rs []*Result
	for {
		select {
		case <-ctx.Done():
			if len(rs) > 0 {
				GormDB.Create(&rs)
			}
			logger.Logger.Info("write result to db success")
			return
		case r := <-mp.resultWriter:
			rs = append(rs, r)
			if len(rs) >= config.WTesterConfig.Result.DbConfig.GetBatchSize() {
				GormDB.Create(&rs)
				rs = make([]*Result, 0)
			}
		}
	}
}

// 异常信息写入
func (mp *MysqlProcessor) writeError(ctx context.Context, group *sync.WaitGroup) {
	defer group.Done()
	var es []*Error
	for {
		select {
		case <-ctx.Done():
			if len(es) > 0 {
				GormDB.Create(&es)
			}
			logger.Logger.Info("write error to db success")
			return
		case err := <-mp.errorWriter:
			es = append(es, err)
			if len(es) >= config.WTesterConfig.Result.DbConfig.GetBatchSize() {
				GormDB.Create(&es)
				es = make([]*Error, 0)
			}
		}
	}
}

// 统计信息写入
func (mp *MysqlProcessor) writeStatistic(ctx context.Context, group *sync.WaitGroup) {
	defer group.Done()
	var ss []*Statistic
	for {
		select {
		case <-ctx.Done():
			if len(ss) > 0 {
				GormDB.Create(&ss)
			}
			logger.Logger.Info("write statistic to db success")
			return
		case s := <-mp.statisticWriter:
			ss = append(ss, s)
			if len(ss) >= config.WTesterConfig.Result.DbConfig.GetBatchSize() {
				GormDB.Create(&ss)
				ss = make([]*Statistic, 0)
			}
		}
	}
}

// 会总结过处理
func (mp *MysqlProcessor) dealTd(statistic map[string]*Statistic, td map[string]*tdigest.TDigest) {
	for k, ttd := range td {
		P99 := ttd.Quantile(0.99)
		if P99 < 0 || math.IsNaN(P99) {
			P99 = 0
		}
		P95 := ttd.Quantile(0.95)
		if P95 < 0 || math.IsNaN(P95) {
			P95 = 0
		}
		P50 := ttd.Quantile(0.50)
		if P50 < 0 || math.IsNaN(P50) {
			P50 = 0
		}
		statistic[k].P99 = P99
		statistic[k].P95 = P95
		statistic[k].P50 = P50
		statistic[k].CC = statistic[k].CC / statistic[k].Count
		ttd.Reset()
		statistic[k].Avg = int(statistic[k].Sum) / (statistic[k].SuccessCount + statistic[k].FailureCount)
		statistic[k].Datetime = time.Now()
		mp.statisticWriter <- statistic[k]
	}
}

func (mp *MysqlProcessor) Process(ctx context.Context, results chan *Result) {
	ticker := time.NewTicker(mp.Sampling)
	defer ticker.Stop()
	// 确保程序总是正常中止
	var group sync.WaitGroup
	group.Add(3)
	go mp.writeResult(ctx, &group)
	go mp.writeError(ctx, &group)
	go mp.writeStatistic(ctx, &group)
	defer group.Wait()
	td := map[string]*tdigest.TDigest{}
	statistic := map[string]*Statistic{}
	for {
		select {
		case <-ctx.Done():
			// 接收终止信号，会丢弃终止信号之后的结果
			mp.dealTd(statistic, td)
			return
		case result := <-results:
			// 如果结果为空，停止接收
			if result == nil {
				mp.dealTd(statistic, td)
				return
			}
			key := result.Stage + "&" + result.Task
			if _, ok := td[key]; !ok {
				td[key] = tdigest.New()
				statistic[key] = &Statistic{
					Stage: result.Stage, Task: result.Task, SuccessCount: 0, FailureCount: 0, Avg: 0, P50: 0, P99: 0, P95: 0, Datetime: time.Now(), CC: 0,
				}
			}
			statistic[key].CC += result.CC
			statistic[key].Count++
			// 更新统计结果
			if !result.IsSuccess {
				statistic[key].FailureCount++
				mp.errorWriter <- result.ToError()
			} else {
				td[key].Add(float64(result.ExecutionTime), 1)
				statistic[key].SuccessCount++
				statistic[key].Sum += result.ExecutionTime
				mp.resultWriter <- result
			}
		case <-ticker.C:
			mp.dealTd(statistic, td)
		}
	}
}
