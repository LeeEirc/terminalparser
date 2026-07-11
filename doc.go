// Package terminalparser interprets ANSI and VT byte streams and exposes the
// resulting terminal screen as plain text.
//
// Use Parse or ParseString for a complete command output buffer. Use New when
// output arrives incrementally from an SSH session, PTY, process pipe, or
// websocket. The parser is backed by Ghostty's libghostty-vt implementation,
// supports modern terminal control sequences, and is safe for concurrent use.
//
// This package uses cgo through go.mitchellh.com/libghostty. Applications must
// make libghostty-vt available to pkg-config when they build; see the project
// README for installation and cross-compilation instructions.
package terminalparser
