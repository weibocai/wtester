package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/wtester/pkg/config"
	"github.com/wtester/pkg/logger"
	"github.com/wtester/pkg/result"
)

var configFile string

var rootCmd = &cobra.Command{
	Use:   "wtester",
	Short: "测试工具",
	Long:  "测试工具",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&configFile, "cf", "", "config flies")
}

func initConfig() {
	viper.SetConfigType("yaml")
	if configFile == "" {
		configFile = filepath.Join("configs", ".cobra.yaml")
	}
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		panic("config flies does not exist")
	}

	viper.SetConfigFile(configFile)
	viper.SetConfigFile(configFile)
	viper.AutomaticEnv()
	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Sprintf("Error reading config file: %v", err))
	}
	if err := viper.Unmarshal(&config.WTesterConfig); err != nil {
		panic(fmt.Sprintf("Error parsing config file: %v", err))
	}

	logger.InitLogger()
	result.InitResult()
	logger.Logger.Info("初始化完成")
}
