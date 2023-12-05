package command

import (
	"gossh/interface/remotestream"
	"gossh/interface/shell"
	"gossh/pkg/core"
	"gossh/pkg/e"
	"gossh/pkg/mysftp"
	"gossh/pkg/shell/local"
	"gossh/pkg/shell/remote"
)

type Cmd struct {
	Exec       shell.IExec
	WorkDir    string
	SSHClient  *core.Cli
	Location   int
	Client     remotestream.IRemoteStream
	RemoteType int // 远程文件管理的类型 e.E_SERVER_FTP e.E_SERVER_SFTP 默认sftp
}

func GetCmd(cmdType int, addr core.Address) (Cmd, error) {
	var cmd Cmd
	if cmdType == e.FILE_LOCATION_LOCAL {
		cmd = Cmd{Exec: &local.Local{}, Location: cmdType}
		return cmd, nil
	} else if cmdType == e.FILE_LOCATION_REMOTE {
		server := core.Server{
			Address: addr,
		}
		cli, err := server.Init()
		if err != nil {
			return cmd, err
		}

		cmd = Cmd{Exec: &remote.Remote{Cli: cli}, SSHClient: cli, Location: cmdType, Client: &mysftp.MySftp{SSH: cli.Client}}
		return cmd, nil
	}
	return cmd, nil
}
