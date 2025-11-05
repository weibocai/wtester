package library

import (
	"os"
	"path/filepath"
)

// CreateFileIfNotExists 创建文件，如果文件不存在
func CreateFileIfNotExists(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		dirPath := filepath.Dir(path)
		if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
			return err
		}
		file, err := os.Create(path)
		if err != nil {
			return err
		}
		defer func(file *os.File) {
			err := file.Close()
			if err != nil {
			}
		}(file)
	}
	return nil
}

// CreateDirectoryIfNotExists 创建文件夹，如果文件夹不存在
func CreateDirectoryIfNotExists(dirPath string) error {
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
			return err
		}
	}
	return nil
}
