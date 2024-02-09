package token

import "errors"

// `<` - terminal symbol
// `<a>` - lexeme
// `<a>` tag is sequence of chars it may be different also containsa.

type Cursor struct {
	Start, End int
}

type TerminalSymbol = byte

const (
	L_BRACKET    TerminalSymbol = '<'
	R_BRACKET    TerminalSymbol = '>'
	SLASH        TerminalSymbol = '/'
	SPACE        TerminalSymbol = ' '
	NEW_LINE     TerminalSymbol = '\n'
	C_RETURN     TerminalSymbol = '\r'
	TAB          TerminalSymbol = '\t'
	FORM_FEED    TerminalSymbol = '\f'
	EQUALS       TerminalSymbol = '='
	SINGLE_QUOTE TerminalSymbol = '\''
	DOUBLE_QUOTE TerminalSymbol = '"'
)

var (
	ErrMssingLeftBracket = errors.New("missing left bracket")
	ErrMssingSlash       = errors.New("missing slash")
	ErrCursorEnd         = errors.New("end of cursor")
)
