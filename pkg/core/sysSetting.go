package core

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"unsafe"
)

type ProxyConfig struct {
	HTTPProxy  string
	HTTPSProxy string
	Port       string
	Enable     bool
}

// setSystemProxy 设置系统代理
func SetSystemProxy(config ProxyConfig) error {
	switch runtime.GOOS {
	case "windows":
		// SetWindowsProxy(config.HTTPProxy+":"+config.Port, config.HTTPProxy+":"+config.Port, "localhost;127.0.0.1;<-loopback>", true)
		return SetWindowsProxy(config.HTTPProxy+":"+config.Port, true)
	case "darwin":
		return setMacOSProxy(config)
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

// disableSystemProxy 禁用系统代理
func DisableSystemProxy() error {
	switch runtime.GOOS {
	case "windows":
		return SetWindowsProxy("", false)
	case "darwin":
		return disableMacOSProxy()
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

// setWindowsProxy 设置Windows系统代理
func setWindowsProxy(config ProxyConfig) error {
	// 启用代理
	cmd := exec.Command("reg", "add",
		"HKCU\\Software\\Microsoft\\Windows\\CurrentVersion\\Internet Settings",
		"/v", "ProxyEnable",
		"/t", "REG_DWORD",
		"/d", "1",
		"/f")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to enable proxy: %v", err)
	}

	// 设置代理服务器
	proxyServer := fmt.Sprintf("%s:%s", config.HTTPProxy, config.HTTPSProxy)
	cmd = exec.Command("reg", "add",
		"HKCU\\Software\\Microsoft\\Windows\\CurrentVersion\\Internet Settings",
		"/v", "ProxyServer",
		"/t", "REG_SZ",
		"/d", proxyServer,
		"/f")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set proxy server: %v", err)
	}

	// 刷新系统设置
	cmd = exec.Command("RunDll32.exe", "inetcpl.cpl,ClearMyTracksByProcess", "255")
	cmd.Run()

	fmt.Println("Windows系统代理已设置")
	return nil
}

// disableWindowsProxy 禁用Windows系统代理
func disableWindowsProxy() error {
	cmd := exec.Command("reg", "add",
		"HKCU\\Software\\Microsoft\\Windows\\CurrentVersion\\Internet Settings",
		"/v", "ProxyEnable",
		"/t", "REG_DWORD",
		"/d", "0",
		"/f")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to disable proxy: %v", err)
	}

	// 刷新系统设置
	cmd = exec.Command("RunDll32.exe", "inetcpl.cpl,ClearMyTracksByProcess", "255")
	cmd.Run()

	fmt.Println("Windows系统代理已禁用")
	return nil
}

// setMacOSProxy 设置macOS系统代理
func setMacOSProxy(config ProxyConfig) error {
	// 获取当前网络服务
	networkService := getMacNetworkService()
	fmt.Println(networkService)
	if len(networkService) == 0 {
		return fmt.Errorf("failed to get network service")
	}
	for _, v := range networkService {
		// 设置HTTP代理
		cmd := exec.Command("networksetup", "-setwebproxy", v, config.HTTPProxy, config.Port)
		if err := cmd.Run(); err != nil {
			fmt.Println(v)
		}

		// 设置HTTPS代理
		cmd = exec.Command("networksetup", "-setsecurewebproxy", v, config.HTTPSProxy, config.Port)
		if err := cmd.Run(); err != nil {
			fmt.Println(v)
		}
	}

	fmt.Println("macOS系统代理已设置")
	return nil
}

// disableMacOSProxy 禁用macOS系统代理
func disableMacOSProxy() error {
	networkService := getMacNetworkService()
	if len(networkService) == 0 {
		return fmt.Errorf("failed to get network service")
	}
	for _, v := range networkService {
		// 禁用HTTP代理
		cmd := exec.Command("networksetup", "-setwebproxystate", v, "off")
		if err := cmd.Run(); err != nil {
			fmt.Println(v)
		}

		// 禁用HTTPS代理
		cmd = exec.Command("networksetup", "-setsecurewebproxystate", v, "off")
		if err := cmd.Run(); err != nil {
			fmt.Println(v)
		}
	}

	fmt.Println("macOS系统代理已禁用")
	return nil
}

// getMacNetworkService 获取macOS网络服务名称
func getMacNetworkService() []string {
	cmd := exec.Command("networksetup", "-listallnetworkservices")
	output, err := cmd.Output()
	if err != nil {
		return []string{}
	}

	// 解析输出，获取第一个启用的网络服务

	services := strings.Split(string(output), "\n")

	return services
}

var (
	advapi32           = syscall.NewLazyDLL("advapi32.dll")
	regOpenKeyExW      = advapi32.NewProc("RegOpenKeyExW")
	regSetValueExW     = advapi32.NewProc("RegSetValueExW")
	regCloseKey        = advapi32.NewProc("RegCloseKey")
	shell32            = syscall.NewLazyDLL("shell32.dll")
	SHChangeNotify     = shell32.NewProc("SHChangeNotify")
	wininet            = syscall.NewLazyDLL("wininet.dll")
	InternetSetOptionW = wininet.NewProc("InternetSetOptionW")

	// 预定义例外列表（避免每次动态生成，Clash 也是内置模板）
	defaultExceptions = "localhost;127.*;192.168.*;172.16.*;172.17.*;172.18.*;172.19.*;172.20.*"
)

const (
	HKEY_CURRENT_USER = 0x80000001
	KEY_WRITE         = 0x20013
	REG_DWORD         = 4
	REG_SZ            = 1
	proxyRegPath      = `Software\Microsoft\Windows\CurrentVersion\Internet Settings`

	// WinINet 同步常量（预定义，避免魔法数字）
	INTERNET_OPTION_SETTINGS_CHANGED = 0x00002500
	INTERNET_OPTION_REFRESH          = 0x00002501
)

// 设置 Windows 系统代理（当前用户生效，无需管理员）
// proxyServer: HTTP 代理地址（如 "127.0.0.1:7890"）
// httpsProxy: HTTPS 代理地址（可选，为空则复用 HTTP 代理）
// exceptions: 例外列表（如 "localhost;127.0.0.1;<-loopback>"）
// enable: 是否启用代理（true=启用，false=禁用）
func SetWindowsProxy(proxyServer string, enable bool) error {
	var hKey syscall.Handle
	err := regOpenKeyEx(
		HKEY_CURRENT_USER,
		proxyRegPath,
		0,
		KEY_WRITE,
		&hKey,
	)
	if err != nil {
		return fmt.Errorf("打开注册表失败：%v", err)
	}
	defer regCloseKey.Call(uintptr(hKey))

	// 禁用代理的完整逻辑（重点优化）
	if !enable {
		// 1. 核心：禁用代理开关
		if err := regSetDWORDValue(hKey, "ProxyEnable", 0); err != nil {
			return fmt.Errorf("禁用 ProxyEnable 失败：%v", err)
		}

		// 2. 清除可能干扰的自动配置脚本（关键！避免自动代理覆盖）
		if err := regSetStringValue(hKey, "AutoConfigURL", ""); err != nil {
			return fmt.Errorf("清空 AutoConfigURL 失败：%v", err)
		}

		// 3. 禁用自动检测设置（防止系统自动启用代理）
		if err := regSetDWORDValue(hKey, "AutoDetect", 0); err != nil {
			return fmt.Errorf("禁用 AutoDetect 失败：%v", err)
		}
	} else {
		// 启用代理的逻辑（保持不变）
		if err := regSetDWORDValue(hKey, "ProxyEnable", 1); err != nil {
			return fmt.Errorf("启用 ProxyEnable 失败：%v", err)
		}
		if err := regSetStringValue(hKey, "ProxyServer", proxyServer); err != nil {
			return fmt.Errorf("设置 ProxyServer 失败：%v", err)
		}
		if err := regSetStringValue(hKey, "ProxyOverride", defaultExceptions); err != nil {
			return fmt.Errorf("设置例外列表失败：%v", err)
		}
		// 启用时确保自动检测关闭
		if err := regSetDWORDValue(hKey, "AutoDetect", 0); err != nil {
			return fmt.Errorf("禁用 AutoDetect 失败：%v", err)
		}
		if err := regSetStringValue(hKey, "AutoConfigURL", ""); err != nil {
			return fmt.Errorf("清空 AutoConfigURL 失败：%v", err)
		}
	}

	// 增强系统同步（禁用时也需强制刷新缓存）
	// 1. 刷新 WinINet 缓存（关键：让应用立即读取新配置）
	InternetSetOptionW.Call(0, uintptr(INTERNET_OPTION_SETTINGS_CHANGED), 0, 0)
	InternetSetOptionW.Call(0, uintptr(INTERNET_OPTION_REFRESH), 0, 0)

	// 2. 通知系统代理设置变更（补充更全面的信号）
	SHChangeNotify.Call(0x08000000|0x00001000, 0x0001|0x0002, 0, 0)

	return nil
}

// 辅助函数：打开注册表项（封装 Windows API）
func regOpenKeyEx(hKey syscall.Handle, subKey string, options uint32, access uint32, result *syscall.Handle) error {
	subKeyPtr, err := syscall.UTF16PtrFromString(subKey)
	if err != nil {
		return err
	}
	r1, _, err := regOpenKeyExW.Call(
		uintptr(hKey),
		uintptr(unsafe.Pointer(subKeyPtr)),
		uintptr(options),
		uintptr(access),
		uintptr(unsafe.Pointer(result)),
	)
	if r1 != 0 {
		return fmt.Errorf("regOpenKeyEx failed: %v", err)
	}
	return nil
}

func regSetDWORDValue(hKey syscall.Handle, name string, value uint32) error {
	namePtr, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return err
	}
	r1, _, err := regSetValueExW.Call(
		uintptr(hKey),
		uintptr(unsafe.Pointer(namePtr)),
		0,
		uintptr(REG_DWORD),
		uintptr(unsafe.Pointer(&value)),
		4, // DWORD 固定 4 字节
	)
	if r1 != 0 {
		return fmt.Errorf("regSetDWORDValue failed: %v", err)
	}
	return nil
}

func regSetStringValue(hKey syscall.Handle, name, value string) error {
	namePtr, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return err
	}
	// 用 syscall.StringToUTF16 替代 syscall.UTF16Encode（兼容所有版本）
	valueUTF16 := syscall.StringToUTF16(value) // 转换为 []uint16（含终止符 \x00）
	valuePtr := &valueUTF16[0]                 // 取首地址指针
	// 字符串长度：每个 uint16 占 2 字节，总长度 = 元素个数 * 2
	valueLen := uintptr(len(valueUTF16) * 2)

	r1, _, err := regSetValueExW.Call(
		uintptr(hKey),
		uintptr(unsafe.Pointer(namePtr)),
		0,
		uintptr(REG_SZ),
		uintptr(unsafe.Pointer(valuePtr)),
		valueLen,
	)
	if r1 != 0 {
		return fmt.Errorf("regSetStringValue failed: %v", err)
	}
	return nil
}
