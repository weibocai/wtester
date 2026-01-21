package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/wtester/pkg/config"
)

var version string
var configFile string

var rootCmd = &cobra.Command{
	Use:   "wtester",
	Short: "测试工具",
	Long:  "测试工具",
}

func Execute() {
	err := rootCmd.Execute()
	if config.Logger != nil {
		_ = config.Logger.Sync()
	}
	if err != nil {
		panic(err)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&configFile, "cf", "", "配置文件路径")
	rootCmd.PersistentFlags().StringVar(&version, "v", "", "测试数据保存版本")
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
	config.WTesterConfig.SetVersion(version)
	// 加载日志配置
	config.InitLogger()
}
