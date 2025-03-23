# Compressed Trie

A compressed trie (or radix tree) is a more space efficient form of a trie that retains the fast word and prefix lookup. It has the same big-O performance as a trie but in practice is faster because it uses fewer nodes and stores data in more cache-efficient ways.

Don't just take my word for it, here are some images. Side by side below are the compressed trie (left) and traditional trie (right) representations of the words romane, romanus, romulus, rubens, ruber, rubicon and rubicundus. A double circle represents the end of a word. Note that the compressed trie has 14 nodes, while the more traditional trie has 28. The longest path in the compressed trie is 4 and 10 in the traditional.

<p float="left">
  <img src="images/compressed_trie.png" width="45%" style="vertical-align: top;" />
  <img src="images/traditional_trie.png" width="45%" style="vertical-align: top;" />
</p>

## How to use

Currently the tree only supports the minimal set of features I needed, `Insert()`, `FindWordsWithPrefix()`, `Serialize` and `Deserialize`.

```go
    tree := compressedtrie.NewTree()
    example_words := []string{"test", "toaster", "toasting", "slow", "slowly"}

    for _, word := range example_words {
        tree.Insert(word)
    }

    tree.FindWordsByPrefix("t") // returns []string{"test", "toaster", "toasting"}
```

Serialization and deserialization allows for offline tree building

```go
    # In your offline builder
    f, err := os.Create("prefixes.ctrie")
    defer f.Close()
    tree.Serialize(f)
```

```go
    # In your runtime
    f, err := os.Open("prefixes.ctrie")
    tree := compressedtrie.Deserialize(f)
    defer f.Close()
```

Internally the Serialize and Deserialize routines use buffered I/O to minimize memory overhead while laying out the file.

## Tests

```
go test .
```

Some of the tests use test files, which can be updated. Don't forget to check them in!

```
go test . -update
```
