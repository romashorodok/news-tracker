package parser

import (
	"github.com/romashorodok/news-tracker/worker/pkg/parser/token"
)

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

type Lexeme string

const (
	SCRIPT       Lexeme = "script"
	CLOSE_SCRIPT Lexeme = "</script>"
)

type NodeType string

const (
	CLOSE_NODE NodeType = "CLOSE_NODE"
	OPEN_NODE  NodeType = "OPEN_NODE"
	TEXT_NODE  NodeType = "TEXT_NODE"
)

type Node struct {
	Name    string
	Type    NodeType
	Tag     token.OpenTag
	Content string

	Next *Node
}

type Selector interface {
	OnOpen(Node)
	OnClose(Node)
	GetPendingNode() *Node
}
