package main

import (
	"bytes"
	"fmt"
	"strings"
)

var (
	enterMarks = [][]byte{
		[]byte("\x1b[?1049h"),
		[]byte("\x1b[?1048h"),
		[]byte("\x1b[?1047h"),
		[]byte("\x1b[?47h"),
	}

	exitMarks = [][]byte{
		[]byte("\x1b[?1049l"),
		[]byte("\x1b[?1048l"),
		[]byte("\x1b[?1047l"),
		[]byte("\x1b[?47l"),
	}
	screenMarks = [][]byte{
		{0x1b, 0x5b, 0x4b, 0x0d, 0x0a}, // 4b 0d 0a
		//{0x1b, 0x5b, 0x34, 0x6c}, // 1b 5b 34 6c
	}
	vimMarks = [][]byte{
		{0x1b, 0x5b, 0x32, 0x3b, 0x31}, // ESC ] 2;  设置标题 1b 5b 32 3b 31
		//{0x1b, 0x5b, 0x32, 0x32, 0x3b, 0x30, 0x3b, 0x30, 0x74}, // 1b 5b 32 32 3b 30 3b 30  74  设置标题的控制字符
	}
)

func isNewScreen(p []byte) bool {
	return matchMark(p, screenMarks)
}

func IsEditEnterMode(p []byte) bool {
	return matchMark(p, enterMarks)
}

func IsEditExitMode(p []byte) bool {
	return matchMark(p, exitMarks)
}

func matchMark(p []byte, marks [][]byte) bool {
	for _, item := range marks {
		if bytes.Contains(p, item) {
			return true
		}
	}
	return false
}

// (1, 1) 初始位置 左上角的位置，x 代表列，y 代表行数

type Row struct {
	Line    []rune
	CursorX int // 当前行光标的位置
}

func (r *Row) InsertChars(i int) {
	spaceLines := make([]rune, i)
	for j := range spaceLines {
		spaceLines[j] = ' '
	}

	index := r.GetCurrentX()
	newLine := make([]rune, 0, len(r.Line)+i)
	newLine = append(newLine, r.Line[:index]...)
	newLine = append(newLine, spaceLines...)
	newLine = append(newLine, r.Line[index:]...)
	r.Line = newLine
}

func (r *Row) String() string {
	return strings.TrimSpace(string(r.Line))
}

func (r *Row) Add(c rune) {
	index := r.GetCurrentX()
	if len(r.Line) > index {
		if r.Line == nil {
			r.Line = make([]rune, index+1)
		}
		r.Line[index] = c
	} else {
		r.Line = append(r.Line, c)
	}
	r.CursorX += 1
}

func (r *Row) MoveLeftCurse(i int) {
	r.CursorX -= i
}

func (r *Row) MoveRightCurse(i int) {
	r.CursorX += i
}

func (r *Row) EaseRightCharsAll() {
	index := r.GetCurrentX()
	newLine := make([]rune, 0, len(r.Line))
	newLine = append(newLine, r.Line[:index]...)
	r.Line = newLine
}

func (r *Row) DeleteChars(i int) {
	index := r.GetCurrentX()
	newLine := make([]rune, 0, len(r.Line))
	newLine = append(newLine, r.Line[:index+1]...)
	rest := r.Line[index+1:]

	if len(rest) >= i {
		newLine = append(newLine, rest[i:]...)
	}
	r.Line = newLine
}

func (r *Row) GetCurrentX() int {
	index := r.CursorX - 1
	if index < 0 {
		index = 0
		r.CursorX = 1
	}
	return index
}

type TmuxCursor struct {
	X, Y int
}

func (t *TmuxCursor) MoveLeft(i int) {
	t.X -= i
}
func (t *TmuxCursor) MoveRight(i int) {
	t.X += i
}

type VtTmuxScreen struct {
	rows            []*Row     // r 创建足够多的 rows
	currentRowIndex int        // 根据光标设置，判断当前的row行数
	Cursor          TmuxCursor // 默认从 （1，1） 开始 获取当前值的时候 默认需要 -1
	maxRows         int
}

func (p *VtTmuxScreen) Print(r rune) {
	//fmt.Printf("[Print] %c\n", r)
	currentRow := p.GetCursorRow()
	currentRow.Add(r)
	p.Cursor.X += 1
	fmt.Print(currentRow)
	fmt.Println()
}

func (p *VtTmuxScreen) Execute(b byte) {
	fmt.Printf("[Execute] %02x\n", b)
	switch b {
	case '\r':
		p.Cursor.X = 0
		//p.Cursor.Y += 1
	case '\n':
		p.currentRowIndex++
		p.Cursor.Y += 1
		if len(p.rows) <= p.currentRowIndex {
			p.rows = append(p.rows, &Row{})
		}
		/*
			如果达到了最后一行，如果还有新内容出现，则会触发：
				params=[[1] [100]], intermediates=[], ignore=false, r=r
				params=[[99] [1]], intermediates=[], ignore=false, r=H
			绘画区域是 1-100 行
			跳转到 99 1 的位置
		*/
		//if len(p.rows) <= (p.maxRows - 1) {
		//	newRows := make([]Row, 0, p.maxRows)
		//}

	case 0x08:
		// 光标后退 1 位
		p.Cursor.X--
		currentRow := p.GetCursorRow()
		currentRow.CursorX--
	case 0x07:
		// do nothing
	}
	//p.Display()
}

func (p *VtTmuxScreen) Put(b byte) {
	fmt.Printf("[Put] %02x\n", b)
}

func (p *VtTmuxScreen) Unhook() {
	fmt.Printf("[Unhook]\n")
}

func (p *VtTmuxScreen) Hook(params [][]uint16, intermediates []byte, ignore bool, r rune) {
	fmt.Printf("[Hook] params=%v, intermediates=%v, ignore=%v, r=%c\n", params, intermediates, ignore, r)
}

func (p *VtTmuxScreen) OscDispatch(params [][]byte, bellTerminated bool) {
	fmt.Printf("[OscDispatch] params=%v, bellTerminated=%v\n", params, bellTerminated)
}

func (p *VtTmuxScreen) CsiDispatch(params [][]uint16, intermediates []byte, ignore bool, r rune) {
	fmt.Printf("[CsiDispatch] params=%v, intermediates=%v, ignore=%v, r=%c\n", params, intermediates, ignore, r)
	switch r {
	case 'r':
		start := params[0][0]
		end := params[1][0]
		rowsNum := int(end-start) + 1
		if len(p.rows) < rowsNum {
			rows := make([]*Row, 0, rowsNum)
			rows = append(rows, p.rows...)
			p.rows = rows
		} else {
			startRowIndex := len(p.rows) - rowsNum
			newRows := make([]*Row, 0, rowsNum)
			newRows = append(newRows, p.rows[startRowIndex:]...)
			p.rows = newRows
		}
		p.maxRows = int(end)
	case 'S':
		// 滚动翻页，暂时不处理
		// [CsiDispatch] params=[[7]], intermediates=[], ignore=false, r=S
		//
		pageUpNum := 1
		switch len(params) {
		case 0:
		case 1:
			pageUpNum = int(params[0][0])
		default:

		}
		for i := pageUpNum; i < p.maxRows; i++ {
			p.rows = append(p.rows, &Row{})
		}
	case 'H':
		y := 1
		x := 1
		switch len(params) {
		case 2:
			y = int(params[0][0])
			x = int(params[1][0])
		}
		p.Cursor.X = x
		p.Cursor.Y = y
		p.currentRowIndex = y
		currentRow := p.GetCursorRow()

		// fix tmux last line
		lastY := p.maxRows - 1
		if y == lastY && x == 1 {
			// tmux 最后一行，应该清空最后一行的数据避免显示错误
			currentRow.CursorX = 1
			currentRow.Line = nil
		}
		if y == lastY && x > 1 {
			currentRow.CursorX = x
		}

	case 'K':
		currentRow := p.GetCursorRow()
		switch len(params) {
		case 0:
			// 删除 当前 line 右边所有的字符
			if currentRow != nil {
				currentRow.EaseRightCharsAll()
			}
		case 1:
			charsNum := int(params[0][0])
			currentRow.DeleteChars(charsNum)
		default:
		}
		//fmt.Println(currentRow)
		//[CsiDispatch] params=[[8]], intermediates=[], ignore=false, r=D
		//[CsiDispatch] params=[[5]], intermediates=[], ignore=false, r=P

	case 'C':
		currentRow := p.GetCursorRow()
		switch len(params) {
		case 0:
			currentRow.CursorX++
		case 1:
			charsNum := int(params[0][0])
			currentRow.CursorX += charsNum
		default:

		}
	case 'D':
		/*
			CSI Ps D  CursorX Backward Ps Times (default = 1) (CUB).
		*/
		currentRow := p.GetCursorRow()
		switch len(params) {
		case 0:
			currentRow.CursorX--
		case 1:
			charsNum := int(params[0][0])
			currentRow.CursorX -= charsNum
		default:
		}
	case 'G':
		currentRow := p.GetCursorRow()
		switch len(params) {
		case 0:
			currentRow.CursorX = 1
		case 1:
			charsNum := int(params[0][0])
			currentRow.CursorX = charsNum
		default:
		}
	case 'P':
		/*
			CSI Ps P  Delete Ps Character(s) (default = 1) (DCH).
		*/
		currentRow := p.GetCursorRow()
		switch len(params) {
		case 0:
			currentRow.DeleteChars(1)
		case 1:
			charsNum := int(params[0][0])
			currentRow.DeleteChars(charsNum)
		default:

		}
	case '@':
		currentRow := p.GetCursorRow()
		switch len(params) {
		case 0:
			currentRow.DeleteChars(1)
		case 1:
			charsNum := int(params[0][0])
			currentRow.DeleteChars(charsNum)
		default:
		}

	}
}

func (p *VtTmuxScreen) EscDispatch(intermediates []byte, ignore bool, b byte) {
	fmt.Printf("[EscDispatch] intermediates=%v, ignore=%v, byte=%02x\n", intermediates, ignore, b)
	switch b {
	case 0x08:
		// 光标后退 1 位
		p.Cursor.X--
		currentRow := p.GetCursorRow()
		currentRow.CursorX--

	}
}

func (p *VtTmuxScreen) GetCursorRow() *Row {
	index := p.currentRowIndex - 1
	if index < 0 {
		index = 0
	}
	if index >= len(p.rows) {
		addNums := index - len(p.rows) + 1
		for i := 0; i < addNums; i++ {
			p.rows = append(p.rows, &Row{})
		}
	}
	return p.rows[index]
}

func (p *VtTmuxScreen) Display() {
	for i, r := range p.rows {
		fmt.Printf("[Display index: %d cursorX: %3d]  %s\n", i, r.CursorX, r)
	}
}

/*
	tmux
	展示 tmux bar 的字符 任何时候都可能发送 tmux bar 字符
		1、如果不是最后一行，执行逻辑是，光标位置到最后一行，展示 tmux bar 字符
		2、如果当前输入是在最后一行的上一行，则发生 \r\n ，发送 bar 字符，然后光标再往上移动 一个字符
	维护一个 活动  烈的区域

*/
