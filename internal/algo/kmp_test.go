package algo

import (
	"bytes"
	"slices"
	"testing"
)

func TestNewKMPFailureTable(t *testing.T) {
	tests := []struct {
		pattern string
		want    []int
	}{
		{"", []int{}},
		{"a", []int{0}},
		{"\r\n", []int{0, 0}},
		{"aaaa", []int{0, 1, 2, 3}},
		{"abab", []int{0, 0, 1, 2}},
		{"aabaaab", []int{0, 1, 0, 1, 2, 2, 3}},
		{"abcd", []int{0, 0, 0, 0}},
		{"abcabcabc", []int{0, 0, 0, 1, 2, 3, 4, 5, 6}},
		// Mismatch at the end falls back through several table entries.
		{"aabaabaaa", []int{0, 1, 0, 1, 2, 3, 4, 5, 2}},
		{"abacabab", []int{0, 0, 1, 0, 1, 2, 3, 2}},
	}

	for _, tt := range tests {
		got := NewKMP([]byte(tt.pattern)).lps
		if !slices.Equal(got, tt.want) {
			t.Errorf("NewKMP(%q).lps = %v, want %v", tt.pattern, got, tt.want)
		}
	}
}

func TestKMPIndexMatchesBytesIndex(t *testing.T) {
	tests := []struct {
		text, pattern string
	}{
		{"", ""},
		{"abc", ""},
		{"", "a"},
		{"+OK\r\n", "\r\n"},
		{"OK", "\r\n"},
		{"OK\r", "\r\n"},
		{"\r\r\n", "\r\n"},
		{"$5\r\nhello\r\n", "\r\n"},
		{"aaaab", "aab"},
		{"abababca", "ababca"},
		{"abcabd", "abd"},
		{"ab", "abc"},
		{"xyz", "xyz"},

		// Position of the match
		{"abcxyz", "abc"},
		{"xyzabc", "abc"},
		{"a", "a"},
		{"a", "b"},

		// Several matches: the first one wins
		{"abcabcabc", "abc"},
		{"aaaa", "aa"},
		{"\r\n\r\n", "\r\n"},

		// Partial matches that must fall back without skipping the real match
		{"aaaaaaab", "aaab"},
		{"aabaabaaab", "aabaaab"},
		{"abacababacabab", "abacabab"},
		{"ababababc", "ababc"},
		{"abcabcabd", "abcabd"},
		{"aaaaaaaa", "aaab"},

		// Bytes are compared exactly
		{"ABC", "abc"},
		{"\x00\x00\x01", "\x00\x01"},
		{"\xff\xfe\xff\xff", "\xff\xff"},
		{"héllo wörld", "wörld"},

		// RESP-shaped input
		{"\n\r\n", "\r\n"},
		{"\r\r\r", "\r\n"},
		{"\n\n\n", "\r\n"},
		{"*2\r\n$4\r\nECHO\r\n", "\r\n"},
	}

	for _, tt := range tests {
		want := bytes.Index([]byte(tt.text), []byte(tt.pattern))
		got := NewKMP([]byte(tt.pattern)).Index([]byte(tt.text))
		if got != want {
			t.Errorf("Index(%q, %q) = %d, want %d", tt.text, tt.pattern, got, want)
		}
	}
}

// TestKMPIndexExhaustive compares against bytes.Index for every text up to
// length 10 and every pattern up to length 5 over {a, b}. A two-letter
// alphabet forces many overlapping partial matches.
func TestKMPIndexExhaustive(t *testing.T) {
	texts := allStrings("ab", 10)
	for _, pattern := range allStrings("ab", 5) {
		k := NewKMP([]byte(pattern))
		for _, text := range texts {
			want := bytes.Index([]byte(text), []byte(pattern))
			if got := k.Index([]byte(text)); got != want {
				t.Fatalf("Index(%q, %q) = %d, want %d", text, pattern, got, want)
			}
		}
	}
}

// TestKMPReuse checks that one matcher gives the same answers across calls,
// so it is safe to share like crlfMatcher in the protocol package.
func TestKMPReuse(t *testing.T) {
	k := NewKMP([]byte("\r\n"))
	for range 2 {
		for _, tt := range []struct {
			text string
			want int
		}{
			{"+OK\r\n", 3},
			{"+OK\r", -1},
			{"\r\n", 0},
			{"", -1},
		} {
			if got := k.Index([]byte(tt.text)); got != tt.want {
				t.Errorf("Index(%q) = %d, want %d", tt.text, got, tt.want)
			}
		}
	}
}

// allStrings returns every string over alphabet with length 0 to maxLen.
func allStrings(alphabet string, maxLen int) []string {
	out := []string{""}
	level := []string{""}
	for range maxLen {
		var next []string
		for _, s := range level {
			for _, c := range alphabet {
				next = append(next, s+string(c))
			}
		}
		out = append(out, next...)
		level = next
	}
	return out
}
