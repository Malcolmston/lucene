package lucene

import (
	"sort"
)

// BM25 tuning parameters. K1 controls term-frequency saturation and B controls
// the strength of document-length normalization. These are the widely used
// defaults.
const (
	bm25K1 = 1.2
	bm25B  = 0.75
)

// bm25 computes the BM25 score contribution of a single term occurrence set:
// termIDF is the term's inverse document frequency, tf its frequency in the
// field, docLen the field length for the document, and avgLen the mean field
// length across the collection.
func bm25(termIDF float64, tf, docLen int, avgLen float64) float64 {
	if tf == 0 {
		return 0
	}
	f := float64(tf)
	var norm float64
	if avgLen > 0 {
		norm = float64(docLen) / avgLen
	}
	denom := f + bm25K1*(1-bm25B+bm25B*norm)
	if denom == 0 {
		return 0
	}
	return termIDF * (f * (bm25K1 + 1)) / denom
}

// Hit is a single search result: the external document ID, its relevance score,
// and the internal document number.
type Hit struct {
	ID     string
	Score  float64
	docNum int
}

// Result is the outcome of a search: the total number of documents that matched
// and the top-ranked hits.
type Result struct {
	// Total is the number of documents that matched the query, which may exceed
	// len(Hits) when the caller requested fewer.
	Total int
	Hits  []Hit
}

// Search evaluates q against the index and returns up to topN hits ranked by
// descending BM25 score. Ties are broken by ascending document ID so results
// are fully deterministic. A topN of zero or less returns all matching hits.
func (idx *Index) Search(q Query, topN int) Result {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	if q == nil {
		return Result{}
	}
	scored := q.match(idx)

	hits := make([]Hit, 0, len(scored))
	for num, score := range scored {
		id, ok := idx.docID[num]
		if !ok || !idx.live[num] {
			continue
		}
		hits = append(hits, Hit{ID: id, Score: score, docNum: num})
	}

	sort.Slice(hits, func(i, j int) bool {
		if hits[i].Score != hits[j].Score {
			return hits[i].Score > hits[j].Score
		}
		return hits[i].ID < hits[j].ID
	})

	total := len(hits)
	if topN > 0 && len(hits) > topN {
		hits = hits[:topN]
	}
	return Result{Total: total, Hits: hits}
}

// SearchString parses queryString with the index's analyzer and default field
// and then executes it. It is a convenience wrapper over Parser and Search.
func (idx *Index) SearchString(queryString string, topN int) (Result, error) {
	p := NewParser(idx.analyzer, defaultField)
	q, err := p.Parse(queryString)
	if err != nil {
		return Result{}, err
	}
	return idx.Search(q, topN), nil
}
