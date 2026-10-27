package algo

// KMP finds a fixed pattern in byte slices with the Knuth-Morris-Pratt algorithm.
// Build it once with NewKMP and reuse it: the failure table depends only on the pattern.
type KMP struct {
	pattern []byte
	// lps[i] is the length of the longest proper prefix of pattern[:i+1]
	// that is also a suffix of it.
	lps []int
}

// NewKMP precomputes the failure table for pattern.
// Time: O(m), Space: O(m), where m = len(pattern)
func NewKMP(pattern []byte) *KMP {
	lps := make([]int, len(pattern))
	length := 0
	for i := 1; i < len(pattern); {
		if pattern[i] == pattern[length] {
			length++
			lps[i] = length
			i++
		} else if length > 0 {
			length = lps[length-1]
		} else {
			lps[i] = 0
			i++
		}
	}
	return &KMP{pattern: pattern, lps: lps}
}

// Index returns the index of the first occurrence of the pattern in text, or -1 if absent.
// An empty pattern matches at 0.
// Time: O(n), Space: O(1)
func (k *KMP) Index(text []byte) int {
	m := len(k.pattern)
	if m == 0 {
		return 0
	}

	matched := 0

	for i := 0; i < len(text); {
		if text[i] == k.pattern[matched] {
			i++
			matched++

			if matched == m {
				return i - m
			}
		} else if matched > 0 {
			matched = k.lps[matched-1]
		} else {
			i++
		}
	}

	return -1
}
