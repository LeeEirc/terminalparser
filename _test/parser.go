package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/LeeEirc/terminalparser"
)

const (
	enter = iota + 1
	waitOut
	input

	enterKey = '\r'
)

type Parser struct {
	inputBuf  bytes.Buffer
	outputBuf bytes.Buffer
	Ps1sStr   string
	screen    terminalparser.Screen
	state     int
	once      sync.Once
	mux       sync.Mutex
	fd        *os.File
}

func (s *Parser) Feed(p []byte) {
	s.mux.Lock()
	defer s.mux.Unlock()
	fmt.Println(hex.Dump(p))
	if s.fd == nil {
		s.fd, _ = os.Create("output.txt")
	}
	_, _ = s.fd.Write(p)
	s.screen.Feed(p)
	if s.state == waitOut {
		s.outputBuf.Write(p)
	}
	fmt.Println("===========terminal  start============>")
	rowLen := len(s.screen.Rows)
	start := 0
	if rowLen > 5 {
		start = rowLen - 5
	}
	for i := range s.screen.Rows {
		if i < start {
			continue
		}
		row := s.screen.Rows[i]
		fmt.Println(row.String())
	}
	fmt.Println()
	row := s.screen.GetCursorRow()
	if row.String() == s.Ps1sStr {
		fmt.Println("current output: ", s.outputBuf.String())
	}
	fmt.Println("===>current ps1:  ", s.Ps1sStr)
	fmt.Printf("===========terminal end total {%d}============>\n", len(s.screen.Rows))
}

func (s *Parser) IsEnterKey(p []byte) bool {
	return p[len(p)-1] == enterKey
}

func (s *Parser) WriteInput(chars []byte) (string, bool) {
	if len(chars) == 0 {
		return "", false
	}
	s.mux.Lock()
	defer s.mux.Unlock()
	s.once.Do(func() {
		s.state = input
		s.Ps1sStr = s.GetPs1()
	})
	if s.IsEnterKey(chars) {
		s.state = waitOut
		lastLine := s.screen.GetCursorRow()
		cmd := strings.TrimPrefix(lastLine.String(), s.Ps1sStr)
		fmt.Println("命令： ", cmd)
		fmt.Println("用户输入的：", s.inputBuf.String())
		time.Sleep(time.Millisecond * 1000)
		s.inputBuf.Reset()
		return cmd, true
	}
	if s.state == waitOut {
		s.state = input
		s.Ps1sStr = s.GetPs1()
		s.outputBuf.Reset()
	}
	s.inputBuf.Write(chars)
	return "", false
}

func (s *Parser) GetPs1() string {
	row := s.screen.GetCursorRow()
	rowStr := row.String()
	return strings.TrimSuffix(rowStr, s.inputBuf.String())
}

// rest ps1
/*

1、

*/
