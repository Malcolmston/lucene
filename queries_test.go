package lucene

import (
	"math"
	"testing"
)

// scoreByID returns the score of the hit with the given ID, or 0 if absent.
func scoreByID(r Result, id string) float64 {
	for _, h := range r.Hits {
		if h.ID == id {
			return h.Score
		}
	}
	return 0
}

func TestBoostQuery(t *testing.T) {
	idx := sampleIndex(t)
	base := idx.Search(NewTermQuery("body", "go"), 0)
	boosted := idx.Search(NewBoostQuery(NewTermQuery("body", "go"), 2), 0)
	if base.Total != boosted.Total {
		t.Fatalf("boost changed match set: %d vs %d", base.Total, boosted.Total)
	}
	for _, h := range base.Hits {
		want := 2 * h.Score
		if got := scoreByID(boosted, h.ID); math.Abs(got-want) > 1e-9 {
			t.Errorf("doc %s boosted score = %f, want %f", h.ID, got, want)
		}
	}
	// A zero boost is normalized to one.
	if NewBoostQuery(nil, 0).Boost != 1 {
		t.Error("zero boost not normalized")
	}
	if got := NewBoostQuery(NewTermQuery("text", "go"), 2).String(); got != "(text:go)^2" {
		t.Errorf("String = %q", got)
	}
}

func TestConstantScoreQuery(t *testing.T) {
	idx := sampleIndex(t)
	// "prog" prefix matches the stemmed "program" in all three docs.
	q := NewConstantScoreQuery(NewPrefixQuery("body", "prog"), 1)
	res := idx.Search(q, 0)
	if res.Total != 3 {
		t.Fatalf("total = %d, want 3", res.Total)
	}
	for _, h := range res.Hits {
		if h.Score != 1 {
			t.Errorf("doc %s score = %f, want 1", h.ID, h.Score)
		}
	}
}

func TestDisjunctionMaxQuery(t *testing.T) {
	idx := sampleIndex(t)
	// Doc coverage: go -> {1,3}, rust -> {2}. Union is all three.
	rust := idx.Search(NewTermQuery("body", "rust"), 0)
	dm := NewDisjunctionMaxQuery(0,
		NewTermQuery("body", "go"),
		NewTermQuery("body", "rust"))
	res := idx.Search(dm, 0)
	if res.Total != 3 {
		t.Fatalf("total = %d, want 3", res.Total)
	}
	// Doc 2 matches only the rust clause, so with tieBreaker 0 its score equals
	// the rust term score.
	if got, want := scoreByID(res, "2"), scoreByID(rust, "2"); math.Abs(got-want) > 1e-9 {
		t.Errorf("dismax doc2 score = %f, want %f", got, want)
	}

	// tieBreaker semantics on a doc matching two clauses (both terms occur in
	// every body). tieBreaker 0 = max, tieBreaker 1 = sum.
	prog := idx.Search(NewTermQuery("body", "programming"), 0)
	lang := idx.Search(NewTermQuery("body", "language"), 0)
	p1, l1 := scoreByID(prog, "1"), scoreByID(lang, "1")
	dm0 := idx.Search(NewDisjunctionMaxQuery(0,
		NewTermQuery("body", "programming"), NewTermQuery("body", "language")), 0)
	dm1 := idx.Search(NewDisjunctionMaxQuery(1,
		NewTermQuery("body", "programming"), NewTermQuery("body", "language")), 0)
	if got, want := scoreByID(dm0, "1"), math.Max(p1, l1); math.Abs(got-want) > 1e-9 {
		t.Errorf("tieBreaker 0 doc1 = %f, want max %f", got, want)
	}
	if got, want := scoreByID(dm1, "1"), p1+l1; math.Abs(got-want) > 1e-9 {
		t.Errorf("tieBreaker 1 doc1 = %f, want sum %f", got, want)
	}
}

func TestTermsQuery(t *testing.T) {
	idx := sampleIndex(t)
	q := NewTermsQuery("body", "go", "rust")
	res := idx.Search(q, 0)
	ids := hitIDs(res)
	if res.Total != 3 {
		t.Fatalf("total = %d, want 3 (%v)", res.Total, ids)
	}
	// Doc 2 contains only rust; its score should equal the rust term score.
	rust := idx.Search(NewTermQuery("body", "rust"), 0)
	if got, want := scoreByID(res, "2"), scoreByID(rust, "2"); math.Abs(got-want) > 1e-9 {
		t.Errorf("terms doc2 score = %f, want %f", got, want)
	}
	// Empty / duplicate terms are ignored gracefully.
	if idx.Search(NewTermsQuery("body"), 0).Total != 0 {
		t.Error("empty terms query matched")
	}
}

func TestRegexpQuery(t *testing.T) {
	idx := sampleIndex(t)
	// Anchored exact match of the term "go" -> docs 1 and 3.
	q, err := NewRegexpQuery("body", "go")
	if err != nil {
		t.Fatal(err)
	}
	res := idx.Search(q, 0)
	ids := hitIDs(res)
	if res.Total != 2 || !contains(ids, "1") || !contains(ids, "3") {
		t.Errorf("regexp go = %v, want {1,3}", ids)
	}
	// "prog.*" matches the stemmed "program" in all docs.
	q, _ = NewRegexpQuery("body", "prog.*")
	if idx.Search(q, 0).Total != 3 {
		t.Errorf("regexp prog.* total = %d, want 3", idx.Search(q, 0).Total)
	}
	// Anchoring: "o" must not match "go".
	q, _ = NewRegexpQuery("body", "o")
	if idx.Search(q, 0).Total != 0 {
		t.Error("unanchored regexp matched substring")
	}
	// Invalid pattern.
	if _, err := NewRegexpQuery("body", "("); err == nil {
		t.Error("expected error for invalid regexp")
	}
	if got := q.String(); got != "body:/o/" {
		t.Errorf("String = %q", got)
	}
}
