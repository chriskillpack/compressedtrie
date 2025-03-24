package compressedtrie

// Online dot file viewer https://dreampuf.github.io/GraphvizOnline/?engine=dot#digraph%20G%20%7B%0A%0A%7D

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path"
	"slices"
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

func TestFindWordsWithPrefix(t *testing.T) {
	cases := []struct {
		Name     string
		Words    []string
		Prefix   string
		Expected []string
	}{
		{"Exact match", []string{"test", "toaster", "toasting"}, "test", []string{"test"}},
		{"Matching prefix", []string{"test", "toaster", "toasting"}, "to", []string{"toaster", "toasting"}},
		{"No match", []string{"test", "toaster", "toasting"}, "a", []string{}},
		{"Prefix too long", []string{"test", "toaster", "toasting"}, "toastinger", []string{}},
		{"Everything", []string{"test", "toaster", "toasting"}, "", []string{"test", "toaster", "toasting"}},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			tree := NewTree()

			for _, word := range tc.Words {
				tree.Insert(word)
			}

			actual := tree.FindWordsWithPrefix(tc.Prefix)
			if !slices.Equal(actual, tc.Expected) {
				t.Errorf("Returned words don't match. Expected: %v\nActual: %v\n", tc.Expected, actual)
			}
		})
	}
}

func TestSerialize(t *testing.T) {
	words := []string{"alphabet", "elephant", "alpha"}
	tree := NewTree()
	for _, word := range words {
		tree.Insert(word)
	}

	buf := &bytes.Buffer{}
	if err := tree.Serialize(buf); err != nil {
		t.Fatal(err)
	}
	const filename = "testdata/serialize.ctree"
	if *update {
		// Write the serialized tree to disk
		if err := os.WriteFile(filename, buf.Bytes(), 0666); err != nil {
			t.Fatal(err)
		}
		// Write the serialized tree as a dot file
		if err := os.WriteFile("testdata/serialize.dot", []byte(asDot(tree)), 0666); err != nil {
			t.Fatal(err)
		}
		return
	}

	expected, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(buf.Bytes(), expected) {
		t.Errorf("Actual serialized tree does not match expected")
	}
}

func TestDeserialize(t *testing.T) {
	f, err := os.Open("testdata/serialize.ctree")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	tree, err := DeserializeTree(f)
	if err != nil {
		t.Fatal(err)
	}
	actual := asDot(tree)
	expected, err := os.ReadFile("testdata/serialize.dot")
	if err != nil {
		t.Fatal(err)
	}
	if actual != string(expected) {
		t.Errorf("Differing output\nActual=%q\nExpected=%q\n", actual, expected)
	}
}

func TestPerf(t *testing.T) {
	t.Skip("Disabled") // For performance measurements

	file_suffixes := []string{"10", "100", "200", "500", "1000", "5000", "10000", "20000", "50000", "100000", "all"}
	for _, suffix := range file_suffixes {
		filepath := path.Join("perf", fmt.Sprintf("words_%s.sid", suffix))
		ctree, err := treeFromSID(filepath)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("file %s has %d nodes", filepath, ctree.nodes)
	}
}
