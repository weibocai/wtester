package storge

import "time"

/*集群状态记录*/

type Recode struct {
	ID        uint      `gorm:"primaryKey;<-:false"`
	Version   string    `json:"version" gorm:"type:varchar(18)"`  // 版本
	Event     string    `json:"event" gorm:"type:varchar(255)"`   // 记录事件
	Message   string    `json:"message" gorm:"type:varchar(255)"` // 记录事件
	Name      string    `json:"name" gorm:"type:varchar(255)"`    // 事件主体
	CreatedAt time.Time `json:"created_at" gorm:"type:datetime"`  // 记录时间
}

func (Recode) TableName() string {
	return "recode"
}

type ActorStatus struct {
	Id                 uint      `gorm:"primaryKey;<-:false"`
	Version            string    `json:"version" gorm:"type:varchar(18)"` // 测试版本
	Name               string    `json:"name" gorm:"type:varchar(36)"`    // pod 对应的名称
	NumberOfConcurrent int       `json:"ramp_up" gorm:"type:int"`         // 当前启动进程个数
	CreatedAt          time.Time `json:"created_at" gorm:"type:datetime"` // 记录时间
}
