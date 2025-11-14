package main

import (
	"fmt"
	"os"

	"apimgr/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
