package terminalparser

import (
	"strings"
	"sync"

	"go.mitchellh.com/libghostty"
)

func NewTerminalVT(col, row uint16) (*TerminalVT, error) {
	vt, err := libghostty.NewTerminal(libghostty.WithSize(col, row))
	if err != nil {
		return nil, err
	}
	return &TerminalVT{VT: vt}, nil
}

type TerminalVT struct {
	mutex sync.Mutex
	VT    *libghostty.Terminal
}

func (t *TerminalVT) Write(p []byte) (n int, err error) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	n, err = t.VT.Write(p)
	return
}

func (t *TerminalVT) Resize(col, row uint16, width, height uint32) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	return t.VT.Resize(col, row, width, height)
}

func (t *TerminalVT) Close() error {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.VT.Close()
	return nil
}

func (t *TerminalVT) String() (string, error) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	f, err := libghostty.NewFormatter(t.VT,
		libghostty.WithFormatterFormat(libghostty.FormatterFormatPlain),
		libghostty.WithFormatterTrim(true),
	)
	if err != nil {
		return "", err
	}
	defer f.Close()
	ret, err := f.FormatString()
	if err != nil {
		return "", err
	}
	return ret, nil
}

func (t *TerminalVT) ScreenRows() ([]string, error) {
	screenStr, err := t.String()
	if err != nil {
		return nil, err
	}
	return strings.Split(screenStr, "\n"), nil
}

func (t *TerminalVT) Title() (string, error) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	return t.VT.Title()
}

func (t *TerminalVT) CursorX() (uint16, error) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	return t.VT.CursorX()
}

func (t *TerminalVT) CursorY() (uint16, error) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	return t.VT.CursorY()
}

func (t *TerminalVT) IsScreenAlternate() (bool, error) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	screen, err := t.VT.ActiveScreen()
	if err != nil {
		return false, err
	}
	return screen == libghostty.ScreenAlternate, nil
}

func (t *TerminalVT) Lock() {
	// Lock the mutex to ensure thread safety when accessing the raw terminal
	t.mutex.Lock()
}

func (t *TerminalVT) Unlock() {
	// Unlock the mutex after accessing the raw terminal
	t.mutex.Unlock()
}
