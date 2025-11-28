package terminalparser

import (
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
