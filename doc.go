// Package lucene is a small, dependency-free, in-memory full-text search
// engine in the style of Apache Lucene. It provides a configurable text
// analysis pipeline, an inverted index with term frequencies and positions,
// a rich query model with a query-string parser, and BM25 relevance ranking.
//
// # Model
//
// A Document has a unique ID and a set of named Fields, each holding raw text.
// When a document is added, every field is run through an Analyzer, which
// tokenizes the text, lowercases it, drops stop words, and stems the remaining
// tokens. The resulting terms are stored in an inverted Index that records, per
// field and per term, which documents contain the term, how often, and at which
// positions. Positions enable phrase matching; frequencies and field lengths
// feed BM25 scoring.
//
// Queries implement the Query interface. The provided implementations are:
//
//   - TermQuery      - a single analyzed term in a field.
//   - PhraseQuery    - terms occurring adjacently and in order.
//   - PrefixQuery    - all terms in a field sharing a prefix (wildcard "net*").
//   - RangeQuery     - all terms within a lexical range [a TO z] or {1 TO 9}.
//   - BooleanQuery   - AND/OR/NOT combination via Should/Must/MustNot clauses.
//   - MatchAllQuery  - every live document.
//
// The Parser turns a query string into a Query tree. Its grammar supports field
// qualifiers (title:go), phrases ("hello world"), required (+) and prohibited
// (-) clauses, prefix wildcards (net*), ranges ([a TO z]), and parenthesized
// grouping.
//
// Searching a query returns a Result containing the total number of matching
// documents and the top-ranked Hits. Ranking is by descending BM25 score with
// ties broken by ascending document ID, so results are fully deterministic.
//
// # Usage
//
//	idx := lucene.NewIndex(lucene.NewStandardAnalyzer())
//	_ = idx.Add(lucene.Document{
//		ID: "1",
//		Fields: map[string]string{
//			"title": "The Go Programming Language",
//			"body":  "Go is an open source programming language.",
//		},
//	})
//	_ = idx.Add(lucene.Document{
//		ID: "2",
//		Fields: map[string]string{
//			"title": "Rust in Action",
//			"body":  "Rust is a systems programming language.",
//		},
//	})
//
//	res, _ := idx.SearchString("body:programming +language", 10)
//	for _, hit := range res.Hits {
//		fmt.Printf("%s %.3f\n", hit.ID, hit.Score)
//	}
//
// The engine is safe for concurrent use: an Index may be read and written from
// multiple goroutines. All operations are deterministic given the same inputs.
package lucene
