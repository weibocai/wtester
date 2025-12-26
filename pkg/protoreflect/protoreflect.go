package protoreflect

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/protoparse"
	"github.com/wtester/pkg/logger"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"google.golang.org/protobuf/types/descriptorpb"
)

// getMessageDescriptorCode 写入定义的数据类型，用于传参
func getMessageDescriptorCode(msg []*desc.MessageDescriptor, writer *bufio.Writer) error {
	for _, m := range msg {
		logger.Logger.Info(m.GetName())
		_, err := writer.WriteString(fmt.Sprintf("func (x *%s) SetPara(para map[string]any){\n", m.GetName()))
		c := cases.Title(language.English)
		for _, field := range m.GetFields() {
			switch field.GetType() {
			case descriptorpb.FieldDescriptorProto_TYPE_STRING:
				_, err = writer.WriteString(fmt.Sprintf("    x.%s = para[\"%s\"].(string)\n", c.String(field.GetName()), field.GetName()))
			case descriptorpb.FieldDescriptorProto_TYPE_INT32, descriptorpb.FieldDescriptorProto_TYPE_SFIXED32, descriptorpb.FieldDescriptorProto_TYPE_SINT32:
				_, err = writer.WriteString(fmt.Sprintf("    x.%s = para[\"%s\"].(int32)\n", c.String(field.GetName()), field.GetName()))
			case descriptorpb.FieldDescriptorProto_TYPE_UINT32, descriptorpb.FieldDescriptorProto_TYPE_FIXED32:
				_, err = writer.WriteString(fmt.Sprintf("    x.%s = para[\"%s\"].(uint32)\n", c.String(field.GetName()), field.GetName()))
			case descriptorpb.FieldDescriptorProto_TYPE_INT64, descriptorpb.FieldDescriptorProto_TYPE_SFIXED64, descriptorpb.FieldDescriptorProto_TYPE_SINT64:
				_, err = writer.WriteString(fmt.Sprintf("    x.%s = para[\"%s\"].(int64)\n", c.String(field.GetName()), field.GetName()))
			case descriptorpb.FieldDescriptorProto_TYPE_UINT64, descriptorpb.FieldDescriptorProto_TYPE_FIXED64:
				_, err = writer.WriteString(fmt.Sprintf("    x.%s = para[\"%s\"].(uint64)\n", c.String(field.GetName()), field.GetName()))
			case descriptorpb.FieldDescriptorProto_TYPE_DOUBLE:
				_, err = writer.WriteString(fmt.Sprintf("    x.%s = para[\"%s\"].(float64)\n", c.String(field.GetName()), field.GetName()))
			case descriptorpb.FieldDescriptorProto_TYPE_FLOAT:
				_, err = writer.WriteString(fmt.Sprintf("    x.%s = para[\"%s\"].(float32)\n", c.String(field.GetName()), field.GetName()))
			case descriptorpb.FieldDescriptorProto_TYPE_BOOL:
				_, err = writer.WriteString(fmt.Sprintf("    x.%s = para[\"%s\"].(bool)\n", c.String(field.GetName()), field.GetName()))
			case descriptorpb.FieldDescriptorProto_TYPE_BYTES:
				_, err = writer.WriteString(fmt.Sprintf("    x.%s = para[\"%s\"].([]byte)\n", c.String(field.GetName()), field.GetName()))
			default:
				return fmt.Errorf("暂不支持 %s", field.GetType())
			}
			if err != nil {
				return err
			}
		}

		_, err = writer.WriteString("}\n")
		if err != nil {
			return err
		}
	}
	return nil
}

func ParseProtoFile(dirname, savePath string) error {
	var protoFiles []string

	fSys := os.DirFS(dirname)
	// 自动递归处理文件
	err := fs.WalkDir(fSys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".proto") {
			protoFiles = append(protoFiles, path)
		}
		return nil
	})
	if err != nil {
		return err
	}
	allImportPaths := []string{dirname}
	parser := &protoparse.Parser{
		ImportPaths: allImportPaths, IncludeSourceCodeInfo: true,
	}
	fileDescriptors, err := parser.ParseFiles(protoFiles...)
	if err != nil {
		return fmt.Errorf("解析失败: %v", err)
	}
	messages := make([]*desc.MessageDescriptor, 0)
	services := make([]*desc.ServiceDescriptor, 0)
	for _, fd := range fileDescriptors {
		services = append(services, fd.GetServices()...)
		messages = append(messages, fd.GetMessageTypes()...)
	}

	file, err := os.OpenFile(savePath, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)
	writer := bufio.NewWriterSize(file, 2048)

	// 写入头信息
	if _, err = writer.WriteString(`package proto
import (
	"context"
	"time"

	"google.golang.org/grpc"
)

// GrpcServerCallFunc grpc 客户端调用函数定义
type GrpcServerCallFunc func(conn *grpc.ClientConn, requestData map[string]any) (string, error)

var grpcServiceFactories map[string]GrpcServerCallFunc

// RegisterGrpcServiceFactories 客户端注册
func RegisterGrpcServiceFactories(service string, f GrpcServerCallFunc) {
	if _, ok := grpcServiceFactories[service]; ok {
		return
	}
	grpcServiceFactories[service] = f
}

// GetGrpcServiceFactories 获取请求客户端
func GetGrpcServiceFactories(service string) GrpcServerCallFunc {
	if f, ok := grpcServiceFactories[service]; ok {
		return f
	}
	return nil
}
`); err != nil {
		return err
	}
	_, err = writer.WriteString(`func init() {
	grpcServiceFactories = make(map[string]GrpcServerCallFunc)
`)
	for _, svc := range services {
		for _, method := range svc.GetMethods() {
			_, err = writer.WriteString(fmt.Sprintf("\tRegisterGrpcServiceFactories(\"%s.%s\", %s%s)\n", svc.GetName(), method.GetName(), svc.GetName(), method.GetName()))
		}
	}
	_, err = writer.WriteString("}\n")

	// 写入消息定义
	if err = getMessageDescriptorCode(messages, writer); err != nil {
		return err
	}

	// 写入客户端请求函数
	for _, svc := range services {
		for _, method := range svc.GetMethods() {
			c := fmt.Sprintf(`
func %s%s(conn *grpc.ClientConn, requestData map[string]any) (string, error) {
	c := New%sClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	para := %s{}
	para.SetPara(requestData)
	r, err := c.%s(ctx, &para)
	if err != nil {
		return "", err
	}
	return r.String(), nil
}

`, svc.GetName(), method.GetName(), svc.GetName(), method.GetInputType().GetName(), method.GetName())
			_, _ = writer.WriteString(c)
		}
	}
	if err = writer.Flush(); err != nil {
		return err
	}
	return nil
}
