package task

import (
	"fmt"
	"strings"
	"time"

	"github.com/wtester/pkg/logger"
	"github.com/wtester/pkg/request"
	"github.com/wtester/pkg/task/parameter"
	"go.yaml.in/yaml/v3"
)

type tempPR struct {
	Type  string    `yaml:"type"`
	Param yaml.Node `yaml:"param"`
}

// Request http 请求型任务
type HttpRequest struct {
	Name         string            `yaml:"name"`
	Description  string            `yaml:"description"`
	TempParams   []yaml.Node       `yaml:"params,omitempty"`
	TempResponse yaml.Node         `yaml:"response,omitempty"`
	Weight       int               `yaml:"weight"` // 只针对时间停止型任务有效
	Method       string            `yaml:"method"`
	Url          string            `yaml:"url"`
	Headers      map[string]string `yaml:"headers,omitempty"`

	params      []*parameter.Parameter `yaml:"-"` // 解析后的参数
	response    *parameter.Response    `yaml:"-"` // 解析后的返回内容
	paramLength int                    `yaml:"-"`
}

func (r *HttpRequest) GetClientPoolKey() string {
	return r.Url
}

func (r *HttpRequest) AddClientPool() error {
	if _, err := request.RegisterHttpClient(r.Url); err != nil {
		return err
	}
	return nil
}

func (r *HttpRequest) GetParamsLength() int {
	return r.paramLength
}

// GetParam 获取task指定轮次的参数
func (r *HttpRequest) getParam(index int) (map[string]any, error) {
	params := make(map[string]any)
	for _, param := range r.params {
		p, err := (*param).GetParam(index)
		if err != nil {
			logger.Logger.Error(fmt.Sprintf("task:%s 参数获取失败：%v", r.Name, err.Error()))
			return nil, err
		}
		for k, v := range p {
			params[k] = v
		}
	}
	return params, nil
}

func (r *HttpRequest) GetDescription() string {
	return r.Description
}

func (r *HttpRequest) Doc() string {
	var doc strings.Builder
	doc.WriteString(fmt.Sprintf("\n----------- 任务%s（http请求型任务）----------- \n", r.GetName()))
	doc.WriteString(fmt.Sprintf("%s\n", r.GetDescription()))
	doc.WriteString(fmt.Sprintf("请求地址: %s\n", r.Url))
	doc.WriteString(fmt.Sprintf("请求方法: %s\n", r.Method))
	doc.WriteString(fmt.Sprintf("请求头: %s\n", r.Headers))
	for i := range r.params {
		if *r.params[i] == nil {
			continue
		}
		doc.WriteString(fmt.Sprintf("参数%d: \n%s", i+1, (*r.params[i]).Doc()))
	}
	doc.WriteString(fmt.Sprintf("\n----------- 任务%s（http请求型任务）----------- \n", r.GetName()))
	return doc.String()
}

func (r *HttpRequest) GetWeight() int {
	if r.Weight > 0 {
		return r.Weight
	}
	return 1
}

func (r *HttpRequest) Init() error {
	length := 1
	isInit := false
	for _, param := range r.params {
		if err := (*param).Init(); err != nil {
			return fmt.Errorf("%s 参数解析异常：%s", r.GetName(), err.Error())
		}
		if (*param).GetType() != "MapParam" {
			if !isInit {
				length = (*param).GetLength()
				isInit = true
			}
			logger.Logger.Info(fmt.Sprintf("%d %d, %s", length, (*param).GetLength(), (*param).GetType()))
			if length != (*param).GetLength() {
				return fmt.Errorf("task: 如果提供多种序列函数，序列的长度必须一致")
			}
		}
	}
	r.paramLength = length
	if r.response == nil {
		return nil
	}
	if err := (*r.response).Init(); err != nil {
		return fmt.Errorf("%s 参数解析异常：%s", r.GetName(), err.Error())
	}
	baseType := map[string]int8{"int": 1, "MapParam": 1, "string": 1, "float": 1}
	if _, ok := baseType[(*r.response).GetType()]; ok {
		return nil
	}
	if (*r.response).GetLength() != length {
		return fmt.Errorf("task: 如果需要校验返回值，且定义了一系列的返回值，则必须和请求参数的长度一致")
	}
	return nil
}
func (r *HttpRequest) GetName() string {
	return r.Name
}

func (r *HttpRequest) SetParam() error {
	if len(r.TempParams) == 0 {
		return nil
	}
	para := make([]*parameter.Parameter, len(r.TempParams))
	for i := range r.TempParams {
		var tp tempPR
		if err := r.TempParams[i].Decode(&tp); err != nil {
			return err
		}
		if tp.Type == "file" {
			// 单独解析文件型参数
			var path string
			if err := tp.Param.Decode(&path); err != nil {
				return err
			}
			logger.Logger.Info(path)
			if pp, err := parameter.GetParam(tp.Type, path); err != nil {
				return err
			} else {
				para[i] = &pp
			}
		} else {
			// 其他类型参数解析
			pp, err := parameter.GetParam(tp.Type)
			if err != nil {
				return err
			}
			if err = tp.Param.Decode(pp); err != nil {
				return err
			}
			para[i] = &pp
		}
	}
	r.params = para
	return nil
}

func (r *HttpRequest) SetResponse() error {
	if r.TempResponse.Content == nil {
		return nil
	}
	var tp tempPR
	if err := r.TempResponse.Decode(&tp); err != nil {
		return err
	}
	// 单独解析文件型参数
	if tp.Type == "file" {
		var path string
		if err := tp.Param.Decode(&path); err != nil {
			return err
		}
		if pp, err := parameter.GetResponse(tp.Type, path); err != nil {
			return err
		} else {
			r.response = &pp
		}
	} else {
		// 其他类型参数解析
		pp, err := parameter.GetResponse(tp.Type)
		if err != nil {
			return err
		}
		if err = tp.Param.Decode(pp); err != nil {
			return err
		}
		r.response = &pp
	}
	return nil
}

func (r *HttpRequest) compare(index int, res string) bool {
	if r.response == nil {
		return true
	}
	if equal, err := (*r.response).Verify(index, res); err != nil {
		logger.Logger.Error(fmt.Sprintf("预计返回信息对比失败：%s, %d, %s", r.GetName(), index, err.Error()))
		return false
	} else {
		return equal
	}
}

func (r *HttpRequest) Runner(index int) (bool, int64, error) {
	now := time.Now()
	isEqual := false
	if para, err := r.getParam(index); err != nil {
		return false, 0, err
	} else {
		if res, err := request.HttpRequest(r.Method, r.Url, para); err != nil {
			return false, 0, err
		} else {
			isEqual = r.compare(index, res)
		}
	}

	duration := time.Since(now).Milliseconds()
	return isEqual, duration, nil
}
