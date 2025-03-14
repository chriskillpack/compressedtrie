package compressedtrie

// A compressed Trie (CTrie) is a Trie variant that uses fewer nodes and memory.
// This is achieved by storing shared prefixes through the tree along the edges
// between nodes.
//
// The words "HELLO" and "HELPER" will be stored in a Trie, 8 nodes.
// +---+      +---+      +---+      +---+      +---+
// | H | ---> | E | ---> | L | ---> | L | ---> | O |
// +---+      +---+      +---+      +---+      +---+
//                         |
//                       +---+      +---+      +---+
//                       | P | ---> | E | ---> | R |
//                       +---+      +---+      +---+
//
// And stored in a CTrie in only 4 nodes.
// +---+  HEL  +---+  LO  +---+
// |   | ----> |   | ---> |   |
// +---+       +---+      +---+
//               | PER
//             +---+
//             |   |
//             +---+
// In addition finding the word HELPER in a Trie required visiting 8 nodes, but
// only 3 in a compressed trie. These factors plus the lack of guarantee nodes
// are laid out linearly in memory so cache coherence is low when using a Trie.
// In a CTrie the prefix parts stored on edges (called labels) have their
// characters sequentially in memory.

import (
	"maps"
	"slices"
	"strings"
)

type Node struct {
	label    string
	children map[byte]*Node
	isWord   bool
}

type Tree struct {
	root *Node
}

func NewTree() *Tree {
	return &Tree{root: &Node{children: make(map[byte]*Node)}}
}

func (t *Tree) Insert(word string) {
	cur := t.root

	for {
		if word == "" {
			// Trivial case, we have reached the end of the word so mark the
			// current node as a word (by definition) and return.
			cur.isWord = true
			return
		}

		// Check if the current node has a child that starts with the first
		// character of the word
		firstChar := word[0]
		child, exists := cur.children[firstChar]

		if !exists {
			// No child exists, add a child with the word as the label. From the
			// definition this also means that the child is a word.
			cur.children[firstChar] = &Node{
				children: make(map[byte]*Node),
				label:    word,
				isWord:   true,
			}

			return
		}

		// A child does exist, find the common prefix between the child's label
		// and the word
		label := child.label
		commonLen := 0
		for commonLen < len(word) && commonLen < len(label) && word[commonLen] == label[commonLen] {
			commonLen++
		}

		if commonLen == len(label) {
			// The word fully contains the label as a prefix. Discard the common
			// part and descend into the child.
			word = word[commonLen:]
			cur = child
			continue
		}

		// Word/prefix comparison stopped before reaching the end of the label so either there is a partial match or
		// the end of word was reached first. Example of partial match: comparing word 'octopus' and label 'octonaut'.
		// Comparison stops at index 4 'o' and 'n' respectively. Example of reaching the end of word first: comparing
		// word 'alpha' with label 'alphabet'.
		//
		// In either case we need to create a new node between the current and child nodes that holds the common prefix,
		// 'octo' and 'alpha' from the two examples. The new node will replace child in the current node (the parent).
		// In the parial match case the child's label is updated to the remainder, 'naut'.
		commonPrefix := label[:commonLen]
		remainder := label[commonLen:]
		newNode := &Node{
			label:    commonPrefix,
			children: make(map[byte]*Node),
			isWord:   remainder == "",
		}
		newNode.children[remainder[0]] = child
		child.label = remainder

		cur.children[firstChar] = newNode
	}
}

func (t *Tree) FindWordsWithPrefix(prefix string) []string {
	var words []string

	// Starting at the root, descend by prefix
	cur := t.root
	currentPath := ""
	for {
		if prefix == "" {
			// Search prefix exhausted. At this point we traverse the tree below
			// this to recover the words
			t.gatherWords(cur, currentPath, &words)
			return words
		}
		firstChar := prefix[0]
		child, exists := cur.children[firstChar]
		if !exists {
			// Cannot go any further, nothing to return
			return nil
		}

		// Check if the remaining prefix entirely covers the child's label, e.g.
		// prefix="buller" entirely contains the label "bull".
		label := child.label
		if len(prefix) >= len(label) && prefix[:len(label)] == label {
			// It does, move into the child. To do that update the currently
			// accumulated path, and update the remaining prefix.
			currentPath += label
			prefix = prefix[len(label):]
			cur = child
			continue
		}

		// Next case: the label is longer than the path prefix. Gather all words
		// under the child and we are finished.
		if strings.HasPrefix(label, prefix) {
			t.gatherWords(child, currentPath+label, &words)
			return words
		}
	}
}

func (t *Tree) gatherWords(node *Node, currentPath string, words *[]string) {
	// If this node marks a word then add it
	if node.isWord {
		*words = append(*words, currentPath)
	}

	// Iterate over the children
	for _, k := range slices.Sorted(maps.Keys(node.children)) {
		child := node.children[k]
		t.gatherWords(child, currentPath+child.label, words)
	}
}
