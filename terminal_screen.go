package terminalparser

import (
	"container/ring"
	"strings"

	"github.com/danielgatis/go-vte"
	"github.com/mattn/go-runewidth"
)

func NewTerminalParser() *TerminalParser {
	t := TerminalScreen{
		Rows:   NewTRingRowBuffer(1000),
		Cursor: TmuxCursor{1, 1},
	}
	vtParser := vte.NewParser(&t)
	return &TerminalParser{
		VtParser: vtParser,
		TScreen:  &t,
	}
}

type TRow struct {
	Line    []rune
	CursorX int // 当前行光标的位置

	currentRuneIndex int
}

func (r *TRow) String() string {
	return strings.TrimSpace(string(r.Line))
}

func (r *TRow) InsertSpaces(spaces int) {
	spacesRunes := make([]rune, spaces)
	for i := 0; i < spaces; i++ {
		spacesRunes[i] = ' '
	}
	index := r.GetCurrentX()
	newLine := make([]rune, len(r.Line))
	copy(newLine, r.Line[:index])
	copy(newLine[index:], spacesRunes)
	maxIndex := index + spaces
	if maxIndex <= len(r.Line) {
		copy(newLine[index+spaces:], r.Line[index:])
	}
	r.Line = newLine
}

func (r *TRow) Add(c rune) {
	index := r.GetCurrentX()
	cWidth := runewidth.RuneWidth(c)
	Printf("CursorX=%d, index=%d", r.CursorX, index)
	if len(r.Line) > index {
		if r.Line == nil {
			r.Line = make([]rune, index+1)
		}
		r.Line[index] = c
	} else {
		r.Line = append(r.Line, c)
	}
	r.CursorX += cWidth
	r.currentRuneIndex++
}

func (r *TRow) LineX() int {
	n := 0
	for i := len(r.Line) - 1; i >= 0; i-- {
		n += runewidth.RuneWidth(r.Line[i])
	}
	return n
}

func (r *TRow) EaseRightCharsAll() {
	index := r.GetCurrentX()
	newLine := make([]rune, 0, len(r.Line))
	line := r.Line
	if len(line) > index {
		line = r.Line[:index]
	}
	newLine = append(newLine, line...)
	r.Line = newLine
}

func (r *TRow) EaseAll() {
	r.CursorX = 1
	r.currentRuneIndex = 0
	r.Line = nil
}

func (r *TRow) DeleteChars(i int) {
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

func (r *TRow) DeleteAllLeft() {
	index := r.GetCurrentX()
	newLine := make([]rune, 0, len(r.Line))
	line := r.Line
	if len(line) > index {
		newLine = append(newLine, line[:index]...)
	}
	r.Line = newLine
}

func (r *TRow) GetCurrentX() int {
	index := r.CursorX - 1
	if index < 0 {
		index = 0
		r.CursorX = 1
		r.currentRuneIndex = 0
		return r.currentRuneIndex
	}
	posX := 1
	r.currentRuneIndex = 0
	for r.currentRuneIndex < len(r.Line) {
		if posX == r.CursorX {
			break
		}
		cWidth := runewidth.RuneWidth(r.Line[r.currentRuneIndex])
		posX += cWidth
		r.currentRuneIndex++
	}
	return r.currentRuneIndex
}

func (r *TRow) ChangeCursorToX(x int) {
	if r.CursorX == x {
		return
	}
	r.CursorX = x
}

type TerminalParser struct {
	VtParser *vte.Parser
	TScreen  *TerminalScreen
}

func (t *TerminalParser) Feed(p []byte) {
	for i := range p {
		t.VtParser.Advance(p[i])
	}

}

type TerminalScreen struct {
	Rows   *TRingRowBuffer
	Cursor TmuxCursor // 默认从 （1，1） 开始 获取当前值的时候 默认需要 -1
}

func (t *TerminalScreen) eraseBelow() {
	t.Rows.EraseBelow(t.Cursor.Y - 1)

}

func (t *TerminalScreen) eraseAbove() {
	t.Rows.EraseAbove(t.Cursor.Y - 1)
}

func (t *TerminalScreen) eraseAll() {
	t.Rows.EraseAll()
}

func (t *TerminalScreen) increaseCursorY() {
	if t.Rows.full {
		t.Cursor.Y = t.Rows.Len()
		return
	}
	t.Cursor.Y++
}

func (t *TerminalScreen) Print(r rune) {
	currentRow := t.GetCursorRow()
	currentRow.Add(r)
	t.Cursor.X += runewidth.RuneWidth(r)
	Println("Print Current row: ", currentRow)
}

func (t *TerminalScreen) Execute(b byte) {
	Printf("[Execute] %02x", b)
	switch b {
	case '\r':
		t.Cursor.X = 1
		currentRow := t.GetCursorRow()
		currentRow.CursorX = 1
		currentRow.currentRuneIndex = 0
	case '\n':
		t.Rows.Append(&TRow{
			CursorX: 1,
		})
		t.increaseCursorY()

	case 0x08:
		// 光标后退 1 位
		t.Cursor.X--
		currentRow := t.GetCursorRow()
		currentRow.CursorX--
		currentRow.currentRuneIndex--
	case 0x07:
		//
	default:
		Printf("Unexpect Execute: %02x", b)
	}
}

func (t *TerminalScreen) Put(b byte) {
	Printf("[Put] %02x", b)
}

func (t *TerminalScreen) Unhook() {
	Printf("[Unhook]")
}

func (t *TerminalScreen) Hook(params [][]uint16, intermediates []byte, ignore bool, r rune) {
	Printf("[Hook] params=%v, intermediates=%v, ignore=%v, r=%c", params, intermediates, ignore, r)
}

func (t *TerminalScreen) OscDispatch(params [][]byte, bellTerminated bool) {
	//Printf("OscDispatch params=%v, bellTerminated= %v", params, bellTerminated)
}

func (t *TerminalScreen) CsiDispatch(params [][]uint16, intermediates []byte, ignore bool, r rune) {
	Printf("[CsiDispatch] params=%v, intermediates=%v, ignore=%v, r=%c", params, intermediates, ignore, r)
	switch r {
	case 'm', 'h':
	case 'A':
		switch len(params) {
		case 0:
			// default =1
			t.Cursor.Y--
		case 1:
			charsNum := int(params[0][0])
			t.Cursor.Y -= charsNum
		default:
			Printf("Unexpect Execute A: params=%v", params)
		}
	case 'B':
		switch len(params) {
		case 0:
			t.Cursor.Y++
		case 1:
			charsNum := int(params[0][0])
			t.Cursor.Y += charsNum
		default:
			Printf("Unexpect Execute B: params=%v", params)
		}
	case 'C':
		currentRow := t.GetCursorRow()
		switch len(params) {
		case 0:
			t.Cursor.X++
			currentRow.CursorX = t.Cursor.X
		case 1:
			charsNum := int(params[0][0])
			t.Cursor.X += charsNum
			currentRow.CursorX = t.Cursor.X
		default:
			Printf("Unexpect Execute C: params=%v", params)
		}

	case 'D':
		/*
			CSI Ps D  CursorX Backward Ps Times (default = 1) (CUB).
		*/
		currentRow := t.GetCursorRow()
		switch len(params) {
		case 0:
			t.Cursor.X--
			currentRow.CursorX = t.Cursor.X
		case 1:
			charsNum := int(params[0][0])
			t.Cursor.X -= charsNum
			currentRow.CursorX = t.Cursor.X
		default:
			Printf("Unexpect Execute D: params=%v", params)
		}
	case 'J':
		switch len(params) {
		case 0:
			// cursor 保持不动
			// Erase from current position to end
			t.eraseBelow()
		case 1:
			charsNum := int(params[0][0])
			if charsNum == 1 {
				// [1J = Erase from beginning to current position (inclusive)
				t.eraseAbove()
			} else if charsNum == 2 {
				t.eraseAll()
			} else {
				Printf("Unexpect Execute J params=%v", params)
			}
		default:
			Printf("Unexpect Execute J params=%v", params)
		}
	case 'K':
		currentRow := t.GetCursorRow()
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
	case 'H':
		y := 1
		x := 1
		switch len(params) {
		case 2:
			y = int(params[0][0])
			x = int(params[1][0])
		}
		t.Cursor.X = x
		t.Cursor.Y = y
		currentRow := t.GetCursorRow()
		if currentRow == nil {
			Println("H Can not get current row ")
		} else {
			currentRow.CursorX = x
		}
	case 'P':
		currentRow := t.GetCursorRow()
		switch len(params) {
		case 0:

			if currentRow != nil {
				currentRow.DeleteChars(1)
			}
		case 1:
			charsNum := int(params[0][0])
			currentRow.DeleteChars(charsNum)
		default:

		}
	case '@':
		currentRow := t.GetCursorRow()
		switch len(params) {
		case 0:
			currentRow.InsertSpaces(1)
		case 1:
			charsNum := int(params[0][0])
			currentRow.InsertSpaces(charsNum)
		default:
		}
	default:
		Printf("Unhandle [CsiDispatch] params=%v, intermediates=%v, ignore=%v, r=%c",
			params, intermediates, ignore, r)
	}
}

func (t *TerminalScreen) EscDispatch(intermediates []byte, ignore bool, b byte) {
}

func (t *TerminalScreen) GetCursorRow() *TRow {
	vals := t.Rows.Values()
	index := t.Cursor.Y - 1
	if index < 0 {
		index = 0
	}
	if len(vals) > 0 && index < len(vals) {
		index = index - 1
		if index < 0 {
			index = 0
		}
		return vals[index]
	}
	return t.Rows.Current()
}

type TRingRowBuffer struct {
	start   *ring.Ring
	current *ring.Ring
	full    bool
	size    int
}

func NewTRingRowBuffer(size int) *TRingRowBuffer {
	r := ring.New(size)
	return &TRingRowBuffer{
		start:   r,
		current: r,
		full:    false,
		size:    size,
	}
}

func (r *TRingRowBuffer) Len() int {
	return r.current.Len()
}

func (r *TRingRowBuffer) Append(v *TRow) {
	r.current.Value = v
	r.current = r.current.Next()
	if r.current == r.start {
		r.full = true
	}
}

func (r *TRingRowBuffer) Values() []*TRow {
	vals := make([]*TRow, 0, 1000)
	if r.full {
		r.current.Do(func(v any) {
			if v != nil {
				vals = append(vals, v.(*TRow))
			}
		})
	} else {
		p := r.start
		for p != r.current {
			if p.Value != nil {
				vals = append(vals, p.Value.(*TRow))
			}
			p = p.Next()
		}
	}
	return vals
}

func (r *TRingRowBuffer) ForEach(fn func(*TRow)) {
	if r.full {
		r.current.Do(func(v any) {
			if v != nil {
				fn(v.(*TRow))
			}
		})
	} else {
		p := r.start
		for p != r.current {
			if p.Value != nil {
				fn(p.Value.(*TRow))
			}
			p = p.Next()
		}
	}
}

func (r *TRingRowBuffer) Last() *TRow {
	prev := r.current.Prev()
	if prev.Value == nil {
		prev.Value = &TRow{Line: make([]rune, 0, 10),
			CursorX: 1}
	}
	return prev.Value.(*TRow)
}

func (r *TRingRowBuffer) Current() *TRow {
	return r.Last()
}

func (r *TRingRowBuffer) EraseAll() {
	// 将所有 Value 清空
	p := r.start
	for i := 0; i < r.size; i++ {
		p.Value = nil
		p = p.Next()
	}
	// 重置指针与状态
	r.current = r.start
	r.full = false
}

func (r *TRingRowBuffer) EraseAbove(idx int) {
	vals := r.Values()
	n := len(vals)
	if n == 0 {
		return
	}
	if idx <= 0 {
		// 不删
		return
	}
	if idx >= n {
		r.EraseAll()
		return
	}
	valNils := vals[:idx]
	for i := idx; i < n; i++ {
		valNils[i].Line = make([]rune, 0)
		valNils[i].CursorX = 0
	}
	keep := vals[idx:]
	r.rebuild(keep)
}

func (rb *TRingRowBuffer) EraseBelow(idx int) {
	vals := rb.Values()
	n := len(vals)
	if n == 0 {
		return
	}
	if idx <= 0 {
		// 不删
		return
	}
	if idx >= n {
		return
	}
	keep := vals[:idx]
	rb.rebuild(keep)
}

func (rb *TRingRowBuffer) rebuild(ordered []*TRow) {
	rb.EraseAll()
	for _, v := range ordered {
		rb.Append(v)
	}
}
