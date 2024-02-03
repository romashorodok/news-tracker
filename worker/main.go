package main

import (
	"bytes"
	"io"
	"log"
	"os"
)

func removeNewLine(source []byte) []byte {
	source = bytes.Replace(source, []byte{'\n'}, []byte{}, -1)
	source = bytes.Replace(source, []byte{'\r'}, []byte{}, -1)
	source = bytes.Replace(source, []byte("\r\n"), []byte{}, -1)
	return source
}

func main() {
	file, err := os.Open("text.html")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	MyTokenizer(file)
}

// token -> lexem
// lexeme is a sequence of characters in the source that matches the pattern
// printf("Total = %d\n",score) ;
// In this case printf in C. Literal pattern wrapped by " "

func MyTokenizer(file io.Reader) {
	tok := NewTokenizer(file)

	for {
		tok.Next()
		// log.Println(tok.tt)

		if tok.Err != nil {
			return
		}

		tagName := tok.TagName()
		if string(tagName) == " " || string(tagName) == "\n" {
			continue
		}

		if tok.tt == CLOSE_TAG_TOKEN {
			log.Println("Close token", string(tagName))
		}

		if tok.tt == OPEN_TAG_TOKEN {
			log.Println("Open token", string(tagName))
			for _, attrCur := range tok.attr {
				key, value := tok.TagAttr(attrCur)
				_, _ = key, value
				log.Println("attr", string(key), string(value))
			}
		}

		if tok.tt == TEXT_TOKEN {
			log.Println("Text", string(tok.Text()))
		}
	}
}

type TokenType uint8

const (
	ERROR_TOKEN TokenType = iota
	SKIP_TOKEN
	TEXT_TOKEN
	OPEN_TAG_TOKEN
	CLOSE_TAG_TOKEN
	SELF_CLOSING_TAG_TOKEN
	COMMENT_TOKEN
	DOCTYPE_TOKEN
)

type ParserState uint8

const (
	NORMAL ParserState = iota
	SCRIPT_CONTENT
)

// `<` - terminal symbol
// `<a>` - lexeme
// `<a>` tag is sequence of chars it may be different also containsa.

// Terminal symbols

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

type Lexeme string

const (
	SCRIPT Lexeme = "script"
)

type cursor struct {
	start, end int
}

type Tokenizer struct {
	source io.Reader
	buf    []byte
	Err    error

	reader cursor
	data   cursor
	tt     TokenType

	pendidngAttrKey   cursor
	pendidngAttrValue cursor
	attr              [][]cursor
	state             ParserState
}

func (tok *Tokenizer) TagAttr(cur []cursor) (key, val []byte) {
	switch tok.tt {
	case OPEN_TAG_TOKEN, SELF_CLOSING_TAG_TOKEN:
		key := tok.buf[cur[0].start:cur[0].end]
		val := tok.buf[cur[1].start:cur[1].end]
		return key, val
	}
	return nil, nil
}

func (tok *Tokenizer) TagName() []byte {
	if tok.data.start < tok.data.end {
		switch tok.tt {
		case OPEN_TAG_TOKEN, CLOSE_TAG_TOKEN, SELF_CLOSING_TAG_TOKEN:
			b := tok.buf[tok.data.start:tok.data.end]
			tok.data.start = tok.reader.end
			tok.data.end = tok.reader.end
			return b
		}
	}
	return nil
}

func (tok *Tokenizer) GetBuffer() ([]byte, int) {
	capacity := cap(tok.buf)
	numElems := tok.reader.end - tok.reader.start

	var buf []byte
	if 2*numElems > capacity {
		buf = make([]byte, numElems, 2*capacity)
	} else {
		buf = tok.buf[:numElems]
	}

	copy(buf, tok.buf[tok.reader.start:tok.reader.end])

	return buf, numElems
}

func (tok *Tokenizer) readByte() byte {
	if tok.reader.end >= len(tok.buf) {
		if tok.Err != nil {
			return 0
		}

		buf, numElems := tok.GetBuffer()

		if x := tok.reader.start; x != 0 {
			tok.adjustRanges(x)
		}

		tok.reader.start, tok.reader.end, tok.buf = 0, numElems, buf[:numElems]

		var n int
		n, tok.Err = tok.source.Read(buf[numElems:cap(buf)])
		if n == 0 {
			return 0
		}

		tok.buf = buf[:numElems+n]
	}

	b := tok.buf[tok.reader.end]
	tok.reader.end++
	return b
}

func (tok *Tokenizer) adjustRanges(offset int) {
	tok.data.start -= offset
	tok.data.end -= offset

	tok.pendidngAttrKey.start -= offset
	tok.pendidngAttrKey.end -= offset
	tok.pendidngAttrValue.start -= offset
	tok.pendidngAttrValue.end -= offset

	for _, attr := range tok.attr {
		attr[0].start -= offset
		attr[0].end -= offset
		attr[1].start -= offset
		attr[1].end -= offset
	}
}

func (tok *Tokenizer) trace(b byte) {
	log.Println(
		string(b),
		"tok.reader.start", tok.reader.start,
		"tok.reader.end", tok.reader.end,
		"tok.data.start", tok.data.start,
		"tok.datta.end", tok.data.end,
	)
}

func (tok *Tokenizer) tagName() {
	for {
		// Read ahead because a tag may consist of only one symbol,
		// and we need to check now if the tag is complete or not.
		var symbol TerminalSymbol = tok.readByte()
		if tok.Err != nil {
			tok.data.end = tok.reader.end
			return
		}

		// tok.trace(symbol)

		switch symbol {
		case SPACE, NEW_LINE, C_RETURN, TAB, FORM_FEED: /* tag name is done, but it may have attribs */
			tok.data.end = tok.reader.end - 1
			return

		case SLASH, R_BRACKET: /* Tag name actually done */
			tok.reader.end--
			tok.data.end = tok.reader.end
			return
		}
	}
}

func (tok *Tokenizer) tagAttrKey() {
	for {
		var symbol TerminalSymbol = tok.readByte()
		if tok.Err != nil {
			tok.pendidngAttrKey.end = tok.reader.end
			return
		}

		switch symbol {
		case SPACE, NEW_LINE, C_RETURN, TAB, FORM_FEED, SLASH: /* Attr key without value */
			tok.pendidngAttrKey.end = tok.reader.end - 1
			return
		case EQUALS:
			// Current symbol `=` go to next symbol
			if tok.pendidngAttrKey.start+1 == tok.reader.end {
				continue
			}
			fallthrough
		case R_BRACKET:
			tok.reader.end--
			tok.pendidngAttrKey.end = tok.reader.end
			return
		}
	}
}

func (tok *Tokenizer) tagAttrVal() {
	var symbol TerminalSymbol = tok.readByte()
	if symbol != EQUALS {
		tok.reader.end--
		return
	}
	if tok.Err != nil {
		tok.reader.end--
		return
	}

	var quote TerminalSymbol = tok.readByte()
	if tok.Err != nil {
		tok.reader.end--
		return
	}

	switch quote {
	case R_BRACKET:
		tok.reader.end--
		return
	case SINGLE_QUOTE, DOUBLE_QUOTE:
		tok.pendidngAttrValue.start = tok.reader.end
		for {
			var symbol TerminalSymbol = tok.readByte()
			if tok.Err != nil {
				tok.pendidngAttrValue.end = tok.reader.end
				return
			}

			if symbol == quote {
				tok.pendidngAttrValue.end = tok.reader.end - 1
				return
			}
		}
		// TODO: make defautl case
	}
}

func (tok *Tokenizer) tag() {
	// Remember start pos of the tag name in buffer. -1 because first letter already consumed
	tok.data.start = tok.reader.end - 1
	tok.data.end = tok.reader.end

	tok.attr = tok.attr[:0]

	tok.tagName()
	tok.skipSpace()
	if tok.Err != nil {
		return
	}

	for {
		var symbol TerminalSymbol = tok.readByte()
		if tok.Err != nil || symbol == '>' {
			break
		} else {
			tok.reader.end--
		}

		tok.pendidngAttrKey.start = tok.reader.end
		tok.pendidngAttrKey.end = tok.reader.end
		tok.tagAttrKey()

		tok.pendidngAttrValue.start = tok.reader.end
		tok.pendidngAttrValue.end = tok.reader.end
		tok.tagAttrVal()

		if tok.pendidngAttrKey.start != tok.pendidngAttrValue.end {
			attr := make([]cursor, 2)
			attr[0] = tok.pendidngAttrKey
			attr[1] = tok.pendidngAttrValue
			tok.attr = append(tok.attr, attr)
		}

		tok.skipSpace()
	}
}

func (tok *Tokenizer) Next() TokenType {
	tok.reader.start = tok.reader.end
	tok.data.start = tok.reader.end
	tok.data.end = tok.reader.end
	if tok.Err != nil {
		tok.tt = ERROR_TOKEN
		return tok.tt
	}

	for {
		var symbol TerminalSymbol = tok.readByte()
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
		case symbol == '/':
			tokenType = CLOSE_TAG_TOKEN
		case symbol == '!' || symbol == '?':
			tokenType = COMMENT_TOKEN
		default:
			tok.reader.end--
			continue
		}

		switch tokenType {
		case OPEN_TAG_TOKEN:
			if tok.state != NORMAL {
				tok.tt = SKIP_TOKEN
				return tok.tt
			}
			tok.tag()

			var lexeme Lexeme = Lexeme(tok.buf[tok.data.start:tok.data.end])
			log.Println("Lexme", lexeme)

			switch lexeme {
			case SCRIPT:
				tok.state = SCRIPT_CONTENT
			}

			tok.tt = OPEN_TAG_TOKEN
			return tok.tt
		case CLOSE_TAG_TOKEN:
			// Discard `/` symbol
			_ = tok.readByte()

			tok.tag()
			if tok.Err != nil {
				tok.tt = ERROR_TOKEN
				return tok.tt
			}

			tok.state = NORMAL
			tok.tt = CLOSE_TAG_TOKEN
			return tok.tt
		default:
		}
	}

	tok.tt = ERROR_TOKEN
	return tok.tt
}

func (tok *Tokenizer) skipSpace() {
	if tok.Err != nil {
		return
	}
	// NOTE: may be here error
	for {
		b := tok.readByte()
		if tok.Err != nil {
			return
		}

		switch b {
		case ' ', '\n', '\r', '\t', '\f':
		default:
			// If space not found return reader cursor back
			tok.reader.end--
			return
		}
	}
}

func (tok *Tokenizer) Text() []byte {
	switch tok.tt {
	case TEXT_TOKEN:
		b := tok.buf[tok.data.start:tok.data.end]
		tok.data.start = tok.reader.end
		tok.data.end = tok.reader.end
		b = removeNewLine(b)
		return b
	}
	return nil
}

func NewTokenizer(reader io.Reader) *Tokenizer {
	return &Tokenizer{
		source: reader,
		buf:    make([]byte, 0, 24),
	}
}
