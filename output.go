package terminalparser

import (
	"strings"

	"github.com/danielgatis/go-vte"
)

func ParseOutput(p []byte) []string {
	out := NewOutPutScreen()
	vtParser := vte.NewParser(&out)
	for i := range p {
		vtParser.Advance(p[i])
	}
	defer func() {
		out.Release()
	}()
	ret := make([]string, 0, 1000)
	rows := out.Rows.Values()
	for i := range rows {
		row := rows[i]
		rowStr := strings.TrimSpace(row.String())
		if rowStr == "" {
			continue
		}
		ret = append(ret, rowStr)
	}

	return ret
}

func NewOutPutScreen() OutPutScreen {
	return OutPutScreen{
		Rows:    NewTRingRowBuffer(1000),
		maxRows: 1000,
	}
}

type OutPutScreen struct {
	Rows            *TRingRowBuffer // r 创建足够多的 rows
	CurrentRowIndex int             // 根据光标设置，判断当前的row行数
	Cursor          TmuxCursor      // 默认从 （1，1） 开始 获取当前值的时候 默认需要 -1
	maxRows         int
}

func (o *OutPutScreen) Print(r rune) {
	if o.CurrentRowIndex >= o.maxRows {
		Printf("Output exceed max rows %d to Print", o.maxRows)
		return
	}
	currentRow := o.GetCursorRow()
	currentRow.Add(r)
	o.Cursor.X += 1
}

func (o *OutPutScreen) Execute(b byte) {
	switch b {
	case '\r':
		o.Cursor.X = 0
		currentRow := o.GetCursorRow()
		currentRow.CursorX = 0
	case '\n':
		o.CurrentRowIndex++
		o.Cursor.Y += 1
		if o.Rows.full {
			return
		}
		o.Rows.Append(&TRow{CursorX: 1, Line: []rune{' '}})

	case 0x08:
		// 光标后退 1 位
		o.Cursor.X--
		currentRow := o.GetCursorRow()
		currentRow.CursorX--
	case 0x07:
		// do nothing
	}
	//p.Display()
}

func (o *OutPutScreen) Put(b byte) {

}

func (o *OutPutScreen) Unhook() {

}

func (o *OutPutScreen) Hook(params [][]uint16, intermediates []byte, ignore bool, r rune) {

}

func (o *OutPutScreen) OscDispatch(params [][]byte, bellTerminated bool) {

}

func (o *OutPutScreen) CsiDispatch(params [][]uint16, intermediates []byte, ignore bool, r rune) {
}

func (o *OutPutScreen) EscDispatch(intermediates []byte, ignore bool, b byte) {
}

func (p *OutPutScreen) GetCursorRow() *TRow {
	index := p.CurrentRowIndex - 1
	if index < 0 {
		index = 0
	}
	if p.CurrentRowIndex >= p.maxRows {
		return p.Rows.Current()
	}
	rows := p.Rows.Values()
	if index >= p.maxRows {
		return p.Rows.Current()
	}
	if index < len(rows) {
		return rows[index]
	}

	return p.Rows.Current()
}

func (p *OutPutScreen) Release() {
	p.Rows.EraseAll()
}
