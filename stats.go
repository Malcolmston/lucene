package lucene

import "sort"

// DocFreq returns the number of live documents that contain term in the given
// field, the document frequency used by IDF weighting. The term is analyzed
// with the index's analyzer so it lines up with the indexed form. It returns 0
// for an unknown field or term. It mirrors the docFreq statistic exposed by
// Lucene's IndexReader.
func (idx *Index) DocFreq(field, term string) int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	t := idx.analyzer.AnalyzeTerm(term)
	if t == "" {
		return 0
	}
	fi, ok := idx.fields[field]
	if !ok {
		return 0
	}
	ti, ok := fi.terms[t]
	if !ok {
		return 0
	}
	return len(ti.postings)
}

// TotalTermFreq returns the total number of occurrences of term across every
// live document in the given field, summing per-document term frequencies. The
// term is analyzed with the index's analyzer. It returns 0 for an unknown field
// or term. It mirrors the totalTermFreq statistic in Lucene's IndexReader.
func (idx *Index) TotalTermFreq(field, term string) int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	t := idx.analyzer.AnalyzeTerm(term)
	if t == "" {
		return 0
	}
	fi, ok := idx.fields[field]
	if !ok {
		return 0
	}
	ti, ok := fi.terms[t]
	if !ok {
		return 0
	}
	total := 0
	for _, p := range ti.postings {
		total += p.freq
	}
	return total
}

// TermCount returns the number of distinct terms currently indexed in the given
// field. It returns 0 for an unknown field. It corresponds to the size of a
// field's term dictionary in Lucene.
func (idx *Index) TermCount(field string) int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	fi, ok := idx.fields[field]
	if !ok {
		return 0
	}
	return len(fi.terms)
}

// Terms returns the distinct terms indexed in the given field, sorted in
// ascending lexical order. The returned slice is freshly allocated and may be
// modified by the caller. An unknown field yields an empty (non-nil) slice. It
// exposes the field's term dictionary, akin to iterating Lucene's TermsEnum.
func (idx *Index) Terms(field string) []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	out := []string{}
	fi, ok := idx.fields[field]
	if !ok {
		return out
	}
	for t := range fi.terms {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}
