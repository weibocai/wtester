package parameter

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
)

// SequenceMapParam 每个元素作为依次任务的参数
type SequenceMapParam []map[string]interface{}

func (smp *SequenceMapParam) Doc() string {
	return fmt.Sprintf("  %s 型参数，总共包含：%d个参数\n", smp.GetType(), smp.GetLength())
}

func (smp *SequenceMapParam) Init() error {
	return nil
}

func (smp *SequenceMapParam) GetLength() int {
	return len(*smp)
}

func (smp *SequenceMapParam) GetType() string {
	return "SequenceMapParam"
}

func (smp *SequenceMapParam) GetParam(index int) (map[string]interface{}, error) {
	if index < 0 || index >= smp.GetLength() {
		return nil, errors.New("index is out of range")
	}
	return (*smp)[index], nil
}
func (smp *SequenceMapParam) Verify(index int, res string) (bool, error) {
	rr := make(map[string]interface{})
	err := json.Unmarshal([]byte(res), &rr)
	if err != nil {
		return false, err
	}
	if vRes, err := smp.GetParam(index); err != nil {
		return false, err
	} else {
		return reflect.DeepEqual(vRes, rr), nil
	}
}
