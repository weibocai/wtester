package parameter

import "fmt"

type Parameter interface {
	Init() error                                        // 初始化参数：解析+校验
	Doc() string                                        // 参数说明
	GetLength() int                                     // 获取参数的长度
	GetType() string                                    // 获取参数类型
	GetParam(index int) (map[string]interface{}, error) // 通过索引，获取指定的参数
}

// 参数定义工厂
type ParamFactory func(para ...string) (Parameter, error)

var paramFactories = make(map[string]ParamFactory)

func RegisterParam(name string, factory ParamFactory) bool {
	if _, ok := paramFactories[name]; ok {
		return false
	}
	paramFactories[name] = factory
	return false
}

func GetParam(name string, para ...string) (Parameter, error) {
	if factory, ok := paramFactories[name]; ok {
		return factory(para...)
	}
	return nil, fmt.Errorf("参数类型不存在：%s", name)
}

func init() {
	RegisterParam("map", func(para ...string) (Parameter, error) {
		return &MapParam{}, nil
	})
	RegisterParam("file", FileParam2RealParam)
	RegisterParam("sequence_map", func(para ...string) (Parameter, error) { return &SequenceMapParam{}, nil })
	RegisterParam("sequence", func(para ...string) (Parameter, error) { return &SequenceParam{}, nil })
}
