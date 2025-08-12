package terminalparser

import (
	"github.com/danielgatis/go-vte"
)

func NewMongoShParser() *MongoShParser {
	tmuxScreen := MongoShScreen{
		Rows:            make([]*VTRow, 0, 10),
		CurrentRowIndex: 0,
		Cursor:          TmuxCursor{1, 1},
	}
	vtParser := vte.NewParser(&tmuxScreen)
	return &MongoShParser{VtParser: vtParser, TmuxScreen: &tmuxScreen}
}

type MongoShParser struct {
	VtParser   *vte.Parser
	TmuxScreen *MongoShScreen
}

func (t *MongoShParser) Feed(p []byte) {
	for i := range p {
		t.VtParser.Advance(p[i])
	}
}

type MongoShScreen struct {
	Rows            []*VTRow   // r 创建足够多的 rows
	CurrentRowIndex int        // 根据光标设置，判断当前的row行数
	Cursor          TmuxCursor // 默认从 （1，1） 开始 获取当前值的时候 默认需要 -1
	maxRows         int
}

func (p *MongoShScreen) Print(r rune) {
	//Printf("[Print] %c\n", r)
	currentRow := p.GetCursorRow()
	currentRow.Add(r)
	p.Cursor.X += 1
	Println(currentRow)
}

func (p *MongoShScreen) Execute(b byte) {
	Printf("[Execute] %02x\n", b)
	switch b {
	case '\r':
		p.Cursor.X = 0
		currentRow := p.GetCursorRow()
		currentRow.CursorX = 0
		//p.Cursor.Y += 1
	case '\n':
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

func (p *MongoShScreen) Put(b byte) {
	Printf("[Put] %02x\n", b)
}

func (p *MongoShScreen) Unhook() {
	Printf("[Unhook]\n")
}

func (p *MongoShScreen) Hook(params [][]uint16, intermediates []byte, ignore bool, r rune) {
	Printf("[Hook] params=%v, intermediates=%v, ignore=%v, r=%c\n", params, intermediates, ignore, r)
}

func (p *MongoShScreen) OscDispatch(params [][]byte, bellTerminated bool) {
	Printf("[OscDispatch] params=%v, bellTerminated=%v\n", params, bellTerminated)
}

func (p *MongoShScreen) CsiDispatch(params [][]uint16, intermediates []byte, ignore bool, r rune) {
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
			currentRow.DeleteChars(charsNum)
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

func (p *MongoShScreen) EscDispatch(intermediates []byte, ignore bool, b byte) {
	Printf("[EscDispatch] intermediates=%v, ignore=%v, byte=%02x\n", intermediates, ignore, b)
	switch b {
	case 0x08:
		// 光标后退 1 位
		p.Cursor.X--
		currentRow := p.GetCursorRow()
		currentRow.CursorX--

	}
}

func (p *MongoShScreen) GetCursorRow() *VTRow {
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
