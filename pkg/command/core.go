package command

import (
	"errors"
	"fmt"
	"gossh/pkg/core"
	"gossh/pkg/e"
	"gossh/pkg/mysftp"
	"gossh/pkg/util"
	"path"
	"strconv"
	"strings"
)

func (c Cmd) Ls() (string, error) {
	return c.Exec.Run(c.GetCmdStr("ls -lh", c.WorkDir))
}

func (c Cmd) IsDirectory(pathDir string) bool {
	dir := false
	if c.Location == e.FILE_LOCATION_LOCAL {
		dir = util.IsDir(pathDir)
	} else {
		dir, _ = c.Client.IsDir(pathDir)
	}
	return dir
}

func (c Cmd) GetFiles(spath string) ([]core.File, error) {
	var files []core.File
	workDir := c.WorkDir
	if spath != "" {
		workDir = spath
	}
	resultStr, err := c.Exec.Run(c.GetCmdStr("ls -al|awk '{print $1\" \"$2\" \"$3\" \"$4\" \"$5\" \"$6\" \"$7\" \"$8\" \"$9}'", workDir))
	if err != nil {
		return files, err
	}
	fileRows := strings.Split(resultStr, "\n")
	for _, row := range fileRows {

		if row != "" && !strings.Contains(row, "total") {
			file := core.File{}
			file.Source = row
			fileSplit := strings.Split(row, " ")
			file.Name = fileSplit[8]
			if file.Name == "." || file.Name == ".." {
				continue
			}
			ftype := string(fileSplit[0][0])
			file.Dir = workDir
			file.Author = fileSplit[0]
			file.Type = ftype

			file.Location = c.Location
			size, err := strconv.ParseInt(fileSplit[4], 10, 64)
			if err != nil {
				size = 0
			}
			file.Size = size
			file.Hommization = util.HommizationSize(size)
			file.Ext = path.Ext(fileSplit[8])
			// if ftype == "d" && file.Name != "." && file.Name != ".." {
			// 	subFiles, err := c.GetFiles(file.Path())
			// 	// fmt.Println(subFiles, err, resultStr)
			// 	if err == nil {
			// 		file.Files = subFiles
			// 	}
			// }
			files = append(files, file)
		}

	}
	return files, nil
}

func (c Cmd) RunFile(path string) (string, error) {
	return c.Exec.RunFile(path)
}

func (c Cmd) GetCmdStr(cmd, workDir string) string {
	if workDir != "" {
		cmd = fmt.Sprintf("cd %s&&%s", workDir, cmd)
	} else {
		dir := strings.Trim(c.GetPwd(), "\n")
		if dir != "" {
			c.WorkDir = dir
			cmd = fmt.Sprintf("cd %s&&%s", dir, cmd)
		}
	}
	return cmd
}

func (c Cmd) GetPwd() string {
	result, err := c.Run("pwd")
	if err != nil {
		return ""
	}
	return result
}

func (c Cmd) Run(cmd string) (string, error) {
	return c.Exec.Run(cmd)
}

func (c Cmd) Close() error {
	if !util.IsNil(c.Client) && c.Client != nil {
		c.Client.Close()
	}
	return c.Exec.Close()
}

func (c Cmd) Cp(src, dst core.File) (string, error) {
	if c.Location == e.FILE_LOCATION_LOCAL {
		if src.Location == e.FILE_LOCATION_LOCAL && dst.Location == e.FILE_LOCATION_LOCAL {
			if src.ExistFile() {
				err := dst.CreateDir()
				if err != nil {
					return "", err
				}
				err = dst.RmFile()
				if err != nil {
					return "", err
				}
				result, err := c.Run(fmt.Sprintf("cp %s %s", src.Path(), dst.Path()))
				if err != nil {
					return result, err
				}

			} else {
				return "", errors.New("本地文件不存在")
			}
		} else {
			return "", errors.New("本地拷贝需都是本地路径")
		}
	} else if c.Location == e.FILE_LOCATION_REMOTE {
		if util.IsNil(c.Client) || c.Client == nil {
			if c.RemoteType == e.E_SERVER_SFTP {
				c.Client = &mysftp.MySftp{SSH: c.SSHClient.Client}
			}
			// } else if c.RemoteType == e.E_SERVER_FTP {
			// 	// ftp模式 暂无
			// }
		}
		if src.Location == e.FILE_LOCATION_LOCAL {
			err := c.Client.Upload(src, dst, func(v ...interface{}) interface{} { return v })
			if err != nil {
				return "", err
			}
		} else if dst.Location == e.FILE_LOCATION_LOCAL {
			err := c.Client.Download(src, dst, func(v ...interface{}) interface{} { return v })
			if err != nil {
				return "", err
			}
		} else if dst.Location == e.FILE_LOCATION_LOCAL && src.Location == e.FILE_LOCATION_LOCAL {
			if src.ExistFile() {
				err := dst.CreateDir()
				if err != nil {
					return "", err
				}
				err = dst.RmFile()
				if err != nil {
					return "", err
				}
				result, err := c.Run(fmt.Sprintf("\\cp %s %s", src.Path(), dst.Path()))
				if err != nil {
					return result, err
				}

			} else {
				return "", errors.New("本地文件不存在")
			}
		} else if src.Location == e.FILE_LOCATION_REMOTE && dst.Location == e.FILE_LOCATION_REMOTE {
			result, err := c.Run(fmt.Sprintf("\\cp %s %s", src.Path(), dst.Path()))
			if err != nil {
				return result, err
			}
		} else {
			return "", errors.New("请检查文件位置")
		}
	} else {
		return "", errors.New("未知操作")
	}
	return "", nil
}
