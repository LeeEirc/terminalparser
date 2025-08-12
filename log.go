package terminalparser

import (
	"fmt"
	"os"
)

var terminalDebug = false

func init() {
	if os.Getenv("TERMINALPARSER") != "" {
		terminalDebug = true
	}
}

func Printf(format string, args ...interface{}) {
	if !terminalDebug {
		return
	}
	if len(args) == 0 {
		fmt.Println(format)
		return
	}
	fmt.Printf(format, args...)
}

func Println(args ...interface{}) {
	if !terminalDebug {
		return
	}
	fmt.Println(args...)
}
