package client

import (
	"fmt"
	"gossh/pkg/core"
	"gossh/pkg/util"
	"os"
	"path/filepath"
	"strconv"

	"github.com/jinzhu/configor"
	"github.com/spf13/cobra"
)

type SSHConfig struct {
	Addresss []core.Address
}

var (
	AppVersion     = "Build: 2023-11-10 V1.0.0"
	SshConfig      SSHConfig
	ServerAddresss map[string]core.Address
	downDir        bool
	Config         string
	Threads        int
	Replace        bool
	Port           string
)

func init() {
	if util.ExistFile("config.yml") {
		configor.Load(&SshConfig, "config.yml")
	} else {
		configor.Load(&SshConfig, fmt.Sprintf("%s/config.yml", GetCurrentDir()))
	}
	ServerAddresss = make(map[string]core.Address)
	for _, v := range SshConfig.Addresss {
		ServerAddresss[v.Key] = v
	}

	RootCmd.AddCommand(copyCmd)
	RootCmd.AddCommand(serveCmd)
	RootCmd.AddCommand(porxyCmd)
	copyCmd.Flags().BoolVarP(&downDir, "dir", "d", false, "下载文件夹")
	copyCmd.Flags().StringVarP(&Config, "config", "c", "", "配置文件")
	copyCmd.Flags().IntVarP(&Threads, "thread", "t", 10, "线程数")
	copyCmd.Flags().BoolVarP(&Replace, "replace", "r", false, "是否替换文件")
	serveCmd.Flags().StringVarP(&Config, "config", "c", "", "配置文件")
	RootCmd.Flags().StringVarP(&Config, "config", "c", "", "配置文件")
	porxyCmd.Flags().StringVarP(&Port, "port", "p", "8080", "端口")
}

func GetCurrentDir() string {
	dir, _ := os.Executable()
	exPath := filepath.Dir(dir)
	return exPath
}

func ShowServers() {
	fmt.Println("索引\tkey\t\tip\t服务名")
	for i, v := range SshConfig.Addresss {
		fmt.Printf("%d\t%s\t%s\t%s\n", i, v.Key, v.IP, v.ServerName)
	}
}

var RootCmd = &cobra.Command{
	Use:   "gossh",
	Short: ``,
	Long:  `登录到服务器，gossh [索引]|[key]`,
	Run: func(cmd *cobra.Command, args []string) {
		if Config != "" {
			configor.Load(&SshConfig, Config)
		}
		fmt.Printf("%s\n", AppVersion)
		if len(args) == 0 {
			ShowServers()
			cmd.Help()
		}

	},
}

var copyCmd = &cobra.Command{
	Use:   "cp",
	Short: `拷贝文件到服务器或者本地`,
	Long:  `拷贝文件到服务器或者本地，gossh cp [-d是否拷贝文件夹] [本地文件｜服务器文件] [服务器文件｜本地文件]...`,
	Run: func(cmd *cobra.Command, args []string) {
		if Config != "" {
			configor.Load(&SshConfig, Config)
		}
		if Threads < 1 {
			Threads = 1
		}
		if Threads > 100 {
			Threads = 100
		}
		CpChan = make(chan int, Threads)
		if len(args) > 1 {
			if downDir {
				CopyDir(args...)
			} else {
				Copy(args...)
			}

		} else {
			cmd.Help()
		}
	},
}

var serveCmd = &cobra.Command{
	Use:   "link",
	Short: `登录服务器`,
	Long:  `登录服务器，gossh serve [key]|[索引]`,
	Run: func(cmd *cobra.Command, args []string) {
		if Config != "" {
			configor.Load(&SshConfig, Config)
		}
		if len(args) > 0 {
			index, err := strconv.Atoi(args[0])
			if err != nil {
				if addr, ok := ServerAddresss[args[0]]; ok {
					fmt.Println(addr.ServerName)
					Terminal(addr)
				} else {
					fmt.Println("服务不存在请重新输入！")
				}
			} else if len(SshConfig.Addresss) > index {
				fmt.Println(SshConfig.Addresss[index].ServerName)
				Terminal(SshConfig.Addresss[index])
			} else {
				fmt.Println("服务不存在请重新输入！")
			}
		}
	},
}

var porxyCmd = &cobra.Command{
	Use:   "proxy",
	Short: `启动代理服务`,
	Long:  `启动代理服务，gossh proxy [key]|[索引]`,
	Run: func(cmd *cobra.Command, args []string) {
		if Config != "" {
			configor.Load(&SshConfig, Config)
		}
		if len(args) > 0 {
			index, err := strconv.Atoi(args[0])
			if err != nil {
				if addr, ok := ServerAddresss[args[0]]; ok {
					fmt.Println(addr.ServerName)
					ProxyTerminal(addr, Port)
				} else {
					fmt.Println("服务不存在请重新输入！")
				}
			} else if len(SshConfig.Addresss) > index {
				fmt.Println(SshConfig.Addresss[index].ServerName)
				ProxyTerminal(SshConfig.Addresss[index], Port)
			} else {
				fmt.Println("服务不存在请重新输入！")
			}
		}
	},
}
