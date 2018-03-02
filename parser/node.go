package parser

import (
	"github.com/cznic/sortutil"
	"sort"
)

const (
	AND     NodeType = "AND"
	OR               = "OR"
	WITHOUT          = "WITHOUT"
	NOT              = "NOT"
	QUERY            = "QUERY"
)

type NodeType string
type NodeMatcher func(*Node) bool

type Node struct {
	Type     NodeType
	Query    string  `json:",omitempty"`
	Children []*Node `json:",omitempty"`
}

func (n *NodeType) String() string {
	return string(*n)
}

func (node *Node) EqualTo(other *Node) bool {
	if node.Type != other.Type {
		return false
	}

	if node.Query != other.Query {
		return false
	}

	if len(node.Children) != len(other.Children) {
		return false
	}

	for idx, child := range node.Children {
		if !child.EqualTo(other.Children[idx]) {
			return false
		}
	}

	return true
}

func (node *Node) LessThan(other *Node) bool {
	if node.Type < other.Type || node.Query < other.Query {
		return true
	}

	for idx := 0; idx < len(node.Children) && idx < len(other.Children); idx++ {
		if node.Children[idx].LessThan(other.Children[idx]) {
			return true
		}
	}

	return len(node.Children) < len(other.Children)
}

func (node *Node) Clone() *Node {
	var children []*Node
	if len(node.Children) > 0 {
		for _, child := range node.Children {
			children = append(children, child.Clone())
		}
	}

	copy := *node
	copy.Children = children
	return &copy
}

func NewQueryNode(query string) *Node {
	return &Node{Type: QUERY, Query: query}
}

func NewOpNode(nodeType NodeType, child *Node, children ...*Node) *Node {
	return &Node{Type: nodeType, Children: append([]*Node{child}, children...)}
}

var AllQueryNode = NewQueryNode("__all")
var EmptyQueryNode = NewQueryNode("__empty")

func everyNode(nodes []*Node, matcher NodeMatcher) bool {
	for _, child := range nodes {
		if !matcher(child) {
			return false
		}
	}

	return true
}

func anyNode(nodes []*Node, matcher NodeMatcher) bool {
	for _, child := range nodes {
		if matcher(child) {
			return true
		}
	}

	return false
}

func filterNodes(nodes []*Node, matcher NodeMatcher) []*Node {
	result := make([]*Node, 0, len(nodes))
	for _, node := range nodes {
		if matcher(node) {
			result = append(result, node)
		}
	}
	return result
}

func deduplicate(nodes []*Node) []*Node {
	n := sortutil.Dedupe(NodePtrSlice(nodes))
	return nodes[:n]
}

func not(matcher NodeMatcher) NodeMatcher {
	return func(node *Node) bool {
		return !matcher(node)
	}
}

func ofType(nodeType NodeType) NodeMatcher {
	return func(node *Node) bool {
		return node.Type == nodeType
	}
}

func SortNodesInPlace(nodes []*Node) {
	sort.Sort(NodePtrSlice(nodes))
}

type NodePtrSlice []*Node

func (s NodePtrSlice) Len() int {
	return len(s)
}

func (s NodePtrSlice) Less(i, j int) bool {
	return s[i].LessThan(s[j])
}

func (s NodePtrSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
