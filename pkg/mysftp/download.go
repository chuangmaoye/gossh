package mysftp

import (
	"fmt"
	"gossh/pkg/core"
	"io"
	"os"
	"path"

	"github.com/gosuri/uiprogress"
)

// Download 文件下载到本地
func (s *MySftp) Download(remoteFile, localFile core.File, callBack core.Callback) error {
	if err := s.connect(); err != nil {
		return err
	}

	srcFile, err := s.Client.Open(path.Join(remoteFile.Dir, remoteFile.Name))
	if err != nil {
		return err
	}
	defer srcFile.Close()

	err = os.MkdirAll(localFile.Dir, os.ModePerm)
	if err != nil {
		return err
	}

	var dstFile *os.File

	info, err := srcFile.Stat()
	if err != nil {
		return err
	}
	bar1 := uiprogress.AddBar(100).AppendCompleted()
	bar1.PrependFunc(func(b *uiprogress.Bar) string {
		return fmt.Sprintf("%s %s", "downloading", srcFile.Name())
	})
	fileSize := info.Size()
	localFileSize, err := s.GetFileSize(localFile)
	if localFileSize == fileSize && !localFile.Replace {
		bar1.Set(100)
		return nil
	}
	if err != nil || localFile.Replace {
		dstFile, err = os.Create(localFile.Path())
		if err != nil {
			return err
		}
		localFileSize = 0
	} else {
		dstFile, err = os.OpenFile(localFile.Path(), os.O_APPEND|os.O_WRONLY, os.ModeAppend)
		if err != nil {
			return err
		}
		// 设置断点位置
		_, err = srcFile.Seek(localFileSize, io.SeekStart)
		if err != nil {
			return err
		}
	}

	defer dstFile.Close()

	progressWriter := &ProgressWriter{
		Writer:       dstFile,
		Total:        fileSize,
		Desc:         "downloading",
		Bar:          bar1,
		TransferSize: localFileSize,
	}
	_, err = io.Copy(progressWriter, srcFile)
	if err != nil {
		fmt.Println("写入文件时发生错误:", err)
		return err
	}
	return nil
}
