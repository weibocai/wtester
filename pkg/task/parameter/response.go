package parameter

import (
	"fmt"
)

type Response interface {
	Init() error                                // 初始化参数：解析+校验
	GetLength() int                             // 获取参数的长度
	GetType() string                            // 获取参数类型
	Verify(index int, res string) (bool, error) // 校验对应的返回结果
}

type ResponseFactory func(para ...string) (Response, error)

var responseFactories = map[string]ResponseFactory{}

func RegisterResponseFactor(name string, factory ResponseFactory) bool {
	if _, ok := responseFactories[name]; ok {
		return false
	}
	responseFactories[name] = factory
	return true
}

func GetResponse(name string, para ...string) (Response, error) {
	if factory, ok := responseFactories[name]; ok {
		return factory(para...)
	}
	return nil, fmt.Errorf("此返回类型定义不存在 %s", name)
}

func init() {
	RegisterResponseFactor("sequence_map", func(para ...string) (Response, error) { return &SequenceMapParam{}, nil })
	RegisterResponseFactor("map", func(para ...string) (Response, error) { return &MapParam{}, nil })
	RegisterResponseFactor("file", FileParam2RealResponse)
	RegisterResponseFactor("sequence", func(para ...string) (Response, error) { return &SequenceParam{}, nil })
	RegisterResponseFactor("string", func(para ...string) (Response, error) { return new(StrParam), nil })
	RegisterResponseFactor("int", func(para ...string) (Response, error) { return new(IntParam), nil })
	RegisterResponseFactor("float", func(para ...string) (Response, error) { return new(FloatParam), nil })
}

type StrParam string

func (s StrParam) Init() error {
	return nil
}
func (s StrParam) GetLength() int {
	return 1
}
func (s StrParam) GetType() string {
	return "string"
}
func (s StrParam) Verify(index int, res string) (bool, error) {
	ss := string(s)
	return res == ss, nil
}

func (s StrParam) ParseResponse() (Response, error) {
	return s, nil
}

type IntParam int

func (i IntParam) Init() error {
	return nil
}
func (i IntParam) GetLength() int {
	return 1
}
func (i IntParam) GetType() string {
	return "int"
}
func (i IntParam) Verify(index int, res string) (bool, error) {
	ss := string(rune(i))
	return res == ss, nil
}

func (i IntParam) ParseResponse() (Response, error) {
	return i, nil
}

type FloatParam int

func (f FloatParam) Init() error {
	return nil
}
func (f FloatParam) GetLength() int {
	return 1
}
func (f FloatParam) GetType() string {
	return "float"
}
func (f FloatParam) Verify(index int, res string) (bool, error) {
	ss := string(rune(f))
	return res == ss, nil
}

func (f FloatParam) ParseResponse() (Response, error) {
	return f, nil
}
