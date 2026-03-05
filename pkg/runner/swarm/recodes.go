package swarm

import (
	"fmt"

	"github.com/wtester/pkg/config"
	"github.com/wtester/pkg/storge"
)

func write2recodes(name, msg string, err error) {
	switch msg {
	case "node_register":
		msg = fmt.Sprintf("%s 注册到服务端，等待执行任务下发", name)
	case "actor_offline":
		msg = fmt.Sprintf("执行端%s：下线", name)
	case "actor_error":
		msg = fmt.Sprintf("执行端%s：消息发送失败：%v", name, err)
	case "had_no_actor":
		msg = fmt.Sprintf("执行端%s：未注册，请检查集群状态是否正确", name)
	case "had_actor":
	default:
		if err != nil {
			msg = fmt.Sprintf("%s %s %v", name, msg, err)
		} else {
			msg = fmt.Sprintf("%s %s", name, msg)
		}
	}
	if err != nil {
		config.Logger.Error(msg)
	} else {
		config.Logger.Info(msg)
	}

	r := storge.Recode{
		Name: name, Message: msg, Version: config.WTesterConfig.Version,
	}
	storge.GormDB.Create(&r)
}
