package client

import (
	"errors"
	"fmt"
	"gossh/pkg/command"
	"gossh/pkg/core"
	"gossh/pkg/e"
	"gossh/pkg/util"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
	"sync"

	"github.com/gosuri/uiprogress"
)

var CpChan chan int
var wg sync.WaitGroup
var fileLocks sync.Map

func clearScreen() {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", "cls")
	} else {
		cmd = exec.Command("clear")
	}

	cmd.Stdout = os.Stdout
	cmd.Run()
}

func Copy(args ...string) error {
	src := args[0]
	// dst := args[1]
	srcs := strings.Split(src, ":")
	var srcAddr core.Address

	srcType := e.FILE_LOCATION_LOCAL

	var (
		cmd command.Cmd
		err error
	)
	if len(srcs) > 1 {
		if addr, ok := ServerAddresss[srcs[0]]; ok {
			srcAddr = addr
			src = srcs[1]
			srcType = e.FILE_LOCATION_REMOTE
		} else {
			return errors.New("服务不存在请重新输入")
		}
	}
	cmd, err = command.GetCmd(srcType, srcAddr)
	if err != nil {
		return err
	}

	srcObj := CopyDstObj{
		PahtDir:  src,
		Addr:     srcAddr,
		Location: srcType,
		Cmd:      cmd,
	}

	var dstObjs []CopyDstObj
	for _, v := range args[1:] {
		dstObj := CopyDstObj{}
		vs := strings.Split(v, ":")
		if len(vs) < 2 {
			dstObj.PahtDir = v
			dstObj.Location = e.FILE_LOCATION_LOCAL
			dstcmd, err := command.GetCmd(e.FILE_LOCATION_LOCAL, core.Address{})
			if err != nil {
				return err
			}
			dstObj.Cmd = dstcmd
			dstObjs = append(dstObjs, dstObj)
			continue
		}
		if addr, ok := ServerAddresss[vs[0]]; ok {
			dstObj.PahtDir = vs[1]
			dstObj.Addr = addr
			dstObj.Location = e.FILE_LOCATION_REMOTE
			dstcmd, err := command.GetCmd(e.FILE_LOCATION_REMOTE, addr)
			if err != nil {
				return err
			}
			dstObj.Cmd = dstcmd
			dstObjs = append(dstObjs, dstObj)
		} else {
			return errors.New("服务不存在请重新输入")
		}

	}

	srcFile := core.File{Dir: path.Dir(srcObj.PahtDir), Name: path.Base(srcObj.PahtDir), Location: srcObj.Location}
	var fileSize int64
	if srcType == e.FILE_LOCATION_LOCAL {
		fileSize = srcFile.GetSize()
	} else {
		fileSize, err = srcObj.Cmd.Client.GetFileSize(srcFile)
	}

	if err != nil {
		return err
	}
	srcFile.Size = fileSize
	tmpdir := fmt.Sprintf(".gosshtmp/%s", util.GenUUID())
	os.MkdirAll(tmpdir, os.ModePerm)
	defer os.RemoveAll(".gosshtmp")
	// fmt.Println(len(dstObjs))
	uiprogress.Start()
	for _, dst := range dstObjs {
		_cpFile(srcObj, dst, srcFile, tmpdir)
	}

	wg.Wait()
	uiprogress.Stop()
	return err
}

type CopyDstObj struct {
	Cmd      command.Cmd
	PahtDir  string
	Addr     core.Address
	Location int
}

func CopyDir(args ...string) error {
	src := args[0]
	// dst := args[1]
	srcs := strings.Split(src, ":")
	var srcAddr core.Address

	srcType := e.FILE_LOCATION_LOCAL

	var (
		cmd   command.Cmd
		files []core.File
		err   error
	)
	if len(srcs) > 1 {
		if addr, ok := ServerAddresss[srcs[0]]; ok {
			srcAddr = addr
			src = srcs[1]
			srcType = e.FILE_LOCATION_REMOTE
		} else {
			return errors.New("服务不存在请重新输入")
		}
	}
	cmd, err = command.GetCmd(srcType, srcAddr)
	if err != nil {
		return err
	}

	srcObj := CopyDstObj{
		PahtDir:  src,
		Addr:     srcAddr,
		Location: srcType,
		Cmd:      cmd,
	}

	if srcType == e.FILE_LOCATION_LOCAL {
		pwdStr := strings.Trim(cmd.GetPwd(), "\n")
		if !strings.Contains(srcObj.PahtDir, pwdStr) {
			srcObj.PahtDir = path.Join(pwdStr, srcObj.PahtDir)
			src = srcObj.PahtDir
			fmt.Println(srcObj.PahtDir, 1)
		}
	}

	var dstObjs []CopyDstObj
	for _, v := range args[1:] {
		dstObj := CopyDstObj{}
		vs := strings.Split(v, ":")
		if len(vs) < 2 {
			dstObj.PahtDir = v
			dstObj.Location = e.FILE_LOCATION_LOCAL
			dstcmd, err := command.GetCmd(e.FILE_LOCATION_LOCAL, core.Address{})
			if err != nil {
				return err
			}
			dstObj.Cmd = dstcmd
			dstObjs = append(dstObjs, dstObj)
			continue
		}
		if addr, ok := ServerAddresss[vs[0]]; ok {
			dstObj.PahtDir = vs[1]
			dstObj.Addr = addr
			dstObj.Location = e.FILE_LOCATION_REMOTE
			dstcmd, err := command.GetCmd(e.FILE_LOCATION_REMOTE, addr)
			if err != nil {
				return err
			}
			dstObj.Cmd = dstcmd
			dstObjs = append(dstObjs, dstObj)
		} else {
			return errors.New("服务不存在请重新输入")
		}

	}
	if !srcObj.Cmd.IsDirectory(src) {
		fmt.Println("不是文件夹")
		return errors.New("不是文件夹")
	}
	// lastDirName := util.GetLastDir(src)

	if srcType == e.FILE_LOCATION_REMOTE {
		files, err = cmd.Client.GetFileList(src)
		// fmt.Printf("%+v,%d", files, len(files))
		if err != nil {
			return err
		}
	} else {
		files, err = cmd.GetFiles(src)
	}
	tmpdir := fmt.Sprintf(".gosshtmp/%s", util.GenUUID())
	os.MkdirAll(tmpdir, os.ModePerm)
	defer os.RemoveAll(".gosshtmp")
	// fmt.Println(len(dstObjs))
	uiprogress.Start()
	for _, dst := range dstObjs {
		for _, file := range files {
			_cp(srcObj, dst, file, tmpdir)
		}
	}
	wg.Wait()
	uiprogress.Stop()
	return err
}

func _cp(src, dst CopyDstObj, file core.File, tmpdir string) {

	if file.Type == "d" {
		clearScreen()
		var files []core.File
		if src.Location == e.FILE_LOCATION_REMOTE {
			files, _ = src.Cmd.Client.GetFileList(file.Path())
		} else {
			files, _ = src.Cmd.GetFiles(file.Path())
			// fmt.Println(files, file.Path())
		}
		if len(files) == 0 {
			return
		}
		for _, v := range files {
			// cpChan <- 1
			_cp(src, dst, v, tmpdir)
		}
		return
	}
	CpChan <- 1
	wg.Add(1)
	RelativePath := strings.Replace(file.Dir, util.GetDir(src.PahtDir), "", 1)
	// fmt.Println(RelativePath)
	if dst.Location == e.FILE_LOCATION_REMOTE && src.Location == e.FILE_LOCATION_REMOTE {
		go func() {
			defer wg.Done()
			tmpFile := core.File{Dir: "./" + tmpdir + RelativePath, Name: file.Name, Location: e.FILE_LOCATION_LOCAL, ServerName: "localhost"}
			if !tmpFile.ExistFile() {
				lock, _ := fileLocks.LoadOrStore(tmpFile.Path(), &sync.Mutex{})
				mutex := lock.(*sync.Mutex)
				mutex.Lock()
				defer mutex.Unlock()
				file.ServerName = src.Cmd.SSHClient.IP
				src.Cmd.Cp(file, tmpFile)
			} else if file.Size != tmpFile.GetSize() {
				lock, _ := fileLocks.LoadOrStore(tmpFile.Path(), &sync.Mutex{})
				mutex := lock.(*sync.Mutex)
				mutex.Lock()
				defer mutex.Unlock()
			}
			dst.Cmd.Cp(tmpFile, core.File{Dir: dst.PahtDir + RelativePath, Name: file.Name, Location: dst.Location, ServerName: dst.Addr.IP, Replace: Replace})
			<-CpChan
		}()

	} else if dst.Location == e.FILE_LOCATION_REMOTE {
		go func() {
			defer wg.Done()
			dst.Cmd.Cp(file, core.File{Dir: dst.PahtDir + RelativePath, Name: file.Name, Location: dst.Location, ServerName: dst.Addr.IP, Replace: Replace})
			<-CpChan
		}()
	} else {
		// fmt.Println("到本地")
		go func() {
			defer wg.Done()
			// fmt.Println(dst.PahtDir + RelativePath)
			// fmt.Printf("%+v", file)
			_, err := src.Cmd.Cp(file, core.File{Dir: dst.PahtDir + RelativePath, Name: file.Name, Location: dst.Location, Replace: Replace})
			if err != nil {
				fmt.Println(err)
			}
			<-CpChan
		}()
	}
}

func _cpFile(src, dst CopyDstObj, srcFile core.File, tmpdir string) {
	CpChan <- 1
	wg.Add(1)
	// fmt.Println(RelativePath)
	if dst.Location == e.FILE_LOCATION_REMOTE && src.Location == e.FILE_LOCATION_REMOTE {
		go func() {
			// fmt.Println("开始")
			defer wg.Done()

			tmpFile := core.File{Dir: "./" + tmpdir, Name: path.Base(dst.PahtDir), Location: e.FILE_LOCATION_LOCAL}
			if !tmpFile.ExistFile() {
				lock, _ := fileLocks.LoadOrStore(tmpFile.Path(), &sync.Mutex{})
				mutex := lock.(*sync.Mutex)
				mutex.Lock()
				defer mutex.Unlock()

				src.Cmd.Cp(srcFile, tmpFile)
			} else if srcFile.Size != tmpFile.GetSize() {
				lock, _ := fileLocks.LoadOrStore(tmpFile.Path(), &sync.Mutex{})
				mutex := lock.(*sync.Mutex)
				mutex.Lock()
				defer mutex.Unlock()
			}
			_, err := dst.Cmd.Cp(tmpFile, core.File{Dir: path.Dir(dst.PahtDir), Name: path.Base(dst.PahtDir), Location: dst.Location, Replace: Replace})
			// fmt.Println(err)
			if err != nil {
				fmt.Println(err)
			}
			<-CpChan
		}()
	} else if dst.Location == e.FILE_LOCATION_REMOTE {
		go func() {
			defer wg.Done()
			dst.Cmd.Cp(srcFile, core.File{Dir: path.Dir(dst.PahtDir), Name: path.Base(dst.PahtDir), Location: dst.Location, Replace: Replace})
			<-CpChan
		}()
	} else {

		go func() {
			defer wg.Done()
			_, err := src.Cmd.Cp(srcFile, core.File{Dir: path.Dir(dst.PahtDir), Name: path.Base(dst.PahtDir), Location: dst.Location, Replace: Replace})
			if err != nil {
				fmt.Println(err)
			}
			<-CpChan
		}()
	}
}
