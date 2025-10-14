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
	ret := make([]string, 0, 10)
	for i := range out.Rows {
		row := out.Rows[i]
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
		Rows: make([]*TmuxRow, 0, 10),
	}
}

type OutPutScreen struct {
	Rows            []*TmuxRow // r 创建足够多的 rows
	CurrentRowIndex int        // 根据光标设置，判断当前的row行数
	Cursor          TmuxCursor // 默认从 （1，1） 开始 获取当前值的时候 默认需要 -1
	maxRows         int
}

func (o *OutPutScreen) Print(r rune) {
	currentRow := o.GetCursorRow()
	currentRow.Add(r)
	o.Cursor.X += 1
}

func (o *OutPutScreen) Execute(b byte) {
	switch b {
	case '\r':
		o.Cursor.X = 0
	case '\n':
		o.CurrentRowIndex++
		o.Cursor.Y += 1
		if len(o.Rows) <= o.CurrentRowIndex {
			o.Rows = append(o.Rows, &TmuxRow{})
		}

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

func (p *OutPutScreen) GetCursorRow() *TmuxRow {
	index := p.CurrentRowIndex - 1
	if index < 0 {
		index = 0
	}
	if index >= len(p.Rows) {
		addNums := index - len(p.Rows) + 1
		for i := 0; i < addNums; i++ {
			p.Rows = append(p.Rows, &TmuxRow{})
		}
	}
	return p.Rows[index]
}

func (p *OutPutScreen) Release() {
	for i := range p.Rows {
		row := p.Rows[i]
		row.Line = nil
	}
	p.Rows = nil
}
