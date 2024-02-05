package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"sync/atomic"
)

var (
	ErrMssingLeftBracket = errors.New("missing left bracket")
	ErrMssingSlash       = errors.New("missing slash")
	ErrCursorEnd         = errors.New("end of cursor")
)

func removeNewLine(source []byte) []byte {
	source = bytes.Replace(source, []byte("  "), []byte(""), -1)
	source = bytes.Replace(source, []byte("\t"), []byte(""), -1)
	source = bytes.Replace(source, []byte{'\n'}, []byte{}, -1)
	source = bytes.Replace(source, []byte{'\r'}, []byte{}, -1)
	source = bytes.Replace(source, []byte("\r\n"), []byte{}, -1)
	return source
}

func main() {
	file, err := os.Open("_text.html")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	var wg sync.WaitGroup
	wg.Add(1)

	result := make(chan *Node)

	go func() {
		defer wg.Done()

		for r := range result {
			traverse(r, 0)
		}
	}()

	selectors := []string{"blog-item"}
	Parse(file, selectors, result)
	close(result)

	wg.Wait()
}

// token -> lexem
// lexeme is a sequence of characters in the source that matches the pattern
// printf("Total = %d\n",score) ;
// In this case printf in C. Literal pattern wrapped by " "

func containsClass(classString string, classSelectors []string) bool {
	for _, selector := range classSelectors {
		if strings.Contains(classString, selector) {
			return true
		}
	}
	return false
}

type NodeType string

const (
	CLOSE_NODE NodeType = "CLOSE_NODE"
	OPEN_NODE  NodeType = "OPEN_NODE"
)

type Node struct {
	Name    string
	Type    NodeType
	Tag     *OpenTag
	Content string

	next *Node
}

var OpenTagNames = make(map[string]*atomic.Int32)

func getIndent(depth int) string {
	return strings.Repeat("  ", depth)
}

func traverse(node *Node, indent int) {
	if node != nil {
		fmt.Printf("%s%s %s %+v\n", getIndent(indent), node.Name, node.Content, node.Tag.Attr)

		if node.next != nil {
			traverse(node.next, indent+1)
		}
	}
}

func lastChildOfNode(node *Node) *Node {
	parentNode := node
	for parentNode.next != nil {
		parentNode = parentNode.next
	}
	return parentNode
}

func Parse(file io.Reader, classSelectors []string, result chan *Node) {
	tok := NewTokenizer(file)

	var nodes []*Node
	var rootNode *Node

	for {
		token := tok.Next()

		if tok.Err != nil {
			return
		}

		switch t := token.(type) {
		case *OpenTag:
			node := &Node{Name: t.Name, Type: OPEN_NODE, Tag: t}
			nodes = append(nodes, node)

			tagNameCounter, tagNameExists := OpenTagNames[node.Name]
			if !tagNameExists {
				var counter atomic.Int32
				counter.Add(1)
				OpenTagNames[node.Name] = &counter
			} else {
				tagNameCounter.Add(1)
			}

			if rootNode == nil && len(classSelectors) > 0 && containsClass(t.Attr["class"], classSelectors) {
				rootNode = node
			}

		case *CloseTag:
			if len(nodes) <= 0 || rootNode == nil {
				continue
			}

			if _, tagNameExists := OpenTagNames[t.Name]; !tagNameExists {
				continue
			}

			childNode := nodes[len(nodes)-1]
			nodes = nodes[:len(nodes)-1]

			for len(nodes) > 0 && childNode.Name != t.Name {
				skippedNode := childNode

				childNode = nodes[len(nodes)-1]
				lastChild := lastChildOfNode(childNode)
				lastChild.next = skippedNode

				nodes = nodes[:len(nodes)-1]
				if childNode.Name == t.Name {
					break
				}
			}

			if childNode.Name == t.Name {
				if len(nodes) > 0 {
					parentNode := nodes[len(nodes)-1]
					lastChild := lastChildOfNode(parentNode)
					lastChild.next = childNode
				}
			}

			if childNode != nil && rootNode == childNode {
				result <- rootNode
				rootNode = nil
				nodes = make([]*Node, 0)
			}

		case *Text:
			if len(nodes) > 0 {
				childNode := nodes[len(nodes)-1]
				childNode.Content = string(removeNewLine(t.Data))
			}

		case *Comment:
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
	SCRIPT       Lexeme = "script"
	CLOSE_SCRIPT Lexeme = "</script>"
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
	state  ParserState
}

type tokenDataByteReader struct {
	data   []byte
	reader cursor
	err    error
}

func (r *tokenDataByteReader) readByte() TerminalSymbol {
	symbol := r.data[r.reader.end]

	if r.reader.end > len(r.data) {
		r.err = ErrCursorEnd
		r.reader.end = len(r.data) - 1
		return symbol
	}

	r.reader.end++
	return symbol
}

type OpenTag struct {
	tokenDataByteReader

	Name string
	Attr map[string]string
}

func (t *OpenTag) unmarshalAttrKey() {
	for {
		symbol := t.readByte()

		switch symbol {
		case SPACE, NEW_LINE, C_RETURN, TAB, FORM_FEED, SLASH:
			t.reader.end--
			return
		case EQUALS:
			fallthrough
		case R_BRACKET:
			t.reader.end--
			return
		}
	}
}

func (t *OpenTag) unmarshalAttrValue() {
	for {
		symbol := t.readByte()
		if symbol != EQUALS {
			t.reader.end--
		}

		quote := t.readByte()

		switch quote {
		case R_BRACKET:
			t.reader.end--
			return
		case SINGLE_QUOTE, DOUBLE_QUOTE:
			t.reader.start = t.reader.end

			for {
				symbol := t.readByte()

				if symbol == quote {
					t.reader.end--
					return
				}
			}
		}

	}
}

func (t *OpenTag) unmarshalAttr() {
	for {
		t.reader.start = t.reader.end

		symbol := t.readByte()
		if symbol == '>' {
			return
		}

		if symbol == SPACE {
			t.reader.start++
		}

		t.unmarshalAttrKey()
		attrKey := string(removeNewLine(t.data[t.reader.start:t.reader.end]))
		t.reader.start = t.reader.end

		t.unmarshalAttrValue()
		attrValue := string(removeNewLine(t.data[t.reader.start:t.reader.end]))

		quote := t.readByte()

		switch quote {
		case SINGLE_QUOTE, DOUBLE_QUOTE:
		default:
			t.reader.end--
		}

		if attrKey == "" || attrValue == "" {
			continue
		}

		t.Attr[attrKey] = attrValue
	}
}

func (t *OpenTag) unmarshalName() {
	t.reader.start = t.reader.end

loop:
	for {
		symbol := t.readByte()

		switch symbol {
		case SLASH, R_BRACKET, SPACE, NEW_LINE, C_RETURN, TAB, FORM_FEED:
			t.reader.end--
			break loop
		default:
		}
	}

	t.Name = string(t.data[t.reader.start:t.reader.end])
}

func (t *OpenTag) Unmarshal(data []byte) (err error) {
	t.data = data

	if symbol := t.readByte(); symbol != L_BRACKET {
		return ErrMssingLeftBracket
	}

	t.unmarshalName()
	if t.reader.end == len(t.data) {
		return nil
	}

	t.Attr = make(map[string]string)

	t.unmarshalAttr()

	return err
}

type CloseTag struct {
	tokenDataByteReader
	Name string
}

func (t *CloseTag) unmarshalName() {
	t.reader.start = t.reader.end

loop:
	for {
		symbol := t.readByte()

		switch symbol {
		case R_BRACKET:
			t.reader.end--
			break loop
		default:
		}
	}

	t.Name = string(t.data[t.reader.start:t.reader.end])
}

func (t *CloseTag) Unmarshal(data []byte) (err error) {
	t.data = data

	if symbol := t.readByte(); symbol != L_BRACKET {
		return ErrMssingLeftBracket
	}

	if symbol := t.readByte(); symbol != SLASH {
		return ErrMssingLeftBracket
	}

	t.unmarshalName()

	return err
}

type Text struct {
	Data []byte
}

type Comment struct {
	Data []byte
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
			tok.data.start -= x
			tok.data.end -= x
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

func (tok *Tokenizer) tag() {
	tok.data.start = tok.reader.end - 2
	tok.data.end = tok.reader.end

loop:
	for {
		var symbol TerminalSymbol = tok.readByte()
		if tok.Err != nil {
			break
		}

		switch symbol {
		case SPACE:
			continue

		case R_BRACKET:
			tok.data.end = tok.reader.end
			break loop
		}
	}
}

func (tok *Tokenizer) readUntilCloseBracket() {
	tok.data.start = tok.reader.end
	var symbol TerminalSymbol
	for {
		symbol = tok.readByte()
		if tok.Err != nil {
			tok.data.end = tok.reader.end
			return
		}

		if symbol == R_BRACKET {
			tok.data.end = tok.reader.end
			return
		}
	}
}

func (tok *Tokenizer) Next() any {
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
		case symbol == SLASH:
			tokenType = CLOSE_TAG_TOKEN
		case symbol == '!' || symbol == '?':
			tokenType = COMMENT_TOKEN
			tok.reader.end = tok.reader.end - 2
		default:
			tok.reader.end--
			continue
		}

		if x := tok.reader.end - 2; tok.reader.start < x && tokenType != COMMENT_TOKEN {
			tok.reader.end = x
			tok.data.end = x

			tok.tt = TEXT_TOKEN
			return &Text{Data: tok.buf[tok.data.start:tok.data.end]}
		}

		switch tokenType {
		case COMMENT_TOKEN:
			tok.readUntilCloseBracket()
			return &Comment{Data: tok.buf[tok.data.start:tok.data.end]}

		case OPEN_TAG_TOKEN:

			if tok.state != NORMAL {
				tok.tt = SKIP_TOKEN
				return tok.tt
			}
			tok.tag()

			bytes := tok.buf[tok.data.start:tok.data.end]

			tag := &OpenTag{}
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

			bytes := tok.buf[tok.data.start:tok.data.end]

			tag := &CloseTag{}
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