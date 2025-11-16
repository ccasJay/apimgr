package main

import (
	"fmt"
	"os"

	"apimgr/cmd"
)

// Version information (set by goreleaser)
var (
	version = "development"
	commit  = ""
	date    = ""
)

func main() {
	cmd.SetVersionInfo(version, commit, date)
	if err := cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
