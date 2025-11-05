package stage

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/wtester/pkg/library"
	"github.com/wtester/pkg/logger"
	"github.com/wtester/pkg/task"
	"go.yaml.in/yaml/v3"
)

type tempTask struct {
	Type string    `yaml:"type"`
	Task yaml.Node `yaml:"task"`
}

type ExecutionModeType string

const (
	ExecutionModeOrder  = ExecutionModeType("order")  // 顺序执行
	ExecutionModeWeight = ExecutionModeType("weight") // 顺序执行，带有权重
	ExecutionModeRandom = ExecutionModeType("random") // 随机执行
)

type ExecutionOrderType string

const (
	ExecutionOrderTypeLoop     = ExecutionOrderType("loop")
	ExecutionOrderTypeDuration = ExecutionOrderType("duration")
)

// Stage 测试阶段
type Stage struct {
	Name               string            `yaml:"name"`
	NumberOfConcurrent int               `yaml:"number_concurrent"` // 最大并发个数
	Sampling           int               `yaml:"sampling"`          // 采样频率，单位为秒
	TempTasks          []yaml.Node       `yaml:"tasks"`             // 临时解析用的task模板
	RampUp             int               `yaml:"ramp_up"`           // 每秒启动的并发数
	Duration           string            `yaml:"duration"`          // 执行时长
	Loop               int               `yaml:"loop"`              // 执行轮次，和Duration互斥，优先使用执行轮次判断执行是否结束
	ExecutionMode      ExecutionModeType `yaml:"execution_mode"`    // 任务执行方式：order（根据指定的顺序执行）、random（随机顺序）

	tasks              []*task.Task       `yaml:"-"`
	duration           int                // 秒
	taskWeight         map[string]int     // 任务对应的权重
	totalWeight        int                // 权重统计
	sampling           time.Duration      // 采样频率
	executionOrderType ExecutionOrderType // 执行模式
}

func (s *Stage) Doc() string {
	var doc strings.Builder
	doc.WriteString(fmt.Sprintf("\n#################### 阶段%s ####################\n", s.Name))
	doc.WriteString(fmt.Sprintf("1. 采样周期为：%d\n", s.Sampling))
	doc.WriteString(fmt.Sprintf("1. 包含%d个任务\n", len(s.tasks)))
	doc.WriteString(fmt.Sprintf("2. 最大启动并发：%d\n", s.NumberOfConcurrent))
	doc.WriteString(fmt.Sprintf("3. 每秒启动并发：%d\n", s.RampUp))
	doc.WriteString(fmt.Sprintf("4. 任务执行顺序：%s\n", s.ExecutionMode))
	doc.WriteString("5. 任务执行权重：\n")
	for k, v := range s.taskWeight {
		doc.WriteString(fmt.Sprintf("  %s权重占比为 %d\n", k, v))
	}

	if s.Duration != "" {
		doc.WriteString(fmt.Sprintf("5. 执行总时长为：%s\n", s.Duration))
	} else {
		doc.WriteString(fmt.Sprintf("5. 执行轮次为：%d\n", s.Loop))
	}

	for i := range s.tasks {
		doc.WriteString((*s.tasks[i]).Doc())
	}
	doc.WriteString(fmt.Sprintf("\n#################### 阶段%s ####################\n", s.Name))
	return doc.String()
}

func (s *Stage) setTaskWeight() (bool, error) {
	weight := 0
	lWeight := -1
	isDiff := false
	for i := range s.tasks {
		w := (*s.tasks[i]).GetWeight()
		if w <= 0 {
			return false, fmt.Errorf("Stage %s - Task %s 权重参数配置异常：%d", s.Name, (*s.tasks[i]).GetName(), w)
		}
		if lWeight < 0 {
			lWeight = w
		} else {
			if lWeight != w {
				isDiff = true
			}
		}
		weight += w
	}
	weightF := float64(weight)
	s.totalWeight = 0
	s.taskWeight = make(map[string]int, len(s.tasks))
	for i := range s.tasks {
		_weight := int(float64((*s.tasks[i]).GetWeight()) / weightF * 100)
		s.taskWeight[(*s.tasks[i]).GetName()] = _weight
		s.totalWeight += _weight
	}
	if !isDiff {
		return false, nil
	}
	return true, nil
}

// Init 初始化阶段
func (s *Stage) Init() error {
	sampling, _, _, _, err := library.ParseDuration(fmt.Sprintf("%ds", s.Sampling))
	if err != nil {
		return err
	}
	s.sampling = *sampling
	if s.RampUp <= 0 {
		return errors.New("ramp_up must be a positive number")
	}
	if s.NumberOfConcurrent <= 0 {
		return errors.New("user number must be a positive number")
	}
	if s.ExecutionMode == "" {
		s.ExecutionMode = ExecutionModeOrder
	}
	if s.ExecutionMode != ExecutionModeOrder && s.ExecutionMode != ExecutionModeRandom {
		return fmt.Errorf("stage:%s execution_mode not supported", s.ExecutionMode)
	}

	if (s.Duration != "" && s.Loop != 0) || (s.Duration == "" && s.Loop == 0) {
		return fmt.Errorf("stage:%s 时间控制和轮次控制只能（必须）选择一种，Duration=%s Loop=%d", s.ExecutionMode, s.Duration, s.Loop)
	}

	if s.Loop > 0 {
		if s.ExecutionMode != ExecutionModeOrder {
			return fmt.Errorf("stage: %s 指定任务执行的轮次，任务执行顺序必须为：order", s.Name)
		}
		s.executionOrderType = ExecutionOrderTypeLoop
	} else {
		_, _, _, duration, err := library.ParseDuration(s.Duration)
		if err != nil {
			return fmt.Errorf("stage: %s Duration=%s 解析异常: %s", s.Name, s.Duration, err.Error())
		}
		s.duration = duration
		s.executionOrderType = ExecutionOrderTypeDuration
	}

	// 配置任务
	if s.TempTasks == nil {
		return fmt.Errorf("%s 必须指定任务", s.Name)

	}
	if err = s.SetTasks(); err != nil {
		return err
	}
	if len(s.tasks) == 0 {
		return fmt.Errorf("%s 未指定执行任务", s.Name)
	}

	taskName := make(map[string]bool)
	for i := range s.tasks {
		if _, ok := taskName[(*s.tasks[i]).GetName()]; ok {
			return fmt.Errorf("stage:%s task already exists", (*s.tasks[i]).GetName())
		}
		taskName[(*s.tasks[i]).GetName()] = true
		if err = (*s.tasks[i]).Init(); err != nil {
			return err
		}
	}
	// 设置任务的执行权重
	if isDiff, err := s.setTaskWeight(); err != nil {
		return err
	} else {
		// 如果是顺序执行，且权重有效，将阶段设置为权重执行模式
		if isDiff && s.ExecutionMode == ExecutionModeOrder {
			s.ExecutionMode = ExecutionModeWeight
		}
	}
	logger.Logger.Info(s.Doc())
	return nil
}

// SetTasks 设置任务
func (s *Stage) SetTasks() error {
	tasks := make([]*task.Task, len(s.TempTasks))
	for i := range s.TempTasks {
		var tt tempTask
		if err := s.TempTasks[i].Decode(&tt); err != nil {
			return fmt.Errorf("任务解析失败：%d, %v", i, err)
		}
		if t, err := task.GetTask(tt.Type); err != nil {
			return fmt.Errorf("任务解析失败：%d, %v", i, err)
		} else {
			if err = tt.Task.Decode(t); err != nil {
				return fmt.Errorf("任务解析失败：%d, %v", i, err)
			}
			if err = t.SetResponse(); err != nil {
				return fmt.Errorf("任务解析失败，参数解析失败：%d, %v", i, err)
			}
			if err = t.SetParam(); err != nil {
				return fmt.Errorf("任务解析失败，参数解析失败：%d, %v", i, err)
			}
			tasks[i] = &t
		}
	}
	s.tasks = tasks
	return nil
}

func (s *Stage) GetSampling() time.Duration {
	return s.sampling
}

func (s *Stage) GetDuration() int {
	return s.duration
}

func (s *Stage) GetTasks() []*task.Task {
	return s.tasks
}

func (s *Stage) GetTaskWeight() map[string]int {
	return s.taskWeight
}

func (s *Stage) GetTotalWeight() int {
	return s.totalWeight
}

func (s *Stage) GetExecutionOrderType() ExecutionOrderType {
	return s.executionOrderType
}
