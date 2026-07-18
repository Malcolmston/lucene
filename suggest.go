package lucene

import (
	"sort"
	"strings"
	"unicode/utf8"
)

// Suggest returns up to max indexed terms in the given field that begin with
// prefix, ordered by descending document frequency (most common first) and then
// ascending term for ties. It is a simple prefix autocompleter in the spirit of
// Lucene's suggest module. The prefix is lowercased to match the indexed form.
// A max of zero or less returns all matching terms. An unknown field or empty
// prefix yields an empty (non-nil) slice.
func (idx *Index) Suggest(field, prefix string, max int) []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	out := []string{}
	prefix = strings.ToLower(strings.TrimSpace(prefix))
	if prefix == "" {
		return out
	}
	fi, ok := idx.fields[field]
	if !ok {
		return out
	}
	type cand struct {
		term string
		df   int
	}
	var cands []cand
	for term, ti := range fi.terms {
		if strings.HasPrefix(term, prefix) {
			cands = append(cands, cand{term: term, df: len(ti.postings)})
		}
	}
	sort.Slice(cands, func(i, j int) bool {
		if cands[i].df != cands[j].df {
			return cands[i].df > cands[j].df
		}
		return cands[i].term < cands[j].term
	})
	for _, c := range cands {
		if max > 0 && len(out) >= max {
			break
		}
		out = append(out, c.term)
	}
	return out
}

// SpellCheck suggests corrections for term drawn from the terms indexed in the
// given field: every indexed term within maxEdits Levenshtein edits of term,
// excluding term itself, ordered by ascending edit distance, then descending
// document frequency, then ascending term. It mirrors the didactic behaviour of
// Lucene's SpellChecker. The term is lowercased (not stemmed) before comparison.
// A max of zero or less returns all candidates; a negative maxEdits is treated
// as one. An unknown field or empty term yields an empty (non-nil) slice.
func (idx *Index) SpellCheck(field, term string, maxEdits, max int) []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	out := []string{}
	term = strings.ToLower(strings.TrimSpace(term))
	if term == "" {
		return out
	}
	if maxEdits < 0 {
		maxEdits = 1
	}
	fi, ok := idx.fields[field]
	if !ok {
		return out
	}
	tlen := utf8.RuneCountInString(term)
	type cand struct {
		term string
		dist int
		df   int
	}
	var cands []cand
	for t, ti := range fi.terms {
		if t == term {
			continue
		}
		if luAbs(utf8.RuneCountInString(t)-tlen) > maxEdits {
			continue
		}
		d := LevenshteinDistance(term, t)
		if d > maxEdits {
			continue
		}
		cands = append(cands, cand{term: t, dist: d, df: len(ti.postings)})
	}
	sort.Slice(cands, func(i, j int) bool {
		if cands[i].dist != cands[j].dist {
			return cands[i].dist < cands[j].dist
		}
		if cands[i].df != cands[j].df {
			return cands[i].df > cands[j].df
		}
		return cands[i].term < cands[j].term
	})
	for _, c := range cands {
		if max > 0 && len(out) >= max {
			break
		}
		out = append(out, c.term)
	}
	return out
}
