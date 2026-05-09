package result

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/influxdata/tdigest"
	"github.com/wtester/pkg/config"
	"github.com/wtester/pkg/storge"
)

// DbProcessor 结果处理器
type DbProcessor struct {
	Sampling time.Duration // 采样频率

	resultWriter    chan *storge.Result
	errorWriter     chan *storge.Error
	statisticWriter chan *storge.Statistic
}

func (mp *DbProcessor) Init() error {
	if storge.GormDB == nil {
		return errors.New("GormDB is nil")
	}
	mp.resultWriter = make(chan *storge.Result, config.WTesterConfig.Db.GetBatchSize())
	mp.errorWriter = make(chan *storge.Error, config.WTesterConfig.Db.GetBatchSize())
	mp.statisticWriter = make(chan *storge.Statistic, config.WTesterConfig.Db.GetBatchSize())
	return nil
}
func (mp *DbProcessor) Done() error {
	close(mp.resultWriter)
	close(mp.errorWriter)
	close(mp.statisticWriter)
	return nil
}

// 结果写入
func (mp *DbProcessor) writeResult(ctx context.Context, group *sync.WaitGroup) {
	defer group.Done()
	var rs []*storge.Result
	for {
		select {
		case <-ctx.Done():
			if len(rs) > 0 {
				storge.GormDB.Create(&rs)
			}
			config.Logger.Info("write result to db success")
			return
		case r := <-mp.resultWriter:
			rs = append(rs, r)
			if len(rs) >= config.WTesterConfig.Db.GetBatchSize() {
				storge.GormDB.Create(&rs)
				rs = make([]*storge.Result, 0)
			}
		}
	}
}

// 异常信息写入
func (mp *DbProcessor) writeError(ctx context.Context, group *sync.WaitGroup) {
	defer group.Done()
	var es []*storge.Error
	for {
		select {
		case <-ctx.Done():
			if len(es) > 0 {
				storge.GormDB.Create(&es)
			}
			config.Logger.Info("write error to db success")
			return
		case err := <-mp.errorWriter:
			es = append(es, err)
			if len(es) >= config.WTesterConfig.Db.GetBatchSize() {
				storge.GormDB.Create(&es)
				es = make([]*storge.Error, 0)
			}
		}
	}
}

// 统计信息写入
func (mp *DbProcessor) writeStatistic(ctx context.Context, group *sync.WaitGroup) {
	defer group.Done()
	var ss []*storge.Statistic
	for {
		select {
		case <-ctx.Done():
			if len(ss) > 0 {
				storge.GormDB.Create(&ss)
			}
			config.Logger.Info("write statistic to db success")
			return
		case s := <-mp.statisticWriter:
			ss = append(ss, s)
			if len(ss) >= config.WTesterConfig.Db.GetBatchSize() {
				storge.GormDB.Create(&ss)
				ss = make([]*storge.Statistic, 0)
			}
		}
	}
}

// 汇总结过处理
func (mp *DbProcessor) dealTd(statistic map[string]*storge.Statistic, td map[string]*tdigest.TDigest) {
	for k, ttd := range td {
		statistic[k].UpdateStatistic(ttd)
		mp.statisticWriter <- statistic[k]
	}
}

// Process 结果处理器主函数
func (mp *DbProcessor) Process(ctx context.Context, results chan *storge.Result) {
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
	statistic := map[string]*storge.Statistic{}
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
				statistic[key] = &storge.Statistic{
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
