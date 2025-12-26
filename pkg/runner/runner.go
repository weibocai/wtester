package runner

import (
	"context"
	"sync"

	"github.com/wtester/pkg/result"
	"github.com/wtester/pkg/stage"
)

type Runner interface {
	PlusCcCount(count int)                                                        // 修改当前并发个数
	GetCcCount() int                                                              // 获取当前并发个数
	GetStage() *stage.Stage                                                       // 获取执行阶段
	Daemon(cf context.CancelFunc, wg *sync.WaitGroup)                             // 守护进程，执行中止的条件
	RunnerOrder(ctx context.Context, wg *sync.WaitGroup, rc chan *result.Result)  //顺序执行任务
	RunnerRandom(ctx context.Context, wg *sync.WaitGroup, rc chan *result.Result) // 随机执行任务
	RunnerWeight(ctx context.Context, wg *sync.WaitGroup, rc chan *result.Result) // 带有权重的顺序执行，且任务之间的权重不一致，有概率不执行
}
