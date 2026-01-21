package runner

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/wtester/pkg/config"
	"github.com/wtester/pkg/result"
	"github.com/wtester/pkg/stage"
	"github.com/wtester/pkg/task"
)

func writeResult(sr Runner, t task.Task, pIndex int, rc chan *result.Result) {
	isEqual, executionTime, err := t.Runner(pIndex)
	exp := ""
	if err != nil {
		exp = fmt.Sprintf("%v", err.Error())
	}
	res := &result.Result{
		Stage: sr.GetStage().Name, Task: t.GetName(), IsSuccess: err == nil, ExecutionTime: executionTime,
		Datetime: time.Now(), IsEqual: isEqual, CC: sr.GetCcCount(), Exception: exp,
	}
	rc <- res
}

func signalRunTicker(ctx context.Context, sr Runner, wg *sync.WaitGroup, rc chan *result.Result) {
	defer wg.Done()
	// 启动计时器
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	sg := sr.GetStage()
	// 任务并发器
	var group sync.WaitGroup
	defer group.Wait()
	tickerTask := func() bool {
		rp := sg.RampUp
		// 本次需要启动的并发的个数
		lrp := sg.NumberOfConcurrent - sr.GetCcCount()
		if lrp > rp {
			lrp = rp
		}
		if lrp <= 0 {
			return false
		}
		for i := 0; i < lrp; i++ {
			group.Add(1)
			switch sg.ExecutionMode {
			case stage.ExecutionModeRandom:
				go sr.RunnerRandom(ctx, &group, rc)
			case stage.ExecutionModeWeight:
				go sr.RunnerWeight(ctx, &group, rc)
			default:
				go sr.RunnerOrder(ctx, &group, rc)
			}
		}
		sr.PlusCcCount(lrp)
		config.Logger.Info(fmt.Sprintf("总并发数：%d, 当前启动并发数：%d", sg.NumberOfConcurrent, sr.GetCcCount()))
		return true
	}
	// 任务启动：在计时器启动之前，启动一次任务
	if tickerTask() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if !tickerTask() {
					return
				}
			}
		}
	}
}

func signalRunner(sr Runner) error {
	// 启动异常收集功能
	sg := sr.GetStage()
	rp, err := result.GetProcessor(config.WTesterConfig.Db.StorageType, sg.GetSampling())
	if err != nil {
		return fmt.Errorf("%s 启动失败: %v", sg.Name, err)
	}
	if err = rp.Init(); err != nil {
		return fmt.Errorf("%s 启动失败: %v", sg.Name, err)
	}
	defer func() {
		if err = rp.Done(); err != nil {
			config.Logger.Error(err.Error())
		}
	}()

	results := make(chan *result.Result, 1000)
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	// 启动结果处理程序
	wg.Add(1)
	go func() {
		defer wg.Done()
		rp.Process(ctx, results)
	}()
	// 定时关闭
	wg.Add(1)
	// 守护协程，监控程序进程
	go sr.Daemon(cancel, &wg)
	wg.Add(1)
	// 执行协程，定时启动协程执行测试
	go signalRunTicker(ctx, sr, &wg, results)

	wg.Wait()
	config.Logger.Info("我结束了")
	return nil
}

// Signal 单节点执行器
func Signal(stages []*stage.Stage) {
	// stage 依次执行测试
	for _, st := range stages {
		var sr Runner
		if st.GetExecutionOrderType() == stage.ExecutionOrderTypeLoop {
			sr = &SingleLoopRunner{Stage: st, DoneChan: make(chan struct{})}
		} else {
			sr = &SingleDurationRunner{Stage: st}
		}
		if err := signalRunner(sr); err != nil {
			config.Logger.Error(err.Error())
		}
	}
}
