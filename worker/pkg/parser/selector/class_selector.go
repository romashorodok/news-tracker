package selector

import "github.com/romashorodok/news-tracker/worker/pkg/parser"

type ClassSelector struct {
	ast            *parser.AstGenerator
	classes        []string
	treeCompleteFn func(*parser.Node)
}

func (s *ClassSelector) OnOpen(node parser.Node) {
	if s.ast.IsBuilding() {
		s.ast.AppendOpenTag(node)
		return
	}

	if len(s.classes) > 0 {
		if parser.ContainsClass(node.Tag.Attr["class"], s.classes) {
			s.ast.AppendOpenTag(node)
		}
	} else {
		s.ast.AppendOpenTag(node)
	}
}

func (s *ClassSelector) OnClose(node parser.Node) {
	s.ast.CloseTag(node)
}

func (s *ClassSelector) GetPendingNode() *parser.Node {
	return s.ast.PendingNode()
}

func (s *ClassSelector) onTreeComplete(node *parser.Node) {
	s.ast.Free()
	s.treeCompleteFn(node)
}

var _ parser.Selector = (*ClassSelector)(nil)

func NewClassSelector(classes []string, treeCompleteFn func(node *parser.Node)) *ClassSelector {
	selector := &ClassSelector{
		ast:            parser.NewAstGenerator(),
		treeCompleteFn: treeCompleteFn,
		classes:        classes,
	}
	selector.ast.OnTreeComplete(selector.onTreeComplete)
	return selector
}
