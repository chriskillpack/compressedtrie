package compressedtrie

// Online dot file viewer https://dreampuf.github.io/GraphvizOnline/?engine=dot#digraph%20G%20%7B%0A%0A%7D

import (
	"flag"
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite testdata/*.dot files")

func TestInsertWord(t *testing.T) {
	cases := []struct {
		Name    string
		Words   []string
		DotFile string
	}{
		{"Simple", []string{"alphabet", "elephant", "alpha"}, "testdata/simple.dot"},
		{"Wikipedia example", []string{"romane", "romanus", "romulus", "rubens", "ruber", "rubicon", "rubicundus"}, "testdata/insert_wiki.dot"},
		{"Wikipedia example 2", []string{"test", "toaster", "toasting", "slow", "slowly"}, "testdata/insert_wiki_2.dot"},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			tree := NewTree()

			for _, word := range tc.Words {
				tree.Insert(word)
			}

			actual := asDot(tree)
			if *update {
				t.Logf("rewriting %s", tc.DotFile)
				if err := os.WriteFile(tc.DotFile, []byte(actual), 0666); err != nil {
					t.Fatal(err)
				}
				return
			}

			expected, err := os.ReadFile(tc.DotFile)
			if err != nil {
				t.Fatal(err)
			}
			if actual != string(expected) {
				t.Errorf("Differing output\nActual=%q\nExpected=%q\n", actual, expected)
			}
		})
	}
}

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
		label := fmt.Sprintf("(%t)", node.isWord)
		sb.WriteString(fmt.Sprintf("  n%d [label=\"%s\"];\n", nodeID, label))

		if parentID >= 0 {
			sb.WriteString(fmt.Sprintf("  n%d -> n%d [label=\"%s\"];\n", parentID, nodeID, node.prefix))
		}

		for _, k := range slices.Sorted(maps.Keys(node.children)) {
			traverse(node.children[k], nodeID)
		}
	}
	traverse(tree.root, -1)

	sb.WriteString("}\n")
	return sb.String()
}
