package compressedtrie

type Node struct {
	prefix   string
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
			// current node as a word (by definition) and return
			cur.isWord = true
			return
		}

		// Check if the current node has a child that starts with the first
		// character of the word
		firstChar := word[0]
		child, exists := cur.children[firstChar]

		if !exists {
			// No child exists let's add a child in with the edge prefix being
			// the word. From the definition this also means that the child is a
			// word.
			cur.children[firstChar] = &Node{
				children: make(map[byte]*Node),
				prefix:   word,
				isWord:   true,
			}

			return
		}

		// A child does exist, find the common prefix along the child's edge
		prefix := child.prefix
		commonLen := 0
		for commonLen < len(word) && commonLen < len(prefix) && word[commonLen] == prefix[commonLen] {
			commonLen++
		}

		if commonLen == len(prefix) {
			// If we completely match the child's prefix then our word could still extend past the prefix. Discard the
			// common part and descend into the child. In the case that the word exactly matches the prefix, it will be
			// handled by the empty string check at the top.
			word = word[commonLen:]
			cur = child
			continue
		}

		// Word/prefix comparison stopped before reaching the end of the prefix so either there is a partial match or
		// the end of word was reached first. Example of partial match: comparing word 'octopus' and prefix 'octonaut'.
		// Comparison stops at index 4 'o' and 'n' respectively. Example of reaching the end of word first: comparing
		// word 'alpha' with prefix 'alphabet'.
		//
		// In either case we need to create a new node between the current and child nodes that holds the common prefix,
		// 'octo' and 'alpha' from the two examples. The new node will replace child in the current node (the parent).
		// In the parial match case the child's prefix is updated to the remainder, 'naut'.
		commonPrefix := prefix[:commonLen]
		remainder := prefix[commonLen:]
		newNode := &Node{
			prefix:   commonPrefix,
			children: make(map[byte]*Node),
			isWord:   remainder == "",
		}
		newNode.children[remainder[0]] = child
		child.prefix = remainder

		cur.children[firstChar] = newNode
	}
}
