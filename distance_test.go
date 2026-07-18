package lucene

import (
	"math"
	"testing"
)

func TestDamerauLevenshteinDistance(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"", "abc", 3},
		{"abc", "", 3},
		{"abc", "abc", 0},
		{"ab", "ba", 1},   // single transposition
		{"teh", "the", 1}, // adjacent swap
		{"abcd", "acbd", 1},
		{"kitten", "sitting", 3}, // no helpful transposition
		{"ca", "abc", 3},
	}
	for _, c := range cases {
		if got := DamerauLevenshteinDistance(c.a, c.b); got != c.want {
			t.Errorf("DamerauLevenshteinDistance(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
		// Symmetry.
		if got := DamerauLevenshteinDistance(c.b, c.a); got != c.want {
			t.Errorf("DamerauLevenshteinDistance(%q, %q) = %d, want %d (symmetry)", c.b, c.a, got, c.want)
		}
	}
}

func approxEqual(a, b float64) bool { return math.Abs(a-b) < 1e-4 }

func TestJaroSimilarity(t *testing.T) {
	cases := []struct {
		a, b string
		want float64
	}{
		{"", "", 1},
		{"abc", "", 0},
		{"abc", "abc", 1},
		{"MARTHA", "MARHTA", 0.944444},
		{"DWAYNE", "DUANE", 0.822222},
		{"DIXON", "DICKSONX", 0.766667},
	}
	for _, c := range cases {
		if got := JaroSimilarity(c.a, c.b); !approxEqual(got, c.want) {
			t.Errorf("JaroSimilarity(%q, %q) = %f, want %f", c.a, c.b, got, c.want)
		}
	}
}

func TestJaroWinklerSimilarity(t *testing.T) {
	cases := []struct {
		a, b string
		want float64
	}{
		{"MARTHA", "MARHTA", 0.961111},
		{"DWAYNE", "DUANE", 0.84},
		{"DIXON", "DICKSONX", 0.813333},
		{"abc", "abc", 1},
	}
	for _, c := range cases {
		if got := JaroWinklerSimilarity(c.a, c.b); !approxEqual(got, c.want) {
			t.Errorf("JaroWinklerSimilarity(%q, %q) = %f, want %f", c.a, c.b, got, c.want)
		}
	}
}

func BenchmarkDamerauLevenshteinDistance(b *testing.B) {
	for i := 0; i < b.N; i++ {
		DamerauLevenshteinDistance("information", "informatoin")
	}
}

func BenchmarkJaroWinklerSimilarity(b *testing.B) {
	for i := 0; i < b.N; i++ {
		JaroWinklerSimilarity("information", "informatoin")
	}
}
