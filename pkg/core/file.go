package core

import (
	"os"
	"path"
)

type File struct {
	Dir         string //目录地址
	Name        string // 文件名
	Author      string //文件权限
	Size        int64  //文件大小 b
	Type        string // 文件类型
	Source      string // 原始形式
	Ext         string // 扩展名
	Hommization string // 人性化展示大小
	Location    int    // 存在位置 e.FILE_LOCATION_LOCAL e.FILE_LOCATION_REMOTE
	ServerName  string // 服务名
	Replace     bool   // 是否替换
	// Files       []File  // 子目录
}

func (f *File) Path() string {
	return path.Join(f.Dir, f.Name)
}

func (f *File) ExistFile() bool {
	return f.exist(f.Path())
}

func (f *File) ExistDir() bool {
	return f.exist(f.Dir)
}

func (f *File) CreateDir() error {
	if f.ExistDir() {
		return nil
	}
	return os.MkdirAll(f.Dir, os.ModePerm)
}

func (f *File) RmFile() error {
	if !f.ExistFile() {
		return nil
	}
	return os.Remove(f.Path())
}

func (f *File) exist(path string) bool {
	_, err := os.Stat(path) //os.Stat获取文件信息
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}

// 文件大小
func (f *File) GetSize() int64 {
	info, _ := os.Stat(f.Path())

	size := info.Size()
	return size

}
