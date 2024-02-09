package parser

import (
	"io"

	"github.com/romashorodok/news-tracker/worker/pkg/parser/token"
)

type Tokenizer struct {
	source io.Reader
	buf    []byte
	Err    error

	reader token.Cursor
	data   token.Cursor
	tt     TokenType
	state  ParserState
}

func (tok *Tokenizer) GetBuffer() ([]byte, int) {
	capacity := cap(tok.buf)
	numElems := tok.reader.End - tok.reader.Start

	var buf []byte
	if 2*numElems > capacity {
		buf = make([]byte, numElems, 2*capacity)
	} else {
		buf = tok.buf[:numElems]
	}

	copy(buf, tok.buf[tok.reader.Start:tok.reader.End])

	return buf, numElems
}

func (tok *Tokenizer) readByte() byte {
	if tok.reader.End >= len(tok.buf) {
		if tok.Err != nil {
			return 0
		}

		buf, numElems := tok.GetBuffer()

		if x := tok.reader.Start; x != 0 {
			tok.data.Start -= x
			tok.data.End -= x
		}

		tok.reader.Start, tok.reader.End, tok.buf = 0, numElems, buf[:numElems]

		var n int
		n, tok.Err = tok.source.Read(buf[numElems:cap(buf)])
		if n == 0 {
			return 0
		}

		tok.buf = buf[:numElems+n]
	}

	b := tok.buf[tok.reader.End]
	tok.reader.End++
	return b
}

func (tok *Tokenizer) tag() {
	tok.data.Start = tok.reader.End - 2
	tok.data.End = tok.reader.End

loop:
	for {
		var symbol token.TerminalSymbol = tok.readByte()
		if tok.Err != nil {
			break
		}

		switch symbol {
		case token.SPACE:
			continue

		case token.R_BRACKET:
			tok.data.End = tok.reader.End
			break loop
		}
	}
}

func (tok *Tokenizer) readUntilCloseBracket() {
	tok.data.Start = tok.reader.End
	var symbol token.TerminalSymbol
	for {
		symbol = tok.readByte()
		if tok.Err != nil {
			tok.data.End = tok.reader.End
			return
		}

		if symbol == token.R_BRACKET {
			tok.data.End = tok.reader.End
			return
		}
	}
}

func (tok *Tokenizer) Next() any {
	tok.reader.Start = tok.reader.End
	tok.data.Start = tok.reader.End
	tok.data.End = tok.reader.End
	if tok.Err != nil {
		tok.tt = ERROR_TOKEN
		return tok.tt
	}

	for {
		var symbol token.TerminalSymbol = tok.readByte()
		if tok.Err != nil {
			break
		}

		if symbol != '<' {
			continue
		}

		// Read ahead because if I will catch </
		// My token must be a close tag
		symbol = tok.readByte()
		if tok.Err != nil {
			break
		}

		var tokenType TokenType

		switch {
		case 'a' <= symbol && symbol <= 'z' || 'A' <= symbol && symbol <= 'Z':
			tokenType = OPEN_TAG_TOKEN
		case symbol == token.SLASH:
			tokenType = CLOSE_TAG_TOKEN
		case symbol == '!' || symbol == '?':
			tokenType = COMMENT_TOKEN
			tok.reader.End = tok.reader.End - 2
		default:
			tok.reader.End--
			continue
		}

		if x := tok.reader.End - 2; tok.reader.Start < x && tokenType != COMMENT_TOKEN {
			tok.reader.End = x
			tok.data.End = x

			tok.tt = TEXT_TOKEN
			return token.Text{Data: tok.buf[tok.data.Start:tok.data.End]}
		}

		switch tokenType {
		case COMMENT_TOKEN:
			tok.readUntilCloseBracket()
			return token.Comment{Data: tok.buf[tok.data.Start:tok.data.End]}

		case OPEN_TAG_TOKEN:

			if tok.state != NORMAL {
				tok.tt = SKIP_TOKEN
				return tok.tt
			}
			tok.tag()

			bytes := tok.buf[tok.data.Start:tok.data.End]

			tag := token.OpenTag{}
			_ = tag.Unmarshal(bytes)

			switch Lexeme(tag.Name) {
			case SCRIPT:
				tok.state = SCRIPT_CONTENT
			}

			tok.tt = OPEN_TAG_TOKEN
			return tag
		case CLOSE_TAG_TOKEN:
			tok.tag()
			if tok.Err != nil {
				tok.tt = ERROR_TOKEN
				return tok.tt
			}

			bytes := tok.buf[tok.data.Start:tok.data.End]

			tag := token.CloseTag{}
			_ = tag.Unmarshal(bytes)

			tok.state = NORMAL
			tok.tt = CLOSE_TAG_TOKEN

			return tag
		default:
		}
	}

	tok.tt = ERROR_TOKEN
	return tok.tt
}

func NewTokenizer(reader io.Reader) *Tokenizer {
	return &Tokenizer{
		source: reader,
		buf:    make([]byte, 0, 24),
	}
}
