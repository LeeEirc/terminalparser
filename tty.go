package terminalparser

import (
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

func (t *TerminalVT) Cols() (uint16, error) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	return t.VT.Cols()
}

func (t *TerminalVT) Rows() (uint16, error) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	return t.VT.Rows()
}
