package remotestream

import "gossh/pkg/core"

type IRemoteStream interface {
	Download(remoteFile, localFile core.File, callBack core.Callback) error
	Upload(localFile, remoteFile core.File, callBack core.Callback) error
	Close() error
	GetFileList(pathStr string) ([]core.File, error)
	IsDir(pathStr string) (bool, error)
	GetFileSize(file core.File) (int64, error)
}
