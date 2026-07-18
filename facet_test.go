package lucene

import "testing"

func TestFacetCounts(t *testing.T) {
	idx := sampleIndex(t)
	facets := idx.FacetCounts("title")
	// Six distinct title terms.
	if len(facets) != 6 {
		t.Fatalf("got %d facets, want 6: %+v", len(facets), facets)
	}
	// "go" and "program" each appear in two documents and lead the list, sorted
	// by descending count then ascending value.
	if facets[0] != (FacetResult{Value: "go", Count: 2}) {
		t.Errorf("facets[0] = %+v, want {go 2}", facets[0])
	}
	if facets[1] != (FacetResult{Value: "program", Count: 2}) {
		t.Errorf("facets[1] = %+v, want {program 2}", facets[1])
	}
	// Remaining terms each appear once.
	for _, f := range facets[2:] {
		if f.Count != 1 {
			t.Errorf("facet %+v, want count 1", f)
		}
	}
	if got := idx.FacetCounts("missing"); len(got) != 0 {
		t.Errorf("FacetCounts missing field = %+v, want empty", got)
	}
}
