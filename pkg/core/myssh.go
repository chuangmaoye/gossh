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
	"runtime"
	"syscall"
	"time"
	"unsafe"

	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	"golang.org/x/sys/windows"
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
// func (c Cli) EnterTerminal_back() error {
// 	session, err := c.NewSession()
// 	if err != nil {
// 		return err
// 	}
// 	defer session.Close()

// 	fd := int(os.Stdin.Fd())

// 	// 跨平台验证：标准输入是否为终端/控制台
// 	if runtime.GOOS == "windows" {
// 		// Windows 下验证是否为控制台（通过 GetConsoleMode）
// 		var mode uint32
// 		if err := windows.GetConsoleMode(windows.Handle(fd), &mode); err != nil {
// 			return fmt.Errorf("stdin is not a Windows console: %w", err)
// 		}
// 	} else {
// 		// 非 Windows 系统验证终端
// 		if !term.IsTerminal(fd) {
// 			return fmt.Errorf("file descriptor %d is not a terminal", fd)
// 		}
// 	}

// 	oldState, err := term.MakeRaw(fd)
// 	if err != nil {
// 		return err
// 	}
// 	defer term.Restore(fd, oldState)

// 	session.Stdout = os.Stdout
// 	session.Stderr = os.Stdin
// 	session.Stdin = os.Stdin

// 	termWidth, termHeight, err := getTerminalSize(fd)
// 	if err != nil {
// 		return err
// 	}

// 	modes := ssh.TerminalModes{
// 		ssh.ECHO:          1,
// 		ssh.TTY_OP_ISPEED: 14400,
// 		ssh.TTY_OP_OSPEED: 14400,
// 	}
// 	err = session.RequestPty("xterm-256color", termHeight, termWidth, modes)
// 	if err != nil {
// 		err = session.RequestPty("vt100", termHeight, termWidth, modes)
// 		if err != nil {
// 			return fmt.Errorf("failed to request PTY (xterm-256color/vt100): %w", err)
// 		}
// 	}

// 	err = session.Shell()
// 	if err != nil {
// 		return err
// 	}

// 	return session.Wait()
// }

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
	time.Sleep(20 * time.Second)
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

// 手动定义 Windows 控制台相关结构体
type COORD struct {
	X int16
	Y int16
}

type SMALL_RECT struct {
	Left   int16
	Top    int16
	Right  int16
	Bottom int16
}

type CONSOLE_SCREEN_BUFFER_INFO struct {
	Size              COORD
	CursorPosition    COORD
	Attributes        uint16
	Window            SMALL_RECT
	MaximumWindowSize COORD
}

// 手动声明 Windows 系统调用函数
var (
	kernel32                       = windows.NewLazySystemDLL("kernel32.dll")
	procGetConsoleScreenBufferInfo = kernel32.NewProc("GetConsoleScreenBufferInfo")
	procSetConsoleOutputCP         = kernel32.NewProc("SetConsoleOutputCP") // 设置控制台编码
	procSetConsoleCP               = kernel32.NewProc("SetConsoleCP")       // 设置控制台输入编码
)

// getConsoleScreenBufferInfo 调用 Windows 原生 API 获取控制台信息
func getConsoleScreenBufferInfo(handle windows.Handle, csbi *CONSOLE_SCREEN_BUFFER_INFO) error {
	r1, _, err := procGetConsoleScreenBufferInfo.Call(
		uintptr(handle),
		uintptr(unsafe.Pointer(csbi)),
	)
	if r1 == 0 {
		return err
	}
	return nil
}

// setWindowsConsoleUTF8 强制 Windows 控制台使用 UTF-8 编码（核心修复乱码）
func setWindowsConsoleUTF8() error {
	if runtime.GOOS != "windows" {
		return nil
	}
	// 设置 UTF-8 编码
	r1, _, err := procSetConsoleOutputCP.Call(65001)
	if r1 == 0 {
		return fmt.Errorf("set console output UTF-8 failed: %w", err)
	}
	r2, _, err := procSetConsoleCP.Call(65001)
	if r2 == 0 {
		return fmt.Errorf("set console input UTF-8 failed: %w", err)
	}
	// 提示用户切换字体
	fmt.Println("tip: please set CMD font to 'Consolas' or 'Microsoft YaHei Mono' to display UTF-8 correctly")
	return nil
}

// getTerminalSize 兼容 Windows/Linux 的终端大小获取
func getTerminalSize() (width, height int, err error) {
	if runtime.GOOS != "windows" {
		fd := int(os.Stdin.Fd())
		return term.GetSize(fd)
	}

	consoleHandle, err := windows.GetStdHandle(windows.STD_OUTPUT_HANDLE)
	if err != nil || consoleHandle == windows.InvalidHandle {
		consoleHandle, err = windows.GetStdHandle(windows.STD_INPUT_HANDLE)
		if err != nil || consoleHandle == windows.InvalidHandle {
			return 0, 0, fmt.Errorf("both stdin/stdout handles are invalid: %w", err)
		}
	}

	var csbi CONSOLE_SCREEN_BUFFER_INFO
	err = getConsoleScreenBufferInfo(consoleHandle, &csbi)
	if err != nil {
		fmt.Printf("warning: failed to get console size (use default 80x24): %v\n", err)
		return 80, 24, nil
	}

	window := csbi.Window
	width = int(window.Right - window.Left + 1)
	height = int(window.Bottom - window.Top + 1)

	if width <= 0 || height <= 0 {
		width = int(csbi.Size.X)
		height = int(csbi.Size.Y)
	}

	if width < 40 {
		width = 40
	}
	if height < 10 {
		height = 10
	}

	return width, height, nil
}

// isWindowsConsole 验证 Windows 下是否为真实控制台
func isWindowsConsole() bool {
	handleTypes := []uint32{windows.STD_INPUT_HANDLE, windows.STD_OUTPUT_HANDLE}

	for _, hType := range handleTypes {
		handle, err := windows.GetStdHandle(hType)
		if err != nil || handle == windows.InvalidHandle {
			continue
		}
		var mode uint32
		if windows.GetConsoleMode(handle, &mode) == nil {
			return true
		}
	}
	return false
}

func (c Cli) EnterTerminal() error {
	// 第一步：强制 Windows 控制台使用 UTF-8 编码（解决乱码核心）
	if runtime.GOOS == "windows" {
		if err := setWindowsConsoleUTF8(); err != nil {
			fmt.Printf("warning: set console UTF-8 failed (may cause garbled): %v\n", err)
		}
	}

	session, err := c.NewSession()
	if err != nil {
		return fmt.Errorf("create session failed: %w", err)
	}
	defer session.Close()

	// 跨平台终端验证
	if runtime.GOOS == "windows" {
		if !isWindowsConsole() {
			return fmt.Errorf("must run in Windows CMD/PowerShell (not WSL/third-party terminal)")
		}
	} else {
		fd := int(os.Stdin.Fd())
		if !term.IsTerminal(fd) {
			return fmt.Errorf("file descriptor %d is not a terminal", fd)
		}
	}

	// 配置终端 Raw 模式（保留 UTF-8 字符完整性）
	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		fmt.Printf("warning: failed to set raw mode (input may be abnormal): %v\n", err)
		oldState = nil
	} else {
		defer term.Restore(fd, oldState)
	}

	// 获取终端大小
	termWidth, termHeight, err := getTerminalSize()
	if err != nil {
		fmt.Printf("warning: %v, force use 80x24\n", err)
		termWidth, termHeight = 80, 24
	}

	// 优化 SSH 终端模式（禁用字符转换，确保 UTF-8 完整传输）
	modes := ssh.TerminalModes{
		ssh.ECHO:          1, // 开启回显（必须）
		ssh.ICRNL:         1, // 输入 \r 转 \n（兼容 Windows 输入）
		ssh.ONLCR:         0, // 禁用输出 \n 转 \r\n（避免双重换行）
		ssh.OCRNL:         0, // 禁用输出 \r 转 \n（防止字符错乱）
		ssh.ISTRIP:        0, // 禁用字符剥离（保留 UTF-8 多字节）
		ssh.INLCR:         0, // 禁用输入 \n 转 \r（避免转换错误）
		ssh.TTY_OP_ISPEED: 115200,
		ssh.TTY_OP_OSPEED: 115200,
		ssh.IXANY:         1, // 允许任意字符重启输入
		ssh.IXOFF:         0, // 禁用流控制
		ssh.IXON:          0, // 禁用 XON/XOFF（避免字符被吞）
		ssh.CS8:           1, // 强制 8 位字符（UTF-8 必需）
	}

	// 请求 PTY（指定 UTF-8 兼容的终端类型）
	ptyType := "xterm-256color" // 支持 UTF-8 的终端类型
	err = session.RequestPty(ptyType, termHeight, termWidth, modes)
	if err != nil {
		ptyType = "vt100"
		err = session.RequestPty(ptyType, termHeight, termWidth, modes)
		if err != nil {
			ptyType = "dumb"
			err = session.RequestPty(ptyType, termHeight, termWidth, modes)
			if err != nil {
				fmt.Printf("warning: request PTY failed, try without PTY: %v\n", err)
			}
		}
	}
	fmt.Printf("Using terminal: %s (size: %dx%d, encoding: UTF-8)\n", ptyType, termWidth, termHeight)

	// 绑定标准流（确保 UTF-8 字符无转换传输）
	session.Stdin = os.Stdin
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr

	// 启动 Shell
	err = session.Shell()
	if err != nil {
		fmt.Printf("warning: start shell with PTY failed, try without PTY: %v\n", err)
		err = session.Run("bash -i")
		if err != nil {
			return fmt.Errorf("start shell failed: %w", err)
		}
	}

	return session.Wait()
}
