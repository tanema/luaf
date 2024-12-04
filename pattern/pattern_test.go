package pattern

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFind(t *testing.T) {
	matchTests := []struct {
		str, pat string
		results  []*Match
	}{
		{"Apple", "A*pple", []*Match{{Start: 0, End: 5}}},
		{"Apple", "apple", []*Match{}},
		{"Apple", "(Ap)ple", []*Match{{Start: 0, End: 5}}},
		{"Apple", "(Ap)p(le)", []*Match{{Start: 3, End: 5}}},
		{"Apple", "A(pp)(le)", []*Match{{Start: 3, End: 5}}},
		// {"apple", "a[Pp][Pp]le", []string{"apple"}},
		// {"1", "%a", []string{}},
		// {"a", "%c", []string{}},
		// {"a", "%d", []string{}},
		// {"A", "%l", []string{}},
		// {"a", "%p", []string{}},
		// {"a", "%s", []string{}},
		// {"a", "%u", []string{}},
		// {"aaab", "a*", "aaa"},
		// {"aaa", "^.*$", "aaa"},
		// {"aaa", "b*", ""},
		// {"aaa", "ab*a", "aa"},
		// {"aba", "ab*a", "aba"},
		// {"aaab", "a+", "aaa"},
		// {"aaa", "^.+$", "aaa"},
		// {"aaa", "b+", ""},
		// {"aaa", "ab+a", ""},
		// {"aba", "ab+a", "aba"},
		// {"a$a", ".$", "a"},
		// {"a$a", ".%$", "a$"},
		// {"a$a", ".$.", "a$a"},
		// {"a$a", "$$", ""},
		// {"a$b", "a$", ""},
		// {"a$a", "$", ""},
		// {"", "b*", ""},
		// {"aaa", "bb*", ""},
		// {"aaab", "a-", ""},
		// {"aaa", "^.-$", "aaa"},
		// {"aabaaabaaabaaaba", "b.*b", "baaabaaabaaab"},
		// {"aabaaabaaabaaaba", "b.-b", "baaab"},
		// {"alo xo", ".o$", "xo"},
		// {" \n isto \x82 assim", "%S%S*", "isto"},
		// {" \n isto \x82 assim", "%S*$", "assim"},
		// {" \n isto \x82 assim", "[a-z]*$", "assim"},
		// {"im caracter ? extra", "[^%sa-z]", "?"},
		// {"", "a?", ""},
		// {"\225", "\225?", "\225"},
		// {"\225bl", "\225?b?l?", "\225bl"},
		// {"  \225bl", "\225?b?l?", ""},
		// {"aa", "^aa?a?a", "aa"},
		// {"]]]\225b", "[^]]", "\225"},
		// {"0alo alo", "%x*", "0alo"},
		// {"alo alo", "%C+", "alo alo"},
		// {"axz123= 4= 4 34", "(.+)=(.*)=%2 %1", "3= 4= 4 3"},
	}

	for i, test := range matchTests {
		match, err := Find(test.pat, test.str, 0, -1)
		require.NoError(t, err)
		assert.Equal(t, test.results, match, "[%v]", i)
	}
}
