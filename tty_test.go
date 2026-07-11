package terminalparser

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
)

func TestParseInterpretsVTSequences(t *testing.T) {
	rows, err := Parse([]byte("hello \x1b[31mred\x1b[0m\r\nworld"))
	if err != nil {
		t.Fatal(err)
	}

	joined := strings.Join(rows, "\n")
	if strings.Contains(joined, "\x1b[") {
		t.Fatalf("formatted output contains an escape sequence: %q", joined)
	}
	if !strings.Contains(joined, "hello red") || !strings.Contains(joined, "world") {
		t.Fatalf("formatted output lost text: %q", joined)
	}
}

func TestTerminalState(t *testing.T) {
	terminal, err := New(WithSize(20, 4))
	if err != nil {
		t.Fatal(err)
	}
	defer terminal.Close()

	if _, err := fmt.Fprint(terminal, "\x1b]2;build log\x07abc"); err != nil {
		t.Fatal(err)
	}

	title, err := terminal.Title()
	if err != nil {
		t.Fatal(err)
	}
	if title != "build log" {
		t.Fatalf("Title() = %q, want %q", title, "build log")
	}

	x, err := terminal.CursorX()
	if err != nil {
		t.Fatal(err)
	}
	if x != 3 {
		t.Fatalf("CursorX() = %d, want 3", x)
	}
}

func TestCursorRowUsesCursorInsteadOfLastScreenRow(t *testing.T) {
	terminal, err := New(WithSize(24, 4), WithMaxScrollback(0))
	if err != nil {
		t.Fatal(err)
	}
	defer terminal.Close()

	if _, err := terminal.Write([]byte(
		"\x1b[?1049h" +
			"\x1b[1;1Huser@tmux$ echo hi" +
			"\x1b[4;1H[0] bash status" +
			"\x1b[1;19H",
	)); err != nil {
		t.Fatal(err)
	}

	row, err := terminal.CursorRow()
	if err != nil {
		t.Fatal(err)
	}
	if row != "user@tmux$ echo hi" {
		t.Fatalf("CursorRow() = %q", row)
	}
}

func TestCursorRowRestoresPrimaryScreen(t *testing.T) {
	terminal, err := New(WithSize(24, 4), WithMaxScrollback(0))
	if err != nil {
		t.Fatal(err)
	}
	defer terminal.Close()

	if _, err := terminal.Write([]byte("primary$ \x1b[?1049halternate$ \x1b[?1049l")); err != nil {
		t.Fatal(err)
	}
	row, err := terminal.CursorRow()
	if err != nil {
		t.Fatal(err)
	}
	if row != "primary$" {
		t.Fatalf("CursorRow() = %q", row)
	}
}

func TestSizeTracksResize(t *testing.T) {
	terminal, err := New(WithSize(80, 24))
	if err != nil {
		t.Fatal(err)
	}
	defer terminal.Close()

	columns, rows, err := terminal.Size()
	if err != nil {
		t.Fatal(err)
	}
	if columns != 80 || rows != 24 {
		t.Fatalf("Size() = %dx%d", columns, rows)
	}
	if err := terminal.Resize(132, 43, 0, 0); err != nil {
		t.Fatal(err)
	}
	columns, rows, err = terminal.Size()
	if err != nil {
		t.Fatal(err)
	}
	if columns != 132 || rows != 43 {
		t.Fatalf("Size() after Resize = %dx%d", columns, rows)
	}
}

func TestAlternateScreen(t *testing.T) {
	terminal, err := New()
	if err != nil {
		t.Fatal(err)
	}
	defer terminal.Close()

	if _, err := terminal.Write([]byte("\x1b[?1049h")); err != nil {
		t.Fatal(err)
	}
	alternate, err := terminal.IsScreenAlternate()
	if err != nil {
		t.Fatal(err)
	}
	if !alternate {
		t.Fatal("alternate screen is not active")
	}

	if _, err := terminal.Write([]byte("\x1b[?1049l")); err != nil {
		t.Fatal(err)
	}
	alternate, err = terminal.IsScreenAlternate()
	if err != nil {
		t.Fatal(err)
	}
	if alternate {
		t.Fatal("alternate screen is still active")
	}
}

func TestCloseIsIdempotent(t *testing.T) {
	terminal, err := New()
	if err != nil {
		t.Fatal(err)
	}
	if err := terminal.Close(); err != nil {
		t.Fatal(err)
	}
	if err := terminal.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := terminal.Write([]byte("late")); !errors.Is(err, ErrClosed) {
		t.Fatalf("Write() error = %v, want ErrClosed", err)
	}
	if _, err := terminal.String(); !errors.Is(err, ErrClosed) {
		t.Fatalf("String() error = %v, want ErrClosed", err)
	}
	if _, err := terminal.CursorRow(); !errors.Is(err, ErrClosed) {
		t.Fatalf("CursorRow() error = %v, want ErrClosed", err)
	}
	if _, _, err := terminal.Size(); !errors.Is(err, ErrClosed) {
		t.Fatalf("Size() error = %v, want ErrClosed", err)
	}
}

func TestConcurrentAccess(t *testing.T) {
	terminal, err := New(WithSize(80, 24))
	if err != nil {
		t.Fatal(err)
	}
	defer terminal.Close()

	var group sync.WaitGroup
	for range 8 {
		group.Add(1)
		go func() {
			defer group.Done()
			for range 20 {
				if _, err := terminal.Write([]byte("x")); err != nil {
					t.Errorf("Write() error: %v", err)
					return
				}
				if _, err := terminal.CursorX(); err != nil {
					t.Errorf("CursorX() error: %v", err)
					return
				}
				if _, err := terminal.CursorRow(); err != nil {
					t.Errorf("CursorRow() error: %v", err)
					return
				}
			}
		}()
	}
	group.Wait()
}

func TestInvalidSize(t *testing.T) {
	if _, err := New(WithSize(0, 24)); !errors.Is(err, ErrInvalidSize) {
		t.Fatalf("New() error = %v, want ErrInvalidSize", err)
	}
	if _, err := NewTerminalVT(80, 0); !errors.Is(err, ErrInvalidSize) {
		t.Fatalf("NewTerminalVT() error = %v, want ErrInvalidSize", err)
	}
}

func TestParseOutputCompatibility(t *testing.T) {
	rows := ParseOutput([]byte("  first  \r\n\r\n second \r\n"))
	if len(rows) != 2 || rows[0] != "first" || rows[1] != "second" {
		t.Fatalf("ParseOutput() = %#v", rows)
	}
}

func TestParseIncludesScrollback(t *testing.T) {
	var input strings.Builder
	for index := range 30 {
		fmt.Fprintf(&input, "line-%02d\r\n", index)
	}

	rows, err := Parse(
		[]byte(input.String()),
		WithSize(20, 3),
		WithMaxScrollback(100),
	)
	if err != nil {
		t.Fatal(err)
	}
	output := strings.Join(rows, "\n")
	if !strings.Contains(output, "line-00") || !strings.Contains(output, "line-29") {
		t.Fatalf("Parse() did not retain the full configured history: %q", output)
	}
}

func benchmarkTerminalWithScrollback(b *testing.B) *TerminalVT {
	b.Helper()
	terminal, err := New(WithSize(80, 24), WithMaxScrollback(2000))
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { _ = terminal.Close() })
	for index := range 2000 {
		if _, err := fmt.Fprintf(terminal, "line-%04d build output\r\n", index); err != nil {
			b.Fatal(err)
		}
	}
	return terminal
}

func BenchmarkTerminalCursorRow(b *testing.B) {
	terminal := benchmarkTerminalWithScrollback(b)
	b.ResetTimer()
	for b.Loop() {
		if _, err := terminal.CursorRow(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkTerminalScreenRows(b *testing.B) {
	terminal := benchmarkTerminalWithScrollback(b)
	b.ResetTimer()
	for b.Loop() {
		if _, err := terminal.ScreenRows(); err != nil {
			b.Fatal(err)
		}
	}
}
