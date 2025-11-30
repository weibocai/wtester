package runner

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/wtester/pkg/result"
	"github.com/wtester/pkg/stage"
)

// 定义轮次配置执行器
type SingleLoopRunner struct {
	Stage    *stage.Stage `json:"stage"`
	DoneChan chan struct{}
	ccCount  int // 当前并发个数
	loop     int // 执行轮次记录器
	lock     sync.Mutex
	once     sync.Once // 添加 once
}

// 更新执行轮次
func (sr *SingleLoopRunner) updateLoop(ctx context.Context) bool {
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

// Done 测试结束执行的操作
func (sr *SingleLoopRunner) Done(cf context.CancelFunc, wg *sync.WaitGroup) {
	defer wg.Done()
	<-sr.DoneChan // 关闭通道后，这里会立即返回
	cf()
}

func (sr *SingleLoopRunner) GetStage() *stage.Stage {
	return sr.Stage
}

func (sr *SingleLoopRunner) GetCcCount() int {
	return sr.ccCount
}

func (sr *SingleLoopRunner) PlusCcCount() {
	sr.ccCount++
}

func (sr *SingleLoopRunner) RunnerRandom(ctx context.Context, wg *sync.WaitGroup, rc chan *result.Result) {
	wg.Done()
}

func (sr *SingleLoopRunner) RunnerWeight(ctx context.Context, wg *sync.WaitGroup, rc chan *result.Result) {
	defer wg.Done()
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
				writeResult(sr, t, i, rc)
			}
		}
	}
}

func (sr *SingleLoopRunner) RunnerOrder(ctx context.Context, wg *sync.WaitGroup, rc chan *result.Result) {
	defer wg.Done()
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
				writeResult(sr, t, i, rc)
			}
		}
	}
}
