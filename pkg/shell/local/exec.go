package local

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os/exec"
	"strings"
)

func (l *Local) Run(cmdStr string) (string, error) {
	var outInfo bytes.Buffer
	var outErr bytes.Buffer
	cmd := exec.Command("/bin/sh", "-c", cmdStr)
	cmd.Dir = l.Dir
	cmd.Stdout = &outInfo
	cmd.Stderr = &outErr
	cmd.Run()
	cmd.Wait()
	cmds := []string{}
	cmdStrs := strings.Split(cmdStr, "|")
	for _, v := range cmdStrs {
		cmds = append(cmds, strings.Split(v, "&&")...)
	}
	// cmds := []*exec.Cmd{}
	for _, v := range cmds {
		qs := strings.Split(strings.Trim(v, " "), " ")
		if strings.Trim(qs[0], " ") == "cd" && len(qs) > 1 {
			dir := strings.Trim(qs[1], " ")
			dirSplit := strings.Split(dir, "/")
			if len(dirSplit) > 1 && dirSplit[0] == "." {
				l.Dir = fmt.Sprintf("%s/%s", l.Dir, strings.Join(dirSplit[1:], "/"))
			} else if len(dirSplit) > 1 && dirSplit[0] == ".." {
				sdirSplit := strings.Split(l.Dir, "/")
				l.Dir = fmt.Sprintf("%s/%s", strings.Join(sdirSplit[0:len(sdirSplit)-1], "/"), strings.Join(dirSplit[1:], "/"))
			} else if dirSplit[0] == ".." {
				sdirSplit := strings.Split(l.Dir, "/")
				l.Dir = strings.Join(sdirSplit[0:len(sdirSplit)-1], "/")
			} else {
				l.Dir = strings.Trim(qs[1], " ")
			}

		}
	}

	return outInfo.String(), nil
}

func (l *Local) RunFile(path string) (string, error) {
	var outInfo string
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return outInfo, err
	}
	str := string(b)
	commands := strings.Split(str, "\n")
	for _, c := range commands {
		out, err := l.Run(c)
		if err != nil {
			return outInfo, err
		}
		outInfo += out + "\n"
	}
	return outInfo, nil
}

func (l *Local) Close() error {
	return nil
}
