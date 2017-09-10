package parser

import (
	"github.com/sirupsen/logrus"
)

func Optimize(root *Node) *Node {
	root = root.Clone()
	canonicalizeNodeSortOrder(root)

	for pass := 0; pass < 16; pass++ {
		ctx := optimizeContext{}
		functions := []NodeTransformer{
			ctx.optRemoveUnnecessaryNodes,
			ctx.optSimplifyFlags,
			ctx.optImplementNotUsingWithout,
			ctx.optCombineHierarchy,
			ctx.optRemoveSelfCancelingWithout,
			ctx.optSimplifyChildren,
			ctx.optSimplifyCancelingOperationAndWithout,
			ctx.optMoveWithoutOutOfAnd,
		}

		changed := false
		for _, fn := range functions {
			ctx.changed = false
			root = TreeWalk(root, fn)

			if ctx.changed {
				changed = true
				//bytes, _ := json.MarshalIndent(root, "", "  ")
				//logrus.Debug(string(bytes))
			}
		}

		if !changed {
			break
		}
	}

	return root
}

type optimizeContext struct {
	changed bool
}

type NodeTransformer func(*Node) *Node

func TreeWalk(node *Node, transform NodeTransformer) *Node {
	children := node.Children
	if len(node.Children) > 0 {
		children = make([]*Node, 0, len(node.Children))
		for _, child := range node.Children {
			if result := TreeWalk(child, transform); result != nil {
				children = append(children, result)
			}
		}

		node.Children = children
	}

	return transform(node)
}

func canonicalizeNodeSortOrder(root *Node) {
	TreeWalk(root, func(node *Node) *Node {
		switch {
		case node.Type == AND || node.Type == OR:
			SortNodesInPlace(node.Children)
			return node

		case node.Type == WITHOUT:
			SortNodesInPlace(node.Children[1:])
			return node

		default:
			return node
		}
	})
}

func (ctx *optimizeContext) markChanged(msg string) {
	logrus.Debug(msg)
	ctx.changed = true
}

func (ctx *optimizeContext) optRemoveSelfCancelingWithout(node *Node) *Node {
	if node.Type == WITHOUT {
		if anyNode(node.Children[1:], node.Children[0].EqualTo) {
			ctx.markChanged("Delete WITHOUT that is canceling itself out")
			return EmptyQueryNode
		}
	}

	return node
}

func (ctx *optimizeContext) optCombineHierarchy(node *Node) *Node {
	if node.Type == WITHOUT {
		// if the first child is a without node, we append our other children
		// to it and return this first child
		candidate := node.Children[0]
		if candidate.Type == WITHOUT {
			candidate.Children = append(candidate.Children, node.Children[1:]...)
			SortNodesInPlace(candidate.Children[1:])

			ctx.markChanged("Combine nested WITHOUTs.")
			return candidate
		}
	}

	if node.Type == AND || node.Type == OR {
		if anyNode(node.Children, ofType(node.Type)) {
			children := []*Node{}
			for _, child := range node.Children {
				if node.Type == child.Type {
					children = append(children, child.Children...)
				} else {
					children = append(children, child)
				}
			}

			SortNodesInPlace(children)
			node.Children = children
			ctx.markChanged("Combine nested " + node.Type.String())
		}
	}

	return node
}

func (ctx *optimizeContext) optRemoveUnnecessaryNodes(node *Node) *Node {
	if node.Type == AND && len(node.Children) == 0 {
		ctx.markChanged("Replace empty AND with 'all' node")
		return AllQueryNode
	}

	if node.Type == OR && len(node.Children) == 0 {
		ctx.markChanged("Replace empty OR with 'empty' node")
		return EmptyQueryNode
	}

	if node.Type == WITHOUT || node.Type == OR || node.Type == AND {
		if len(node.Children) == 1 {
			ctx.markChanged("Remove one-child node of type " + node.Type.String())
			return node.Children[0]
		}
	}

	if node.Type == NOT && node.Children[0].Type == NOT {
		ctx.markChanged("Remove double negation.")
		return node.Children[0].Children[0]
	}

	switch {
	case node.Type == WITHOUT && node.Children[0] == EmptyQueryNode:
		ctx.markChanged("Simplify WITHOUT that has no elements")
		return EmptyQueryNode

	case node.Type == WITHOUT && anyNode(node.Children[1:], AllQueryNode.EqualTo):
		ctx.markChanged("Simplify WITHOUT that would remove all elements")
		return EmptyQueryNode

	case node.Type == AND && anyNode(node.Children, EmptyQueryNode.EqualTo):
		ctx.markChanged("Simplify AND with 'empty' node")
		return EmptyQueryNode

	case node.Type == OR && anyNode(node.Children, AllQueryNode.EqualTo):
		ctx.markChanged("Simplify OR with 'all' node")
		return AllQueryNode
	}

	return node
}

func (ctx *optimizeContext) optSimplifyChildren(node *Node) *Node {
	count := len(node.Children)
	switch node.Type {
	case OR:
		node.Children = deduplicate(filterNodes(node.Children, not(EmptyQueryNode.EqualTo)))

	case AND:
		node.Children = deduplicate(filterNodes(node.Children, not(AllQueryNode.EqualTo)))

	case WITHOUT:
		children := deduplicate(filterNodes(node.Children[1:], not(EmptyQueryNode.EqualTo)))
		if len(children) != len(node.Children)-1 {
			node.Children = append(node.Children[:1], children...)
		}
	}

	if count != len(node.Children) {
		ctx.markChanged("Remove noop-nodes and deduplicate nodes from " + node.Type.String())
	}
	return node
}

func (ctx *optimizeContext) optSimplifyCancelingOperationAndWithout(node *Node) *Node {
	if node.Type == WITHOUT && node.Children[0].Type == AND {
		isNoop := anyNode(filterNodes(node.Children[1:], ofType(QUERY)), func(child *Node) bool {
			return containsOnly(child.Query)(node.Children[0])
		})

		if isNoop {
			ctx.markChanged("Remove a WITHOUT node that is always empty")
			return EmptyQueryNode
		}
	}

	if node.Type == OR && anyNode(node.Children, ofType(WITHOUT)) {
		for _, termNode := range filterNodes(node.Children, not(ofType(WITHOUT))) {
			for _, woNode := range filterNodes(node.Children, ofType(WITHOUT)) {
				if anyNode(woNode.Children[1:], termNode.EqualTo) {
					// we now know, that termNode cancels itself out inside of the
					// WITHOUT and the OR. We can remove it from both.

					node.Children = filterNodes(node.Children, not(termNode.EqualTo))
					woNode.Children = filterNodes(woNode.Children, not(termNode.EqualTo))
					ctx.markChanged("Remove a term that has no effect in combination with OR/WITHOUT")
				}
			}
		}
	}

	return node
}

func (ctx *optimizeContext) optImplementNotUsingWithout(node *Node) *Node {
	if node.Type == NOT {
		ctx.markChanged("Replace NOT with a WITHOUT node")
		return NewOpNode(WITHOUT, AllQueryNode, node.Children[0])
	}

	return node
}

var nodeSfw = NewQueryNode("f:sfw")
var nodeNsfw = NewQueryNode("f:nsfw")
var nodeNsfl = NewQueryNode("f:nsfl")
var nodeNsfp = NewQueryNode("f:nsfp")

func (ctx *optimizeContext) optSimplifyFlags(node *Node) *Node {
	if node.Type == OR && len(node.Children) == 3 {
		if node.Children[0].EqualTo(nodeNsfp) && node.Children[1].EqualTo(nodeNsfw) && node.Children[2].EqualTo(nodeSfw) {
			ctx.markChanged("Replace nsfp/nsfw/sfw with NOT nsfl")
			return NewOpNode(NOT, nodeNsfl)
		}

		if node.Children[0].EqualTo(nodeNsfl) && node.Children[1].EqualTo(nodeNsfw) && node.Children[2].EqualTo(nodeSfw) {
			ctx.markChanged("Replace nsfl/nsfw/sfw with NOT nsfp")
			return NewOpNode(NOT, nodeNsfp)
		}
	}

	return node
}

func (ctx *optimizeContext) optMoveWithoutOutOfAnd(node *Node) *Node {
	if node.Type == AND && anyNode(node.Children, ofType(WITHOUT)) {
		for idx, child := range node.Children {
			if child.Type == WITHOUT {
				ctx.markChanged("Moving WITHOUT out of an AND")

				// remove the without node and add its children to the and node.
				node.Children = append(node.Children[:idx], node.Children[idx+1:]...)
				node.Children = append(node.Children, child.Children[0])
				SortNodesInPlace(node.Children)
				return NewOpNode(WITHOUT, node, child.Children[1:]...)
			}
		}
	}

	return node
}

func containsOnly(query string) NodeMatcher {
	return func(node *Node) bool {
		switch node.Type {
		case QUERY:
			return node.Query == query

		case AND:
			return anyNode(node.Children, containsOnly(query))

		case WITHOUT:
			return containsOnly(query)(node.Children[0])
		}

		return false
	}
}
