package terminalparser

import (
	"log"
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
		log.Println(format)
		return
	}
	log.Printf(format, args...)
}
