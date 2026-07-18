package lucene

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// BoostQuery wraps another query and multiplies every score it produces by a
// constant factor. It mirrors Lucene's BoostQuery: the set of matching
// documents is unchanged, only their relevance scores are scaled. A boost
// greater than one promotes the wrapped clause; a boost between zero and one
// demotes it.
type BoostQuery struct {
	Query Query
	Boost float64
}

// NewBoostQuery builds a BoostQuery that scales the scores of sub by boost.
// A boost of zero is treated as one (no scaling) to avoid silently discarding
// all scores.
func NewBoostQuery(sub Query, boost float64) *BoostQuery {
	if boost == 0 {
		boost = 1
	}
	return &BoostQuery{Query: sub, Boost: boost}
}

func (q *BoostQuery) match(idx *Index) map[int]float64 {
	out := map[int]float64{}
	if q.Query == nil {
		return out
	}
	for num, s := range q.Query.match(idx) {
		out[num] = s * q.Boost
	}
	return out
}

// String returns a text representation of the boost query, for example
// "(text:go)^2".
func (q *BoostQuery) String() string {
	if q.Query == nil {
		return fmt.Sprintf("()^%g", q.Boost)
	}
	return fmt.Sprintf("(%s)^%g", q.Query.String(), q.Boost)
}

// ConstantScoreQuery wraps another query and assigns every matching document
// the same fixed score, discarding the wrapped query's own scoring. It mirrors
// Lucene's ConstantScoreQuery and is useful for pure filtering, where relevance
// ordering among matches is irrelevant.
type ConstantScoreQuery struct {
	Query Query
	Score float64
}

// NewConstantScoreQuery builds a ConstantScoreQuery that gives each document
// matched by sub the given score. A score of zero is treated as one so that
// matches remain distinguishable from non-matches.
func NewConstantScoreQuery(sub Query, score float64) *ConstantScoreQuery {
	if score == 0 {
		score = 1
	}
	return &ConstantScoreQuery{Query: sub, Score: score}
}

func (q *ConstantScoreQuery) match(idx *Index) map[int]float64 {
	out := map[int]float64{}
	if q.Query == nil {
		return out
	}
	for num := range q.Query.match(idx) {
		out[num] = q.Score
	}
	return out
}

// String returns a text representation of the constant-score query.
func (q *ConstantScoreQuery) String() string {
	if q.Query == nil {
		return fmt.Sprintf("ConstantScore()^=%g", q.Score)
	}
	return fmt.Sprintf("ConstantScore(%s)^=%g", q.Query.String(), q.Score)
}

// DisjunctionMaxQuery combines several sub-queries so that a document's score
// is the maximum score across the clauses that match it, plus a configurable
// fraction (the tie breaker) of the scores of the other matching clauses. It
// mirrors Lucene's DisjunctionMaxQuery and is the standard way to search the
// same terms across multiple fields without letting a document that matches in
// two fields dominate one that matches strongly in a single field. A document
// matches if it matches at least one clause.
type DisjunctionMaxQuery struct {
	Disjuncts  []Query
	TieBreaker float64
}

// NewDisjunctionMaxQuery builds a DisjunctionMaxQuery from the given clauses.
// tieBreaker must be in [0,1]: 0 uses the pure maximum clause score, 1 sums all
// matching clause scores. Values outside the range are clamped.
func NewDisjunctionMaxQuery(tieBreaker float64, disjuncts ...Query) *DisjunctionMaxQuery {
	if tieBreaker < 0 {
		tieBreaker = 0
	}
	if tieBreaker > 1 {
		tieBreaker = 1
	}
	return &DisjunctionMaxQuery{Disjuncts: disjuncts, TieBreaker: tieBreaker}
}

func (q *DisjunctionMaxQuery) match(idx *Index) map[int]float64 {
	maxScore := map[int]float64{}
	sumScore := map[int]float64{}
	seen := map[int]bool{}
	for _, sub := range q.Disjuncts {
		if sub == nil {
			continue
		}
		for num, s := range sub.match(idx) {
			sumScore[num] += s
			if !seen[num] || s > maxScore[num] {
				maxScore[num] = s
			}
			seen[num] = true
		}
	}
	out := make(map[int]float64, len(maxScore))
	for num := range seen {
		out[num] = maxScore[num] + q.TieBreaker*(sumScore[num]-maxScore[num])
	}
	return out
}

// String returns a text representation of the disjunction-max query.
func (q *DisjunctionMaxQuery) String() string {
	parts := make([]string, 0, len(q.Disjuncts))
	for _, sub := range q.Disjuncts {
		if sub != nil {
			parts = append(parts, sub.String())
		}
	}
	return fmt.Sprintf("(%s)~%g", strings.Join(parts, " | "), q.TieBreaker)
}

// TermsQuery matches documents that contain any one of several terms in a
// field, scoring each match as the sum of the BM25 contributions of the terms
// it contains. It mirrors Lucene's TermInSetQuery and is a compact alternative
// to a BooleanQuery of many Should term clauses.
type TermsQuery struct {
	Field string
	Terms []string
	Boost float64
}

// NewTermsQuery builds a TermsQuery over the given terms. An empty field
// defaults to "text". Terms are analyzed with the index's analyzer at match
// time, exactly as a TermQuery would be.
func NewTermsQuery(field string, terms ...string) *TermsQuery {
	if field == "" {
		field = defaultField
	}
	return &TermsQuery{Field: field, Terms: terms, Boost: 1}
}

func (q *TermsQuery) boost() float64 {
	if q.Boost == 0 {
		return 1
	}
	return q.Boost
}

func (q *TermsQuery) match(idx *Index) map[int]float64 {
	out := map[int]float64{}
	seen := map[string]bool{}
	for _, raw := range q.Terms {
		t := idx.analyzer.AnalyzeTerm(raw)
		if t == "" || seen[t] {
			continue
		}
		seen[t] = true
		for num, s := range scoreTerm(idx, q.Field, t, q.boost()) {
			out[num] += s
		}
	}
	return out
}

// String returns a text representation of the terms query.
func (q *TermsQuery) String() string {
	return fmt.Sprintf("%s:(%s)", q.Field, strings.Join(q.Terms, " "))
}

// RegexpQuery matches every document containing any term in a field that
// matches an anchored regular expression, mirroring Lucene's RegexpQuery. The
// pattern uses Go's regexp/syntax (RE2) and must match a whole term, not a
// substring, so "colou?r" matches "color" and "colour" but not "colored".
// Scores sum the BM25 contributions of all matching terms.
type RegexpQuery struct {
	Field string
	re    *regexp.Regexp
	src   string
	Boost float64
}

// NewRegexpQuery compiles pattern and builds a RegexpQuery over field. The
// pattern is automatically anchored to match a complete term. An invalid
// pattern returns an *Error. An empty field defaults to "text". Terms in the
// index are lowercased, so patterns should be written in lower case.
func NewRegexpQuery(field, pattern string) (*RegexpQuery, error) {
	if field == "" {
		field = defaultField
	}
	re, err := regexp.Compile("^(?:" + pattern + ")$")
	if err != nil {
		return nil, &Error{Op: "regexp", Msg: err.Error()}
	}
	return &RegexpQuery{Field: field, re: re, src: pattern, Boost: 1}, nil
}

func (q *RegexpQuery) boost() float64 {
	if q.Boost == 0 {
		return 1
	}
	return q.Boost
}

func (q *RegexpQuery) match(idx *Index) map[int]float64 {
	out := map[int]float64{}
	if q.re == nil {
		return out
	}
	fi, ok := idx.fields[q.Field]
	if !ok {
		return out
	}
	// Iterate terms in sorted order for deterministic score accumulation.
	terms := make([]string, 0, len(fi.terms))
	for t := range fi.terms {
		terms = append(terms, t)
	}
	sort.Strings(terms)
	for _, t := range terms {
		if !q.re.MatchString(t) {
			continue
		}
		for num, s := range scoreTermIndex(idx, fi, fi.terms[t], q.boost()) {
			out[num] += s
		}
	}
	return out
}

// String returns a text representation of the regexp query in the conventional
// "field:/pattern/" form.
func (q *RegexpQuery) String() string {
	return fmt.Sprintf("%s:/%s/", q.Field, q.src)
}
