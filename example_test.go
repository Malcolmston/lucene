package lucene_test

import (
	"fmt"

	"github.com/malcolmston/lucene"
)

// Example demonstrates indexing a few documents and running a parsed query with
// BM25 ranking.
func Example() {
	idx := lucene.NewIndex(lucene.NewStandardAnalyzer())

	_ = idx.Add(lucene.Document{ID: "1", Fields: map[string]string{
		"title": "The Go Programming Language",
		"body":  "Go is an open source programming language designed at Google.",
	}})
	_ = idx.Add(lucene.Document{ID: "2", Fields: map[string]string{
		"title": "Rust in Action",
		"body":  "Rust is a systems programming language focused on safety.",
	}})
	_ = idx.Add(lucene.Document{ID: "3", Fields: map[string]string{
		"title": "Network Programming with Go",
		"body":  "Building network servers and clients using the Go programming language.",
	}})

	// Documents mentioning "go" in the body, ranked by relevance.
	res, err := idx.SearchString("body:go", 10)
	if err != nil {
		panic(err)
	}
	fmt.Println("matches:", res.Total)
	for _, hit := range res.Hits {
		fmt.Println(hit.ID)
	}

	// Highlight the query term in a snippet.
	h := lucene.NewHighlighter(idx.Analyzer(), "[", "]")
	fmt.Println(h.Highlight("Go is an open source programming language.", lucene.NewTermQuery("body", "go")))

	// Output:
	// matches: 2
	// 1
	// 3
	// [Go] is an open source programming language.
}
