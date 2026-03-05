package runner

import (
	"context"
	"fmt"
	"time"

	"github.com/schollz/progressbar/v3"
	"github.com/wtester/pkg/stage"
	"github.com/wtester/pkg/storge"
	"github.com/wtester/pkg/task"
)

type Runner interface {
	PlusCcCount(count int)                                       // 修改当前并发个数
	GetCcCount() int                                             // 获取当前并发个数
	GetStage() *stage.Stage                                      // 获取执行阶段
	GetRandSource() int64                                        // 获取随机种子
	GetRemainingCoroutinesNumber() int                           // 获取当前需要启动的线程数
	Daemon(ctx context.Context, cf context.CancelFunc)           // 守护进程，执行中止的条件
	WriteResult(t task.Task, pIndex int, rc chan *storge.Result) // 结果写入
	RunnerOrder(ctx context.Context, rc chan *storge.Result)     // 顺序执行任务
	RunnerRandom(ctx context.Context, rc chan *storge.Result)    // 随机执行任务
	RunnerWeight(ctx context.Context, rc chan *storge.Result)    // 带有权重的顺序执行，且任务之间的权重不一致，有概率不执行
}

type SwarmRunner interface {
	PlusRealCoroutinesNumber(count int)
}

type BaseRunner struct {
	Stage                *stage.Stage `json:"stage"`
	Bar                  *progressbar.ProgressBar
	CoroutinesNumber     int   // 当前并发个数
	RealCoroutinesNumber int   // 实际启动并发个数
	RandSource           int64 // 随机种子，确保节点的一致性
}

func (sr *BaseRunner) GetRemainingCoroutinesNumber() int {
	lrp := sr.Stage.NumberOfConcurrent - sr.GetCcCount()
	if lrp > sr.Stage.RampUp {
		lrp = sr.Stage.RampUp
	}
	return lrp
}

func (sr *BaseRunner) WriteResult(t task.Task, pIndex int, rc chan *storge.Result) {
	isEqual, executionTime, err := t.Runner(pIndex)
	exp := ""
	if err != nil {
		exp = fmt.Sprintf("%v", err.Error())
	}
	res := &storge.Result{
		Stage: sr.GetStage().Name, Task: t.GetName(), IsSuccess: err == nil, ExecutionTime: executionTime,
		Datetime: time.Now(), IsEqual: isEqual, CC: sr.GetCcCount(), Exception: exp,
	}
	rc <- res
}
func (sr *BaseRunner) GetStage() *stage.Stage {
	return sr.Stage
}

func (sr *BaseRunner) GetRandSource() int64 {
	return sr.RandSource
}

func (sr *BaseRunner) GetCcCount() int {
	return sr.CoroutinesNumber
}

func (sr *BaseRunner) PlusCcCount(count int) {
	sr.CoroutinesNumber += count

}

func (sr *BaseRunner) PlusRealCoroutinesNumber(count int) {
	sr.RealCoroutinesNumber += count
}

func (sr *BaseRunner) RunnerRandom(ctx context.Context, rc chan *storge.Result) {
}
