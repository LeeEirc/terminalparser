package terminalparser

import (
	"os"
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

func TestParse(t *testing.T) {
	outfile := "/Users/eric/Documents/codes/fit2cloud/terminalparser/_test/output_windows.txt"
	s := NewScreen(60, 80)
	buf, _ := os.ReadFile(outfile)
	s.Parse(buf)
	for i := range s.Rows {
		t.Logf("Row %d: %s", i, s.Rows[i].String())
	}
	t.Logf("%+v", s.Cursor)
}
