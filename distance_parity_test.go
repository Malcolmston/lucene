package lucene

import (
	"math"
	"testing"
)

// This file encodes known-answer vectors taken directly from Apache Lucene's
// own unit tests, so the Go port's string-distance functions can be checked for
// parity against the upstream library. Sources:
//
//   - lucene/suggest .../spell/TestJaroWinklerDistance.java
//   - lucene/suggest .../spell/TestLevenshteinDistance.java
//   - lucene/suggest .../spell/TestNGramDistance.java
//   - lucene/analysis/phonetic .../TestPhoneticFilter.java (Soundex vectors)
//
// The tolerances mirror Lucene's own (0.001 for distance assertions).

const parityTol = 0.001

func parityClose(a, b float64) bool { return math.Abs(a-b) < parityTol }

// TestParityJaroWinklerDistance mirrors TestJaroWinklerDistance.testGetDistance.
func TestParityJaroWinklerDistance(t *testing.T) {
	if d := JaroWinklerSimilarity("al", "al"); d != 1.0 {
		t.Errorf("jw(al,al)=%v want 1.0", d)
	}
	cases := []struct {
		a, b   string
		lo, hi float64
	}{
		{"martha", "marhta", 0.961, 0.962},
		{"jones", "johnson", 0.832, 0.833},
		{"abcvwxyz", "cabvwxyz", 0.958, 0.959},
		{"dwayne", "duane", 0.84, 0.841},
		{"dixon", "dicksonx", 0.813, 0.814},
	}
	for _, c := range cases {
		d := JaroWinklerSimilarity(c.a, c.b)
		// Widen the lower bound by the shared tolerance so exact-boundary
		// values (Lucene asserts strict >) do not flap on float rounding.
		if d < c.lo-parityTol || d > c.hi {
			t.Errorf("jw(%q,%q)=%v want in (%v,%v)", c.a, c.b, d, c.lo, c.hi)
		}
	}
	if d := JaroWinklerSimilarity("fvie", "ten"); d != 0 {
		t.Errorf("jw(fvie,ten)=%v want 0", d)
	}
	// Relational assertions from the upstream test.
	if !(JaroWinklerSimilarity("zac ephron", "zac efron") > JaroWinklerSimilarity("zac ephron", "kai ephron")) {
		t.Error("jw: zac efron should beat kai ephron")
	}
	if !(JaroWinklerSimilarity("brittney spears", "britney spears") > JaroWinklerSimilarity("brittney spears", "brittney startzman")) {
		t.Error("jw: britney spears should beat brittney startzman")
	}
}

// TestParityLevenshteinDistance mirrors TestLevenshteinDistance.testGetDistance
// and testEmpty (Lucene's normalized similarity).
func TestParityLevenshteinDistance(t *testing.T) {
	cases := []struct {
		a, b string
		want float64
	}{
		{"al", "al", 1.0},
		{"martha", "marhta", 0.6666},
		{"jones", "johnson", 0.4285},
		{"abcvwxyz", "cabvwxyz", 0.75},
		{"dwayne", "duane", 0.666},
		{"dixon", "dicksonx", 0.5},
		{"six", "ten", 0},
		{"", "al", 0},
	}
	for _, c := range cases {
		if d := LevenshteinSimilarity(c.a, c.b); !parityClose(d, c.want) {
			t.Errorf("lev(%q,%q)=%v want %v", c.a, c.b, d, c.want)
		}
	}
	// zac efron and kai ephron are equidistant from zac ephron for plain
	// Levenshtein (two edits into a length-10 target).
	if d1, d2 := LevenshteinSimilarity("zac ephron", "zac efron"), LevenshteinSimilarity("zac ephron", "kai ephron"); !parityClose(d1, d2) {
		t.Errorf("lev: expected %v == %v", d1, d2)
	}
	if !(LevenshteinSimilarity("brittney spears", "britney spears") > LevenshteinSimilarity("brittney spears", "brittney startzman")) {
		t.Error("lev: britney spears should beat brittney startzman")
	}
}

// TestParityNGramDistance mirrors TestNGramDistance for n = 1, 2 and 3.
func TestParityNGramDistance(t *testing.T) {
	type tc struct {
		a, b string
		want float64
	}
	n1 := []tc{
		{"al", "al", 1.0}, {"a", "a", 1.0}, {"b", "a", 0.0},
		{"martha", "marhta", 0.6666}, {"jones", "johnson", 0.4285},
		{"natural", "contrary", 0.25}, {"abcvwxyz", "cabvwxyz", 0.75},
		{"dwayne", "duane", 0.666}, {"dixon", "dicksonx", 0.5}, {"six", "ten", 0},
	}
	n2 := []tc{
		{"al", "al", 1.0}, {"a", "a", 1.0}, {"b", "a", 0.0}, {"a", "aa", 0.5},
		{"martha", "marhta", 0.6666}, {"jones", "johnson", 0.4285},
		{"natural", "contrary", 0.25}, {"abcvwxyz", "cabvwxyz", 0.625},
		{"dwayne", "duane", 0.5833}, {"dixon", "dicksonx", 0.5}, {"six", "ten", 0},
	}
	n3 := []tc{
		{"al", "al", 1.0}, {"a", "a", 1.0}, {"b", "a", 0.0},
		{"martha", "marhta", 0.7222}, {"jones", "johnson", 0.4762},
		{"natural", "contrary", 0.2083}, {"abcvwxyz", "cabvwxyz", 0.5625},
		{"dwayne", "duane", 0.5277}, {"dixon", "dicksonx", 0.4583}, {"six", "ten", 0},
	}
	for n, cases := range map[int][]tc{1: n1, 2: n2, 3: n3} {
		for _, c := range cases {
			if d := NGramSimilarity(c.a, c.b, n); !parityClose(d, c.want) {
				t.Errorf("ngram%d(%q,%q)=%v want %v", n, c.a, c.b, d, c.want)
			}
		}
	}
	// testEmpty: NGram(1) getDistance("","al") == 0.
	if d := NGramSimilarity("", "al", 1); !parityClose(d, 0) {
		t.Errorf("ngram1(\"\",al)=%v want 0", d)
	}
}

// TestParitySoundex mirrors the Soundex vectors asserted in
// TestPhoneticFilter.testAlgorithms (commons-codec Soundex encoder).
func TestParitySoundex(t *testing.T) {
	cases := map[string]string{
		"aaa":     "A000",
		"bbb":     "B000",
		"ccc":     "C000",
		"easgasg": "E220",
	}
	for in, want := range cases {
		if got := Soundex(in); got != want {
			t.Errorf("Soundex(%q)=%q want %q", in, got, want)
		}
	}
}
