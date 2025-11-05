package library

import (
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"time"
)

// ParseDuration 解析时间：1s,1m,1h,1d,1w,1y;
func ParseDuration(duration string) (*time.Duration, string, uint8, int, error) {
	// 时间单位
	unit := duration[len(duration)-1]
	// 对应的值
	value, err := strconv.Atoi(duration[:len(duration)-1])
	if value < 0 {
		return nil, "", 0, 0, fmt.Errorf("%s 必须大于等于0", duration)
	}
	if err != nil {
		return nil, "", 0, 0, err
	}
	// 将大于小时的时间，转成以h为单位的值
	switch unit {
	case 's':
	case 'm':
		value = value * 60
	case 'h':
		value = value * 60 * 60
	case 'd':
		value = value * 60 * 60 * 24
		duration = fmt.Sprintf("%ds", value)
	case 'w':
		value = value * 60 * 60 * 24 * 7
		duration = fmt.Sprintf("%ds", value)
	case 'y':
		value = value * 60 * 60 * 24 * 365
		duration = fmt.Sprintf("%ds", value)
	default:
		return nil, "", 0, 0, errors.New("%s 只支持：s,m,h,d,w,y 单位")
	}
	step, err := time.ParseDuration(duration)
	return &step, duration, unit, value, err
}

type WeightedItem struct{}

// SelectWeightedRandom 获取加权随机数
func SelectWeightedRandom(r *rand.Rand, totalWeight int, items map[string]int) string {
	value := r.Intn(totalWeight)
	cSum := 0
	for k, v := range items {
		if cSum+v > value {
			return k
		}
		cSum += v
	}
	return "Unknown"
}
