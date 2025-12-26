package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wtester/pkg/logger"
	"github.com/wtester/pkg/protoreflect"
)

var protoPath string
var needGo bool
var needClient bool

var protoCmd = &cobra.Command{
	Use:   "proto",
	Short: "grpc proto 文件解析以及编译",
	Long:  "grpc proto 文件解析以及编译；只支持proto3、文件结构简单布局；如果比较复杂，建议自定义请求文件，请参考proto/proto.go",
	Run: func(cmd *cobra.Command, args []string) {
		pwd, _ := os.Getwd()
		if needGo {
			if protoPath == "" {
				protoPath = filepath.Join(pwd, "proto")
			}
			saveProtoPath := filepath.Join(pwd, "proto")
			matches := make([]string, 0)
			err := filepath.Walk(protoPath, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if !info.IsDir() && strings.HasSuffix(info.Name(), ".proto") {
					matches = append(matches, path)
				}
				return nil
			})
			if err != nil {
				logger.Logger.Error(fmt.Sprintf("未找到有效的proto文件： %s", protoPath))
				panic(err)
			}
			if len(matches) == 0 {
				logger.Logger.Error(fmt.Sprintf("未找到有效的proto文件： %s", protoPath))
				return
			}
			sArgs := []string{
				"protoc", "--proto_path=" + protoPath, "--go_out=" + saveProtoPath, "--go_opt=paths=source_relative", "--go-grpc_out=" + saveProtoPath, "--go-grpc_opt=paths=source_relative",
			}
			sArgs = append(sArgs, matches...)
			logger.Logger.Info("grpc 构建  " + strings.Join(sArgs, " "))
			sCmd := exec.Command(sArgs[0], sArgs[1:]...)
			if output, err := sCmd.CombinedOutput(); err != nil {
				fmt.Println(string(output), err)
				logger.Logger.Error(string(output))
				panic(err)
			}
		}
		if needClient {
			savePath := filepath.Join(pwd, "proto/proto.go")
			if err := protoreflect.ParseProtoFile(protoPath, savePath); err != nil {
				logger.Logger.Error("请求客户端构建失败")
				panic(err)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(protoCmd)
	protoCmd.Flags().StringVarP(&protoPath, "path", "p", "", "proto file path")
	protoCmd.Flags().BoolVarP(&needGo, "needGo", "g", true, "是否需要执行：protoc，如果不需要，请将对应的代码放到proto文件夹下")
	protoCmd.Flags().BoolVarP(&needClient, "needClient", "c", true, "是否需要生成客户端，如果不需要，请将对应的客户端放到proto文件夹下")
}