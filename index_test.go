package lucene

import (
	"fmt"
	"sync"
	"testing"
)

func sampleIndex(t *testing.T) *Index {
	t.Helper()
	idx := NewIndex(NewStandardAnalyzer())
	docs := []Document{
		{ID: "1", Fields: map[string]string{
			"title": "The Go Programming Language",
			"body":  "Go is an open source programming language designed at Google.",
		}},
		{ID: "2", Fields: map[string]string{
			"title": "Rust in Action",
			"body":  "Rust is a systems programming language focused on safety.",
		}},
		{ID: "3", Fields: map[string]string{
			"title": "Network Programming with Go",
			"body":  "Building network servers and clients using the Go programming language.",
		}},
	}
	for _, d := range docs {
		if err := idx.Add(d); err != nil {
			t.Fatalf("add %s: %v", d.ID, err)
		}
	}
	return idx
}

func TestAddValidation(t *testing.T) {
	idx := NewIndex(nil)
	if err := idx.Add(Document{ID: "", Fields: map[string]string{"x": "y"}}); err == nil {
		t.Error("expected error for empty ID")
	}
	if err := idx.Add(Document{ID: "1"}); err == nil {
		t.Error("expected error for nil fields")
	}
}

func TestNumDocsAndHas(t *testing.T) {
	idx := sampleIndex(t)
	if idx.NumDocs() != 3 {
		t.Errorf("NumDocs = %d, want 3", idx.NumDocs())
	}
	if !idx.Has("2") {
		t.Error("Has(2) = false")
	}
	if idx.Has("nope") {
		t.Error("Has(nope) = true")
	}
}

func TestFields(t *testing.T) {
	idx := sampleIndex(t)
	got := idx.Fields()
	want := []string{"body", "title"}
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("Fields = %v, want %v", got, want)
	}
	if idx.Analyzer() == nil {
		t.Error("Analyzer nil")
	}
}

func TestUpdateReplacesDocument(t *testing.T) {
	idx := sampleIndex(t)
	// Replace doc 2 with content that no longer mentions "rust".
	if err := idx.Add(Document{ID: "2", Fields: map[string]string{
		"title": "Python Basics",
		"body":  "Python is a scripting language.",
	}}); err != nil {
		t.Fatal(err)
	}
	if idx.NumDocs() != 3 {
		t.Errorf("NumDocs after update = %d, want 3", idx.NumDocs())
	}
	res := idx.Search(NewTermQuery("body", "rust"), 10)
	if res.Total != 0 {
		t.Errorf("rust still found after update: %d", res.Total)
	}
	res = idx.Search(NewTermQuery("body", "python"), 10)
	if res.Total != 1 || res.Hits[0].ID != "2" {
		t.Errorf("python not found in updated doc: %+v", res)
	}
}

func TestDelete(t *testing.T) {
	idx := sampleIndex(t)
	if !idx.Delete("1") {
		t.Error("Delete(1) = false")
	}
	if idx.Delete("1") {
		t.Error("second Delete(1) = true")
	}
	if idx.NumDocs() != 2 {
		t.Errorf("NumDocs = %d, want 2", idx.NumDocs())
	}
	res := idx.Search(NewTermQuery("body", "google"), 10)
	if res.Total != 0 {
		t.Errorf("deleted doc still matched: %+v", res)
	}
}

func TestConcurrentAccess(t *testing.T) {
	idx := NewIndex(nil)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_ = idx.Add(Document{ID: fmt.Sprintf("d%d", n), Fields: map[string]string{
				"body": fmt.Sprintf("document number %d about programming", n),
			}})
		}(i)
	}
	wg.Wait()
	if idx.NumDocs() != 50 {
		t.Fatalf("NumDocs = %d, want 50", idx.NumDocs())
	}
	var rg sync.WaitGroup
	for i := 0; i < 20; i++ {
		rg.Add(1)
		go func() {
			defer rg.Done()
			_ = idx.Search(NewTermQuery("body", "programming"), 5)
		}()
	}
	rg.Wait()
}

func TestIdf(t *testing.T) {
	// Rarer terms should have higher idf.
	if idf(100, 1) <= idf(100, 50) {
		t.Error("idf not monotonic in df")
	}
}
