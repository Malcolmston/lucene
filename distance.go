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
	t := float64(transpositions) / 2
	return (m/float64(la) + m/float64(lb) + (m-t)/m) / 3
}

// JaroWinklerSimilarity returns the Jaro-Winkler similarity of a and b, a value
// in [0,1]. It boosts the Jaro similarity when the strings share a common prefix
// (up to four characters), reflecting the observation that human errors are less
// likely at the start of a word. It uses the standard scaling factor of 0.1.
// This mirrors the Jaro-Winkler distance offered by Lucene's suggest module.
func JaroWinklerSimilarity(a, b string) float64 {
	jaro := JaroSimilarity(a, b)
	if jaro <= 0 {
		return jaro
	}
	ra, rb := []rune(a), []rune(b)
	maxPrefix := 4
	prefix := 0
	for prefix < len(ra) && prefix < len(rb) && prefix < maxPrefix && ra[prefix] == rb[prefix] {
		prefix++
	}
	return jaro + float64(prefix)*0.1*(1-jaro)
}
