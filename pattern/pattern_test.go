package pattern

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatch(t *testing.T) {
	matchTests := []struct {
		str  string
		pat  string
		succ bool
		caps []string
	}{
		{"Apple", "[Aa]pple", true, []string{"Apple"}},
		{"Apple", "apple", false, []string{}},
		{"Apple", "(Ap)ple", true, []string{"Ap"}},
		{"Apple", "(Ap)p(le)", true, []string{"Ap", "le"}},
		{"Apple", "A(pp)(le)", true, []string{"pp", "le"}},
		{"apple", "a[Pp][Pp]le", true, []string{"apple"}},
		{"1", "%a", false, []string{}},
		{"a", "%c", false, []string{}},
		{"a", "%d", false, []string{}},
		{"A", "%l", false, []string{}},
		{"a", "%p", false, []string{}},
		{"a", "%s", false, []string{}},
		{"a", "%u", false, []string{}},
	}
	for i, test := range matchTests {
		succ, caps := Match(test.str, test.pat)
		assert.Equal(t, test.succ, succ, "match('%s', '%s') returned %t instead of expected %t", test.str, test.pat, succ, test.succ)
		assert.Equal(t, test.caps, caps, "%v Captures do not match: got %s expected %s", i, caps, test.caps)
	}
}

func TestPlainFind(t *testing.T) {
	plainTests := []struct {
		s1    string
		s2    string
		succ  bool
		start int
		end   int
	}{
		{"", "", true, 0, 0},
		{"a", "a", true, 0, 1},
		{"a", "b", false, -1, -1},
		{"ab", "b", true, 1, 2},
		{"ab", "a", true, 0, 1},
		{"aaa", "aaa", true, 0, 3},
		{"aaabaa", "aaa", true, 0, 3},
		{"aaabaa", "baa", true, 3, 6},
		{"aaa", "b", false, -1, -1},
		{"aaaba", "baa", false, -1, -1},
		{"aaabbaba", "aba", true, 5, 8},
	}

	for _, test := range plainTests {
		found, start, end, _ := Find(test.s1, test.s2, true)
		assert.Equal(t, test.succ, found)
		assert.Equal(t, test.start, start, "Fail in Find('%s', '%s', true) => %d instead of %d\n", test.s1, test.s2, start, test.start)
		assert.Equal(t, test.end, end, "Fail in Find('%s', '%s', true) => %d instead of %d\n", test.s1, test.s2, end, test.end)
	}
}

func TestFrontier(t *testing.T) {
	t.Skip()
	var frontierTests = []struct {
		src string
		pat string
		rep string
		res string
	}{
		{"aaa aa a aaa a", "%f[%w]a", "x", "xaa xa x xaa x"},
		{"[[]] [][] [[[[", "%f[[].", "x", "x[]] x]x] x[[["},
		{"01abc45de3", "%f[%d]", ".", ".01abc.45de.3"},
		{"function", "%f[\x01-\xff]%w", ".", ".unction"},
		{"function", "%f[^\x01-\xff]", ".", "function."},
	}
	for _, test := range frontierTests {
		res, _ := Replace(test.src, test.pat, test.rep, -1)
		if res != test.res {
			t.Errorf("replace('%s', '%s', '%s', %d) returned '%s', expected '%s'",
				test.src, test.pat, test.rep, -1, res, test.res)
			return
		}
	}
}

func TestGmatch(t *testing.T) {
	var gmatchTests = []struct {
		str  string
		pat  string
		vals [][]string
	}{
		{"Apple", "%w", [][]string{{"A"}, {"p"}, {"p"}, {"l"}, {"e"}}},
		{"Apple", "(%w)(%w)", [][]string{{"A", "p"}, {"p", "l"}}},
	}

	for _, test := range gmatchTests {
		idx := 0
		for caps := range Gmatch(test.str, test.pat) {
			assert.Equal(t, test.vals[idx], caps, "Captures do not match: got %s espected %s", caps, test.vals)
			idx = idx + 1
		}
	}
}

func TestPatternPos(t *testing.T) {
	posTests := []struct {
		str, pat   string
		succ       bool
		start, end int
	}{
		{"", "", true, 0, 0},    // special case
		{"alo", "", true, 0, 0}, // special case
		{"a\x00o a\x00o a\x00o", "a", true, 0, 1},
		{"o a\x00o a\x00o", "a\x00o", true, 2, 5},
		{"a\x00o", "a\x00o", true, 0, 3},
		{"oa\x00a\x00a\x00\x00ab", "\x00ab", true, 7, 10},
		{"a\x00oa\x00a\x00a\x00\x00ab", "b", true, 11, 12},
		{"a\x00oa\x00a\x00a\x00\x00ab", "b\x00", false, -1, -1},
		{"", "\x00", false, -1, -1},
		{"alo123alo", "12", true, 3, 5},
		{"alo123alo", "^12", false, -1, -1},
	}

	for _, test := range posTests {
		succ, start, end, _ := Find(test.str, test.pat, false)
		assert.Equal(t, test.succ, succ)
		assert.Equal(t, test.start, start, "Fail in Find('%s', '%s', true) => %d instead of %d\n", test.str, test.pat, start, test.start)
		assert.Equal(t, test.end, end, "Fail in Find('%s', '%s', true) => %d instead of %d\n", test.str, test.pat, end, test.end)
	}
}

func TestReplace(t *testing.T) {
	replaceTests := []struct {
		src string
		pat string
		rep string
		max int
		res string
		n   int
	}{
		{"alo alo", "a", "x", -1, "xlo xlo", 2},
		{"alo alo  ", " +$", "", -1, "alo alo", 1},              // trim
		{"  alo alo  ", "^%s*(.-)%s*$", "%1", -1, "alo alo", 1}, // double trim
		// POSITION CAPTURES NOT SUPPORTED
		//{"abc=xyz", "(%w*)(%p)(%w+)", "%3%2%1", -1, "xyz=abc", 1},
		{"aei", "$", "\x00ou", -1, "aei\x00ou", 1},
		{"", "^", "r", -1, "r", 1},
		{"", "$", "r", -1, "r", 1},
	}

	for _, test := range replaceTests {
		res, n := Replace(test.src, test.pat, test.rep, test.max)
		if res != test.res {
			t.Errorf("replace('%s', '%s', '%s', %d) returned '%s', expected '%s'",
				test.src, test.pat, test.rep, test.max, res, test.res)
			return
		} else if n != test.n {
			t.Errorf("replace('%s', '%s', '%s', %d) performed %d replacements, expected %d",
				test.src, test.pat, test.rep, test.max, n, test.n)
			return
		}
	}
}

func TestSubtring(t *testing.T) {
	subTests := []struct {
		str  string
		pat  string
		succ bool
		cap  string
	}{
		{"aaab", "a*", true, "aaa"},
		{"aaa", "^.*$", true, "aaa"},
		{"aaa", "b*", true, ""},
		{"aaa", "ab*a", true, "aa"},
		{"aba", "ab*a", true, "aba"},
		{"aaab", "a+", true, "aaa"},
		{"aaa", "^.+$", true, "aaa"},
		{"aaa", "b+", false, ""},
		{"aaa", "ab+a", false, ""},
		{"aba", "ab+a", true, "aba"},
		{"a$a", ".$", true, "a"},
		{"a$a", ".%$", true, "a$"},
		{"a$a", ".$.", true, "a$a"},
		{"a$a", "$$", false, ""},
		{"a$b", "a$", false, ""},
		{"a$a", "$", true, ""},
		{"", "b*", true, ""},
		{"aaa", "bb*", false, ""},
		{"aaab", "a-", true, ""},
		{"aaa", "^.-$", true, "aaa"},
		{"aabaaabaaabaaaba", "b.*b", true, "baaabaaabaaab"},
		{"aabaaabaaabaaaba", "b.-b", true, "baaab"},
		{"alo xo", ".o$", true, "xo"},
		{" \n isto \x82 assim", "%S%S*", true, "isto"},
		{" \n isto \x82 assim", "%S*$", true, "assim"},
		{" \n isto \x82 assim", "[a-z]*$", true, "assim"},
		{"im caracter ? extra", "[^%sa-z]", true, "?"},
		{"", "a?", true, ""},

		// These tests don't work 100% correctly if you use Unicode á instead
		// of the ASCII value \225. In particular the third test fails
		{"\225", "\225?", true, "\225"},
		{"\225bl", "\225?b?l?", true, "\225bl"},
		{"  \225bl", "\225?b?l?", true, ""},
		{"aa", "^aa?a?a", true, "aa"},
		{"]]]\225b", "[^]]", true, "\225"},
		{"0alo alo", "%x*", true, "0alo"},
		{"alo alo", "%C+", true, "alo alo"},

		{"axz123= 4= 4 34", "(.+)=(.*)=%2 %1", true, "3= 4= 4 3"},
	}

	for _, test := range subTests {
		succ, start, end, _ := Find(test.str, test.pat, false)
		assert.Equal(t, test.succ, succ, "find('%s', '%s') returned %t, expected %t", test.str, test.pat, succ, test.succ)
		if succ {
			substr := test.str[start:end]
			assert.Equal(t, test.cap, substr, "find('%s', '%s') => substr '%s' does not match expected '%s' (got %d and %d as start/end)", test.str, test.pat, substr, test.cap, start, end)
		}
	}
}

func BenchmarkLongBytes(b *testing.B) {
	limit := int(1e6)
	longBytes := make([]byte, limit+1)
	for i := 0; i < limit; i++ {
		longBytes[i] = 'a'
	}
	longBytes[limit] = 'b'
	FindBytes(longBytes, []byte(".-b"), false)
}

func BenchmarkLongString(b *testing.B) {
	limit := int(1e6)
	longString := strings.Repeat("a", limit) + "b"
	Find(longString, ".-b", false)
}
