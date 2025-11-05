package runner

import (
	"context"
	"sync"

	"github.com/wtester/pkg/result"
	"github.com/wtester/pkg/stage"
)

type Runner interface {
	PlusCcCount()
	GetCcCount() int
	GetStage() *stage.Stage
	Done(cf context.CancelFunc, wg *sync.WaitGroup)
	RunnerOrder(ctx context.Context, wg *sync.WaitGroup, rc chan *result.Result)  //顺序执行任务
	RunnerRandom(ctx context.Context, wg *sync.WaitGroup, rc chan *result.Result) // 随机执行任务
	RunnerWeight(ctx context.Context, wg *sync.WaitGroup, rc chan *result.Result) // 带有权重的顺序执行，且任务之间的权重不一致，有概率不执行
}
