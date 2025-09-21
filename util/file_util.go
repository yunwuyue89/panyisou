package util

import (
	"os"
	"path/filepath"
)

// FileExists 检查文件是否存在
func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

// DirExists 检查目录是否存在
func DirExists(dirname string) bool {
	info, err := os.Stat(dirname)
	return !os.IsNotExist(err) && info.IsDir()
}

// EnsureDir 确保目录存在，如果不存在则创建
func EnsureDir(dirname string) error {
	if DirExists(dirname) {
		return nil
	}
	return os.MkdirAll(dirname, 0755)
}

// GetFileSize 获取文件大小
func GetFileSize(filename string) (int64, error) {
	info, err := os.Stat(filename)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// WriteFile 写入文件，如果目录不存在则创建
func WriteFile(filename string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(filename)
	if err := EnsureDir(dir); err != nil {
		return err
	}
	return os.WriteFile(filename, data, perm)
}

// ReadFile 读取文件
func ReadFile(filename string) ([]byte, error) {
	return os.ReadFile(filename)
}

// RemoveFile 删除文件
func RemoveFile(filename string) error {
	return os.Remove(filename)
}

// GetFileExt 获取文件扩展名
func GetFileExt(filename string) string {
	return filepath.Ext(filename)
}

// GetFileName 获取文件名（不含扩展名）
func GetFileName(filename string) string {
	base := filepath.Base(filename)
	ext := filepath.Ext(base)
	return base[:len(base)-len(ext)]
}
