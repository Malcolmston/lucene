package lucene

import (
	"fmt"
	"sort"
	"strings"
)

// Query is the interface implemented by all query types. A Query knows how to
// match documents in an index and produce per-document scores. Implementations
// are produced either directly or by the query Parser.
type Query interface {
	// match evaluates the query against the index and returns a map from
	// internal document number to that document's score contribution. The
	// caller (Searcher) holds the index read lock.
	match(idx *Index) map[int]float64
	// String returns a canonical, human-readable form of the query.
	String() string
}

// defaultField is used by queries that were constructed without an explicit
// field, and by the parser when no field qualifier is present.
const defaultField = "text"

// TermQuery matches documents that contain a single analyzed term in a field.
// It scores matches with BM25.
type TermQuery struct {
	Field string
	Term  string
	Boost float64
}

// NewTermQuery builds a TermQuery. An empty field defaults to "text".
func NewTermQuery(field, term string) *TermQuery {
	if field == "" {
		field = defaultField
	}
	return &TermQuery{Field: field, Term: term, Boost: 1}
}

func (q *TermQuery) boost() float64 {
	if q.Boost == 0 {
		return 1
	}
	return q.Boost
}

func (q *TermQuery) match(idx *Index) map[int]float64 {
	term := idx.analyzer.AnalyzeTerm(q.Term)
	return scoreTerm(idx, q.Field, term, q.boost())
}

// scoreTerm computes BM25 scores for a single analyzed term in a field.
func scoreTerm(idx *Index, field, term string, boost float64) map[int]float64 {
	out := map[int]float64{}
	if term == "" {
		return out
	}
	fi, ok := idx.fields[field]
	if !ok {
		return out
	}
	ti, ok := fi.terms[term]
	if !ok {
		return out
	}
	n := idx.numDocs
	df := len(ti.postings)
	termIDF := idf(n, df)
	avg := fi.avgFieldLen()
	for num, p := range ti.postings {
		out[num] = boost * bm25(termIDF, p.freq, fi.lengths[num], avg)
	}
	return out
}

// String returns a text representation of the term query.
func (q *TermQuery) String() string {
	return fmt.Sprintf("%s:%s", q.Field, q.Term)
}

// PhraseQuery matches documents in which the given terms occur adjacently and
// in order within a single field. Positional information recorded at index time
// drives the match. Scoring sums the BM25 contributions of the constituent
// terms for documents where the phrase is present.
type PhraseQuery struct {
	Field string
	Terms []string
	Boost float64
}

// NewPhraseQuery builds a PhraseQuery from raw terms. An empty field defaults to
// "text".
func NewPhraseQuery(field string, terms ...string) *PhraseQuery {
	if field == "" {
		field = defaultField
	}
	return &PhraseQuery{Field: field, Terms: terms, Boost: 1}
}

func (q *PhraseQuery) boost() float64 {
	if q.Boost == 0 {
		return 1
	}
	return q.Boost
}

func (q *PhraseQuery) match(idx *Index) map[int]float64 {
	out := map[int]float64{}
	// Analyze terms; drop empties.
	terms := make([]string, 0, len(q.Terms))
	for _, t := range q.Terms {
		if at := idx.analyzer.AnalyzeTerm(t); at != "" {
			terms = append(terms, at)
		}
	}
	if len(terms) == 0 {
		return out
	}
	if len(terms) == 1 {
		return scoreTerm(idx, q.Field, terms[0], q.boost())
	}
	fi, ok := idx.fields[q.Field]
	if !ok {
		return out
	}
	// Gather term indexes; all terms must exist.
	tis := make([]*termIndex, len(terms))
	for i, t := range terms {
		ti, ok := fi.terms[t]
		if !ok {
			return out
		}
		tis[i] = ti
	}
	// Candidate docs are those containing the first term.
	avg := fi.avgFieldLen()
	for num := range tis[0].postings {
		if !phraseMatch(tis, num) {
			continue
		}
		var score float64
		for i, t := range terms {
			p := tis[i].postings[num]
			score += bm25(idf(idx.numDocs, len(fi.terms[t].postings)), p.freq, fi.lengths[num], avg)
		}
		out[num] = q.boost() * score
	}
	return out
}

// phraseMatch reports whether the terms occur consecutively in the given
// document, using the recorded positions.
func phraseMatch(tis []*termIndex, num int) bool {
	first, ok := tis[0].postings[num]
	if !ok {
		return false
	}
	for _, start := range first.positions {
		if consecutiveFrom(tis, num, start) {
			return true
		}
	}
	return false
}

func consecutiveFrom(tis []*termIndex, num, start int) bool {
	for i := 1; i < len(tis); i++ {
		p, ok := tis[i].postings[num]
		if !ok {
			return false
		}
		if !containsInt(p.positions, start+i) {
			return false
		}
	}
	return true
}

// containsInt reports whether sorted-or-unsorted slice s contains v. Positions
// are appended in increasing order at index time, so a binary search is valid.
func containsInt(s []int, v int) bool {
	i := sort.SearchInts(s, v)
	return i < len(s) && s[i] == v
}

// String returns a text representation of the phrase query.
func (q *PhraseQuery) String() string {
	return fmt.Sprintf("%s:%q", q.Field, strings.Join(q.Terms, " "))
}

// PrefixQuery matches every document containing any term in a field that begins
// with the given prefix. It is the engine for wildcard-prefix queries such as
// "net*". Scores sum the BM25 contributions of all matching terms.
type PrefixQuery struct {
	Field  string
	Prefix string
	Boost  float64
}

// NewPrefixQuery builds a PrefixQuery. An empty field defaults to "text".
func NewPrefixQuery(field, prefix string) *PrefixQuery {
	if field == "" {
		field = defaultField
	}
	return &PrefixQuery{Field: field, Prefix: prefix, Boost: 1}
}

func (q *PrefixQuery) boost() float64 {
	if q.Boost == 0 {
		return 1
	}
	return q.Boost
}

func (q *PrefixQuery) match(idx *Index) map[int]float64 {
	out := map[int]float64{}
	// Prefixes are lowercased but not stemmed, so that "runn*" still matches
	// the stemmed term "run"... but stemming would defeat prefixing. We
	// lowercase only.
	prefix := strings.ToLower(strings.TrimSpace(q.Prefix))
	if prefix == "" {
		return out
	}
	fi, ok := idx.fields[q.Field]
	if !ok {
		return out
	}
	for term, ti := range fi.terms {
		if !strings.HasPrefix(term, prefix) {
			continue
		}
		sub := scoreTermIndex(idx, fi, ti, q.boost())
		for num, s := range sub {
			out[num] += s
		}
	}
	return out
}

// scoreTermIndex scores an already-resolved term index within a field.
func scoreTermIndex(idx *Index, fi *fieldIndex, ti *termIndex, boost float64) map[int]float64 {
	out := map[int]float64{}
	termIDF := idf(idx.numDocs, len(ti.postings))
	avg := fi.avgFieldLen()
	for num, p := range ti.postings {
		out[num] = boost * bm25(termIDF, p.freq, fi.lengths[num], avg)
	}
	return out
}

// String returns a text representation of the prefix query.
func (q *PrefixQuery) String() string {
	return fmt.Sprintf("%s:%s*", q.Field, q.Prefix)
}

// RangeQuery matches documents containing at least one term in a field that
// falls within a lexical range [Lower, Upper]. Bounds may be inclusive or
// exclusive, and either bound may be empty to leave that side unbounded. Terms
// are compared as strings, which yields correct numeric ordering for
// zero-padded or fixed-width numbers.
type RangeQuery struct {
	Field        string
	Lower        string
	Upper        string
	IncludeLower bool
	IncludeUpper bool
	Boost        float64
}

// NewRangeQuery builds an inclusive-by-default range query. An empty field
// defaults to "text".
func NewRangeQuery(field, lower, upper string, includeLower, includeUpper bool) *RangeQuery {
	if field == "" {
		field = defaultField
	}
	return &RangeQuery{
		Field:        field,
		Lower:        lower,
		Upper:        upper,
		IncludeLower: includeLower,
		IncludeUpper: includeUpper,
		Boost:        1,
	}
}

func (q *RangeQuery) boost() float64 {
	if q.Boost == 0 {
		return 1
	}
	return q.Boost
}

func (q *RangeQuery) inRange(term string) bool {
	if q.Lower != "" {
		if q.IncludeLower {
			if term < q.Lower {
				return false
			}
		} else if term <= q.Lower {
			return false
		}
	}
	if q.Upper != "" {
		if q.IncludeUpper {
			if term > q.Upper {
				return false
			}
		} else if term >= q.Upper {
			return false
		}
	}
	return true
}

func (q *RangeQuery) match(idx *Index) map[int]float64 {
	out := map[int]float64{}
	fi, ok := idx.fields[q.Field]
	if !ok {
		return out
	}
	for term, ti := range fi.terms {
		if !q.inRange(term) {
			continue
		}
		sub := scoreTermIndex(idx, fi, ti, q.boost())
		for num, s := range sub {
			out[num] += s
		}
	}
	return out
}

// String returns a text representation of the range query.
func (q *RangeQuery) String() string {
	lb, rb := "{", "}"
	if q.IncludeLower {
		lb = "["
	}
	if q.IncludeUpper {
		rb = "]"
	}
	lo, hi := q.Lower, q.Upper
	if lo == "" {
		lo = "*"
	}
	if hi == "" {
		hi = "*"
	}
	return fmt.Sprintf("%s:%s%s TO %s%s", q.Field, lb, lo, hi, rb)
}

// Occur specifies how a clause participates in a BooleanQuery.
type Occur int

const (
	// Should clauses contribute to the score; a document matches the boolean
	// query if it satisfies at least one Should clause (when there are no Must
	// clauses) or any combination alongside the Must clauses.
	Should Occur = iota
	// Must clauses are required: a matching document must satisfy every Must
	// clause.
	Must
	// MustNot clauses are prohibited: a matching document must satisfy none of
	// them. They never contribute to the score.
	MustNot
)

// Clause pairs a sub-query with its Occur semantics inside a BooleanQuery.
type Clause struct {
	Query Query
	Occur Occur
}

// BooleanQuery combines sub-queries with AND/OR/NOT semantics expressed through
// Occur values. Its score for a document is the sum of the scores of the
// Should and Must clauses that the document satisfies.
type BooleanQuery struct {
	Clauses []Clause
}

// NewBooleanQuery builds an empty boolean query. Use Add to attach clauses.
func NewBooleanQuery() *BooleanQuery {
	return &BooleanQuery{}
}

// Add appends a clause and returns the query for chaining.
func (q *BooleanQuery) Add(sub Query, occur Occur) *BooleanQuery {
	q.Clauses = append(q.Clauses, Clause{Query: sub, Occur: occur})
	return q
}

func (q *BooleanQuery) match(idx *Index) map[int]float64 {
	scores := map[int]float64{}
	// Track which documents satisfy each requirement.
	var mustSets []map[int]float64
	prohibited := map[int]struct{}{}
	shouldMatched := map[int]struct{}{}
	hasShould := false

	for _, c := range q.Clauses {
		sub := c.Query.match(idx)
		switch c.Occur {
		case Must:
			mustSets = append(mustSets, sub)
			for num, s := range sub {
				scores[num] += s
			}
		case Should:
			hasShould = true
			for num, s := range sub {
				scores[num] += s
				shouldMatched[num] = struct{}{}
			}
		case MustNot:
			for num := range sub {
				prohibited[num] = struct{}{}
			}
		}
	}

	out := map[int]float64{}
	for num, s := range scores {
		if _, bad := prohibited[num]; bad {
			continue
		}
		// Must: document must appear in every Must set.
		satisfiesMust := true
		for _, ms := range mustSets {
			if _, ok := ms[num]; !ok {
				satisfiesMust = false
				break
			}
		}
		if !satisfiesMust {
			continue
		}
		// When there are no Must clauses, at least one Should must match.
		if len(mustSets) == 0 && hasShould {
			if _, ok := shouldMatched[num]; !ok {
				continue
			}
		}
		out[num] = s
	}
	return out
}

// String returns a text representation of the boolean query.
func (q *BooleanQuery) String() string {
	parts := make([]string, 0, len(q.Clauses))
	for _, c := range q.Clauses {
		prefix := ""
		switch c.Occur {
		case Must:
			prefix = "+"
		case MustNot:
			prefix = "-"
		}
		parts = append(parts, prefix+c.Query.String())
	}
	return strings.Join(parts, " ")
}

// MatchAllQuery matches every live document with a constant score of 1. It is
// useful as a base for filtering with MustNot clauses.
type MatchAllQuery struct{}

func (q *MatchAllQuery) match(idx *Index) map[int]float64 {
	out := make(map[int]float64, idx.numDocs)
	for num, live := range idx.live {
		if live {
			out[num] = 1
		}
	}
	return out
}

// String returns the "*:*" text form of the match-all query.
func (q *MatchAllQuery) String() string { return "*:*" }
