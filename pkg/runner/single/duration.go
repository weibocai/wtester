package single

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/wtester/pkg/config"
	"github.com/wtester/pkg/library"
	"github.com/wtester/pkg/runner"
	"github.com/wtester/pkg/storge"
	"github.com/wtester/pkg/task"
)

// DurationRunner 定义时间执行器
type DurationRunner struct {
	runner.BaseRunner
}

func (sr *DurationRunner) Daemon(ctx context.Context, cf context.CancelFunc) {
	ticker := time.NewTicker(1 * time.Second)
	config.Logger.Info("starting daemon")
	i := 1
Loop:
	for {
		select {
		case <-ctx.Done():
			break Loop
		case <-ticker.C:
			i += 1
			if i >= sr.Stage.GetDuration() {
				break Loop
			}
			_ = sr.Bar.Add(1)
		}
	}
	_ = sr.Bar.Close()
	ticker.Stop()
	cf()
	config.Logger.Info("stopping daemon")
}

func (sr *DurationRunner) RunnerRandom(ctx context.Context, wg *sync.WaitGroup, rc chan *storge.Result) {
	defer wg.Done()
	// 初始化随机种子
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	tk := sr.Stage.GetTasks()
	tasks := make(map[string]task.Task, len(tk))
	for _, t := range tk {
		tasks[(*t).GetName()] = *t
	}
	taskWeight := sr.Stage.GetTaskWeight()
	totalWeight := sr.Stage.GetTotalWeight()
	for {
		select {
		case <-ctx.Done(): // 接收中止信号，如果中止，则停止执行
			rc <- nil
			return
		default:
			name := library.SelectWeightedRandom(r, totalWeight, taskWeight)
			if name == "Unknown" {
				continue
			}
			t := tasks[name]
			sr.WriteResult(t, rand.Intn(t.GetParamsLength()), rc)
		}
	}
}

func (sr *DurationRunner) RunnerWeight(ctx context.Context, wg *sync.WaitGroup, rc chan *storge.Result) {
	defer wg.Done()
	// 设置随机种子
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	tasks := sr.Stage.GetTasks()
	taskWeight := sr.Stage.GetTaskWeight()
	// 执行到了第几个任务
	index, pIndex, lTask, updateIndex := 0, 0, len(tasks), true
	for {
		select {
		case <-ctx.Done():
			rc <- nil
			return
		default:
			if index >= lTask {
				index = 0
			}
			t := *tasks[index]
			if updateIndex {
				if r.Intn(100) > taskWeight[t.GetName()] {
					index++
					continue
				}
				updateIndex = false
			}
			sr.WriteResult(t, rand.Intn(t.GetParamsLength()), rc)
			pIndex++
			if pIndex >= t.GetParamsLength() {
				pIndex = 0
				index++
				updateIndex = true
			}
		}
	}
}

func (sr *DurationRunner) RunnerOrder(ctx context.Context, wg *sync.WaitGroup, rc chan *storge.Result) {
	defer wg.Done()
	tasks := sr.Stage.GetTasks()
	index, pIndex, lTask := 0, 0, len(tasks)
	for {
		select {
		case <-ctx.Done():
			rc <- nil
			return
		default:
			if index >= lTask {
				index = 0
			}
			t := *tasks[index]
			sr.WriteResult(t, rand.Intn(t.GetParamsLength()), rc)
			pIndex++
			if pIndex >= t.GetParamsLength() {
				pIndex = 0
				index++
			}
		}
	}
}
