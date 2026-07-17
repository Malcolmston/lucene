package lucene

import (
	"testing"
)

func TestTermQuerySearch(t *testing.T) {
	idx := sampleIndex(t)
	res := idx.Search(NewTermQuery("body", "rust"), 10)
	if res.Total != 1 || res.Hits[0].ID != "2" {
		t.Fatalf("term query rust: %+v", res)
	}
	// programming appears in all three docs.
	res = idx.Search(NewTermQuery("body", "programming"), 10)
	if res.Total != 3 {
		t.Errorf("programming total = %d, want 3", res.Total)
	}
	// Scores must be sorted descending.
	for i := 1; i < len(res.Hits); i++ {
		if res.Hits[i-1].Score < res.Hits[i].Score {
			t.Error("hits not sorted by score")
		}
	}
}

func TestTermQueryDefaultFieldAndBoost(t *testing.T) {
	q := NewTermQuery("", "x")
	if q.Field != "text" {
		t.Errorf("default field = %q", q.Field)
	}
	if q.boost() != 1 {
		t.Errorf("boost = %v", q.boost())
	}
	q.Boost = 2
	if q.boost() != 2 {
		t.Errorf("boost = %v", q.boost())
	}
	if q.String() != "text:x" {
		t.Errorf("String = %q", q.String())
	}
}

func TestPhraseQuery(t *testing.T) {
	idx := sampleIndex(t)
	// "programming language" appears in docs 1 and 3 (adjacent), but "language
	// programming" should not.
	res := idx.Search(NewPhraseQuery("body", "programming", "language"), 10)
	ids := hitIDs(res)
	if !contains(ids, "1") || !contains(ids, "3") {
		t.Errorf("phrase query missing docs: %v", ids)
	}
	res = idx.Search(NewPhraseQuery("body", "language", "programming"), 10)
	if res.Total != 0 {
		t.Errorf("reversed phrase matched: %+v", res)
	}
}

func TestPhraseQuerySingleTermAndMissing(t *testing.T) {
	idx := sampleIndex(t)
	res := idx.Search(NewPhraseQuery("body", "rust"), 10)
	if res.Total != 1 {
		t.Errorf("single-term phrase: %+v", res)
	}
	// A phrase containing a non-existent term matches nothing.
	res = idx.Search(NewPhraseQuery("body", "programming", "zebra"), 10)
	if res.Total != 0 {
		t.Errorf("phrase with missing term matched: %+v", res)
	}
	// Empty phrase.
	res = idx.Search(NewPhraseQuery("body"), 10)
	if res.Total != 0 {
		t.Errorf("empty phrase matched: %+v", res)
	}
}

func TestPrefixQuery(t *testing.T) {
	idx := sampleIndex(t)
	res := idx.Search(NewPrefixQuery("body", "sys"), 10)
	if res.Total != 1 || res.Hits[0].ID != "2" {
		t.Errorf("prefix sys: %+v", res)
	}
	// "prog" prefix matches the stemmed "program" in all docs.
	res = idx.Search(NewPrefixQuery("body", "prog"), 10)
	if res.Total != 3 {
		t.Errorf("prefix prog total = %d, want 3", res.Total)
	}
	// Empty prefix and missing field.
	if idx.Search(NewPrefixQuery("body", ""), 10).Total != 0 {
		t.Error("empty prefix matched")
	}
	if idx.Search(NewPrefixQuery("missing", "x"), 10).Total != 0 {
		t.Error("prefix on missing field matched")
	}
	if got := NewPrefixQuery("body", "net").String(); got != "body:net*" {
		t.Errorf("String = %q", got)
	}
}

func TestRangeQuery(t *testing.T) {
	idx := NewIndex(NewAnalyzer(WithStemming(false)))
	for _, d := range []Document{
		{ID: "a", Fields: map[string]string{"year": "2001"}},
		{ID: "b", Fields: map[string]string{"year": "2010"}},
		{ID: "c", Fields: map[string]string{"year": "2020"}},
	} {
		if err := idx.Add(d); err != nil {
			t.Fatal(err)
		}
	}
	res := idx.Search(NewRangeQuery("year", "2005", "2015", true, true), 10)
	if res.Total != 1 || res.Hits[0].ID != "b" {
		t.Errorf("inclusive range: %+v", res)
	}
	// Exclusive bounds.
	res = idx.Search(NewRangeQuery("year", "2001", "2020", false, false), 10)
	if res.Total != 1 || res.Hits[0].ID != "b" {
		t.Errorf("exclusive range: %+v", res)
	}
	// Unbounded upper.
	res = idx.Search(NewRangeQuery("year", "2010", "", true, true), 10)
	if res.Total != 2 {
		t.Errorf("open-ended range total = %d, want 2", res.Total)
	}
	// Missing field.
	if idx.Search(NewRangeQuery("nope", "a", "z", true, true), 10).Total != 0 {
		t.Error("range on missing field matched")
	}
}

func TestRangeQueryString(t *testing.T) {
	q := NewRangeQuery("year", "2001", "2020", true, false)
	if got := q.String(); got != "year:[2001 TO 2020}" {
		t.Errorf("String = %q", got)
	}
	q2 := NewRangeQuery("year", "", "", false, true)
	if got := q2.String(); got != "year:{* TO *]" {
		t.Errorf("String = %q", got)
	}
}

func TestBooleanQuery(t *testing.T) {
	idx := sampleIndex(t)
	// +programming +rust: only doc 2.
	bq := NewBooleanQuery().
		Add(NewTermQuery("body", "programming"), Must).
		Add(NewTermQuery("body", "rust"), Must)
	res := idx.Search(bq, 10)
	if res.Total != 1 || res.Hits[0].ID != "2" {
		t.Errorf("AND query: %+v", res)
	}

	// programming -rust: docs 1 and 3.
	bq = NewBooleanQuery().
		Add(NewTermQuery("body", "programming"), Should).
		Add(NewTermQuery("body", "rust"), MustNot)
	res = idx.Search(bq, 10)
	ids := hitIDs(res)
	if len(ids) != 2 || contains(ids, "2") {
		t.Errorf("NOT query: %v", ids)
	}

	// Should-only OR.
	bq = NewBooleanQuery().
		Add(NewTermQuery("body", "rust"), Should).
		Add(NewTermQuery("body", "google"), Should)
	res = idx.Search(bq, 10)
	if res.Total != 2 {
		t.Errorf("OR query total = %d, want 2", res.Total)
	}

	if got := bq.String(); got == "" {
		t.Error("boolean String empty")
	}
}

func TestMatchAllQuery(t *testing.T) {
	idx := sampleIndex(t)
	res := idx.Search(&MatchAllQuery{}, 10)
	if res.Total != 3 {
		t.Errorf("match all total = %d, want 3", res.Total)
	}
	if (&MatchAllQuery{}).String() != "*:*" {
		t.Error("match all String wrong")
	}
	// MatchAll combined with a MustNot filter.
	bq := NewBooleanQuery().
		Add(&MatchAllQuery{}, Must).
		Add(NewTermQuery("body", "rust"), MustNot)
	res = idx.Search(bq, 10)
	if res.Total != 2 {
		t.Errorf("filtered match all total = %d, want 2", res.Total)
	}
}

func TestSearchTopNAndTieBreak(t *testing.T) {
	idx := NewIndex(NewAnalyzer(WithStemming(false)))
	// Identical docs -> identical scores -> deterministic order by ID.
	for _, id := range []string{"z", "a", "m", "b"} {
		if err := idx.Add(Document{ID: id, Fields: map[string]string{"body": "same content here"}}); err != nil {
			t.Fatal(err)
		}
	}
	res := idx.Search(NewTermQuery("body", "content"), 2)
	if res.Total != 4 {
		t.Errorf("total = %d, want 4", res.Total)
	}
	if len(res.Hits) != 2 {
		t.Fatalf("returned %d hits, want 2", len(res.Hits))
	}
	if res.Hits[0].ID != "a" || res.Hits[1].ID != "b" {
		t.Errorf("tie-break order = %s,%s want a,b", res.Hits[0].ID, res.Hits[1].ID)
	}
}

func TestSearchNilAndEmpty(t *testing.T) {
	idx := sampleIndex(t)
	if idx.Search(nil, 10).Total != 0 {
		t.Error("nil query matched")
	}
	// Query for a term that doesn't exist.
	if idx.Search(NewTermQuery("body", "zebra"), 10).Total != 0 {
		t.Error("nonexistent term matched")
	}
	// topN <= 0 returns all.
	res := idx.Search(NewTermQuery("body", "programming"), 0)
	if len(res.Hits) != 3 {
		t.Errorf("topN=0 returned %d hits", len(res.Hits))
	}
}

func hitIDs(r Result) []string {
	out := make([]string, len(r.Hits))
	for i, h := range r.Hits {
		out[i] = h.ID
	}
	return out
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
