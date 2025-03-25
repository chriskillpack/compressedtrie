// Package compressedtrie implements a compressed Trie which provides the same
// functionality as a traditional Trie but using fewer nodes and less memory.
// Currently it only supports strings.
//
// A compressed Trie, aka a radix tree, achieves compression by storing shared
// prefixes (called labels) on the edges between letters or portions of words.

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
	"bufio"
	"encoding/binary"
	"errors"
	"io"
	"maps"
	"slices"
	"strings"
)

var (
	ErrUnsupportedVersion = errors.New("unsupported version of the file format")
	ErrInvalidFormat      = errors.New("invalid file format")
)

type Node struct {
	label    string
	children map[byte]*Node
	isWord   bool
}

type Tree struct {
	root *Node
	N    int // The number of nodes in the tree
}

type SerializedTreeHeader struct {
	Magic   uint32 // magic number (CtreeMagic)
	Version uint32 // file format version
	Nodes   uint32 // number of nodes in the tree
}

const (
	// 32-bit magic number for the serialized tree binary format
	CtreeMagic uint32 = 'C'<<24 | 'T'<<16 | 'R'<<8 | 'E'
	// File format version
	Version uint32 = 1
)

// NewTree creates an empty instance of Tree, ready for word insertion.
func NewTree() *Tree {
	return &Tree{root: &Node{children: make(map[byte]*Node)}, N: 1}
}

// Insert adds a word into t.
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
			t.N++

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
		t.N++
		newNode.children[remainder[0]] = child
		child.label = remainder

		cur.children[firstChar] = newNode
	}
}

// FindWordsWithPrefix returns all the words in the tree that start with prefix.
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

// Serialize a tree into an io.Writer. The serialized format is binary.
func (t *Tree) Serialize(w io.Writer) error {
	if int(uint32(t.N)) != t.N {
		panic("node count exceeds file format")
	}

	buf := bufio.NewWriter(w)
	hdr := SerializedTreeHeader{
		Magic:   CtreeMagic,
		Version: Version,
		Nodes:   uint32(t.N),
	}
	if err := binary.Write(buf, binary.BigEndian, hdr); err != nil {
		return err
	}

	t.serializeNode(t.root, buf)
	return buf.Flush()
}

// DeserializeTree returns a *Tree from an io.Reader. Returns ErrUnsupportedVersion
// if the serialize format is an unsupported version, ErrInvalidFormat if the
// file is unrecognized.
func DeserializeTree(r io.Reader) (*Tree, error) {
	tree := NewTree()

	buf := bufio.NewReader(r)

	// Read the header in
	hdr := SerializedTreeHeader{}
	if err := binary.Read(buf, binary.BigEndian, &hdr); err != nil {
		return nil, err
	}
	if hdr.Magic != CtreeMagic {
		return nil, ErrInvalidFormat
	}
	if hdr.Version != Version {
		return nil, ErrUnsupportedVersion
	}

	tree.N = int(hdr.Nodes)

	if err := deserializeNode(tree.root, buf); err != nil {
		return nil, err
	}

	return tree, nil
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

func (t *Tree) serializeNode(node *Node, buf *bufio.Writer) error {
	// Each node starts with the node label (u16 length, bytes of label string)
	if _, err := buf.Write(serializeString(node.label)); err != nil {
		return err
	}

	// Followed by u8 for isWord and then u8 for the number of children the node has
	var err error
	switch node.isWord {
	case false:
		err = buf.WriteByte(0)
	case true:
		err = buf.WriteByte(1)
	}
	if err != nil {
		return err
	}
	if err := buf.WriteByte(byte(len(node.children))); err != nil {
		return err
	}

	// Then we iterate over the keys in the node, write out the child key
	// and then recurse into the child.
	for _, k := range slices.Sorted(maps.Keys(node.children)) {
		if err := buf.WriteByte(k); err != nil {
			return err
		}
		if err := t.serializeNode(node.children[k], buf); err != nil {
			return err
		}
	}

	return nil
}

func deserializeNode(node *Node, buf *bufio.Reader) error {
	var (
		err       error
		ncb, w, k byte
	)

	node.label, err = deserializeString(buf)
	if err != nil {
		return err
	}

	if w, err = buf.ReadByte(); err != nil {
		return err
	}
	node.isWord = w == 1

	if ncb, err = buf.ReadByte(); err != nil {
		return err
	}
	node.children = make(map[byte]*Node, int(ncb))
	for range int(ncb) {
		// Read key
		if k, err = buf.ReadByte(); err != nil {
			return err
		}
		node.children[k] = &Node{}
		if err = deserializeNode(node.children[k], buf); err != nil {
			return err
		}

	}
	return err
}

func serializeString(s string) []byte {
	ls := uint16(len(s))
	if int(ls) != len(s) {
		panic("string length exceeds file format")
	}

	out := make([]byte, 2+len(s))
	binary.BigEndian.PutUint16(out, ls)
	copy(out[2:], s)

	return out
}

func deserializeString(r io.Reader) (string, error) {
	// Read the length of the string
	var blen [2]byte
	if _, err := io.ReadFull(r, blen[:]); err != nil {
		return "", err
	}

	slen := int(binary.BigEndian.Uint16(blen[:]))
	scratch := make([]byte, slen)

	if _, err := io.ReadFull(r, scratch); err != nil {
		return "", err
	}

	return string(scratch), nil
}
