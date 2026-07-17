package lucene

import (
	"strings"
	"unicode"
)

// Highlighter produces snippets of field text with query terms wrapped in
// marker strings. It analyzes text the same way the index does so that stemmed
// and lowercased query terms line up with the original words.
type Highlighter struct {
	analyzer *Analyzer
	pre      string
	post     string
}

// NewHighlighter builds a Highlighter that wraps matched terms between pre and
// post markers (for example "<b>" and "</b>"). If analyzer is nil,
// NewStandardAnalyzer is used.
func NewHighlighter(analyzer *Analyzer, pre, post string) *Highlighter {
	if analyzer == nil {
		analyzer = NewStandardAnalyzer()
	}
	return &Highlighter{analyzer: analyzer, pre: pre, post: post}
}

// queryTerms extracts the set of analyzed terms referenced by a query so the
// highlighter knows what to mark.
func queryTerms(a *Analyzer, q Query) map[string]struct{} {
	terms := map[string]struct{}{}
	collectTerms(a, q, terms)
	return terms
}

func collectTerms(a *Analyzer, q Query, out map[string]struct{}) {
	switch t := q.(type) {
	case *TermQuery:
		if term := a.AnalyzeTerm(t.Term); term != "" {
			out[term] = struct{}{}
		}
	case *PhraseQuery:
		for _, raw := range t.Terms {
			if term := a.AnalyzeTerm(raw); term != "" {
				out[term] = struct{}{}
			}
		}
	case *PrefixQuery:
		// Prefixes are marked specially by Highlight via prefix matching.
		out["\x00prefix\x00"+strings.ToLower(t.Prefix)] = struct{}{}
	case *BooleanQuery:
		for _, c := range t.Clauses {
			if c.Occur == MustNot {
				continue
			}
			collectTerms(a, c.Query, out)
		}
	}
}

// Highlight returns text with every word that matches a term in q wrapped in
// the highlighter's markers. Matching is performed on the analyzed form of each
// word, so inflected forms are highlighted when the query term stems to the
// same root. The original text, spacing, and punctuation are preserved.
func (h *Highlighter) Highlight(text string, q Query) string {
	if q == nil {
		return text
	}
	terms := queryTerms(h.analyzer, q)
	if len(terms) == 0 {
		return text
	}

	var prefixes []string
	for t := range terms {
		if strings.HasPrefix(t, "\x00prefix\x00") {
			prefixes = append(prefixes, strings.TrimPrefix(t, "\x00prefix\x00"))
		}
	}

	var out strings.Builder
	runes := []rune(text)
	i := 0
	for i < len(runes) {
		if isWordRune(runes[i]) {
			j := i
			for j < len(runes) && isWordRune(runes[j]) {
				j++
			}
			word := string(runes[i:j])
			if h.matches(word, terms, prefixes) {
				out.WriteString(h.pre)
				out.WriteString(word)
				out.WriteString(h.post)
			} else {
				out.WriteString(word)
			}
			i = j
		} else {
			out.WriteRune(runes[i])
			i++
		}
	}
	return out.String()
}

func (h *Highlighter) matches(word string, terms map[string]struct{}, prefixes []string) bool {
	lower := strings.ToLower(word)
	analyzed := h.analyzer.AnalyzeTerm(word)
	if analyzed != "" {
		if _, ok := terms[analyzed]; ok {
			return true
		}
	}
	for _, pfx := range prefixes {
		if strings.HasPrefix(lower, pfx) {
			return true
		}
	}
	return false
}

func isWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}
