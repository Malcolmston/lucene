package lucene

// DamerauLevenshteinDistance returns the optimal string alignment distance
// between a and b: the minimum number of single-character insertions, deletions,
// substitutions, and transpositions of two adjacent characters needed to turn a
// into b. It extends LevenshteinDistance by counting a swap of neighbouring
// characters (for example "ab" to "ba") as one edit rather than two, which
// better models common typos. It operates on Unicode code points, is symmetric,
// and runs in O(len(a)*len(b)) time.
func DamerauLevenshteinDistance(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	la, lb := len(ra), len(rb)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}
	// d has three rolling rows: prev2, prev1, curr.
	prev2 := make([]int, lb+1)
	prev1 := make([]int, lb+1)
	curr := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev1[j] = j
	}
	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			curr[j] = luMin3(prev1[j]+1, curr[j-1]+1, prev1[j-1]+cost)
			if i > 1 && j > 1 && ra[i-1] == rb[j-2] && ra[i-2] == rb[j-1] {
				if t := prev2[j-2] + 1; t < curr[j] {
					curr[j] = t
				}
			}
		}
		prev2, prev1, curr = prev1, curr, prev2
	}
	return prev1[lb]
}

// JaroSimilarity returns the Jaro similarity of a and b, a value in [0,1] where
// 1 means the strings are identical and 0 means they share no matching
// characters. It accounts for matching characters within a sliding window and
// the number of transpositions among them, and is a common ingredient in
// record-linkage and typo-tolerant matching. Two empty strings are defined to
// have similarity 1.
func JaroSimilarity(a, b string) float64 {
	ra, rb := []rune(a), []rune(b)
	la, lb := len(ra), len(rb)
	if la == 0 && lb == 0 {
		return 1
	}
	if la == 0 || lb == 0 {
		return 0
	}
	window := la
	if lb > window {
		window = lb
	}
	window = window/2 - 1
	if window < 0 {
		window = 0
	}
	aMatched := make([]bool, la)
	bMatched := make([]bool, lb)
	matches := 0
	for i := 0; i < la; i++ {
		lo := i - window
		if lo < 0 {
			lo = 0
		}
		hi := i + window + 1
		if hi > lb {
			hi = lb
		}
		for j := lo; j < hi; j++ {
			if bMatched[j] || ra[i] != rb[j] {
				continue
			}
			aMatched[i] = true
			bMatched[j] = true
			matches++
			break
		}
	}
	if matches == 0 {
		return 0
	}
	// Count transpositions: half the number of matched characters that occur
	// in a different order between the two strings.
	transpositions := 0
	k := 0
	for i := 0; i < la; i++ {
		if !aMatched[i] {
			continue
		}
		for !bMatched[k] {
			k++
		}
		if ra[i] != rb[k] {
			transpositions++
		}
		k++
	}
	m := float64(matches)
	// Lucene truncates the transposition count with integer division, so an
	// odd number of out-of-order matches (for example three) contributes only
	// one transposition. Dividing as a float here would diverge from Lucene's
	// JaroWinklerDistance on such inputs.
	t := float64(transpositions / 2)
	return (m/float64(la) + m/float64(lb) + (m-t)/m) / 3
}

// jaroWinklerThreshold is the Jaro similarity below which Lucene's
// JaroWinklerDistance applies no common-prefix boost, leaving the plain Jaro
// score unchanged.
const jaroWinklerThreshold = 0.7

// JaroWinklerSimilarity returns the Jaro-Winkler similarity of a and b, a value
// in [0,1]. It boosts the Jaro similarity when the strings share a common
// prefix, reflecting the observation that human errors are less likely at the
// start of a word. This is a faithful port of Lucene's JaroWinklerDistance
// (lucene/suggest): the boost is applied only when the Jaro similarity is at
// least the 0.7 threshold; the common prefix is not capped in length; and the
// scaling factor is min(0.1, 1/maxLen) where maxLen is the rune length of the
// longer string. It operates on Unicode code points and is symmetric.
func JaroWinklerSimilarity(a, b string) float64 {
	jaro := JaroSimilarity(a, b)
	if jaro <= 0 {
		return jaro
	}
	if jaro < jaroWinklerThreshold {
		return jaro
	}
	ra, rb := []rune(a), []rune(b)
	minLen := min(len(ra), len(rb))
	maxLen := max(len(ra), len(rb))
	prefix := 0
	for prefix < minLen && ra[prefix] == rb[prefix] {
		prefix++
	}
	scale := 0.1
	if maxLen > 0 {
		if inv := 1.0 / float64(maxLen); inv < scale {
			scale = inv
		}
	}
	return jaro + scale*float64(prefix)*(1-jaro)
}

// LevenshteinSimilarity returns the normalized Levenshtein similarity of a and
// b, a value in [0,1] where 1 means the strings are identical. It is a faithful
// port of Lucene's LevenshteinDistance.getDistance (lucene/suggest): the raw
// edit distance is normalized as 1 - distance/max(len(a), len(b)). Following
// Lucene, if either string is empty the result is 1 when both are empty and 0
// otherwise. It operates on Unicode code points and is symmetric.
func LevenshteinSimilarity(a, b string) float64 {
	ra, rb := []rune(a), []rune(b)
	n, m := len(ra), len(rb)
	if n == 0 || m == 0 {
		if n == m {
			return 1
		}
		return 0
	}
	dist := LevenshteinDistance(a, b)
	return 1 - float64(dist)/float64(max(n, m))
}

// NGramSimilarity returns the n-gram similarity of a and b, a value in [0,1]
// where 1 means the strings are identical. It is a faithful port of Lucene's
// NGramDistance.getDistance (lucene/suggest), a Levenshtein-style edit distance
// computed over overlapping character n-grams rather than single characters,
// with affix padding so that boundary n-grams are compared fairly. Larger n
// weighs longer shared substrings more heavily. If n is less than one it
// defaults to one, in which case the result equals LevenshteinSimilarity.
// Following Lucene, if either string is empty the result is 1 when both are
// empty and 0 otherwise. It operates on Unicode code points and is symmetric in
// value for equal n.
func NGramSimilarity(a, b string, n int) float64 {
	if n < 1 {
		n = 1
	}
	source, target := []rune(a), []rune(b)
	sl, tl := len(source), len(target)
	if sl == 0 || tl == 0 {
		if sl == tl {
			return 1
		}
		return 0
	}
	// When either string is shorter than a single n-gram, Lucene falls back to
	// counting matching leading characters.
	if sl < n || tl < n {
		cost := 0
		for i, ni := 0, min(sl, tl); i < ni; i++ {
			if source[i] == target[i] {
				cost++
			}
		}
		return float64(cost) / float64(max(sl, tl))
	}

	// sa is source prefixed with n-1 NUL sentinels so that the first real
	// n-gram is aligned; matches against a sentinel are discounted below.
	sa := make([]rune, sl+n-1)
	for i := range sa {
		if i < n-1 {
			sa[i] = 0
		} else {
			sa[i] = source[i-n+1]
		}
	}
	p := make([]float64, sl+1) // previous cost row
	d := make([]float64, sl+1) // current cost row
	tj := make([]rune, n)      // jth n-gram of target
	for i := 0; i <= sl; i++ {
		p[i] = float64(i)
	}
	for j := 1; j <= tl; j++ {
		if j < n {
			for ti := 0; ti < n-j; ti++ {
				tj[ti] = 0
			}
			for ti := n - j; ti < n; ti++ {
				tj[ti] = target[ti-(n-j)]
			}
		} else {
			copy(tj, target[j-n:j])
		}
		d[0] = float64(j)
		for i := 1; i <= sl; i++ {
			cost := 0
			tn := n
			for k := 0; k < n; k++ {
				if sa[i-1+k] != tj[k] {
					cost++
				} else if sa[i-1+k] == 0 { // discount matches on the prefix sentinel
					tn--
				}
			}
			ec := float64(cost) / float64(tn)
			d[i] = min(min(d[i-1]+1, p[i]+1), p[i-1]+ec)
		}
		p, d = d, p
	}
	return 1 - p[sl]/float64(max(tl, sl))
}
