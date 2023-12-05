package mysftp

import (
	"gossh/pkg/core"
	"gossh/pkg/e"
	"gossh/pkg/util"
	"io"
	"io/fs"
	"os"
	"path"

	"github.com/gosuri/uiprogress"
	"github.com/pkg/sftp"
	"github.com/tj/go-spin"
	"golang.org/x/crypto/ssh"
)

// var bufSize = 1024 * 1000
var bufSize = 1024 * 100

var SP = spin.New()

type MySftp struct {
	Client *sftp.Client
	SSH    *ssh.Client
}

type ProgressWriter struct {
	Writer       io.Writer
	Total        int64
	Written      int64
	Desc         string
	Bar          *uiprogress.Bar
	TransferSize int64
}

// 链接sftp
func (s *MySftp) connect() error {
	var sftpClient *sftp.Client
	var err error
	if util.IsNil(s.SSH) {
		return &core.SftpError{ErrorInfo: "ssh为初始化!"}
	}
	if util.IsNil(s.Client) {
		if sftpClient, err = sftp.NewClient(s.SSH); err != nil {
			return err
		}
		s.Client = sftpClient
	}
	return nil
}

func (s *MySftp) Close() error {
	return s.Client.Close()
}

func (pw *ProgressWriter) Write(p []byte) (n int, err error) {
	n, err = pw.Writer.Write(p)
	pw.Written += int64(n)
	progress := (float64(pw.Written) + float64(pw.TransferSize)) / float64(pw.Total) * 100
	pw.Bar.Set(int(progress))
	return n, err
}

// 获取文件列表
func (s *MySftp) GetFileList(pathStr string) ([]core.File, error) {
	if err := s.connect(); err != nil {
		return nil, err
	}
	files, err := s.Client.ReadDir(pathStr)
	if err != nil {
		return nil, err
	}
	var fileList []core.File
	for _, file := range files {
		if file.IsDir() {
			if file.Name() == "." || file.Name() == ".." {
				continue
			}
		}
		fileInfo := core.File{}
		if pathStr[len(pathStr)-1:] != "/" {
			pathStr = pathStr + "/"
		}
		fileInfo.Dir = pathStr
		fileInfo.Name = file.Name()
		fileInfo.Size = file.Size()
		fileInfo.Author = file.Mode().String()
		fileInfo.Ext = path.Ext(file.Name())
		fileInfo.Type = file.Mode().Type().String()[:1]
		fileInfo.Hommization = util.HommizationSize(fileInfo.Size)
		fileInfo.Location = e.FILE_LOCATION_REMOTE

		fileList = append(fileList, fileInfo)
	}
	return fileList, nil
}

// 判断是否文件夹
func (s *MySftp) IsDir(pathStr string) (bool, error) {
	if err := s.connect(); err != nil {
		return false, err
	}
	srcFile, err := s.Client.Open(pathStr)
	if err != nil {
		return false, err
	}
	defer srcFile.Close()
	info, err := srcFile.Stat()
	if err != nil {
		return false, err
	}
	return info.IsDir(), nil
}

// 获取文件大小
func (s *MySftp) GetFileSize(file core.File) (int64, error) {
	var (
		err      error
		fileInfo fs.FileInfo
	)
	if file.Location == e.FILE_LOCATION_LOCAL {
		fileInfo, err = os.Stat(file.Path())
	} else {
		if err = s.connect(); err != nil {
			return 0, err
		}
		fileInfo, err = s.Client.Stat(file.Path())
	}
	if err != nil {
		return 0, err
	}
	return fileInfo.Size(), nil
}
