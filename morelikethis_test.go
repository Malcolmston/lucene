package lucene

import "testing"

func TestMoreLikeThis(t *testing.T) {
	idx := sampleIndex(t)
	res := idx.MoreLikeThis("1", "body", 10)
	ids := hitIDs(res)
	// The source document is excluded; the other two docs remain.
	if res.Total != 2 {
		t.Fatalf("total = %d, want 2 (%v)", res.Total, ids)
	}
	if contains(ids, "1") {
		t.Errorf("source document not excluded: %v", ids)
	}
	// Doc 3 shares "go", "program", and "language" with doc 1, whereas doc 2
	// shares only "program" and "language", so doc 3 ranks first.
	if res.Hits[0].ID != "3" {
		t.Errorf("top match = %s, want 3", res.Hits[0].ID)
	}
	// Scores must be sorted descending.
	for i := 1; i < len(res.Hits); i++ {
		if res.Hits[i-1].Score < res.Hits[i].Score {
			t.Error("hits not sorted by score")
		}
	}
}

func TestMoreLikeThisEdgeCases(t *testing.T) {
	idx := sampleIndex(t)
	if idx.MoreLikeThis("missing", "body", 10).Total != 0 {
		t.Error("unknown id produced results")
	}
	if idx.MoreLikeThis("1", "nofield", 10).Total != 0 {
		t.Error("unknown field produced results")
	}
	// topN limits the returned hits.
	res := idx.MoreLikeThis("1", "body", 1)
	if len(res.Hits) != 1 || res.Total != 2 {
		t.Errorf("topN=1: hits=%d total=%d, want 1 and 2", len(res.Hits), res.Total)
	}
}
