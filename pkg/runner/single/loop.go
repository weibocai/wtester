package single

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/wtester/pkg/runner"
	"github.com/wtester/pkg/storge"
)

// LoopRunner 定义轮次配置执行器
type LoopRunner struct {
	runner.BaseRunner

	DoneChan chan struct{}
	loop     int // 执行轮次记录器
	lock     sync.Mutex
	once     sync.Once // 添加 once
}

// 更新执行轮次
func (sr *LoopRunner) updateLoop(ctx context.Context) bool {
	sr.lock.Lock()
	defer sr.lock.Unlock()
	select {
	case <-ctx.Done():
		return false
	default:
	}
	sr.loop++
	if sr.loop >= sr.Stage.Loop {
		sr.once.Do(func() {
			close(sr.DoneChan) // 关闭通道而不是发送数据
		})
		return false
	}
	return true
}

// Daemon 测试结束执行的操作
func (sr *LoopRunner) Daemon(ctx context.Context, cf context.CancelFunc) {
	// defer wg.Done()
	// <-sr.DoneChan // 等待程序结束
	// cf()
}

func (sr *LoopRunner) RunnerRandom(ctx context.Context, rc chan *storge.Result) {
}

func (sr *LoopRunner) RunnerWeight(ctx context.Context, rc chan *storge.Result) {
	// 设置随机种子
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	tasks := sr.Stage.GetTasks()
	taskWeight := sr.Stage.GetTaskWeight()
	// 执行到了第几个任务
	lTask := len(tasks)
	for {
		if !sr.updateLoop(ctx) {
			return
		}
		for i := 0; i < lTask; i++ {
			t := *tasks[i]
			if r.Intn(100) > taskWeight[t.GetName()] {
				continue
			}
			for j := 0; j < t.GetParamsLength(); j++ {
				sr.WriteResult(t, i, rc)

			}
		}
	}
}

func (sr *LoopRunner) RunnerOrder(ctx context.Context, rc chan *storge.Result) {
	tasks := sr.Stage.GetTasks()
	lTask := len(tasks)
	for {
		// 等待结束信号
		if !sr.updateLoop(ctx) {
			return
		}
		for i := 0; i < lTask; i++ {
			t := *tasks[i]
			for j := 0; j < t.GetParamsLength(); j++ {
				sr.WriteResult(t, i, rc)
			}
		}
	}
}
