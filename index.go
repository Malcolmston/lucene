package lucene

import (
	"math"
	"sort"
	"sync"
)

// Document is a unit of content added to an Index. ID must be unique within the
// index; adding a document whose ID already exists replaces the previous
// version. Fields maps field names to their raw (unanalyzed) text.
type Document struct {
	ID     string
	Fields map[string]string
}

// posting records the occurrences of a term within a single field of a single
// document: the term frequency and the positions at which it occurs.
type posting struct {
	freq      int
	positions []int
}

// termIndex holds the postings for one term across all documents, keyed by
// internal document number, plus the document frequency (number of distinct
// documents containing the term).
type termIndex struct {
	postings map[int]*posting
}

// fieldIndex is the inverted index for a single field.
type fieldIndex struct {
	terms map[string]*termIndex
	// lengths maps internal doc number to the field's token count, used for
	// BM25 length normalization.
	lengths  map[int]int
	totalLen int
}

func newFieldIndex() *fieldIndex {
	return &fieldIndex{
		terms:   map[string]*termIndex{},
		lengths: map[int]int{},
	}
}

// Index is an in-memory inverted index over a set of documents. It supports
// concurrent use by multiple goroutines. The zero value is not usable; call
// NewIndex.
type Index struct {
	mu       sync.RWMutex
	analyzer *Analyzer

	fields map[string]*fieldIndex

	// docNum assigns a stable internal integer to each external document ID.
	docNum map[string]int
	docID  map[int]string
	// live marks whether an internal doc number currently refers to a live
	// document (deletes flip this to false).
	live    map[int]bool
	nextNum int
	numDocs int
}

// NewIndex creates an empty index that uses the supplied analyzer for both
// indexing and query analysis. If analyzer is nil, NewStandardAnalyzer is used.
func NewIndex(analyzer *Analyzer) *Index {
	if analyzer == nil {
		analyzer = NewStandardAnalyzer()
	}
	return &Index{
		analyzer: analyzer,
		fields:   map[string]*fieldIndex{},
		docNum:   map[string]int{},
		docID:    map[int]string{},
		live:     map[int]bool{},
	}
}

// Analyzer returns the analyzer used by the index.
func (idx *Index) Analyzer() *Analyzer {
	return idx.analyzer
}

// NumDocs returns the number of live documents currently in the index.
func (idx *Index) NumDocs() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.numDocs
}

// Add inserts or replaces a document. If a document with the same ID already
// exists it is deleted first, so Add doubles as update. An empty ID or a nil
// Fields map is rejected with an error.
func (idx *Index) Add(doc Document) error {
	if doc.ID == "" {
		return &Error{Op: "add", Msg: "document ID must not be empty"}
	}
	if doc.Fields == nil {
		return &Error{Op: "add", Msg: "document fields must not be nil"}
	}

	idx.mu.Lock()
	defer idx.mu.Unlock()

	if _, exists := idx.docNum[doc.ID]; exists {
		idx.deleteLocked(doc.ID)
	}

	num := idx.nextNum
	idx.nextNum++
	idx.docNum[doc.ID] = num
	idx.docID[num] = doc.ID
	idx.live[num] = true
	idx.numDocs++

	// Analyze fields in sorted order for deterministic construction.
	names := make([]string, 0, len(doc.Fields))
	for name := range doc.Fields {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		tokens := idx.analyzer.Analyze(doc.Fields[name])
		fi, ok := idx.fields[name]
		if !ok {
			fi = newFieldIndex()
			idx.fields[name] = fi
		}
		fi.lengths[num] = len(tokens)
		fi.totalLen += len(tokens)
		for _, tok := range tokens {
			ti, ok := fi.terms[tok.Text]
			if !ok {
				ti = &termIndex{postings: map[int]*posting{}}
				fi.terms[tok.Text] = ti
			}
			p, ok := ti.postings[num]
			if !ok {
				p = &posting{}
				ti.postings[num] = p
			}
			p.freq++
			p.positions = append(p.positions, tok.Position)
		}
	}
	return nil
}

// Delete removes the document with the given ID. It reports whether a document
// was actually removed.
func (idx *Index) Delete(id string) bool {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	return idx.deleteLocked(id)
}

// deleteLocked removes a document; the caller must hold the write lock.
func (idx *Index) deleteLocked(id string) bool {
	num, ok := idx.docNum[id]
	if !ok || !idx.live[num] {
		return false
	}
	for _, fi := range idx.fields {
		if l, ok := fi.lengths[num]; ok {
			fi.totalLen -= l
			delete(fi.lengths, num)
		}
		for term, ti := range fi.terms {
			if _, ok := ti.postings[num]; ok {
				delete(ti.postings, num)
				if len(ti.postings) == 0 {
					delete(fi.terms, term)
				}
			}
		}
	}
	idx.live[num] = false
	delete(idx.docNum, id)
	delete(idx.docID, num)
	idx.numDocs--
	return true
}

// Has reports whether a live document with the given ID exists.
func (idx *Index) Has(id string) bool {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	num, ok := idx.docNum[id]
	return ok && idx.live[num]
}

// Fields returns the sorted names of all fields that have been indexed.
func (idx *Index) Fields() []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	names := make([]string, 0, len(idx.fields))
	for name := range idx.fields {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// avgFieldLen returns the mean token count for a field over live documents. The
// caller must hold at least the read lock.
func (fi *fieldIndex) avgFieldLen() float64 {
	if len(fi.lengths) == 0 {
		return 0
	}
	return float64(fi.totalLen) / float64(len(fi.lengths))
}

// idf computes the BM25 inverse document frequency for a term with the given
// document frequency, over a collection of n documents.
func idf(n, df int) float64 {
	return math.Log(1 + (float64(n)-float64(df)+0.5)/(float64(df)+0.5))
}
