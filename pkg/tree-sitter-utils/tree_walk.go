package tree_sitter_utils

import sitter "github.com/smacker/go-tree-sitter"

// Walk traverses the (sub-)tree in post-order traversal (visiting all child nodes before the parent node),
// invoking the provided callback function for each node.
func Walk(node *sitter.Node, fn func(*sitter.Node)) {
	var cursor = sitter.NewTreeCursor(node)
	defer cursor.Close()

	// Walk() traverses the tree using recursion.
	// It attempts to locate the left-most leaf node, and then work its way side-ways (to next sibling).
	// When all siblings are handled, it moves back to the parent node.
	var walk func(*sitter.TreeCursor)
	walk = func(cursor *sitter.TreeCursor) {
		if cursor.GoToFirstChild() {
			walk(cursor) // dive into left-most / first child's subtree
		}

		// after handling the children node(s) (or not if we are leaf),
		// invoke the user-supplied callback function for the current node.
		fn(cursor.CurrentNode())

		// attempt to traverse the subtree of the next sibling,
		// or goto parent node if there aren't any siblings left.
		if hasSibling := cursor.GoToNextSibling(); hasSibling {
			walk(cursor)
		} else {
			cursor.GoToParent()
		}
	}

	// kick-start traversal!
	walk(cursor)
}

// Find returns a list of all nodes for which the given predicate function returns true.
func Find(node *sitter.Node, fn func(*sitter.Node) bool) []*sitter.Node {
	var nodes []*sitter.Node
	Walk(node, func(node *sitter.Node) {
		if fn(node) {
			nodes = append(nodes, node)
		}
	})
	return nodes
}
