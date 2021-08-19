package main

import (
	"github.com/akaspin/logx"
	"os"
)

func main() {
	err := run(os.Stderr, os.Stdout, os.Stdin, os.Args[1:]...)
	if err != nil {
		logx.GetLog("main").Critical(err)
	}
}
