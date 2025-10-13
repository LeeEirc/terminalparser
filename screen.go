package terminalparser

import (
	"bytes"
	"strconv"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
)

func NewScreen(r, c int) *Screen {
	rows := NewRingRowBuffer(500)
	return &Screen{
		Rows:    rows,
		Cursor:  &Cursor{X: 1, Y: 1},
		ColLens: c,
		RowLens: r,
	}
}

type Screen struct {
	Rows *RingRowBuffer

	Cursor *Cursor

	pasteMode bool // Set bracketed paste mode, xterm. ?2004h   reset ?2004l

	title string

	buffer  bytes.Buffer
	ColLens int
	RowLens int
}

func (s *Screen) Feed(p []byte) {
	s.buffer.Write(p)
	s.TryParse()
}

func (s *Screen) TryParse() {
	remainBytes := s.buffer.Bytes()
	remain := s.parse(remainBytes)
	s.buffer.Reset()
	s.buffer.Write(remain)
}

func (s *Screen) GetRows() []*Row {
	return s.Rows.Values()
}

func (s *Screen) parse(data []byte) []byte {
	rest := data
	for len(rest) > 0 {
		code, size := utf8.DecodeRune(rest)
		if code == utf8.RuneError {
			return rest
		}

		rest = rest[size:]
		switch code {
		case ESCKey:
			code, size = utf8.DecodeRune(rest)
			if code == utf8.RuneError {
				return rest
			}
			rest = rest[size:]
			switch code {
			case '[':
				// CSI
				rest = s.parseCSISequence(rest)
				continue
			case ']':
				// OSC
				rest = s.parseOSCSequence(rest)
				continue
			default:
				if existIndex := bytes.IndexRune([]byte(string(Intermediate)), code); existIndex >= 0 {
					// ESC
					rest = s.parseIntermediate(code, rest)
					continue
				}
				if existIndex := bytes.IndexRune([]byte(string(Parameters)), code); existIndex >= 0 {

					Printf("Screen 未解析 ESC `%q` %x Parameters字符\n", code, code)
					continue
				}
				if existIndex := bytes.IndexRune([]byte(string(Uppercase)), code); existIndex >= 0 {
					Printf("Screen 未解析 ESC `%q` %x Uppercase字符\n", code, code)
					continue
				}

				if existIndex := bytes.IndexRune([]byte(string(Lowercase)), code); existIndex >= 0 {
					Printf("Screen 未解析 ESC `%q` %x Lowercase字符\n", code, code)
					continue
				}
				Printf("Screen 未解析 ESC `%q` %x\n", code, code)
			}
			continue
		case Delete:
			continue
		default:
			if existIndex := bytes.IndexRune([]byte(string(C0Control)), code); existIndex >= 0 {
				s.parseC0Sequence(code)
			} else {
				if len(s.Rows) == 0 && s.Cursor.Y == 0 {
					s.Rows = append(s.Rows, &Row{
						dataRune: make([]rune, 0, 1024),
					})
					s.Cursor.Y++
				}
				s.appendCharacter(code)
			}
			continue
		}
	}
	return rest
}

func (s *Screen) Parse(data []byte) []string {
	s.parse(data)
	ret := make([]string, 0, len(s.Rows))
	for _, row := range s.Rows {
		ret = append(ret, row.String())
	}
	return ret
}

func (s *Screen) parseC0Sequence(code rune) {
	switch code {
	case 0x07:
		//bell 忽略
	case 0x08:
		// 后退1光标
		s.Cursor.MoveLeft(1)
	case 0x0d:
		/*
			\r
		*/
		s.Cursor.X = 0
		if s.Cursor.Y > len(s.Rows) {
			s.Rows = append(s.Rows, &Row{
				dataRune: make([]rune, 0, 1024),
			})
		}
	case 0x0a:
		/*
			\n
		*/
		s.Cursor.Y++
		if s.Cursor.Y > len(s.Rows) {
			s.Rows = append(s.Rows, &Row{
				dataRune: make([]rune, 0, 1024),
			})
		}
	default:
		Printf("未处理的字符 %q %v\n", code, code)
	}

}

func (s *Screen) parseCSISequence(p []byte) []byte {
	endIndex := bytes.IndexFunc(p, IsAlphabetic)
	if endIndex == -1 {
		return p
	}
	params := []rune(string(p[:endIndex]))
	switch rune(p[endIndex]) {
	case 'Y':
		//	/*
		//		ESC Y Ps Ps
		//		          Move the cursor to given row and column.
		//	*/
		if len(p[endIndex+1:]) < 2 {
			return p[endIndex+1:]
		}
		if row, err := strconv.Atoi(string(p[endIndex+1])); err == nil {
			s.Cursor.Y = row
		}
		if col, err := strconv.Atoi(string(p[endIndex+2])); err == nil {
			s.Cursor.X = col
		}
		return p[endIndex+3:]

	}

	funcName, ok := CSIFuncMap[rune(p[endIndex])]
	if ok {
		funcName(s, params)
	} else {
		Printf("screen未处理的CSI %s %q\n", DebugString(string(params)), p[endIndex])
	}

	return p[endIndex+1:]
}

func (s *Screen) parseIntermediate(code rune, p []byte) []byte {
	switch code {
	case '(':
		terminationIndex := bytes.IndexFunc(p, func(r rune) bool {
			if insideIndex := bytes.IndexRune([]byte(string(Alphabetic)), r); insideIndex < 0 {
				return false
			}
			return true
		})
		params := p[:terminationIndex+1]
		switch string(params) {
		case "B":
			/*
				ESC ( C   Designate G0 Character Set, VT100, ISO 2022.
						  C = B  ⇒  United States (USASCII), VT100.
			*/
		}
		p = p[terminationIndex+1:]
		return p
	case ')':
		terminationIndex := bytes.IndexFunc(p, func(r rune) bool {
			if insideIndex := bytes.IndexRune([]byte(string(Alphabetic)), r); insideIndex < 0 {
				return false
			}
			return true
		})
		p = p[terminationIndex+1:]
	default:
		Printf("Screen 未解析 ESC `%q` %x Intermediate字符\n", code, code)
	}
	return p
}

func (s *Screen) parseOSCSequence(p []byte) []byte {
	if endIndex := bytes.IndexRune(p, BEL); endIndex >= 0 {
		return p[endIndex+1:]
	}

	if endIndex := bytes.IndexRune(p, ST); endIndex >= 0 {
		return p[endIndex+1:]
	}
	Printf("未处理的 parseOSCSequence")
	return p
}

func (s *Screen) appendCharacter(code rune) {
	currentRow := s.GetCursorRow()
	currentRow.changeCursorToX(s.Cursor.X)
	currentRow.appendCharacter(code)
	width := runewidth.StringWidth(string(code))
	s.Cursor.X += width
}

func (s *Screen) eraseEndToLine() {
	currentRow := s.GetCursorRow()
	currentRow.changeCursorToX(s.Cursor.X)
	currentRow.eraseRight()

}

func (s *Screen) eraseRight() {
	currentRow := s.GetCursorRow()
	currentRow.changeCursorToX(s.Cursor.X)
	currentRow.eraseRight()
}

func (s *Screen) eraseLeft() {
	Printf("Screen %s Erase Left cursor(%d，%d) 总Row数量 %d",
		UnsupportedMsg, s.Cursor.X, s.Cursor.Y, len(s.Rows))
}

func (s *Screen) eraseAbove() {
	s.Rows = s.Rows[s.Cursor.Y-1:]
}

func (s *Screen) eraseBelow() {
	s.Rows = s.Rows[:s.Cursor.Y]
}

func (s *Screen) eraseAll() {
	s.Rows = s.Rows[:0]
	//htop?
	s.Cursor.X = 0
	s.Cursor.Y = 0
}

func (s *Screen) eraseFromCursor() {
	if s.Cursor.Y > len(s.Rows) {
		s.Cursor.Y = len(s.Rows)
	}
	s.Rows = s.Rows[:s.Cursor.Y]
	currentRow := s.GetCursorRow()
	currentRow.changeCursorToX(s.Cursor.X)
	currentRow.eraseRight()
}

func (s *Screen) deleteChars(ps int) {
	currentRow := s.GetCursorRow()
	currentRow.changeCursorToX(s.Cursor.X)
	currentRow.deleteChars(ps)
}

func (s *Screen) GetCursorRow() *Row {
	if s.Cursor.Y == 0 {
		s.Cursor.Y++
	}
	if s.Rows.Len() == 0 {
		s.Rows.Append(&Row{
			dataRune: make([]rune, 0, 1024),
		})
	}
	return s.Rows.Last()
}

const UnsupportedMsg = "Unsupported"
