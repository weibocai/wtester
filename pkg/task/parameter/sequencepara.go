package parameter

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/wtester/pkg/config"
	"github.com/wtester/pkg/library"
)

// Sequence 解析结构型参数：根据start-step-end获取参数；如果指定多个参数，参数个数要能对应
type Sequence struct {
	Start string `json:"start" yaml:"start"` // 支持整形和时间型字符串
	End   string `json:"end" yaml:"end"`     // 支持整形和时间型字符串
	Step  string `json:"step" yaml:"step"`   // 支持整形和时间型（支持：1s、1m、1d、1h、1y）字符串
	Type  string `json:"type" yaml:"type"`   // 指定类型：int、datetime
	Key   string `json:"key" yaml:"key"`     // 对应的参数名

	length int // 参数的长度

	// 整形参数解析后
	endInt   int
	startInt int
	stepInt  int

	// datetime 参数解析后
	endDatetime   *time.Time
	startDatetime *time.Time
	stepDatetime  *time.Duration
	unitDatetime  uint8
}

// 解析整形参数
func (s *Sequence) parseInt() error {
	start, err := strconv.Atoi(s.Start)
	if err != nil {
		config.Logger.Error(fmt.Sprint("参数解析：", s.Key, s.Start, err.Error()))
		return err
	}
	end, err := strconv.Atoi(s.End)
	if err != nil {
		config.Logger.Error(fmt.Sprint("参数解析：", s.Key, s.End, err.Error()))
		return err
	}
	step, err := strconv.Atoi(s.Step)
	if err != nil {
		config.Logger.Error(fmt.Sprint("参数解析：", s.Key, s.Step, err.Error()))
		return err
	}
	if step < 0 {
		return errors.New("step is less than 0")
	}

	if start > end {
		return fmt.Errorf("参数解析：%s start=%s 必须小于等于 end=%s", s.Key, s.Start, s.End)
	}
	if start+step > end {
		return fmt.Errorf("参数解析：%s start=%s + step=%s 必须小于等于 end=%s", s.Key, s.Start, s.Step, s.End)
	}

	s.endInt = end
	s.stepInt = step
	s.startInt = start
	if start == end {
		if step != 0 {
			return fmt.Errorf("参数解析：%s start=%s 等于 end=%s 的情况下，step=%s必须为0", s.Key, s.Start, s.End, s.Step)
		}
		s.length = 1
	} else {
		if step == 0 {
			return fmt.Errorf("参数解析：%s start=%s 大于 end=%s 的情况下，step=%s必须大于0", s.Key, s.Start, s.End, s.Step)
		}
		s.length = (end-start)/step + 1
	}
	return nil
}

// 解析datetime型参数
func (s *Sequence) parsingDatetime() error {
	stepDatetime, step, unitDatetime, value, err := library.ParseDuration(s.Step)
	if err != nil {
		config.Logger.Error(fmt.Sprint("参数解析：", s.Key, err))
		return err
	}
	end, start := time.Now(), time.Now()
	if start, err = time.Parse("2006-01-02 15:04:05", s.Start); err != nil {
		config.Logger.Error(fmt.Sprintf("参数解析: %s %s %s", s.Key, s.Start, err))
		return err
	}

	if end, err = time.Parse("2006-01-02 15:04:05", s.End); err != nil {
		config.Logger.Error(fmt.Sprintf("参数解析: %s %s %s", s.Key, s.End, err))
		return err
	}

	s.Step = step
	s.unitDatetime = unitDatetime
	s.stepDatetime = stepDatetime
	s.endDatetime = &end
	s.startDatetime = &start

	var start2end = start.Sub(end).Seconds()
	if start2end > 0 {
		return fmt.Errorf("参数解析: %s start=%s 必须晚于 end=%s", s.Key, s.Start, s.End)
	}
	if start2end == 0 {
		if value != 0 {
			return fmt.Errorf("参数解析: %s start=%s 等于 end=%s 的情况下，step=%s必须为0", s.Key, s.Start, s.End, s.Step)
		}
		s.length = 1
		return nil
	}
	if value < 0 {
		return fmt.Errorf("参数解析: %s start=%s 大于 end=%s 的情况下，step=%s必须大于0", s.Key, s.Start, s.End, s.Step)
	}
	duration := s.endDatetime.Sub(*s.startDatetime)
	s.length = int(duration.Seconds()) / value
	s.length = s.length + 1
	if s.length <= 0 {
		return fmt.Errorf("序列为空：%s start=%s, end=%s, step=%s", s.Key, s.Start, s.End, s.Step)
	}
	return nil
}

func (s *Sequence) Init() error {
	switch s.Type {
	case "int":
		err := s.parseInt()
		if err != nil {
			return err
		}
	case "datetime":
		err := s.parsingDatetime()
		if err != nil {
			return err
		}
	case "string":

	default:
		return errors.New("只支持：int、datetime")
	}
	return nil
}

func (s *Sequence) GetParam(index int) (string, interface{}) {
	if s.Type == "datetime" {
		return s.Key, s.startDatetime.Add(*s.stepDatetime * time.Duration(index))
	}
	return s.Key, s.startInt + s.stepInt*index
}

// SequenceParam 序列参数：可以按照指定的规则生成对应的参数序列，指定的参数必须一一对应
type SequenceParam struct {
	Sequences []*Sequence `json:"sequences" yaml:"sequences"`
	Length    int         `json:"length" yaml:"-"`
}

func (sp *SequenceParam) Doc() string {
	var doc strings.Builder

	doc.WriteString(fmt.Sprintf("  %s型参数\n", sp.GetType()))
	doc.WriteString(fmt.Sprintf("  总共包含：%d两个参数；包含%d个键\n", len(sp.Sequences), sp.GetLength()))
	for index, s := range sp.Sequences {
		doc.WriteString(fmt.Sprintf("  %d. key=%s：start=%s, step=%s, end=%s\n", index+1, s.Key, s.Start, s.Step, s.End))
	}
	return doc.String()
}

func (sp *SequenceParam) Init() error {
	length := 0
	keyMap := make(map[string]bool)
	for index, sequence := range sp.Sequences {
		if err := sequence.Init(); err != nil {
			return err
		}
		if index == 0 {
			length = sequence.length
		} else {
			if length != sequence.length {
				return fmt.Errorf("提供的参数，长度不对应，无法解析")
			}
		}
		if _, ok := keyMap[sequence.Key]; ok {
			return fmt.Errorf("提供的参数对应的名称重复，请确认后重试")
		}
		keyMap[sequence.Key] = true
	}
	sp.Length = length
	return nil
}

func (sp *SequenceParam) GetLength() int {
	return sp.Length
}

func (sp *SequenceParam) GetType() string {
	return "SequenceParam"
}

func (sp *SequenceParam) GetParam(index int) (map[string]interface{}, error) {
	if index < 0 || index >= sp.Length {
		return nil, errors.New("index is out of range")
	}
	params := make(map[string]interface{})
	for _, sequence := range sp.Sequences {
		k, v := sequence.GetParam(index)
		params[k] = v
	}
	return params, nil
}
func (sp *SequenceParam) Verify(index int, res string) (bool, error) {
	rr := make(map[string]interface{})
	err := json.Unmarshal([]byte(res), &rr)
	if err != nil {
		return false, err
	}
	if vRes, err := sp.GetParam(index); err != nil {
		return false, err
	} else {
		return reflect.DeepEqual(vRes, rr), nil
	}
}
