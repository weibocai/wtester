package swarm

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/k0kubun/go-ansi"
	"github.com/schollz/progressbar/v3"
	"github.com/wtester/pkg/config"
	"github.com/wtester/pkg/runner"
	"github.com/wtester/pkg/runner/single"
	"github.com/wtester/pkg/stage"
	"go.uber.org/zap"
)

func swarmRunTicker(ctx context.Context, sr runner.Runner, stageIndex, actorNum int, actorDeal map[string]*Actor) error {
	// 计时器
	ticker := time.NewTicker(1 * time.Second)
	actorNumHalf := actorNum / 2
	defer ticker.Stop()
	defer func() {
		actorDealNum := uint32(0)
		msg := &StageRequest{Kind: StatementKind_StartCoroutine, CoroutinesNumber: &actorDealNum, StageIndex: uint32(stageIndex)}
		for _, a := range actorDeal {
			ctxA, cancel := context.WithTimeout(context.Background(), time.Second)
			res, err := a.WClient.ActorStage(ctxA, msg)
			if err != nil || res.GetStatus() == Status_Fail {

			}
			cancel()
		}
	}()
	sg := sr.GetStage()
	r := rand.New(rand.NewSource(sr.GetRandSource()))
	// 执行器
	tickerTask := func(tIndex, lrp int) error {
		// 确保进程分配正确
		for {
			actorCurrentNum := GetActiveActor(actorDeal)
			// 检查当前启动的执行端个数，当启动个数小于设置个数的一半的时候，会触发异常，执行结束
			if actorCurrentNum <= actorNumHalf {
				return fmt.Errorf("阶段：%s: 当前启动的执行端个数小于设置执行端个数的一半，退出当前阶段", sr.GetStage().Name)
			}
			keys := make([]string, actorCurrentNum)
			messages := make(map[string]*StageRequest, actorCurrentNum)
			index := 0

			// 计算每个节点要处理的协程的个数
			actorDealNum := uint32(lrp / actorCurrentNum)
			// 平均分配节点需要处理的进程的个数
			for k, v := range actorDeal {
				if !v.IsActiveNode() {
					continue
				}
				keys[index] = k
				messages[k] = &StageRequest{Kind: StatementKind_StartCoroutine, CoroutinesNumber: &actorDealNum, StageIndex: uint32(stageIndex)}
			}
			// 随机分配剩余执行协程
			_lrp := lrp - actorCurrentNum*int(actorDealNum)
			if _lrp > 0 {
				r.Shuffle(len(keys), func(i, j int) { keys[i], keys[j] = keys[j], keys[i] })
				for i := 0; i < _lrp; i++ {
					a, _ := messages[keys[i]]
					rp := *a.CoroutinesNumber + uint32(1)
					a.CoroutinesNumber = &rp
				}
			}
			// 检查任务下发时，节点数是否和计算时一致
			if GetActiveActor(actorDeal) != actorCurrentNum {
				config.Logger.Warn(fmt.Sprintf("%s 启动测试进程时，执行端个数有波动，将重新分配任务", sg.Name))
				continue
			}
			// 下发执行协程，这里默认，下发成功即认为启动成功
			for k, v := range messages {
				if a, ok := actorDeal[k]; ok {
					ctxA, cancel := context.WithTimeout(context.Background(), time.Second)
					res, err := a.WClient.ActorStage(ctxA, v)
					if err != nil || res.GetStatus() == Status_Fail {
						cancel()
						return fmt.Errorf("%s 启动测试进程时，任务分配失败", sg.Name)
					}
					cancel()
				}
			}
			sr.PlusCcCount(lrp)
			write2recodes("director", fmt.Sprintf("阶段：%s, 第%d秒, 执行端：%s", sg.Name, tIndex, strings.Join(keys, ",")), nil)
			config.Logger.Info(fmt.Sprintf("总并发数：%d, 当前启动并发数：%d", sg.NumberOfConcurrent, sr.GetCcCount()))
			return nil
		}
	}
	// 任务启动：在计时器启动之前，启动一次任务
	lrp := sr.GetRemainingCoroutinesNumber()
	if lrp <= 0 {
		return nil
	}
	if err := tickerTask(1, lrp); err != nil {
		write2recodes("director", fmt.Sprintf("%s: 执行端运行异常，中断本阶段测试执行", sg.Name), err)
		return err
	}
	i := 2
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			// 本次需要启动的并发的个数
			lrp = sr.GetRemainingCoroutinesNumber()
			if lrp <= 0 {
				return nil
			}
			if err := tickerTask(i, lrp); err != nil {
				return nil
			}
			i += 1
		}
	}
}

func swarmRunner(ctx context.Context, cf context.CancelFunc, sr runner.Runner, index, actorNum int, actorDeal map[string]*Actor) {
	var wg sync.WaitGroup
	// 测试任务分配
	wg.Go(func() {
		if err := swarmRunTicker(ctx, sr, index, actorNum, actorDeal); err != nil {
			cf()
		}
	})
	// 守护协程，监控测试是否执行完成
	wg.Go(func() {
		sr.Daemon(ctx, cf)
	})
	wg.Wait()
}

// 构建每个阶段的 runner
func initRunner(stages []*stage.Stage, actorNum int, randSource int64) []runner.Runner {
	runnerList := make([]runner.Runner, len(stages))
	for i := range stages {
		st := stages[i]
		if actorNum > -1 {
			if actorNum > st.RampUp {
				config.Logger.Warn(fmt.Sprintf("不建议每秒启动线程个数小于执行端个数：%s RampUp=%d 执行端个数=%d", stages[i].Name, stages[i].RampUp, actorNum))
			}
			if st.RampUp%actorNum != 0 {
				config.Logger.Warn(fmt.Sprintf("建议RampUp为执行端个数的整数倍：%s RampUp=%d 执行端个数=%d", stages[i].Name, stages[i].RampUp, actorNum))
			}
		}
		var sr runner.Runner
		if st.GetExecutionOrderType() == stage.ExecutionOrderTypeLoop {

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

			dr := single.DurationRunner{BaseRunner: runner.BaseRunner{Stage: st, Bar: bar, CoroutinesNumber: 0, RandSource: randSource}}
			sr = &DurationRunner{DurationRunner: dr}
		}
		runnerList[i] = sr
	}
	return runnerList
}

// DirectorActuator 服务端启动程序
func DirectorActuator(actorNum int, signal chan os.Signal, stages []*stage.Stage) {
	// 定义异常
	errChan := make(chan error, 1)
	// runner
	srl := initRunner(stages, actorNum, time.Now().UnixNano())
	// 工作节点：状态记录、消息处理队列
	actorDeal := make(map[string]*Actor, actorNum)
	// 服务节点消息处理队列
	directorDeal := make(chan *DirectorDealMessage, 100)
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	// 确保程序退出前执行
	defer close(directorDeal)
	defer close(errChan)
	defer func() {
		for _, d := range actorDeal {
			d.Close()
		}
	}()
	defer func() {
		if err := recover(); err != nil {
			config.Logger.Error("程序终止失败", zap.Any("err", err))
		}
		BroadcastTerminate(actorDeal)
	}()
	defer wg.Wait()

	// 检测程序执行情况：正常结束 or 检测异常 or 外部终止信号
	wg.Go(func() {
		for {
			select {
			case <-ctx.Done():
				write2recodes("director", "测试结束", nil)
				BroadcastTerminate(actorDeal)
				return
			case err := <-errChan:
				// 处理异常情况，终止程序
				write2recodes("director", "测试异常终止", err)
				BroadcastTerminate(actorDeal)
				cancel()
				return
			case s := <-signal:
				// 处理主动关闭程序的情况
				write2recodes("director", fmt.Sprintf("%v Pod 主动或是异常终止", s), nil)
				BroadcastTerminate(actorDeal)
				cancel()
				return
			}
		}
	})

	// 启动 grpc服务端
	wg.Go(func() {
		directorService(ctx, errChan, directorDeal, actorDeal)
	})

	// 执行端消息处理
	wg.Go(func() {
		for {
			select {
			case <-ctx.Done():
				return
			case m := <-directorDeal:
				switch m.Kind {
				case StatementKind_FbCoroutinesNumber:
					// 执行段反馈启动的协程数
					_sr := srl[int(m.GetStageIndex())].(runner.SwarmRunner)
					_sr.PlusRealCoroutinesNumber(int(m.GetCoroutinesNumber()))
				}
			}
		}
	})

	// 启动测试执行程序
	wg.Go(func() {
		for i := range stages {
			// 新阶段总是等待所有的节点状态正常，否则认为不满足测试条件，中止测试
			ctxActor, cancelActor := context.WithTimeout(ctx, time.Minute*9)
		WaitingActor:
			for {
				select {
				case <-ctxActor.Done():
					break WaitingActor
				default:
					_actorNum := GetActiveActor(actorDeal)
					config.Logger.Info(fmt.Sprintf("应启动客户端/当前启动客户端个数：%d/%d", actorNum, _actorNum))
					if _actorNum >= actorNum {
						break WaitingActor
					}
					time.Sleep(time.Second * 1)
				}
			}
			cancelActor()
			// 检查执行端启动的个数，如果不匹配则终止测试程序
			if GetActiveActor(actorDeal) < actorNum {
				errChan <- fmt.Errorf("timeout 执行端启动失败：应启动%d，实际启动%d", actorNum, len(actorDeal))
				cancel()
				return
			}
			// 设置执行端的执行阶段
			if !NotifyUpdateStage(ctx, actorDeal, uint32(i), errChan) {
				cancel()
				return
			}

			sr := srl[i]
			sctx, cf := context.WithCancel(ctx)
			swarmRunner(sctx, cf, srl[i], i, actorNum, actorDeal)

			// 检查执行端是否执行结束， 总是确保执行端本阶段执行完成
			stctx, tcf := context.WithTimeout(ctx, time.Second*60)
		WaitingActorFinish:
			for {
				select {
				case <-stctx.Done():
					break WaitingActorFinish
				default:
					if IsStateFinish(actorDeal, i) == "" {
						tcf()
						break WaitingActorFinish
					}
					time.Sleep(time.Second * 1)
				}
			}
			tcf()
			_a := IsStateFinish(actorDeal, i)
			if _a != "" {
				errChan <- fmt.Errorf("执行端%s：阶段%s执行超时", _a, sr.GetStage().Name)
				cancel()
			}
			config.Logger.Info("stages 执行结束", zap.Any("stages", sr.GetStage().Name))
		}
		cancel()
	})
}
