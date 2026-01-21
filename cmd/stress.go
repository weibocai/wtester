package cmd

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/wtester/pkg/config"
	"github.com/wtester/pkg/request"
	"github.com/wtester/pkg/runner"
	"github.com/wtester/pkg/stage"
	"github.com/wtester/pkg/storge"
	"go.yaml.in/yaml/v3"
)

var testPlan string

var stressCmd = &cobra.Command{
	Use:   "stress",
	Short: "压力测试工具",
	Long:  "压力测试工具",
	Run: func(cmd *cobra.Command, args []string) {
		err := execute()
		if err != nil {
			panic(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(stressCmd)
	stressCmd.Flags().StringVarP(&testPlan, "plan", "p", "", "测试计划（默认：$PWD/config/wtester.yaml）")
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
		testPlan = filepath.Join(pwd, "configs", "wtester.yaml")
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
	runner.Signal(stageList)
	config.Logger.Info("本次测试结束")
	// 关闭需要关闭
	closeStress()
	return nil
}
