package lucene

import (
	"reflect"
	"testing"
)

func TestNGrams(t *testing.T) {
	cases := []struct {
		s    string
		n    int
		want []string
	}{
		{"hello", 2, []string{"he", "el", "ll", "lo"}},
		{"hello", 3, []string{"hel", "ell", "llo"}},
		{"ab", 3, []string{}},
		{"abc", 0, []string{}},
		{"", 1, []string{}},
		{"café", 2, []string{"ca", "af", "fé"}}, // rune-aware
	}
	for _, c := range cases {
		if got := NGrams(c.s, c.n); !reflect.DeepEqual(got, c.want) {
			t.Errorf("NGrams(%q, %d) = %v, want %v", c.s, c.n, got, c.want)
		}
	}
}

func TestEdgeNGrams(t *testing.T) {
	cases := []struct {
		s          string
		minG, maxG int
		want       []string
	}{
		{"hello", 1, 3, []string{"h", "he", "hel"}},
		{"hello", 2, 4, []string{"he", "hel", "hell"}},
		{"hi", 1, 10, []string{"h", "hi"}}, // maxGram capped at length
		{"hello", 3, 2, []string{}},        // max < min
		{"", 1, 3, []string{}},
	}
	for _, c := range cases {
		if got := EdgeNGrams(c.s, c.minG, c.maxG); !reflect.DeepEqual(got, c.want) {
			t.Errorf("EdgeNGrams(%q, %d, %d) = %v, want %v", c.s, c.minG, c.maxG, got, c.want)
		}
	}
}

func TestShingles(t *testing.T) {
	cases := []struct {
		tokens []string
		n      int
		want   []string
	}{
		{[]string{"a", "b", "c"}, 2, []string{"a b", "b c"}},
		{[]string{"the", "quick", "brown", "fox"}, 3, []string{"the quick brown", "quick brown fox"}},
		{[]string{"a"}, 2, []string{}},
		{[]string{"a", "b"}, 0, []string{"a b"}}, // n<1 defaults to 2
	}
	for _, c := range cases {
		if got := Shingles(c.tokens, c.n); !reflect.DeepEqual(got, c.want) {
			t.Errorf("Shingles(%v, %d) = %v, want %v", c.tokens, c.n, got, c.want)
		}
	}
}

func BenchmarkNGrams(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NGrams("information retrieval", 3)
	}
}
