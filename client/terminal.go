package client

import (
	"fmt"
	"gossh/pkg/core"
	"time"
)

func Terminal(addr core.Address) error {
	c, err := core.New(addr.IP, addr.Name, addr.Password, addr.Pem, addr.Port)
	if err != nil {
		fmt.Println("err", err)
		return err
	}

	time.Sleep(1 * time.Second)

	return c.EnterTerminal()

}

func ProxyTerminal(addr core.Address, port string) error {
	c, err := core.New(addr.IP, addr.Name, addr.Password, addr.Pem, addr.Port)
	if err != nil {
		fmt.Println("err", err)
		return err
	}

	time.Sleep(1 * time.Second)

	return c.ProxyTerminal(port)

}
