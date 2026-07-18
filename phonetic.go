package lucene

import "strings"

// Soundex encodes an English word into the classic four-character Soundex code:
// an uppercase initial letter followed by three digits derived from the
// remaining consonants. It mirrors the phonetic encoding in Lucene's analysis
// module and groups words that sound alike ("Robert" and "Rupert" both encode
// to "R163"). Non-letter characters are ignored. An input with no letters yields
// the empty string. The result is always padded or truncated to exactly four
// characters when any letter is present.
func Soundex(s string) string {
	var letters []rune
	for _, r := range s {
		if r >= 'a' && r <= 'z' {
			r -= 'a' - 'A'
		}
		if r >= 'A' && r <= 'Z' {
			letters = append(letters, r)
		}
	}
	if len(letters) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteRune(letters[0])
	prev := luSoundexCode(letters[0])
	for i := 1; i < len(letters) && b.Len() < 4; i++ {
		code := luSoundexCode(letters[i])
		// 'h' and 'w' are transparent: they do not separate two consonants
		// that share a code, whereas a vowel does.
		if letters[i] == 'H' || letters[i] == 'W' {
			continue
		}
		if code != '0' && code != prev {
			b.WriteRune(code)
		}
		prev = code
	}
	for b.Len() < 4 {
		b.WriteByte('0')
	}
	return b.String()
}

// luSoundexCode returns the Soundex digit for an uppercase letter, or '0' for
// vowels and other non-coded letters.
func luSoundexCode(r rune) rune {
	switch r {
	case 'B', 'F', 'P', 'V':
		return '1'
	case 'C', 'G', 'J', 'K', 'Q', 'S', 'X', 'Z':
		return '2'
	case 'D', 'T':
		return '3'
	case 'L':
		return '4'
	case 'M', 'N':
		return '5'
	case 'R':
		return '6'
	default:
		return '0'
	}
}
