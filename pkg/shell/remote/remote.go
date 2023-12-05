package remote

import (
	"gossh/pkg/core"

	"golang.org/x/crypto/ssh"
)

type Remote struct {
	Dir     string
	Cli     *core.Cli
	Session *ssh.Session
}
