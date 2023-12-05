package remote

import (
	"io/ioutil"
	"strings"
)

func (R *Remote) Run(cmdStr string) (string, error) {
	output, err := R.Cli.Run(cmdStr)

	return output, err
}

func (R *Remote) RunFile(path string) (string, error) {
	var outInfo string
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return outInfo, err
	}
	str := string(b)
	commands := strings.Split(str, "\n")
	for _, c := range commands {
		out, err := R.Run(c)
		if err != nil {
			return outInfo, err
		}
		outInfo += out + "\n"
	}
	return outInfo, nil
}

func (R *Remote) Close() error {
	return nil
}
