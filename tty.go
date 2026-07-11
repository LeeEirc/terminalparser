package terminalparser

import (
	"errors"
	"strings"
	"sync"

	"go.mitchellh.com/libghostty"
)

const (
	// DefaultColumns is the default terminal width used by [New] and [Parse].
	DefaultColumns uint16 = 80

	// DefaultRows is the default terminal height used by [New] and [Parse].
	DefaultRows uint16 = 24

	// DefaultMaxScrollback is the default number of history rows retained by
	// the terminal. Set it to zero with [WithMaxScrollback] to disable history.
	DefaultMaxScrollback uint = 1000
)

var (
	// ErrClosed is returned when an operation needs a terminal that has already
	// been closed.
	ErrClosed = errors.New("terminalparser: terminal is closed")

	// ErrInvalidSize is returned when either terminal dimension is zero.
	ErrInvalidSize = errors.New("terminalparser: columns and rows must be greater than zero")
)

type config struct {
	columns       uint16
	rows          uint16
	maxScrollback uint
	trim          bool
	unwrap        bool
}

func defaultConfig() config {
	return config{
		columns:       DefaultColumns,
		rows:          DefaultRows,
		maxScrollback: DefaultMaxScrollback,
		trim:          true,
	}
}

// Option configures a [TerminalVT] created by [New] or a one-shot parse
// performed by [Parse] or [ParseString]. Options are created by this package's
// With functions.
type Option interface {
	apply(*config) error
}

type optionFunc func(*config) error

func (fn optionFunc) apply(cfg *config) error {
	return fn(cfg)
}

// WithSize sets the virtual terminal dimensions in character cells. Both
// values must be greater than zero.
func WithSize(columns, rows uint16) Option {
	return optionFunc(func(cfg *config) error {
		if columns == 0 || rows == 0 {
			return ErrInvalidSize
		}
		cfg.columns = columns
		cfg.rows = rows
		return nil
	})
}

// WithMaxScrollback sets the maximum number of history rows retained by the
// primary screen. A value of zero disables scrollback.
func WithMaxScrollback(rows uint) Option {
	return optionFunc(func(cfg *config) error {
		cfg.maxScrollback = rows
		return nil
	})
}

// WithTrim controls whether trailing spaces are removed from non-empty lines
// in formatted output. It is enabled by default.
func WithTrim(enabled bool) Option {
	return optionFunc(func(cfg *config) error {
		cfg.trim = enabled
		return nil
	})
}

// WithUnwrap controls whether soft-wrapped terminal rows are joined when the
// terminal is formatted. It is disabled by default so screen rows remain
// visible as separate strings.
func WithUnwrap(enabled bool) Option {
	return optionFunc(func(cfg *config) error {
		cfg.unwrap = enabled
		return nil
	})
}

// New creates a terminal parser with the supplied options. The returned
// parser is safe for concurrent use. Call [TerminalVT.Close] when it is no
// longer needed.
func New(opts ...Option) (*TerminalVT, error) {
	cfg := defaultConfig()
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt.apply(&cfg); err != nil {
			return nil, err
		}
	}

	vt, err := libghostty.NewTerminal(
		libghostty.WithSize(cfg.columns, cfg.rows),
		libghostty.WithMaxScrollback(cfg.maxScrollback),
	)
	if err != nil {
		return nil, err
	}

	return &TerminalVT{
		vt:     vt,
		trim:   cfg.trim,
		unwrap: cfg.unwrap,
	}, nil
}

// NewTerminalVT creates a terminal parser with the requested dimensions and
// otherwise default settings. It is retained for compatibility; new code can
// use New(WithSize(columns, rows)).
func NewTerminalVT(columns, rows uint16) (*TerminalVT, error) {
	return New(WithSize(columns, rows))
}

// Parse feeds data into a new virtual terminal and returns the resulting plain
// text screen as rows. ANSI/VT control sequences are interpreted rather than
// copied into the result. Empty screen output is returned as an empty slice.
func Parse(data []byte, opts ...Option) ([]string, error) {
	terminal, err := New(opts...)
	if err != nil {
		return nil, err
	}
	defer terminal.Close()

	if _, err := terminal.Write(data); err != nil {
		return nil, err
	}
	return terminal.ScreenRows()
}

// ParseString feeds data into a new virtual terminal and returns the resulting
// plain text screen. ANSI/VT control sequences are interpreted rather than
// copied into the result.
func ParseString(data []byte, opts ...Option) (string, error) {
	terminal, err := New(opts...)
	if err != nil {
		return "", err
	}
	defer terminal.Close()

	if _, err := terminal.Write(data); err != nil {
		return "", err
	}
	return terminal.String()
}

// ParseOutput parses command output into at most 500 non-empty, trimmed rows.
// It preserves the error-free signature provided by the dev branch. New code
// should use [Parse], which returns parsing errors and offers configuration.
// ParseOutput returns nil if the parser cannot be initialized.
func ParseOutput(data []byte) []string {
	rows, err := Parse(data)
	if err != nil {
		return nil
	}

	const maxRows = 500
	result := make([]string, 0, min(len(rows), maxRows))
	for _, row := range rows {
		row = strings.TrimSpace(row)
		if row == "" {
			continue
		}
		result = append(result, row)
		if len(result) == maxRows {
			break
		}
	}
	return result
}

// TerminalVT is a stateful VT/ANSI command-output parser backed by
// libghostty. Its methods serialize access to the underlying native terminal,
// so a TerminalVT may be shared by multiple goroutines. A TerminalVT must not
// be copied after first use.
type TerminalVT struct {
	mutex  sync.Mutex
	vt     *libghostty.Terminal
	trim   bool
	unwrap bool
}

// Write implements io.Writer. It feeds bytes through the VT parser and always
// consumes the entire input unless the terminal has been closed.
func (t *TerminalVT) Write(data []byte) (int, error) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if t.vt == nil {
		return 0, ErrClosed
	}
	return t.vt.Write(data)
}

// Resize changes the terminal dimensions and the pixel dimensions of each
// cell. columns and rows must be greater than zero. cellWidthPx and
// cellHeightPx may be zero when pixel dimensions are unknown.
func (t *TerminalVT) Resize(columns, rows uint16, cellWidthPx, cellHeightPx uint32) error {
	if columns == 0 || rows == 0 {
		return ErrInvalidSize
	}

	t.mutex.Lock()
	defer t.mutex.Unlock()

	if t.vt == nil {
		return ErrClosed
	}
	return t.vt.Resize(columns, rows, cellWidthPx, cellHeightPx)
}

// Reset restores the terminal to its initial state while preserving its
// dimensions. Screen contents, scrollback, modes, and cursor state are reset.
func (t *TerminalVT) Reset() error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if t.vt == nil {
		return ErrClosed
	}
	t.vt.Reset()
	return nil
}

// Close releases the native terminal. It is safe to call Close more than once.
// All later operations return [ErrClosed].
func (t *TerminalVT) Close() error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if t.vt == nil {
		return nil
	}
	t.vt.Close()
	t.vt = nil
	return nil
}

// String returns the active terminal screen as plain text. ANSI/VT sequences
// are not included. Formatting follows the options supplied at construction.
func (t *TerminalVT) String() (string, error) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	return t.stringLocked()
}

func (t *TerminalVT) stringLocked() (string, error) {
	if t.vt == nil {
		return "", ErrClosed
	}

	formatter, err := libghostty.NewFormatter(t.vt,
		libghostty.WithFormatterFormat(libghostty.FormatterFormatPlain),
		libghostty.WithFormatterTrim(t.trim),
		libghostty.WithFormatterUnwrap(t.unwrap),
	)
	if err != nil {
		return "", err
	}
	defer formatter.Close()
	return formatter.FormatString()
}

// ScreenRows returns the same formatted snapshot as [TerminalVT.String], split
// on newline boundaries. An empty screen is returned as an empty slice.
func (t *TerminalVT) ScreenRows() ([]string, error) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	screen, err := t.stringLocked()
	if err != nil {
		return nil, err
	}
	if screen == "" {
		return []string{}, nil
	}
	return strings.Split(screen, "\n"), nil
}

// CursorRow returns the plain-text contents of the physical row containing
// the cursor. Unlike ScreenRows, it formats only that row and does not walk or
// copy the terminal's scrollback history.
func (t *TerminalVT) CursorRow() (string, error) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if t.vt == nil {
		return "", ErrClosed
	}
	columns, err := t.vt.Cols()
	if err != nil {
		return "", err
	}
	y, err := t.vt.CursorY()
	if err != nil {
		return "", err
	}
	start, err := t.vt.GridRef(libghostty.Point{
		Tag: libghostty.PointTagActive,
		Y:   uint32(y),
	})
	if err != nil {
		return "", err
	}
	end, err := t.vt.GridRef(libghostty.Point{
		Tag: libghostty.PointTagActive,
		X:   columns - 1,
		Y:   uint32(y),
	})
	if err != nil {
		return "", err
	}
	selection := &libghostty.Selection{Start: *start, End: *end}
	return t.vt.SelectionFormatString(
		libghostty.WithSelection(selection),
		libghostty.WithSelectionFormat(libghostty.FormatterFormatPlain),
		libghostty.WithSelectionTrim(t.trim),
		libghostty.WithSelectionUnwrap(t.unwrap),
	)
}

// Size returns the current terminal dimensions in character cells.
func (t *TerminalVT) Size() (columns, rows uint16, err error) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if t.vt == nil {
		return 0, 0, ErrClosed
	}
	columns, err = t.vt.Cols()
	if err != nil {
		return 0, 0, err
	}
	rows, err = t.vt.Rows()
	if err != nil {
		return 0, 0, err
	}
	return columns, rows, nil
}

// Title returns the title last set by an OSC 0 or OSC 2 sequence. It returns
// an empty string when no title has been set.
func (t *TerminalVT) Title() (string, error) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if t.vt == nil {
		return "", ErrClosed
	}
	return t.vt.Title()
}

// CursorX returns the zero-based cursor column.
func (t *TerminalVT) CursorX() (uint16, error) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if t.vt == nil {
		return 0, ErrClosed
	}
	return t.vt.CursorX()
}

// CursorY returns the zero-based cursor row within the active screen.
func (t *TerminalVT) CursorY() (uint16, error) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if t.vt == nil {
		return 0, ErrClosed
	}
	return t.vt.CursorY()
}

// IsScreenAlternate reports whether the alternate screen buffer is active.
func (t *TerminalVT) IsScreenAlternate() (bool, error) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if t.vt == nil {
		return false, ErrClosed
	}
	screen, err := t.vt.ActiveScreen()
	if err != nil {
		return false, err
	}
	return screen == libghostty.ScreenAlternate, nil
}
