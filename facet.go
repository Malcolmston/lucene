package lucene

import "sort"

// FacetResult pairs a distinct field term with the number of live documents in
// which it appears. It is the unit returned by FacetCounts.
type FacetResult struct {
	// Value is the indexed (analyzed) term.
	Value string
	// Count is the number of live documents containing the term in the field.
	Count int
}

// FacetCounts returns, for the given field, every distinct term together with
// the number of live documents that contain it, sorted by descending count and
// then ascending term for deterministic output. It is a lightweight equivalent
// of Lucene's terms faceting: a quick way to see the value distribution of a
// field. An unknown field yields an empty (non-nil) slice.
func (idx *Index) FacetCounts(field string) []FacetResult {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	out := []FacetResult{}
	fi, ok := idx.fields[field]
	if !ok {
		return out
	}
	for term, ti := range fi.terms {
		out = append(out, FacetResult{Value: term, Count: len(ti.postings)})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Value < out[j].Value
	})
	return out
}
