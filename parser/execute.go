package parser

import (
	"github.com/mopsalarm/go-pr0gramm-tags/store"
	"fmt"
)

type IteratorFactory func(string) store.ItemIterator

func ToIterator(node *Node, makeIter IteratorFactory) store.ItemIterator {
	switch node.Type {
	case QUERY:
		if node.EqualTo(EmptyQueryNode) {
			return store.NewEmptyIterator()
		} else {
			return makeIter(node.Query)
		}

	case AND:
		return store.NewAndIterator(nodesToIterator(node.Children, makeIter)...)

	case OR:
		return store.NewOrIterator(nodesToIterator(node.Children, makeIter)...)

	case WITHOUT:
		return store.NewDiffIterator(nodesToIterator(node.Children, makeIter)...)

	case NOT:
		return ToIterator(NewOpNode(WITHOUT, AllQueryNode, node.Children[0]), makeIter)

	default:
		panic(fmt.Errorf("Can not create iterator for node of type " + node.Type.String()))
	}
}

func nodesToIterator(nodes []*Node, makeIter IteratorFactory) []store.ItemIterator {
	children := make([]store.ItemIterator, len(nodes))
	for idx, child := range nodes {
		children[idx] = ToIterator(child, makeIter)
	}

	return children
}

