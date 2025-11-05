package parameter

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// MapParam 直接指定任务的参数，也可以直接为序列参数添加额外的参数
type MapParam map[string]interface{}

func (mp *MapParam) Init() error {
	return nil
}
func (mp *MapParam) Doc() string {
	return fmt.Sprintf("  %s 型参数: %v\n", mp.GetType(), mp)
}
func (mp *MapParam) GetLength() int {
	return 1
}
func (mp *MapParam) GetType() string {
	return "MapParam"
}
func (mp *MapParam) GetParam(index int) (map[string]interface{}, error) {
	return *mp, nil
}
func (mp *MapParam) Verify(index int, res string) (bool, error) {
	rr := make(map[string]interface{})
	err := json.Unmarshal([]byte(res), &rr)
	if err != nil {
		return false, err
	}
	if vRes, err := mp.GetParam(index); err != nil {
		return false, err
	} else {
		return reflect.DeepEqual(vRes, rr), nil
	}
}
