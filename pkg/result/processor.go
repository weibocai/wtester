package result

import (
	"context"
	"fmt"
	"time"

	"github.com/wtester/pkg/constants"
)

type Processor interface {
	Init() error                                       // 初始化以及参数校验
	Process(ctx context.Context, results chan *Result) // 结果处理过程
	Done() error                                       // 收尾
}

type ProcessorFactory func(sampling time.Duration) Processor

var processorFactories = map[constants.StorageType]ProcessorFactory{}

func RegisterProcessor(name constants.StorageType, factory ProcessorFactory) bool {
	if _, ok := processorFactories[name]; ok {
		return false
	}
	processorFactories[name] = factory
	return true
}

func GetProcessor(name constants.StorageType, sampling time.Duration) (Processor, error) {
	if factory, ok := processorFactories[name]; ok {
		return factory(sampling), nil
	}
	return nil, fmt.Errorf("此类型的%s结果处理器不存在", name)
}

func init() {
	RegisterProcessor(constants.FileStorageType, func(sampling time.Duration) Processor {
		return &FileProcessor{Sampling: sampling}
	})
	RegisterProcessor(constants.MysqlStorageType, func(sampling time.Duration) Processor {
		return &MysqlProcessor{Sampling: sampling}
	})
}
