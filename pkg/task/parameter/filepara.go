package parameter

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
)

func fileParam2RealParam(para ...string) (Parameter, Response, error) {
	if len(para) != 1 {
		return nil, nil, fmt.Errorf("文件型参数解析异常：请提供有效参数")
	}
	path := para[0]
	if _, err := os.Stat(path); err != nil {
		return nil, nil, fmt.Errorf("文件型参数解析失败：%s, %s", path, err.Error())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("文件型参数解析失败：%s, %s", path, err.Error())
	}
	var param interface{}
	if err := json.Unmarshal(data, &param); err != nil {
		return nil, nil, fmt.Errorf("文件型参数解析失败：%s, %s", path, err.Error())
	}
	switch v := param.(type) {
	case map[string]interface{}:
		vv := MapParam(v)
		return &vv, &vv, nil
	case []map[string]interface{}:
		vv := SequenceMapParam(v)
		return &vv, &vv, nil
	case []interface{}:
		// json 不会直接将[{"e": "an"}, {"f": "immortal"}] 解析成[]map[string]interface{}
		smp := make(SequenceMapParam, len(v))
		for i, vv := range v {
			if s, ok := vv.(map[string]interface{}); ok {
				smp[i] = s
			} else {
				t := reflect.TypeOf(vv)
				return nil, nil, fmt.Errorf("%s 文件格式不正确，只支持json：支持map、[map]；不支持：%s", path, t.String())
			}
		}
		return &smp, &smp, nil
	default:
		t := reflect.TypeOf(v)
		return nil, nil, fmt.Errorf("%s 文件格式不正确，只支持json：支持map、[map]；不支持：%s", path, t.String())
	}
}

// FileParam2RealParam 文件型参数，不直接使用，而是转换成对应格式的参数类型，只支持json格式的文件：支持map、[]map[string]string
func FileParam2RealParam(para ...string) (Parameter, error) {
	p, _, err := fileParam2RealParam(para...)
	return p, err
}

// FileParam2RealResponse 文件型参数，不直接使用，而是转换成对应格式的参数类型，只支持json格式的文件：支持map、[]map[string]string
func FileParam2RealResponse(para ...string) (Response, error) {
	_, r, err := fileParam2RealParam(para...)
	return r, err
}
