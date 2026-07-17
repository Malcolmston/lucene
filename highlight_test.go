package lucene

import (
	"strings"
	"testing"
)

func TestHighlightTerm(t *testing.T) {
	h := NewHighlighter(NewStandardAnalyzer(), "<b>", "</b>")
	q := NewTermQuery("body", "programming")
	got := h.Highlight("Go is a programming language.", q)
	if !strings.Contains(got, "<b>programming</b>") {
		t.Errorf("highlight = %q", got)
	}
	// Inflected form should still highlight because stems match.
	got = h.Highlight("He programs daily.", NewTermQuery("body", "program"))
	if !strings.Contains(got, "<b>programs</b>") {
		t.Errorf("inflected highlight = %q", got)
	}
}

func TestHighlightPhraseAndPrefix(t *testing.T) {
	h := NewHighlighter(nil, "[", "]")
	got := h.Highlight("network programming rules", NewPhraseQuery("body", "network", "programming"))
	if !strings.Contains(got, "[network]") || !strings.Contains(got, "[programming]") {
		t.Errorf("phrase highlight = %q", got)
	}
	got = h.Highlight("networking is neat", NewPrefixQuery("body", "net"))
	if !strings.Contains(got, "[networking]") {
		t.Errorf("prefix highlight = %q", got)
	}
}

func TestHighlightBooleanAndMustNot(t *testing.T) {
	h := NewHighlighter(nil, "*", "*")
	bq := NewBooleanQuery().
		Add(NewTermQuery("body", "go"), Should).
		Add(NewTermQuery("body", "rust"), MustNot)
	got := h.Highlight("go and rust", bq)
	if !strings.Contains(got, "*go*") {
		t.Errorf("expected go highlighted: %q", got)
	}
	if strings.Contains(got, "*rust*") {
		t.Errorf("MustNot term should not be highlighted: %q", got)
	}
}

func TestHighlightNilAndNoTerms(t *testing.T) {
	h := NewHighlighter(nil, "<", ">")
	if got := h.Highlight("unchanged", nil); got != "unchanged" {
		t.Errorf("nil query highlight = %q", got)
	}
	// A range query contributes no plain terms, so text is unchanged.
	if got := h.Highlight("unchanged text", NewRangeQuery("year", "1", "9", true, true)); got != "unchanged text" {
		t.Errorf("range highlight = %q", got)
	}
	// Punctuation and spacing preserved.
	got := h.Highlight("go, go!", NewTermQuery("body", "go"))
	if got != "<go>, <go>!" {
		t.Errorf("punctuation preserved = %q", got)
	}
}
