package core

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

// Cli ...
type Cli struct {
	IP       string //IP地址
	Username string //用户名
	Password string //密码
	Port     int    //端口号
	Pem      string
	Client   *ssh.Client //ssh客户端
}

// New 创建命令行对象
// ip IP地址
// username 用户名
// password 密码
// port 端口号,默认22
func New(ip string, username string, password string, pem string, port ...int) (*Cli, error) {
	cli := new(Cli)
	cli.IP = ip
	cli.Username = username
	cli.Password = password
	cli.Pem = pem
	if len(port) <= 0 {
		cli.Port = 22
	} else {
		cli.Port = port[0]
	}
	if password == "" {
		return cli, cli.connectPublicKeys()
	}
	return cli, cli.connect()
}

// Run 执行 shell脚本命令
func (c Cli) Run(shell string) (string, error) {
	session, err := c.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	buf, err := session.CombinedOutput(shell)
	return string(buf), err
}

// RunTerminal 执行带交互的命令
func (c *Cli) RunTerminal(shell string) error {
	session, err := c.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	return c.RunTerminalSession(session, shell)
}

// RunTerminalSession 执行带交互的命令
func (c *Cli) RunTerminalSession(session *ssh.Session, shell string) error {
	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		panic(err)
	}
	defer term.Restore(fd, oldState)

	session.Stdout = os.Stdout
	session.Stderr = os.Stdin
	session.Stdin = os.Stdin

	termWidth, termHeight, err := term.GetSize(fd)
	if err != nil {
		panic(err)
	}
	// Set up terminal modes
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,     // enable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}

	// Request pseudo terminal
	if err := session.RequestPty("xterm-256color", termHeight, termWidth, modes); err != nil {
		return err
	}

	session.Run(shell)
	return nil
}

// EnterTerminal 完全进入终端
func (c Cli) EnterTerminal() error {
	session, err := c.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return err
	}
	defer term.Restore(fd, oldState)

	session.Stdout = os.Stdout
	session.Stderr = os.Stdin
	session.Stdin = os.Stdin

	termWidth, termHeight, err := term.GetSize(fd)
	if err != nil {
		return err
	}

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	err = session.RequestPty("xterm-256color", termHeight, termWidth, modes)
	if err != nil {
		return err
	}

	err = session.Shell()
	if err != nil {
		return err
	}

	return session.Wait()
}

// Enter 完全进入终端
func (c Cli) Enter(w io.Writer, r io.Reader) error {
	session, err := c.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	fd := int(os.Stdin.Fd())
	// oldState, err := terminal.MakeRaw(fd)
	// if err != nil {
	// 	return err
	// }
	// defer terminal.Restore(fd, oldState)

	session.Stdout = w
	session.Stderr = os.Stdin
	session.Stdin = r

	termWidth, termHeight, err := term.GetSize(fd)
	if err != nil {
		return err
	}

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	err = session.RequestPty("xterm-256color", termHeight, termWidth, modes)
	if err != nil {
		return err
	}

	err = session.Shell()
	if err != nil {
		return err
	}

	return session.Wait()
}

// 连接
func (c *Cli) connect() error {
	config := ssh.ClientConfig{
		User: c.Username,
		Auth: []ssh.AuthMethod{ssh.Password(c.Password)},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
		Timeout: 10 * time.Second,
	}
	addr := fmt.Sprintf("%s:%d", c.IP, c.Port)
	sshClient, err := ssh.Dial("tcp", addr, &config)
	if err != nil {
		return err
	}
	c.Client = sshClient
	return nil
}

// connectPublicKeys连接
func (c *Cli) connectPublicKeys() error {

	privateKeyBytes, err := os.ReadFile(c.Pem)
	if err != nil {
		log.Fatal(err)
	}

	key, err := ssh.ParsePrivateKey(privateKeyBytes)
	if err != nil {
		log.Fatal(err)
	}

	addr := fmt.Sprintf("%s:%d", c.IP, c.Port)
	auths := []ssh.AuthMethod{ssh.PublicKeys(key)}

	config := &ssh.ClientConfig{
		User: c.Username,
		Auth: auths,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
		Timeout: 10 * time.Second,
	}
	config.SetDefaults()
	logrus.Infof("tcp dial to %s", addr)
	sshClient, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		logrus.Infof("error login,details: %s", err.Error())
		return err
	}
	c.Client = sshClient
	return nil
}

// newSession new session
func (c Cli) NewSession() (*ssh.Session, error) {
	if c.Client == nil {
		if err := c.connect(); err != nil {
			return nil, err
		}
	}
	session, err := c.Client.NewSession()
	if err != nil {
		return nil, err
	}

	return session, nil
}
