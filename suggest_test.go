package lucene

import (
	"reflect"
	"testing"
)

func TestSuggest(t *testing.T) {
	idx := sampleIndex(t)
	// "prog" -> the stemmed term "program".
	if got := idx.Suggest("body", "prog", 10); !reflect.DeepEqual(got, []string{"program"}) {
		t.Errorf("Suggest body prog = %v, want [program]", got)
	}
	// Prefix ordering favours the higher document frequency. Both "program"
	// (df 3) and "go" (df 2) are present but do not share a prefix; use a shared
	// one-letter prefix to exercise the ordering deterministically.
	got := idx.Suggest("body", "g", 10)
	if len(got) == 0 || got[0] != "go" {
		t.Errorf("Suggest body g = %v, want to start with go", got)
	}
	// Empty prefix and missing field.
	if len(idx.Suggest("body", "", 10)) != 0 {
		t.Error("empty prefix suggested terms")
	}
	if len(idx.Suggest("missing", "x", 10)) != 0 {
		t.Error("missing field suggested terms")
	}
	// max limit is honoured.
	all := idx.Suggest("body", "s", 0)
	if len(all) >= 2 {
		if got := idx.Suggest("body", "s", 1); len(got) != 1 {
			t.Errorf("Suggest with max 1 returned %d", len(got))
		}
	}
}

func TestSpellCheck(t *testing.T) {
	idx := sampleIndex(t)
	// One deletion from the indexed term "program".
	got := idx.SpellCheck("body", "programm", 1, 10)
	if !contains(got, "program") {
		t.Errorf("SpellCheck programm = %v, want to contain program", got)
	}
	// One insertion recovers "google".
	got = idx.SpellCheck("body", "googe", 1, 10)
	if !contains(got, "google") {
		t.Errorf("SpellCheck googe = %v, want to contain google", got)
	}
	// The input term itself is never suggested.
	got = idx.SpellCheck("body", "go", 1, 10)
	if contains(got, "go") {
		t.Errorf("SpellCheck suggested the input term: %v", got)
	}
	// Empty term and missing field.
	if len(idx.SpellCheck("body", "", 1, 10)) != 0 {
		t.Error("empty term produced suggestions")
	}
	if len(idx.SpellCheck("missing", "go", 1, 10)) != 0 {
		t.Error("missing field produced suggestions")
	}
}
