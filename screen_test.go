package terminalparser

import (
	"testing"
)

func TestParser(t *testing.T) {

	var tests = []struct {
		in  string
		out []string
	}{
		{
			"\r(reverse-i-search)`': \x1b[K\b\b\bp': ps -a\b\b\b\b\b\r\x1b[11@[root@FAT00400000 koko-allinone]#\x1b[C\x1b[C\x1b[C\x1b[C\x1b[C\x1b[C",
			[]string{"[root@FAT00400000 koko-allinone]# ps -a"},
		},
	}

	for _, test := range tests {
		s := Screen{Cursor: &Cursor{}}
		out := s.Parse([]byte(test.in))
		if !testEq(out, test.out) {
			t.Errorf("expected %#v got %#v", test.out, out)
		}
	}
}

func testEq(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
