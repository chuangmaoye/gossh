package main

import (
	"fmt"
	"os"

	"gossh/client"
)

func main() {
	if err := client.RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
