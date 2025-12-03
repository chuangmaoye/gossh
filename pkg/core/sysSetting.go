package core

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
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
		return setWindowsProxy(config)
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
		return disableWindowsProxy()
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
