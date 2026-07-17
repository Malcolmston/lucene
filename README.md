# lucene

Embedded full-text search engine for Go — a small, dependency-free, in-memory
search library in the style of Apache Lucene. Standard library only: no cgo, no
third-party modules.

## Features

- **Analysis pipeline** — tokenizer (splits on non-letter/digit), lowercase
  filter, configurable stop-word filter, and a Porter-style suffix-stripping
  stemmer.
- **Inverted index** — add / update / delete documents made of named text
  fields; postings carry term frequencies, positions, and per-field document
  lengths. Safe for concurrent use.
- **Query model** — term, phrase (positional), boolean (AND / OR / NOT via
  `+`/`-`), prefix/wildcard (`net*`), and lexical range (`[a TO z]`) queries.
- **Query parser** — turns a query string into a query tree, with field
  qualifiers, phrases, required/prohibited clauses, prefixes, ranges, and
  parenthesized grouping.
- **BM25 ranking** — top-N hits with scores, deterministic tie-breaking by
  document ID.
- **Highlighting** — wrap matched terms in a snippet with custom markers.

## Install

```
go get github.com/malcolmston/lucene
```

Requires Go 1.24 or later.

## Quick start

```go
package main

import (
	"fmt"

	"github.com/malcolmston/lucene"
)

func main() {
	idx := lucene.NewIndex(lucene.NewStandardAnalyzer())

	_ = idx.Add(lucene.Document{ID: "1", Fields: map[string]string{
		"title": "The Go Programming Language",
		"body":  "Go is an open source programming language designed at Google.",
	}})
	_ = idx.Add(lucene.Document{ID: "2", Fields: map[string]string{
		"title": "Network Programming with Go",
		"body":  "Building network servers and clients using the Go programming language.",
	}})

	// Parse and run a query: must contain "go", must not contain "rust".
	res, err := idx.SearchString("body:programming +body:go -rust", 10)
	if err != nil {
		panic(err)
	}
	fmt.Println("matches:", res.Total)
	for _, hit := range res.Hits {
		fmt.Printf("  %s  score=%.3f\n", hit.ID, hit.Score)
	}

	// Highlight matches in a snippet.
	h := lucene.NewHighlighter(idx.Analyzer(), "<b>", "</b>")
	fmt.Println(h.Highlight("Go is a great programming language.",
		lucene.NewTermQuery("body", "programming")))
}
```

## Query syntax

| Syntax        | Meaning                                  |
|---------------|------------------------------------------|
| `word`        | term query (default field)               |
| `title:word`  | term query on a specific field           |
| `"a b c"`     | phrase query (terms adjacent, in order)  |
| `+word`       | required clause (AND)                     |
| `-word`       | prohibited clause (NOT)                   |
| `net*`        | prefix / wildcard query                  |
| `[a TO z]`    | inclusive range query                    |
| `{a TO z}`    | exclusive range query                    |
| `(a b) +c`    | grouping                                 |

Unqualified, space-separated clauses combine as OR by default.

## Building

```
go build ./...
go vet ./...
go test ./...
```

## License

See repository.
