package lucene

import (
	"strings"
	"unicode"
)

// Token is a single unit of text produced by an Analyzer. Position records the
// ordinal offset of the token within the analyzed field (starting at zero) and
// is used by phrase queries for positional matching.
type Token struct {
	Text     string
	Position int
}

// Analyzer converts raw field text into a normalized stream of tokens. The
// standard pipeline is: tokenize on non-letter/digit boundaries, lowercase,
// drop stop words, then stem. Analyze is deterministic: the same input always
// yields the same tokens in the same order.
type Analyzer struct {
	stopWords map[string]struct{}
	stem      bool
}

// AnalyzerOption customizes the construction of an Analyzer.
type AnalyzerOption func(*Analyzer)

// WithStopWords configures the analyzer to drop the supplied stop words. The
// words are matched after lowercasing. Passing an empty slice disables stop
// word filtering.
func WithStopWords(words []string) AnalyzerOption {
	return func(a *Analyzer) {
		a.stopWords = make(map[string]struct{}, len(words))
		for _, w := range words {
			a.stopWords[strings.ToLower(w)] = struct{}{}
		}
	}
}

// WithStemming enables or disables the suffix-stripping stemmer. Stemming is on
// by default.
func WithStemming(on bool) AnalyzerOption {
	return func(a *Analyzer) {
		a.stem = on
	}
}

// DefaultStopWords is a small, conventional English stop word list. It is a
// copy; callers may modify the returned slice freely.
func DefaultStopWords() []string {
	return []string{
		"a", "an", "and", "are", "as", "at", "be", "but", "by",
		"for", "if", "in", "into", "is", "it", "no", "not", "of",
		"on", "or", "such", "that", "the", "their", "then", "there",
		"these", "they", "this", "to", "was", "will", "with",
	}
}

// NewAnalyzer builds an Analyzer. With no options it lowercases, applies no stop
// words, and stems. Use WithStopWords and WithStemming to customize behavior.
func NewAnalyzer(opts ...AnalyzerOption) *Analyzer {
	a := &Analyzer{
		stopWords: map[string]struct{}{},
		stem:      true,
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

// NewStandardAnalyzer returns an Analyzer configured with DefaultStopWords and
// stemming enabled. It is a convenient default for English text.
func NewStandardAnalyzer() *Analyzer {
	return NewAnalyzer(WithStopWords(DefaultStopWords()), WithStemming(true))
}

// Analyze runs the full pipeline on text and returns the resulting tokens with
// their positions. Positions count only tokens that survive filtering, so a
// stop word does not create a positional gap.
func (a *Analyzer) Analyze(text string) []Token {
	raw := tokenize(text)
	tokens := make([]Token, 0, len(raw))
	pos := 0
	for _, w := range raw {
		w = strings.ToLower(w)
		if _, stop := a.stopWords[w]; stop {
			continue
		}
		if a.stem {
			w = stem(w)
		}
		if w == "" {
			continue
		}
		tokens = append(tokens, Token{Text: w, Position: pos})
		pos++
	}
	return tokens
}

// AnalyzeTerm runs the pipeline on a single query term and returns the single
// normalized term, or the empty string if the term was filtered out entirely
// (for example, a stop word). It never applies stop word filtering so that an
// explicit single-term query is not silently dropped, but it does lowercase and
// stem to match the indexed form.
func (a *Analyzer) AnalyzeTerm(term string) string {
	term = strings.ToLower(strings.TrimSpace(term))
	if term == "" {
		return ""
	}
	if a.stem {
		term = stem(term)
	}
	return term
}

// tokenize splits text into maximal runs of letters and digits, discarding all
// other characters as separators.
func tokenize(text string) []string {
	return strings.FieldsFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}

// stem applies a light suffix-stripping stemmer inspired by the Porter
// algorithm. It handles common English inflections while remaining fully
// deterministic and dependency-free. It is intentionally conservative: short
// words are left untouched.
func stem(word string) string {
	if len(word) <= 3 {
		return word
	}

	// Step 1a: plurals and -ed/-ing handled below; first normalize plural s.
	switch {
	case strings.HasSuffix(word, "sses"):
		word = word[:len(word)-2] // sses -> ss
	case strings.HasSuffix(word, "ies"):
		word = word[:len(word)-3] + "i" // ponies -> poni
	case strings.HasSuffix(word, "ss"):
		// leave as-is
	case strings.HasSuffix(word, "s"):
		word = word[:len(word)-1]
	}

	// Step 1b: -eed, -ed, -ing.
	switch {
	case strings.HasSuffix(word, "eed"):
		if measure(word[:len(word)-3]) > 0 {
			word = word[:len(word)-1] // agreed -> agree
		}
	case strings.HasSuffix(word, "ed"):
		if stem := word[:len(word)-2]; containsVowel(stem) {
			word = fixup(stem)
		}
	case strings.HasSuffix(word, "ing"):
		if stem := word[:len(word)-3]; containsVowel(stem) {
			word = fixup(stem)
		}
	}

	// Step 2/3: common derivational suffixes, longest first.
	word = stripDerivational(word)

	// Step 4: remove a trailing -e when the stem is long enough.
	if strings.HasSuffix(word, "e") && len(word) > 4 && measure(word[:len(word)-1]) > 1 {
		word = word[:len(word)-1]
	}

	return word
}

// derivationalSuffixes are stripped when the preceding stem has a positive
// measure. They are ordered longest-first so the greediest match wins.
var derivationalSuffixes = []struct {
	suffix  string
	replace string
}{
	{"ational", "ate"},
	{"tional", "tion"},
	{"fulness", "ful"},
	{"ousness", "ous"},
	{"iveness", "ive"},
	{"ization", "ize"},
	{"alize", "al"},
	{"icate", "ic"},
	{"iciti", "ic"},
	{"ical", "ic"},
	{"ness", ""},
	{"ment", ""},
	{"ance", ""},
	{"ence", ""},
	{"able", ""},
	{"ible", ""},
	{"ant", ""},
	{"ent", ""},
	{"er", ""},
	{"or", ""},
	{"ly", ""},
	{"ful", ""},
	{"ize", ""},
	{"ity", ""},
	{"ous", ""},
	{"ive", ""},
}

func stripDerivational(word string) string {
	for _, d := range derivationalSuffixes {
		if strings.HasSuffix(word, d.suffix) {
			stem := word[:len(word)-len(d.suffix)]
			if measure(stem) > 0 {
				return stem + d.replace
			}
			return word
		}
	}
	return word
}

// fixup restores a canonical form after -ed/-ing removal: undoubles a final
// double consonant and re-adds a dropped -e in common cases.
func fixup(word string) string {
	if strings.HasSuffix(word, "at") || strings.HasSuffix(word, "bl") || strings.HasSuffix(word, "iz") {
		return word + "e"
	}
	if n := len(word); n >= 2 && word[n-1] == word[n-2] && !isVowel(rune(word[n-1])) &&
		word[n-1] != 'l' && word[n-1] != 's' && word[n-1] != 'z' {
		return word[:n-1]
	}
	return word
}

// measure counts vowel-consonant sequence transitions (the Porter "m" value),
// a rough proxy for the number of syllables in the stem.
func measure(word string) int {
	m := 0
	prevVowel := false
	started := false
	for _, r := range word {
		v := isVowel(r)
		if started && prevVowel && !v {
			m++
		}
		prevVowel = v
		started = true
	}
	return m
}

func containsVowel(word string) bool {
	for _, r := range word {
		if isVowel(r) {
			return true
		}
	}
	return false
}

func isVowel(r rune) bool {
	switch r {
	case 'a', 'e', 'i', 'o', 'u':
		return true
	default:
		return false
	}
}
