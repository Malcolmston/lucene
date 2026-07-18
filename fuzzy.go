package lucene

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// LevenshteinDistance returns the minimum number of single-character
// insertions, deletions, or substitutions required to transform a into b. It
// operates on Unicode code points (runes), so multi-byte characters count as a
// single edit. The result is symmetric: LevenshteinDistance(a, b) equals
// LevenshteinDistance(b, a). It is deterministic and runs in O(len(a)*len(b))
// time using two rolling rows of O(len(b)) space.
func LevenshteinDistance(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	la, lb := len(ra), len(rb)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}
	prev := make([]int, lb+1)
	curr := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}
	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			curr[j] = luMin3(prev[j]+1, curr[j-1]+1, prev[j-1]+cost)
		}
		prev, curr = curr, prev
	}
	return prev[lb]
}

// FuzzyQuery matches documents containing any term in a field whose Levenshtein
// edit distance from the query term is at most MaxEdits. It is the engine for
// approximate ("did you mean") matching that tolerates typos. Each matching
// term contributes its BM25 score, so a document that contains several near
// variants scores higher.
type FuzzyQuery struct {
	Field    string
	Term     string
	MaxEdits int
	Boost    float64
}

// NewFuzzyQuery builds a FuzzyQuery that matches terms within maxEdits edits of
// term. A negative maxEdits is treated as zero (exact match). An empty field
// defaults to "text". The term is compared against indexed terms after
// lowercasing but without stemming, so it matches the surface form recorded in
// the index.
func NewFuzzyQuery(field, term string, maxEdits int) *FuzzyQuery {
	if field == "" {
		field = defaultField
	}
	if maxEdits < 0 {
		maxEdits = 0
	}
	return &FuzzyQuery{Field: field, Term: term, MaxEdits: maxEdits, Boost: 1}
}

func (q *FuzzyQuery) boost() float64 {
	if q.Boost == 0 {
		return 1
	}
	return q.Boost
}

func (q *FuzzyQuery) match(idx *Index) map[int]float64 {
	out := map[int]float64{}
	term := strings.ToLower(strings.TrimSpace(q.Term))
	if term == "" {
		return out
	}
	fi, ok := idx.fields[q.Field]
	if !ok {
		return out
	}
	tlen := utf8.RuneCountInString(term)
	for t, ti := range fi.terms {
		// Length pruning: a term whose rune length differs by more than
		// MaxEdits can never be within edit distance MaxEdits.
		if luAbs(utf8.RuneCountInString(t)-tlen) > q.MaxEdits {
			continue
		}
		if LevenshteinDistance(term, t) > q.MaxEdits {
			continue
		}
		for num, s := range scoreTermIndex(idx, fi, ti, q.boost()) {
			out[num] += s
		}
	}
	return out
}

// String returns a text representation of the fuzzy query in the conventional
// "field:term~N" form.
func (q *FuzzyQuery) String() string {
	return fmt.Sprintf("%s:%s~%d", q.Field, q.Term, q.MaxEdits)
}

// luMin3 returns the smallest of three integers.
func luMin3(a, b, c int) int {
	m := a
	if b < m {
		m = b
	}
	if c < m {
		m = c
	}
	return m
}

// luAbs returns the absolute value of an integer.
func luAbs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
