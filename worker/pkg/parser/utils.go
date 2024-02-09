package parser

import (
	"strings"
)

func LastChildOfNode(node *Node) *Node {
	parentNode := node
	for parentNode.Next != nil {
		parentNode = parentNode.Next
	}
	return parentNode
}

func ContainsClass(classString string, classSelectors []string) bool {
	for _, selector := range classSelectors {
		if strings.Contains(classString, selector) {
			return true
		}
	}
	return false
}

// func getIndent(depth int) string {
// 	return strings.Repeat("  ", depth)
// }
//
// func traverse(node *Node, indent int) {
// 	if node != nil {
// 		fmt.Printf("%s%s %s %+v\n", getIndent(indent), node.Name, node.Content, node.Tag.Attr)
//
// 		if node.Next != nil {
// 			traverse(node.Next, indent+1)
// 		}
// 	}
// }
