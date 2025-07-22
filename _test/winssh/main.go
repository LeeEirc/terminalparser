package main

import (
	"bytes"
	"fmt"
	"github.com/LeeEirc/terminalparser"
	"github.com/danielgatis/go-vte"
	"io"
	"os"
	"strings"
	"time"
)

var _ (vte.Performer) = (*performer)(nil)

type Row struct {
	data []rune
}

func (r *Row) String() string {
	return string(r.data)
}

type Cursor struct {
	x, y int
}

type performer struct {
	row          []*Row
	currentIndex int
	Cursor       Cursor
}

func (p *performer) Print(r rune) {
	fmt.Printf("[Print] %c\n", r)
	row := p.row[p.currentIndex]
	row.data = append(row.data, r)
	//fmt.Println(row.data)
}

func (p *performer) Execute(b byte) {
	fmt.Printf("[Execute] %02x\n", b)
	switch b {
	case '\r':
		p.Cursor.x = 0
	case '\n':
		p.row = append(p.row, &Row{})
		p.currentIndex++
	}
}

func (p *performer) Put(b byte) {
	fmt.Printf("[Put] %02x\n", b)
	time.Sleep(1000 * time.Millisecond)
}

func (p *performer) Unhook() {
	fmt.Printf("[Unhook]\n")
}

func (p *performer) Hook(params [][]uint16, intermediates []byte, ignore bool, r rune) {
	fmt.Printf("[Hook] params=%v, intermediates=%v, ignore=%v, r=%c\n", params, intermediates, ignore, r)
}

func (p *performer) OscDispatch(params [][]byte, bellTerminated bool) {
	fmt.Printf("[OscDispatch] params=%v, bellTerminated=%v\n", params, bellTerminated)
}

func (p *performer) CsiDispatch(params [][]uint16, intermediates []byte, ignore bool, r rune) {
	fmt.Printf("[CsiDispatch] params=%v, intermediates=%v, ignore=%v, r=%c\n", params, intermediates, ignore, r)
}

func (p *performer) EscDispatch(intermediates []byte, ignore bool, b byte) {
	fmt.Printf("[EscDispatch] intermediates=%v, ignore=%v, byte=%02x\n", intermediates, ignore, b)
}

func main() {
	winTxt := "output_windows.txt"
	buf, _ := os.ReadFile(winTxt)

	reader := bytes.NewReader(buf)
	per := &performer{
		row: []*Row{{}},
	}
	parser := vte.NewParser(per)

	buff := make([]byte, 2048)

	for {
		n, err := reader.Read(buff)

		if err != nil {
			if err == io.EOF {
				break
			}

			fmt.Printf("Err %v:", err)
			break
		}

		for _, b := range buff[:n] {
			parser.Advance(b)
		}
	}
	fmt.Println("=====per=====")
	for _, row := range per.row {
		if strings.TrimSpace(string(row.data)) == "" {
			continue
		}
		fmt.Printf("%s\n", row.String())

	}
	fmt.Println("=====screen=====")
	s := terminalparser.NewScreen(100, 1000)
	lines := s.Parse(buff)
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		fmt.Printf("%s\n", line)
	}

}
