package lucene

import (
	"fmt"
	"strings"
)

// WildcardMatch reports whether s matches pattern, where the pattern may
// contain two wildcard metacharacters: '*' matches any (possibly empty)
// sequence of characters, and '?' matches exactly one character. All other
// characters match literally. Matching is performed over Unicode code points
// and is anchored: the whole of s must be consumed. It runs in linear time with
// backtracking on '*'.
func WildcardMatch(pattern, s string) bool {
	return luWildcard([]rune(pattern), []rune(s))
}

func luWildcard(p, s []rune) bool {
	pi, si := 0, 0
	star := -1
	sStar := 0
	for si < len(s) {
		switch {
		case pi < len(p) && (p[pi] == '?' || p[pi] == s[si]):
			pi++
			si++
		case pi < len(p) && p[pi] == '*':
			star = pi
			sStar = si
			pi++
		case star != -1:
			pi = star + 1
			sStar++
			si = sStar
		default:
			return false
		}
	}
	for pi < len(p) && p[pi] == '*' {
		pi++
	}
	return pi == len(p)
}

// WildcardQuery matches every document containing any term in a field that
// matches a glob-style pattern using '*' (any run of characters) and '?' (a
// single character). Unlike PrefixQuery the wildcards may appear anywhere in
// the pattern, for example "n?t*rk". Scores sum the BM25 contributions of all
// matching terms.
type WildcardQuery struct {
	Field   string
	Pattern string
	Boost   float64
}

// NewWildcardQuery builds a WildcardQuery. The pattern is lowercased before
// matching so it lines up with the lowercased terms in the index. An empty
// field defaults to "text".
func NewWildcardQuery(field, pattern string) *WildcardQuery {
	if field == "" {
		field = defaultField
	}
	return &WildcardQuery{Field: field, Pattern: pattern, Boost: 1}
}

func (q *WildcardQuery) boost() float64 {
	if q.Boost == 0 {
		return 1
	}
	return q.Boost
}

func (q *WildcardQuery) match(idx *Index) map[int]float64 {
	out := map[int]float64{}
	pattern := strings.ToLower(strings.TrimSpace(q.Pattern))
	if pattern == "" {
		return out
	}
	fi, ok := idx.fields[q.Field]
	if !ok {
		return out
	}
	pr := []rune(pattern)
	for t, ti := range fi.terms {
		if !luWildcard(pr, []rune(t)) {
			continue
		}
		for num, s := range scoreTermIndex(idx, fi, ti, q.boost()) {
			out[num] += s
		}
	}
	return out
}

// String returns a text representation of the wildcard query.
func (q *WildcardQuery) String() string {
	return fmt.Sprintf("%s:%s", q.Field, q.Pattern)
}
