package cmd

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/wtester/pkg/config"
	"github.com/wtester/pkg/request"
	"github.com/wtester/pkg/runner/single"
	"github.com/wtester/pkg/runner/swarm"
	"github.com/wtester/pkg/stage"
	"github.com/wtester/pkg/storge"
	"go.yaml.in/yaml/v3"
)

var testPlan string
var isSwarm bool
var swarmRole string
var swarmNodeCount int
var actorName string
var swarmRoles = []string{"director", "actor"}

var stressCmd = &cobra.Command{
	Use:   "stress",
	Short: "压力测试工具",
	Long:  "压力测试工具",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		mode, err := cmd.Flags().GetString("role")
		if err != nil {
			return err
		}

		// 检查 mode 是否在合法范围内
		found := false
		for _, v := range swarmRoles {
			if v == mode {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("invalid mode %q, must be one of: %s", mode, strings.Join(swarmRoles, ", "))
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		err := execute()
		if err != nil {
			panic(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(stressCmd)
	stressCmd.Flags().StringVarP(&testPlan, "plan", "p", "", "测试计划（默认：$PWD/config/theforce.yaml）")
	stressCmd.Flags().StringVarP(&swarmRole, "role", "r", "director", "服务端：director；执行端：actor")
	stressCmd.Flags().StringVarP(&actorName, "name", "n", "", "执行端的名称")
	stressCmd.Flags().BoolVarP(&isSwarm, "swarm", "s", false, "是否使用集群运行方式")
	stressCmd.Flags().IntVarP(&swarmNodeCount, "count", "c", 1, "集群执行节点的个数，默认为1")
}

// 执行完之后，需要做一些清理的工作，在这里完成
func closeStress() {
	if storge.GormDB != nil {
		if sqlDB, err := storge.GormDB.DB(); err == nil {
			_ = sqlDB.Close()
		}
	}
	request.CloseGrpcReflectionClient()
	request.CloseGrpcClient()
	request.CloseHtpClient()
}

func execute() error {
	// 加载结果保存配置
	config.Logger.Info("初始化结果储存")
	if err := storge.InitStorge(); err != nil {
		return err
	}
	config.Logger.Info("开始加载测试方案")
	if testPlan == "" {
		pwd, _ := os.Getwd()
		testPlan = filepath.Join(pwd, "configs", "theforce.yaml")
	}
	yamlStage, err := os.ReadFile(testPlan)
	if err != nil {
		config.Logger.Error(err.Error())
		return err
	}
	var stageList []*stage.Stage
	if err = yaml.Unmarshal(yamlStage, &stageList); err != nil {
		config.Logger.Error(err.Error())
		return err
	}
	for index := range stageList {
		if err = stageList[index].Init(); err != nil {
			return err
		}
	}

	if isSwarm {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			sig := <-sigCh
			config.Logger.Info(fmt.Sprintf("%v 外部主动中止程序", sig))
			os.Exit(0)
		}()
		if swarmRole == "director" {
			//go func() {
			//	// 启动调试端口，通常是 6060
			//	http.ListenAndServe("localhost:6060", nil)
			//}()
			swarm.DirectorActuator(swarmNodeCount, sigCh, stageList)
		} else {
			go func() {
				// 启动调试端口，通常是 6060
				http.ListenAndServe("localhost:6061", nil)
			}()
			swarm.ActorActuator(actorName, "127.0.0.1", "127.0.0.1:50052", 50053, sigCh, stageList)
		}
	} else {
		single.Single(stageList)
	}

	config.Logger.Info("本次测试结束")
	// 关闭需要关闭
	closeStress()
	return nil
}
