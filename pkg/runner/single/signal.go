package single

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/k0kubun/go-ansi"
	"github.com/schollz/progressbar/v3"
	"github.com/wtester/pkg/config"
	"github.com/wtester/pkg/result"
	"github.com/wtester/pkg/runner"
	"github.com/wtester/pkg/stage"
	"github.com/wtester/pkg/storge"
)

func signalRunTicker(ctx context.Context, sr runner.Runner, rc chan *storge.Result) {
	// 启动计时器
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	sg := sr.GetStage()
	// 任务并发器
	var group sync.WaitGroup
	defer group.Wait()
	tickerTask := func() bool {
		// 本次需要启动的并发的个数
		lrp := sr.GetRemainingCoroutinesNumber()
		if lrp <= 0 {
			return false
		}
		for i := 0; i < lrp; i++ {
			group.Add(1)
			switch sg.ExecutionMode {
			case stage.ExecutionModeRandom:
				group.Go(func() {
					sr.RunnerRandom(ctx, rc)
				})
			case stage.ExecutionModeWeight:
				group.Go(func() {
					sr.RunnerWeight(ctx, rc)
				})
			default:
				group.Go(func() {
					sr.RunnerOrder(ctx, rc)
				})
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

func signalRunner(sr runner.Runner) error {
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

	results := make(chan *storge.Result, 1000)
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	// 启动结果处理程序
	wg.Go(func() {
		rp.Process(ctx, results)
	})
	// 守护协程，监控测试是否执行完成
	wg.Go(func() { sr.Daemon(ctx, cancel) })
	// 执行协程，定时启动协程执行测试
	wg.Go(func() { signalRunTicker(ctx, sr, results) })

	wg.Wait()
	config.Logger.Info("我结束了")
	return nil
}

// Single 单节点执行器
func Single(stages []*stage.Stage) {
	// stage 依次执行测试
	randSource := time.Now().UnixNano()
	for _, st := range stages {
		var sr runner.Runner
		// 分别执行轮次测试+随机测试
		if st.GetExecutionOrderType() == stage.ExecutionOrderTypeLoop {
			bar := progressbar.NewOptions(st.GetLoop(),
				progressbar.OptionSetWriter(ansi.NewAnsiStdout()),
				progressbar.OptionEnableColorCodes(true),
				progressbar.OptionSetWidth(15),
				progressbar.OptionSetDescription("任务执行："),
				progressbar.OptionSetTheme(progressbar.Theme{
					Saucer:        "[green]=[reset]",
					SaucerHead:    "[green]>[reset]",
					SaucerPadding: " ",
					BarStart:      "[",
					BarEnd:        "]",
				}))
			br := runner.BaseRunner{Stage: st, Bar: bar, CoroutinesNumber: 0, RandSource: randSource}
			sr = &LoopRunner{BaseRunner: br, DoneChan: make(chan struct{})}
		} else {
			bar := progressbar.NewOptions(st.GetDuration(),
				progressbar.OptionSetWriter(ansi.NewAnsiStdout()),
				progressbar.OptionEnableColorCodes(true),
				progressbar.OptionSetWidth(15),
				progressbar.OptionSetDescription("任务执行："),
				progressbar.OptionSetTheme(progressbar.Theme{
					Saucer:        "[green]=[reset]",
					SaucerHead:    "[green]>[reset]",
					SaucerPadding: " ",
					BarStart:      "[",
					BarEnd:        "]",
				}))
			br := runner.BaseRunner{Stage: st, Bar: bar, CoroutinesNumber: 0, RandSource: randSource}
			sr = &DurationRunner{BaseRunner: br}
		}
		if err := signalRunner(sr); err != nil {
			config.Logger.Error(fmt.Sprintf("%s 测试失败异常退出: %v", st.Name, err))
		}
	}
}
