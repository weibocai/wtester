package runner

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/k0kubun/go-ansi"
	"github.com/schollz/progressbar/v3"
	"github.com/wtester/pkg/library"
	"github.com/wtester/pkg/stage"
	"github.com/wtester/pkg/storge"
	"github.com/wtester/pkg/task"
)

// SingleDurationRunner 定义时间执行器
type SingleDurationRunner struct {
	Stage   *stage.Stage `json:"stage"`
	ccCount int          // 当前并发个数
}

func (sr *SingleDurationRunner) Daemon(cf context.CancelFunc, wg *sync.WaitGroup) {
	defer wg.Done()
	bar := progressbar.NewOptions(sr.Stage.GetDuration(),
		progressbar.OptionSetWriter(ansi.NewAnsiStdout()), //you should install "github.com/k0kubun/go-ansi"
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
	for i := 0; i < sr.Stage.GetDuration(); i++ {
		_ = bar.Add(1)
		time.Sleep(time.Second * 1)
	}
	cf()
	_ = bar.Close()
}

func (sr *SingleDurationRunner) GetStage() *stage.Stage {
	return sr.Stage
}

func (sr *SingleDurationRunner) GetCcCount() int {
	return sr.ccCount
}

func (sr *SingleDurationRunner) PlusCcCount(count int) {
	sr.ccCount += count
}

func (sr *SingleDurationRunner) RunnerRandom(ctx context.Context, wg *sync.WaitGroup, rc chan *storge.Result) {
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
			writeResult(sr, t, rand.Intn(t.GetParamsLength()), rc)
		}
	}
}

func (sr *SingleDurationRunner) RunnerWeight(ctx context.Context, wg *sync.WaitGroup, rc chan *storge.Result) {
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
			writeResult(sr, t, rand.Intn(t.GetParamsLength()), rc)
			pIndex++
			if pIndex >= t.GetParamsLength() {
				pIndex = 0
				index++
				updateIndex = true
			}
		}
	}
}

func (sr *SingleDurationRunner) RunnerOrder(ctx context.Context, wg *sync.WaitGroup, rc chan *storge.Result) {
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
			writeResult(sr, t, rand.Intn(t.GetParamsLength()), rc)
			pIndex++
			if pIndex >= t.GetParamsLength() {
				pIndex = 0
				index++
			}
		}
	}
}
