package parser

import (
	"sync"
	"sync/atomic"
)

type AstGenerator struct {
	tagNames        map[string]*atomic.Int32
	rootNode        *Node
	nodes           []*Node
	nodesMu         sync.Mutex
	autoClosingNode bool
	onTreeComplete  func(*Node)
}

func (t *AstGenerator) AppendOpenTag(node Node) {
	t.nodesMu.Lock()
	defer t.nodesMu.Unlock()

	if t.nodes == nil {
		t.nodes = make([]*Node, 0)
	}

	tagNameCounter, tagNameExists := t.tagNames[node.Name]
	if !tagNameExists {
		var counter atomic.Int32
		counter.Add(1)
		t.tagNames[node.Name] = &counter
	} else {
		tagNameCounter.Add(1)
	}

	t.nodes = append(t.nodes, &node)

	if t.rootNode == nil {
		t.rootNode = &node
	}
}

func (t *AstGenerator) nextNode() *Node {
	if len(t.nodes) > 0 {
		node := t.nodes[len(t.nodes)-1]
		t.nodes = t.nodes[:len(t.nodes)-1]
		return node
	}
	return nil
}

func (t *AstGenerator) CloseTag(closingNode Node) {
	t.nodesMu.Lock()
	defer t.nodesMu.Unlock()

	tagNameCounter, tagNameExist := t.tagNames[closingNode.Name]
	if !tagNameExist {
		return
	}

	tagNameCounter.Add(-1)
	if tagNameCounter.Load() == 0 {
		delete(t.tagNames, closingNode.Name)
	}

	targetNode := t.nextNode()

	// If target node is not correspond to the closing node.
	// It may be by inconsistent stack or somewhere was lost closing tag.
	// And need to complete open tag anyway. But what to do with lost open tag?
	for len(t.nodes) > 0 && targetNode.Name != closingNode.Name {
		nodeToSkip := targetNode

		targetNode = t.nextNode()

		// Link lost node as the Next node of the tree stack
		if t.autoClosingNode {
			targetNodeLastChild := LastChildOfNode(targetNode)
			targetNodeLastChild.Next = nodeToSkip
		}

		if targetNode.Name == closingNode.Name {
			break
		}
	}

	if targetNode.Name == closingNode.Name {
		if parentNode := t.PendingNode(); parentNode != nil {
			parentNodeLastChild := LastChildOfNode(parentNode)
			parentNodeLastChild.Next = targetNode
		}
	}

	if targetNode != nil && t.rootNode == targetNode {
		t.onTreeComplete(t.rootNode)
	}
}

func (t *AstGenerator) PendingNode() *Node {
	if len(t.nodes) > 0 {
		return t.nodes[len(t.nodes)-1]
	}
	return nil
}

func (t *AstGenerator) OnTreeComplete(fn func(*Node)) {
	t.onTreeComplete = fn
}

func (t *AstGenerator) IsBuilding() bool {
	return t.rootNode != nil
}

func (t *AstGenerator) Free() {
	t.rootNode = nil
	t.nodes = make([]*Node, 0)
	t.tagNames = make(map[string]*atomic.Int32)
}

func NewAstGenerator() *AstGenerator {
	return &AstGenerator{
		tagNames:        make(map[string]*atomic.Int32),
		nodes:           make([]*Node, 0),
		autoClosingNode: true,
	}
}
