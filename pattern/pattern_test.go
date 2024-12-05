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
		{"Apple", "A*pple", []*Match{{Subs: "Apple", Start: 0, End: 5}}},
		{"Apple", "apple", []*Match{}},
		{"Apple", "(Ap)ple", []*Match{{Subs: "Apple", Start: 0, End: 5}, {Subs: "Ap", Start: 0, End: 2}}},
		{"Apple", "(Ap)p(le)", []*Match{{Subs: "Apple", Start: 0, End: 5}, {Subs: "Ap", Start: 0, End: 2}, {Subs: "le", Start: 3, End: 5}}},
		{"Apple", "A(pp)(le)", []*Match{{Subs: "Apple", Start: 0, End: 5}, {Subs: "pp", Start: 1, End: 3}, {Subs: "le", Start: 3, End: 5}}},
		{"apple", "a[Pp][Pp]le", []*Match{{Subs: "apple", Start: 0, End: 5}}},
		{"1", "%x", []*Match{{Subs: "1", Start: 0, End: 1}}},
		{"a", "%x", []*Match{{Subs: "a", Start: 0, End: 1}}},
		{"1", "%a", []*Match{}},
		{"1", ".", []*Match{{Subs: "1", Start: 0, End: 1}}},
		{"a", "%a", []*Match{{Subs: "a", Start: 0, End: 1}}},
		{"a", "%c", []*Match{}},
		{"a", "%d", []*Match{}},
		{"2", "%d", []*Match{{Subs: "2", Start: 0, End: 1}}},
		{"A", "%l", []*Match{}},
		{"a", "%l", []*Match{{Subs: "a", Start: 0, End: 1}}},
		{"a", "%p", []*Match{}},
		{"!", "%p", []*Match{{Subs: "!", Start: 0, End: 1}}},
		{"a", "%s", []*Match{}},
		{" ", "%s", []*Match{{Subs: " ", Start: 0, End: 1}}},
		{"a", "%u", []*Match{}},
		{"A", "%u", []*Match{{Subs: "A", Start: 0, End: 1}}},
		{"aaab", "a*", []*Match{{Subs: "aaa", Start: 0, End: 3}}},
		{"aaa", "^.*$", []*Match{{Subs: "aaa", Start: 0, End: 3}}},
		{"aaa", "b*", []*Match{{}}},
		{"aaa", "ab*a", []*Match{{Subs: "aa", Start: 0, End: 2}}},
		{"aba", "ab*a", []*Match{{Subs: "aba", Start: 0, End: 3}}},
		{"aaab", "a+", []*Match{{Subs: "aaa", Start: 0, End: 3}}},
		{"aaa", "^.+$", []*Match{{Subs: "aaa", Start: 0, End: 3}}},
		{"aaa", "b+", []*Match{}},
		{"aaa", "ab+a", []*Match{}},
		{"aba", "ab+a", []*Match{{Subs: "aba", Start: 0, End: 3}}},
		{"a$a", ".$", []*Match{{Subs: "a", Start: 2, End: 3}}},
		{"a$a", ".%$", []*Match{{Subs: "a$", Start: 0, End: 2}}},
		// {"a$a", ".$.", []*Match{{Subs: "a$a", Start: 0, End: 3}}},
		// {"a$a", "$$", []*Match{}},
		{"a$b", "a$", []*Match{}},
		{"a$a", "$", []*Match{{Start: 3, End: 3}}},
		{"", "b*", []*Match{{}}},
		{"aaa", "bb*", []*Match{}},
		{"aaab", "a-", []*Match{{}}},
		{"aaa", "^.-$", []*Match{{Subs: "aaa", Start: 0, End: 3}}},
		{"aabaaabaaabaaaba", "b.*b", []*Match{{Subs: "baaabaaabaaab", Start: 2, End: 15}}},
		{"aabaaabaaabaaaba", "b.-b", []*Match{{Subs: "baaab", Start: 2, End: 7}}},
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
		match, err := Find(test.pat, test.str, 0, 1)
		require.NoError(t, err)
		assert.Equal(t, test.results, match, "[%v]", i)
	}
}
