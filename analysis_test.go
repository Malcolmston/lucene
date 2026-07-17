package lucene

import (
	"reflect"
	"testing"
)

func tokenTexts(toks []Token) []string {
	out := make([]string, len(toks))
	for i, t := range toks {
		out[i] = t.Text
	}
	return out
}

func TestTokenize(t *testing.T) {
	got := tokenize("Hello, World! 123_foo")
	want := []string{"Hello", "World", "123", "foo"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("tokenize = %v, want %v", got, want)
	}
}

func TestAnalyzeLowercaseAndStop(t *testing.T) {
	a := NewAnalyzer(WithStopWords([]string{"the", "a"}), WithStemming(false))
	toks := a.Analyze("The Quick a BROWN fox")
	got := tokenTexts(toks)
	want := []string{"quick", "brown", "fox"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
	// Positions should be contiguous despite dropped stop words.
	for i, tok := range toks {
		if tok.Position != i {
			t.Errorf("position[%d] = %d, want %d", i, tok.Position, i)
		}
	}
}

func TestAnalyzeStemming(t *testing.T) {
	a := NewAnalyzer(WithStemming(true))
	cases := map[string]string{
		"running":  "run",
		"runs":     "run",
		"happily":  "happi",
		"agreed":   "agree",
		"ponies":   "poni",
		"caresses": "caress",
		"national": "national",
		"systems":  "system",
	}
	for in, want := range cases {
		toks := a.Analyze(in)
		if len(toks) != 1 {
			t.Fatalf("Analyze(%q) produced %d tokens", in, len(toks))
		}
		if toks[0].Text != want {
			t.Errorf("stem(%q) = %q, want %q", in, toks[0].Text, want)
		}
	}
}

func TestAnalyzeTerm(t *testing.T) {
	a := NewStandardAnalyzer()
	if got := a.AnalyzeTerm("  Running  "); got != "run" {
		t.Errorf("AnalyzeTerm = %q, want run", got)
	}
	if got := a.AnalyzeTerm("   "); got != "" {
		t.Errorf("AnalyzeTerm(blank) = %q, want empty", got)
	}
	// AnalyzeTerm must NOT drop stop words.
	if got := a.AnalyzeTerm("the"); got == "" {
		t.Errorf("AnalyzeTerm(the) unexpectedly empty")
	}
}

func TestStemShortWord(t *testing.T) {
	if got := stem("go"); got != "go" {
		t.Errorf("stem(go) = %q", got)
	}
	if got := stem("cat"); got != "cat" {
		t.Errorf("stem(cat) = %q", got)
	}
}

func TestDefaultStopWordsCopy(t *testing.T) {
	a := DefaultStopWords()
	b := DefaultStopWords()
	if len(a) == 0 {
		t.Fatal("empty stop words")
	}
	a[0] = "MUTATED"
	if b[0] == "MUTATED" {
		t.Error("DefaultStopWords returns shared slice")
	}
}

func TestMeasureAndVowel(t *testing.T) {
	if measure("tr") != 0 {
		t.Errorf("measure(tr) = %d", measure("tr"))
	}
	if measure("tree") != 0 {
		t.Errorf("measure(tree) = %d", measure("tree"))
	}
	if measure("trouble") != 1 {
		t.Errorf("measure(trouble) = %d, want 1", measure("trouble"))
	}
	if !containsVowel("cat") || containsVowel("xyz") {
		t.Error("containsVowel wrong")
	}
}
