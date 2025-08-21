package terminalparser

import (
	"strings"

	"github.com/danielgatis/go-vte"
)

func NewUSqlParser() *USqlParser {
	tmuxScreen := USqlScreen{
		Rows:            make([]*VTRow, 0, 10),
		CurrentRowIndex: 0,
		Cursor:          TmuxCursor{1, 1},
	}
	vtParser := vte.NewParser(&tmuxScreen)
	return &USqlParser{VtParser: vtParser, TmuxScreen: &tmuxScreen}
}

type USqlParser struct {
	VtParser   *vte.Parser
	TmuxScreen *USqlScreen
}

func (t *USqlParser) Feed(p []byte) {
	for i := range p {
		t.VtParser.Advance(p[i])
	}
}

// (1, 1) 初始位置 左上角的位置，x 代表列，y 代表行数

type VTRow struct {
	Line    []rune
	CursorX int // 当前行光标的位置
}

func (r *VTRow) InsertChars(i int) {
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

func (r *VTRow) String() string {
	return strings.TrimSpace(string(r.Line))
}

func (r *VTRow) Add(c rune) {
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

func (r *VTRow) MoveLeftCurse(i int) {
	r.CursorX -= i
}

func (r *VTRow) MoveRightCurse(i int) {
	r.CursorX += i
}

func (r *VTRow) EaseRightCharsAll() {
	index := r.GetCurrentX()
	newLine := make([]rune, 0, len(r.Line))
	line := r.Line
	if len(line) > index {
		line = r.Line[:index]
	}
	newLine = append(newLine, line...)
	r.Line = newLine
}

func (r *VTRow) EaseAll() {
	newLine := make([]rune, 0, len(r.Line))
	r.CursorX = 0
	r.Line = newLine
}

func (r *VTRow) DeleteChars(i int) {
	index := r.GetCurrentX()
	newLine := make([]rune, 0, len(r.Line))
	line := r.Line
	if len(line) > index {
		newLine = append(newLine, line[:index+1]...)
		rest := line[index+1:]
		if len(rest) >= i {
			newLine = append(newLine, rest[i:]...)
		}
	} else {
		newLine = append(newLine, line...)
	}
	r.Line = newLine
}

func (r *VTRow) DeleteAllLeft() {
	index := r.GetCurrentX()
	newLine := make([]rune, 0, len(r.Line))
	line := r.Line
	if len(line) > index {
		newLine = append(newLine, line[:index]...)
	}
	r.Line = newLine
}

func (r *VTRow) GetCurrentX() int {
	index := r.CursorX - 1
	if index < 0 {
		index = 0
		r.CursorX = 1
	}
	return index
}

type USqlScreen struct {
	Rows            []*VTRow   // r 创建足够多的 rows
	CurrentRowIndex int        // 根据光标设置，判断当前的row行数
	Cursor          TmuxCursor // 默认从 （1，1） 开始 获取当前值的时候 默认需要 -1
	maxRows         int
}

func (p *USqlScreen) Print(r rune) {
	//fmt.fmt.Printf("[Print] %c\n", r)
	currentRow := p.GetCursorRow()
	currentRow.Add(r)
	p.Cursor.X += 1
	Println(currentRow)
}

func (p *USqlScreen) Execute(b byte) {
	Printf("[Execute] %02x\n", b)
	switch b {
	case 0x0d: //'\r'
		p.Cursor.X = 0
		currentRow := p.GetCursorRow()
		currentRow.CursorX = 0
		//p.Cursor.Y += 1
	case 0x0a: // '\n'
		p.CurrentRowIndex++
		p.Cursor.Y += 1
		if len(p.Rows) <= p.CurrentRowIndex {
			p.Rows = append(p.Rows, &VTRow{})
		}

	case 0x08:
		// 光标后退 1 位
		p.Cursor.X--
		currentRow := p.GetCursorRow()
		currentRow.CursorX--
	case 0x07:
		// do nothing
	}
}

func (p *USqlScreen) Put(b byte) {
	Printf("[Put] %02x\n", b)
}

func (p *USqlScreen) Unhook() {
	Printf("[Unhook]\n")
}

func (p *USqlScreen) Hook(params [][]uint16, intermediates []byte, ignore bool, r rune) {
	Printf("[Hook] params=%v, intermediates=%v, ignore=%v, r=%c\n", params, intermediates, ignore, r)
}

func (p *USqlScreen) OscDispatch(params [][]byte, bellTerminated bool) {
	Printf("[OscDispatch] params=%v, bellTerminated=%v\n", params, bellTerminated)
}

func (p *USqlScreen) CsiDispatch(params [][]uint16, intermediates []byte, ignore bool, r rune) {
	Printf("[CsiDispatch] params=%v, intermediates=%v, ignore=%v, r=%c\n", params, intermediates, ignore, r)
	switch r {
	case 'r':
		start := params[0][0]
		end := params[1][0]
		rowsNum := int(end-start) + 1
		if len(p.Rows) < rowsNum {
			rows := make([]*VTRow, 0, rowsNum)
			rows = append(rows, p.Rows...)
			p.Rows = rows
		} else {
			startRowIndex := len(p.Rows) - rowsNum
			newRows := make([]*VTRow, 0, rowsNum)
			newRows = append(newRows, p.Rows[startRowIndex:]...)
			p.Rows = newRows
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
			p.Rows = append(p.Rows, &VTRow{})
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
		p.CurrentRowIndex = y
		currentRow := p.GetCursorRow()

		// fix tmux last line
		lastY := p.maxRows - 1
		if y == lastY && x == 1 {
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
			switch charsNum {
			case 0:
			case 1:
				currentRow.DeleteChars(charsNum)
			case 2:
				currentRow.EaseAll()
			default:
				currentRow.DeleteChars(charsNum)
			}

		default:
		}

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
	case 'J':
		currentRow := p.GetCursorRow()
		switch len(params) {
		case 0:
			if currentRow != nil {
				currentRow.EaseRightCharsAll()
			}
		case 1:
			charsNum := int(params[0][0])
			if charsNum == 0 {
				currentRow.EaseRightCharsAll()
			} else {
				currentRow.DeleteChars(charsNum)
			}

		default:

		}

	}
}

func (p *USqlScreen) EscDispatch(intermediates []byte, ignore bool, b byte) {
	Printf("[EscDispatch] intermediates=%v, ignore=%v, byte=%02x\n", intermediates, ignore, b)
	switch b {
	case 0x08:
		// 光标后退 1 位
		p.Cursor.X--
		currentRow := p.GetCursorRow()
		currentRow.CursorX--

	}
}

func (p *USqlScreen) GetCursorRow() *VTRow {
	index := p.CurrentRowIndex - 1
	if index < 0 {
		index = 0
	}
	if len(p.Rows) > 5000 {
		// 减少内存
		newRows := make([]*VTRow, 0, 2000)
		start := len(p.Rows) - 2000
		newRows = append(newRows, p.Rows[start:]...)
		p.Rows = newRows
	}
	if index >= len(p.Rows) {
		addNums := index - len(p.Rows) + 1
		for i := 0; i < addNums; i++ {
			p.Rows = append(p.Rows, &VTRow{})
		}
	}

	return p.Rows[index]
}

/*
[CsiDispatch] params=[], intermediates=[], ignore=false, r=J
[CsiDispatch] params=[[2]], intermediates=[], ignore=false, r=K
	usql 每次都会清空当前行，然后再填充所有的字符
*/
