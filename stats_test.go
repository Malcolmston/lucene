package lucene

import (
	"reflect"
	"sort"
	"testing"
)

func TestDocFreq(t *testing.T) {
	idx := sampleIndex(t)
	// "programming" (stemmed) appears in every body.
	if got := idx.DocFreq("body", "programming"); got != 3 {
		t.Errorf("DocFreq body programming = %d, want 3", got)
	}
	// "go" appears in the bodies of docs 1 and 3.
	if got := idx.DocFreq("body", "go"); got != 2 {
		t.Errorf("DocFreq body go = %d, want 2", got)
	}
	if got := idx.DocFreq("title", "programming"); got != 2 {
		t.Errorf("DocFreq title programming = %d, want 2", got)
	}
	if got := idx.DocFreq("body", "nonexistent"); got != 0 {
		t.Errorf("DocFreq missing term = %d, want 0", got)
	}
	if got := idx.DocFreq("nofield", "go"); got != 0 {
		t.Errorf("DocFreq missing field = %d, want 0", got)
	}
}

func TestTotalTermFreq(t *testing.T) {
	idx := sampleIndex(t)
	// "go" occurs once in doc1 body and once in doc3 body.
	if got := idx.TotalTermFreq("body", "go"); got != 2 {
		t.Errorf("TotalTermFreq body go = %d, want 2", got)
	}
	if got := idx.TotalTermFreq("body", "nonexistent"); got != 0 {
		t.Errorf("TotalTermFreq missing = %d, want 0", got)
	}
}

func TestTermCountAndTerms(t *testing.T) {
	idx := sampleIndex(t)
	terms := idx.Terms("title")
	if !sort.StringsAreSorted(terms) {
		t.Errorf("Terms not sorted: %v", terms)
	}
	if got := idx.TermCount("title"); got != len(terms) {
		t.Errorf("TermCount = %d, len(Terms) = %d", got, len(terms))
	}
	for _, want := range []string{"go", "program", "rust", "action", "network"} {
		if !contains(terms, want) {
			t.Errorf("title terms missing %q: %v", want, terms)
		}
	}
	// Distinct title terms: go, program, language-stem, rust, action, network.
	if got := idx.TermCount("title"); got != 6 {
		t.Errorf("TermCount title = %d, want 6 (%v)", got, terms)
	}
	if got := idx.Terms("missing"); !reflect.DeepEqual(got, []string{}) {
		t.Errorf("Terms missing field = %v, want []", got)
	}
}
