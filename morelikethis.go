package lucene

import "sort"

// mltMaxQueryTerms bounds how many of a source document's highest-weighted
// terms drive a MoreLikeThis query, keeping the comparison focused and fast.
const mltMaxQueryTerms = 25

// MoreLikeThis finds the documents most similar to the document identified by
// id, comparing them on the terms of the given field. It mirrors Lucene's
// MoreLikeThis feature: the source document's most distinctive terms (ranked by
// term frequency times inverse document frequency) become an implicit query,
// and the top matching documents are returned, ranked by descending BM25 score
// with ties broken by ascending document ID. The source document itself is
// always excluded from the results. A topN of zero or less returns all matches.
// If id is unknown, the field is missing, or the document has no terms in the
// field, an empty Result is returned.
func (idx *Index) MoreLikeThis(id, field string, topN int) Result {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	num, ok := idx.docNum[id]
	if !ok || !idx.live[num] {
		return Result{}
	}
	fi, ok := idx.fields[field]
	if !ok {
		return Result{}
	}

	// Collect the source document's terms weighted by tf * idf.
	type weighted struct {
		term string
		w    float64
	}
	var terms []weighted
	for term, ti := range fi.terms {
		p, ok := ti.postings[num]
		if !ok {
			continue
		}
		w := float64(p.freq) * idf(idx.numDocs, len(ti.postings))
		terms = append(terms, weighted{term: term, w: w})
	}
	if len(terms) == 0 {
		return Result{}
	}
	sort.Slice(terms, func(i, j int) bool {
		if terms[i].w != terms[j].w {
			return terms[i].w > terms[j].w
		}
		return terms[i].term < terms[j].term
	})
	if len(terms) > mltMaxQueryTerms {
		terms = terms[:mltMaxQueryTerms]
	}

	// Score every other document over the selected terms.
	scores := map[int]float64{}
	for _, wt := range terms {
		for n, s := range scoreTermIndex(idx, fi, fi.terms[wt.term], 1) {
			if n == num {
				continue
			}
			scores[n] += s
		}
	}

	hits := make([]Hit, 0, len(scores))
	for n, s := range scores {
		docID, ok := idx.docID[n]
		if !ok || !idx.live[n] {
			continue
		}
		hits = append(hits, Hit{ID: docID, Score: s, docNum: n})
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
