package terminalparser

import (
	"container/ring"
	"strings"

	"github.com/mattn/go-runewidth"
)

type Row struct {
	dataRune         []rune
	currentX         int
	currentRuneIndex int

	// fish shell 补全提示
	tipRune   []rune
	tipRecord bool
	MaxColNum int
}

func (r *Row) String() string {
	return strings.TrimSuffix(string(r.dataRune), string(r.tipRune))
}

func (r *Row) appendCharacter(code rune) {
	width := runewidth.RuneWidth(code)
	if r.currentRuneIndex < len(r.dataRune) {
		r.dataRune[r.currentRuneIndex] = code
	} else {
		r.dataRune = append(r.dataRune, code)
	}
	r.currentRuneIndex++
	r.currentX += width
	r.addTipRune(code)
}

func (r *Row) insertCharacters(data []rune) {
	result := make([]rune, len(r.dataRune)+len(data))
	copy(result, r.dataRune[:r.currentRuneIndex])
	copy(result[r.currentRuneIndex:], data)
	copy(result[r.currentRuneIndex+len(data):], r.dataRune[r.currentRuneIndex:])
	for i := range data {
		r.currentRuneIndex++
		r.currentX += runewidth.RuneWidth(data[i])
	}
	r.dataRune = result
}

func (r *Row) eraseRight() {
	r.dataRune = r.dataRune[:r.currentRuneIndex]
}

func (r *Row) deleteChars(ps int) {
	result := make([]rune, r.currentRuneIndex, len(r.dataRune))
	copy(result, r.dataRune[:r.currentRuneIndex])
	rest := r.dataRune[r.currentRuneIndex:]
	inits := ps
	for i := range rest {
		inits -= runewidth.RuneWidth(rest[i])
		if inits == 0 {
			result = append(result, rest[i+1:]...)
			break
		}
	}
	r.dataRune = result
}

func (r *Row) changeCurrentRuneIndex() {
	if r.currentX < 0 {
		r.currentX = 0
	}
	currentRuneIndex := 0
	for i := range r.dataRune {
		currentRuneIndex += runewidth.RuneWidth(r.dataRune[i])
		if currentRuneIndex > r.currentX {
			r.currentRuneIndex = i
			return
		}
	}
	r.currentRuneIndex = len(r.dataRune)
}

func (r *Row) changeCursorToX(x int) {
	if r.currentX == x {
		return
	}
	r.currentX = x
	r.changeCurrentRuneIndex()
}

func (r *Row) addTipRune(code rune) {
	if r.tipRecord {
		r.tipRune = append(r.tipRune, code)
	}

}

type RingRowBuffer struct {
	start   *ring.Ring
	current *ring.Ring
	full    bool
	size    int
}

func NewRingRowBuffer(size int) *RingRowBuffer {
	r := ring.New(size)
	return &RingRowBuffer{
		start:   r,
		current: r,
		full:    false,
		size:    size,
	}
}

func (r *RingRowBuffer) Len() int {
	return r.current.Len()
}

func (r *RingRowBuffer) Append(v *Row) {
	r.current.Value = v
	r.current = r.current.Next()
	if r.current == r.start {
		r.full = true
	}
}

func (r *RingRowBuffer) Values() []*Row {
	var vals []*Row

	if r.full {
		r.current.Do(func(v any) {
			if v != nil {
				vals = append(vals, v.(*Row))
			}
		})
	} else {
		p := r.start
		for p != r.current {
			if p.Value != nil {
				vals = append(vals, p.Value.(*Row))
			}
			p = p.Next()
		}
	}
	return vals
}

func (r *RingRowBuffer) Last() *Row {
	prev := r.current.Prev()
	if prev.Value == nil {
		prev.Value = &Row{dataRune: make([]rune, 0, 1024)}
	}
	return prev.Value.(*Row)
}

func (r *RingRowBuffer) Current() *Row {
	return r.Last()
}

func (r *RingRowBuffer) EraseAll() {
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

func (r *RingRowBuffer) EraseAbove(idx int) {
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
	keep := vals[idx:]
	r.rebuild(keep)
}

func (rb *RingRowBuffer) EraseBelow(idx int) {
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

func (rb *RingRowBuffer) rebuild(ordered []*Row) {
	rb.EraseAll()
	for _, v := range ordered {
		rb.Append(v)
	}
}
