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
	width := runewidth.StringWidth(string(code))
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
		r.currentX += runewidth.StringWidth(string(data[i]))
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
		inits -= runewidth.StringWidth(string(rest[i]))
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
		currentRuneIndex += runewidth.StringWidth(string(r.dataRune[i]))
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

func (r *Row) startRecord() {
	r.tipRecord = true
	r.tipRune = make([]rune, 0, 100)
}

func (r *Row) stopRecord() {
	if !r.tipRecord {
		r.tipRune = make([]rune, 0, 100)
	}
	r.tipRecord = false
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

func (rb *RingRowBuffer) Append(v *Row) {
	rb.current.Value = v
	rb.current = rb.current.Next()
	if rb.current == rb.start {
		rb.full = true
	}
}

func (rb *RingRowBuffer) Values() []*Row {
	var vals []*Row

	if rb.full {
		rb.current.Do(func(v any) {
			if v != nil {
				vals = append(vals, v.(*Row))
			}
		})
	} else {
		p := rb.start
		for p != rb.current {
			if p.Value != nil {
				vals = append(vals, p.Value.(*Row))
			}
			p = p.Next()
		}
	}
	return vals
}

func (rb *RingRowBuffer) Last() *Row {
	prev := rb.current.Prev()
	return prev.Value.(*Row)
}
