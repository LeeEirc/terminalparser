package terminalparser

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

func DebugString(p string) string {
	var s strings.Builder
	for _, v := range []rune(p) {
		if unicode.IsPrint(v) {
			s.WriteRune(v)
		} else {
			s.WriteString(fmt.Sprintf("%q", v))
		}
	}
	return s.String()
}

// ASCII helper checks to avoid per-call allocations in hot paths.
func IsAlphabetic(r rune) bool {
	// Alphabetic range per table: 0x40 ('@') to 0x7E ('~')
	return r >= 0x40 && r <= 0x7E
}

func isIntermediate(r rune) bool { // 0x20..0x2F
	return r >= 0x20 && r <= 0x2F
}

func isParameters(r rune) bool { // 0x30..0x3F
	return r >= 0x30 && r <= 0x3F
}

func isUppercase(r rune) bool { // 0x40..0x5F
	return r >= 0x40 && r <= 0x5F
}

func isLowercase(r rune) bool { // 0x60..0x7E
	return r >= 0x60 && r <= 0x7E
}

func isC0Control(r rune) bool { // 0x00..0x1F
	return r >= 0x00 && r <= 0x1F
}

func ReadRunePacket(p []byte) (code rune, rest []byte) {
	r, l := utf8.DecodeRune(p)
	if r == utf8.RuneError {
		return utf8.RuneError, p
	}
	return r, p[l:]
}
