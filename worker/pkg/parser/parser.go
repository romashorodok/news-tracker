package parser

import (
	"io"
	"sync"

	"github.com/romashorodok/news-tracker/worker/pkg/parser/token"
)

func Parse(file io.Reader, selectors ...Selector) {
	tok := NewTokenizer(file)
	var wg sync.WaitGroup

	for {
		t := tok.Next()

		if tok.Err != nil {
			return
		}

		switch t := t.(type) {
		case token.OpenTag:
			node := Node{Name: t.Name, Type: OPEN_NODE, Tag: t}
			for _, selector := range selectors {
				wg.Add(1)
				go func(selector Selector, node Node) {
					defer wg.Done()
					selector.OnOpen(node)
				}(selector, node)
			}
			wg.Wait()

		case token.CloseTag:
			node := Node{Name: t.Name, Type: CLOSE_NODE}
			for _, selector := range selectors {
				wg.Add(1)
				go func(selector Selector, node Node) {
					defer wg.Done()
					if selector.GetPendingNode() == nil {
						return
					}
					selector.OnClose(node)
				}(selector, node)
			}
			wg.Wait()

		case token.Text:
			for _, selector := range selectors {
				wg.Add(1)
				go func(selector Selector, text token.Text) {
					defer wg.Done()
					pendingNode := selector.GetPendingNode()
					if pendingNode == nil {
						return
					}
					// Also that approach may be used.
					// pendingNode.Content += string(removeNewLine(text.Data))

					// Create separated text node may be more error resistant.
					// That may prevent losing the text if something goes wrong.
					selector.OnOpen(Node{Name: string(TEXT_NODE), Type: TEXT_NODE, Content: string(token.RemoveNewLine(text.Data))})
				}(selector, t)
			}
			wg.Wait()

		case token.Comment:
		}
	}
}
