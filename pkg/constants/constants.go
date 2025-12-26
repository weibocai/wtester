package constants

const (
	ResultFilePath    string = "results" // 项目主路径的相对路径，结果存储路径
	ResultFileBufSize int    = 64 * 1024 // 结果文件存储缓冲区
	LogPath           string = "logs"    // 项目主路径的相对路径
	ConfigPath        string = "configs" // 项目主路径的相对路径
)

// StorageType 结果存储类型
type StorageType string

const (
	FileStorageType  StorageType = "file"       // 本地存储
	MysqlStorageType StorageType = "mysql"      // mysql
	AllStorageType   StorageType = "file,mysql" // 声明支持的保存方式
)
