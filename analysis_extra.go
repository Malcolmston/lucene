package lucene

import "strings"

// NGrams splits s into overlapping substrings of exactly n Unicode code points
// (character n-grams), in left-to-right order. It mirrors Lucene's NGramTokenizer
// and is useful for substring and fuzzy matching. If n is less than one, or if s
// has fewer than n runes, NGrams returns an empty (non-nil) slice. The input is
// used verbatim: callers that want case-insensitive n-grams should lowercase s
// first.
func NGrams(s string, n int) []string {
	out := []string{}
	if n < 1 {
		return out
	}
	runes := []rune(s)
	if len(runes) < n {
		return out
	}
	for i := 0; i+n <= len(runes); i++ {
		out = append(out, string(runes[i:i+n]))
	}
	return out
}

// EdgeNGrams returns the prefixes of s whose length in Unicode code points is
// between minGram and maxGram inclusive, shortest first. It mirrors Lucene's
// EdgeNGramTokenFilter, the standard building block for autocomplete: indexing
// the edge n-grams of a term lets a prefix be matched as an ordinary term. If
// minGram is less than one it is raised to one; if maxGram is less than minGram,
// or s is empty, an empty (non-nil) slice is returned. maxGram is capped at the
// rune length of s.
func EdgeNGrams(s string, minGram, maxGram int) []string {
	out := []string{}
	if minGram < 1 {
		minGram = 1
	}
	runes := []rune(s)
	if len(runes) == 0 || maxGram < minGram {
		return out
	}
	hi := maxGram
	if hi > len(runes) {
		hi = len(runes)
	}
	for size := minGram; size <= hi; size++ {
		out = append(out, string(runes[:size]))
	}
	return out
}

// Shingles combines adjacent tokens into word n-grams of size n, joining the
// members of each shingle with a single space. It mirrors Lucene's
// ShingleFilter and captures short phrases as single index terms, improving
// phrase-like matching. If n is less than one it defaults to two; if fewer than
// n tokens are supplied, an empty (non-nil) slice is returned.
func Shingles(tokens []string, n int) []string {
	out := []string{}
	if n < 1 {
		n = 2
	}
	if len(tokens) < n {
		return out
	}
	for i := 0; i+n <= len(tokens); i++ {
		out = append(out, strings.Join(tokens[i:i+n], " "))
	}
	return out
}
