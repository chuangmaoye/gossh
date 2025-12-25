package mysftp

import (
	"fmt"
	"gossh/pkg/core"
	"io"
	"os"

	"github.com/gosuri/uiprogress"
	"github.com/pkg/sftp"
)

// Upload 文件上传到远程服务器
func (s *MySftp) Upload(localFile, remoteFile core.File, callBack core.Callback) error {
	if err := s.connect(); err != nil {
		return err
	}

	srcFile, err := os.Open(localFile.Path())
	if err != nil {
		fmt.Println(1)
		return err
	}
	defer srcFile.Close()

	err = s.Client.MkdirAll(remoteFile.Dir)
	if err != nil {
		return err
	}

	var dstFile *sftp.File

	info, err := srcFile.Stat()
	if err != nil {
		fmt.Println(2)
		return err
	}
	bar1 := uiprogress.AddBar(100).AppendCompleted()
	bar1.PrependFunc(func(b *uiprogress.Bar) string {
		return fmt.Sprintf("%s %s", "uploading", srcFile.Name())
	})
	fileSize := info.Size()

	remoteFileSize, err := s.GetFileSize(remoteFile)
	if remoteFileSize == fileSize && !remoteFile.Replace {
		bar1.Set(100)
		return nil
	}

	if err != nil || remoteFile.Replace {

		dstFile, err = s.Client.Create(remoteFile.Path())
		remoteFileSize = 0
		if err != nil {

			return err
		}
	} else {
		dstFile, err = s.Client.OpenFile(remoteFile.Path(), os.O_APPEND|os.O_WRONLY)
		if err != nil {
			// fmt.Println(4)
			return err
		}
		// 设置断点位置
		_, err = srcFile.Seek(remoteFileSize, io.SeekStart)
		if err != nil {
			// fmt.Println(5)
			return err
		}
	}

	defer dstFile.Close()

	progressWriter := &ProgressWriter{
		Writer:       dstFile,
		Total:        fileSize,
		Desc:         "uploading-" + remoteFile.ServerName,
		Bar:          bar1,
		TransferSize: remoteFileSize,
	}
	_, err = io.Copy(progressWriter, srcFile)
	if err != nil {
		fmt.Println("写入文件时发生错误:", err)
		return err
	}
	return nil
}
