package swarm

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/wtester/pkg/config"
	"github.com/wtester/pkg/result"
	"github.com/wtester/pkg/runner"
	"github.com/wtester/pkg/stage"
	"github.com/wtester/pkg/storge"
)

func swarmActorRunner(ctx context.Context, cf context.CancelFunc, errChan chan error, sr runner.Runner, coroutinesNumber chan int, actor *Actor) {
	sg := sr.GetStage()
	rp, err := result.GetProcessor(config.WTesterConfig.Db.StorageType, sg.GetSampling())
	if err != nil {
		errChan <- fmt.Errorf("%s 结果收集程序启动失败: %v", sg.Name, err)
		return
	}
	if err = rp.Init(); err != nil {
		errChan <- fmt.Errorf("%s 结果收集失败: %v", sg.Name, err)
		return
	}
	results := make(chan *storge.Result, 1000)
	var wg sync.WaitGroup
	defer func() {
		_ = rp.Done()
	}()
	defer wg.Wait()

	// 结果收集程序
	wg.Go(func() {
		rp.Process(ctx, results)
	})
	// 守护协程，监控测试是否执行完成
	wg.Go(func() {
		sr.Daemon(ctx, cf)
	})
	wg.Go(func() {
		var twg sync.WaitGroup
		defer twg.Wait()
		for {
			select {
			case <-ctx.Done():
				return
			case num := <-coroutinesNumber:
				if num == 0 {
					return
				}
				// 启动测试协程
				for i := 0; i < num; i++ {
					switch sg.ExecutionMode {
					case stage.ExecutionModeRandom:
						twg.Go(func() {
							sr.RunnerRandom(ctx, results)
						})
					case stage.ExecutionModeWeight:
						twg.Go(func() {
							sr.RunnerWeight(ctx, results)
						})
					default:
						twg.Go(func() {
							sr.RunnerOrder(ctx, results)
						})
					}
				}
				if !actor.NotifyCoroutinesNumber(ctx, uint32(num), errChan) {
					cf()
					return
				}
			}
		}
	})
}

// ActorActuator 客户端执行器
func ActorActuator(name, host, sHost string, port int, signal chan os.Signal, stages []*stage.Stage) {
	// 定义异常
	errChan := make(chan error, 10) // runner
	defer close(errChan)
	srl := initRunner(stages, -1, time.Now().UnixNano())
	// 定义 context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 构建执行端对象
	actor := InitActor(ctx, sHost, name, uint32(len(stages)+1))
	if actor == nil {
		write2recodes("actor", fmt.Sprintf("%s：连接到服务端失败", name), nil)
		return
	}

	defer actor.Close()
	// 总是确保已注册的执行端收到失败的请求
	defer func() {
		if err := recover(); err != nil {
			actor.NotifyRegister(ctx, ActorStatus_Abnormal, fmt.Sprintf("%s:%d", host, port), errChan)
			write2recodes("actor", fmt.Sprintf("%s：节点下线", name), nil)

		} else {
			actor.NotifyRegister(ctx, ActorStatus_Offline, fmt.Sprintf("%s:%d", host, port), errChan)
			write2recodes("actor", fmt.Sprintf("%s：节点下线", name), nil)

		}
	}()

	var wg sync.WaitGroup
	defer func() {
		wg.Wait()
	}()
	// 检测异常，如果有异常触发，停止整个测试流程
	wg.Go(func() {
		for {
			select {
			case <-ctx.Done():
				return
			case err := <-errChan:
				if err != nil {
					cancel()
					if err != io.EOF {
						write2recodes("actor", "程序异常终止", err)
						actor.NotifyRegister(ctx, ActorStatus_Abnormal, fmt.Sprintf("%s:%d", host, port), errChan)
					}
					return
				}
			case s := <-signal:
				// 处理主动关闭程序的情况
				cancel()
				write2recodes("actor", fmt.Sprintf("%v Pod 主动或是异常终止", s), nil)
				actor.NotifyRegister(ctx, ActorStatus_Abnormal, fmt.Sprintf("%s:%d", host, port), errChan)
				return
			}
		}
	})
	// 启动执行端服务
	wg.Go(func() {
		actorService(ctx, actor, host, port, errChan)
	})

	// 接收服务端终止信号
	wg.Go(func() {
		actor.WaitingEndSignal(ctx, cancel)
	})

	// 启动测试执行程序
	wg.Go(func() {
		defer cancel()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				// 等待 director创建新阶段指令
				if !actor.WaitingUpdateStage(ctx, actor.StageIndex, errChan) {
					return
				}
				currentStage := int(actor.StageIndex)
				sr := srl[currentStage]
				cCtx, cCancel := context.WithCancel(ctx)
				coroutinesNumber := make(chan int, 10)
				swarmActorRunner(cCtx, cCancel, errChan, sr, coroutinesNumber, actor)
				// 通知服务端，本阶段执行完成
				if !actor.NotifyUpdateStage(ctx, errChan) {
					return
				}
			}
		}
	})
}
