package core

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"syscall"
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
		Timeout: 30 * time.Second,
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

func (c Cli) ProxyTerminal(port string) error {
	sigChan := make(chan os.Signal, 1)
	// 使用signal.Notify函数将收到的信号发送到sigChan
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	addr := "127.0.0.1"
	config := ProxyConfig{
		HTTPProxy:  addr,
		HTTPSProxy: addr,
		Port:       port,
		Enable:     true,
	}
	if err := SetSystemProxy(config); err != nil {
		log.Printf("设置代理失败: %v", err)
		return err
	}
	server := &http.Server{
		Addr: ":" + port,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, err := c.Run("echo 1")
			if err != nil {
				log.Println("error: 链接已断开", err)
				// 重新连接
				c.connect()
			}
			if r.Method == http.MethodConnect {
				c.handleTunneling(w, r)
			} else {
				c.handleHTTP(w, r)
			}
		}),
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("服务器启动失败: %v", err)
		}
	}()
	<-sigChan
	fmt.Println("接收到中断信号，正在关闭服务...")

	// 优雅关闭服务器
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("服务器关闭失败: %v", err)
	}
	fmt.Println("服务退出")
	DisableSystemProxy()
	return nil
}

func (c Cli) handleHTTP(w http.ResponseWriter, req *http.Request) {
	// 检查是否需要通过SSH隧道转发
	if c.Client != nil {
		// log.Printf("进入了http函数")
		c.handleSSHTunnelHTTP(w, req)
		return
	}

	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()

	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func (c Cli) handleTunneling(w http.ResponseWriter, req *http.Request) {
	// 检查是否需要通过SSH隧道转发
	if c.Client != nil {
		// log.Printf("进入了https函数")
		c.handleSSHTunnelHTTPS(w, req)
		return
	}

	destConn, err := net.DialTimeout("tcp", req.Host, 10*time.Second)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	go transfer(destConn, clientConn)
	go transfer(clientConn, destConn)
}

// 通过SSH隧道处理HTTP请求
func (c Cli) handleSSHTunnelHTTP(w http.ResponseWriter, req *http.Request) {
	// 通过SSH连接建立到目标服务器的连接
	addr := req.Host
	if req.URL.Port() == "" {
		addr += ":80"
	}
	remoteConn, err := c.Client.Dial("tcp", addr)
	// log.Printf("host: %s", addr)
	if err != nil {
		http.Error(w, fmt.Sprintf("SSH tunnel error: %v", err), http.StatusServiceUnavailable)
		return
	}
	// defer remoteConn.Close()

	// 发送HTTP请求
	err = req.Write(remoteConn)
	if err != nil {
		http.Error(w, fmt.Sprintf("Write error: %v", err), http.StatusServiceUnavailable)
		return
	}

	// 读取响应
	resp, err := http.ReadResponse(bufio.NewReader(remoteConn), req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Read response error: %v", err), http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()

	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// 通过SSH隧道处理HTTPS请求
func (c Cli) handleSSHTunnelHTTPS(w http.ResponseWriter, req *http.Request) {
	// 通过SSH连接建立到目标服务器的连接

	remoteConn, err := c.Client.Dial("tcp", req.Host)
	// log.Printf("host: %s", req.Host)

	if err != nil {
		http.Error(w, fmt.Sprintf("SSH tunnel error: %v", err), http.StatusServiceUnavailable)
		return
	}
	// defer remoteConn.Close()

	w.WriteHeader(http.StatusOK)

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	// startLocalListener(remoteConn, clientConn)
	go transfer(remoteConn, clientConn)
	go transfer(clientConn, remoteConn)
	// go transfer(clientConn, remoteConn)
}

func transfer(destination io.WriteCloser, source io.ReadCloser) {
	defer destination.Close()
	defer source.Close()
	io.Copy(destination, source)
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func (c Cli) NewMultipleHostsReverseProxy(targets []*url.URL) *httputil.ReverseProxy {
	director := func(req *http.Request) {
		target := targets[rand.Int()%len(targets)]
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.URL.Path = target.Path
	}
	return &httputil.ReverseProxy{
		Director: director,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				// 如果配置了SSH客户端且是内部请求，通过SSH隧道转发
				// if sshClient != nil && isInternalRequest(req) {
				if c.Client != nil {
					return c.Client.Dial(network, addr)
				}
				return net.Dial(network, addr)
			},
		},
	}
}
