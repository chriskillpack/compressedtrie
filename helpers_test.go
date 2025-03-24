// Test helpers

package compressedtrie

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"slices"
	"strings"
)

// Generate a DOT file for this tree
func asDot(tree *Tree) string {
	var sb strings.Builder
	sb.WriteString("digraph Trie {\n")
	sb.WriteString("  node [shape=circle];\n")

	nodeCounter := 0
	nodeIDs := make(map[*Node]int)
	var traverse func(node *Node, parentID int)
	traverse = func(node *Node, parentID int) {
		nodeID, exists := nodeIDs[node]
		if !exists {
			nodeID = nodeCounter
			nodeIDs[node] = nodeCounter
			nodeCounter++
		}

		// Label with prefix and isWord status
		label := " [label=\"\"]"
		if node.isWord {
			label = " [label=\"\", shape=doublecircle]"
		}
		sb.WriteString(fmt.Sprintf("  n%d%s;\n", nodeID, label))

		if parentID >= 0 {
			sb.WriteString(fmt.Sprintf("  n%d -> n%d [label=\"%s\"];\n", parentID, nodeID, node.label))
		}

		for _, k := range slices.Sorted(maps.Keys(node.children)) {
			traverse(node.children[k], nodeID)
		}
	}
	traverse(tree.root, -1)

	sb.WriteString("}\n")
	return sb.String()
}

const stringSetMagic uint32 = 'S'<<24 | 'T'<<16 | 'R'<<8 | 'S'

type serializedStringSetHeader struct {
	Magic    uint32
	Version  uint32 // currently 1
	NStrings uint32
	MaxLen   uint16

	// Followed by each string one after the other
	// Each string is of the form byte length (int16) and then the bytes of the string
	// Strings are stored as UTF-8
}

func treeFromSID(sidfile string) (*Tree, error) {
	f, err := os.Open(sidfile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	buf := bufio.NewReader(f)
	hdr := serializedStringSetHeader{}
	if err := binary.Read(buf, binary.BigEndian, &hdr); err != nil {
		return nil, err
	}

	if hdr.Version != 1 || hdr.Magic != stringSetMagic {
		return nil, errors.New("bad file format")
	}
	scratch := make([]byte, hdr.MaxLen)

	tree := NewTree()
	for range hdr.NStrings {
		slen, err := binary.ReadUvarint(buf)
		if err != nil {
			return nil, err
		}
		io.ReadFull(buf, scratch[:slen])
		tree.Insert(string(scratch[:slen]))
	}

	return tree, nil
}
