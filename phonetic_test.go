package lucene

import "testing"

func TestSoundex(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"Robert", "R163"},
		{"Rupert", "R163"},
		{"Rubin", "R150"},
		{"Ashcraft", "A261"}, // h/w transparency rule
		{"Tymczak", "T522"},  // vowel separates equal codes
		{"Honeyman", "H555"},
		{"Lee", "L000"},
		{"", ""},
		{"123", ""},
	}
	for _, c := range cases {
		if got := Soundex(c.in); got != c.want {
			t.Errorf("Soundex(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestSoundexHomophones(t *testing.T) {
	if Soundex("Robert") != Soundex("Rupert") {
		t.Errorf("expected Robert and Rupert to share a Soundex code")
	}
}

func BenchmarkSoundex(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Soundex("Ashcraft")
	}
}
