package pattern

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFind(t *testing.T) {
	t.Parallel()
	matchTests := []struct {
		str, pat string
		results  []*Match
	}{
		{"Apple", "A*pple", []*Match{{Subs: "Apple", Start: 0, End: 5}}},
		{"Apple", "apple", []*Match{}},
		{"Apple", "(Ap)ple", []*Match{
			{Subs: "Apple", Start: 0, End: 5},
			{Subs: "Ap", Start: 0, End: 2},
		}},
		{"Apple", "(Ap)p(le)", []*Match{
			{Subs: "Apple", Start: 0, End: 5},
			{Subs: "Ap", Start: 0, End: 2},
			{Subs: "le", Start: 3, End: 5},
		}},
		{"Apple", "A(pp)(le)", []*Match{
			{Subs: "Apple", Start: 0, End: 5},
			{Subs: "pp", Start: 1, End: 3},
			{Subs: "le", Start: 3, End: 5},
		}},
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
		{"aaa", "b*", []*Match{}},
		{"aaa", "ab*a", []*Match{{Subs: "aa", Start: 0, End: 2}}},
		{"aba", "ab*a", []*Match{{Subs: "aba", Start: 0, End: 3}}},
		{"aaab", "a+", []*Match{{Subs: "aaa", Start: 0, End: 3}}},
		{"aaa", "^.+$", []*Match{{Subs: "aaa", Start: 0, End: 3}}},
		{"aaa", "b+", []*Match{}},
		{"aaa", "ab+a", []*Match{}},
		{"aba", "ab+a", []*Match{{Subs: "aba", Start: 0, End: 3}}},
		{"a$a", ".$", []*Match{{Subs: "a", Start: 2, End: 3}}},
		{"a$a", ".%$", []*Match{{Subs: "a$", Start: 0, End: 2}}},
		{"a$a", ".$.", []*Match{{Subs: "a$a", Start: 0, End: 3}}},
		{"a$a", "$$", []*Match{}},
		{"a$b", "a$", []*Match{}},
		{"a$a", "$", []*Match{}},
		{"", "b*", []*Match{}},
		{"aaa", "bb*", []*Match{}},
		{"aaab", "a-", []*Match{}},
		{"aaa", "^.-$", []*Match{{Subs: "aaa", Start: 0, End: 3}}},
		{"aabaaabaaabaaaba", "b.*b", []*Match{{Subs: "baaabaaabaaab", Start: 2, End: 15}}},
		{"aabaaabaaabaaaba", "b.-b", []*Match{{Subs: "baaab", Start: 2, End: 7}}},
		{"alo xo", ".o$", []*Match{{Subs: "xo", Start: 4, End: 6}}},
		{" \n isto \x82 assim", "%S%S*", []*Match{{Subs: "isto", Start: 3, End: 7}}},
		{" \n isto \x82 assim", "%S*$", []*Match{{Subs: "assim", Start: 10, End: 15}}},
		{" \n isto \x82 assim", "[a-z]*$", []*Match{{Subs: "assim", Start: 10, End: 15}}},
		{"im caracter ? extra", "[^%sa-z]", []*Match{{Subs: "?", Start: 12, End: 13}}},
		{"", "a?", []*Match{}},
		{"bab", "ba?b", []*Match{{Subs: "bab", Start: 0, End: 3}}},
		{"bb", "ba?b", []*Match{{Subs: "bb", Start: 0, End: 2}}},
		{"baaab", "ba?b", []*Match{}},
		{"\225", "\225?", []*Match{{Subs: string([]rune("\225")), Start: 0, End: 1}}},
		{"\225bl", "\225?b?l?", []*Match{{Subs: string([]rune("\225bl")), Start: 0, End: 3}}},
		{"  abl", "a?b?l?", []*Match{{Subs: "abl", Start: 2, End: 5}}},
		{"aa", "^aa?a?a", []*Match{{Subs: "aa", Start: 0, End: 2}}},
		{"]]]bc", "[^]]", []*Match{{Subs: "b", Start: 3, End: 4}}},
		{"dead beef", "%x*", []*Match{{Subs: "dead", Start: 0, End: 4}}},
		{"x=x", "(.)=%1", []*Match{{Subs: "x=x", Start: 0, End: 3}, {Subs: "x", Start: 0, End: 1}}},
		{"hello world from Lua", "%a+", []*Match{{Subs: "hello", Start: 0, End: 5}}},
		{"=", "^[=-]", []*Match{{Subs: "=", Start: 0, End: 1}}},
		// {"  \n\r*&\n\r   xuxu  \n\n", "%g%g%g+", []*Match{{Subs: "xuxu", Start: 11, End: 15}}},
		// {" \n isto é assim", "%S*$", []*Match{{Subs: "assim", Start: 11, End: 16}}},
		// {" \n isto é assim", "[a-z]*$", []*Match{{Subs: "assim", Start: 11, End: 16}}},
	}

	for i, test := range matchTests {
		pat, err := Parse(test.pat)
		require.NoError(t, err)
		match, err := pat.Find(test.str, 1)
		require.NoError(t, err)
		assert.Equal(t, test.results, match, "[%v]", i)
	}
}

func TestFindAll(t *testing.T) {
	t.Parallel()
	pat, err := Parse("%a+")
	require.NoError(t, err)
	matches, err := pat.Find("hello world from Lua", -1)
	require.NoError(t, err)
	assert.Equal(t, []*Match{
		{Subs: "hello", Start: 0, End: 5},
		{Subs: "world", Start: 6, End: 11},
		{Subs: "from", Start: 12, End: 16},
		{Subs: "Lua", Start: 17, End: 20},
	}, matches)
}

func TestPatternIterator(t *testing.T) {
	t.Parallel()
	pat, err := Parse("%a+")
	require.NoError(t, err)
	iter := pat.Iter("hello world from Lua")
	match, err := iter.Next()
	require.NoError(t, err)
	assert.Equal(t, []*Match{{Subs: "hello", Start: 0, End: 5}}, match)
	match, err = iter.Next()
	require.NoError(t, err)
	assert.Equal(t, []*Match{{Subs: "world", Start: 6, End: 11}}, match)
	match, err = iter.Next()
	require.NoError(t, err)
	assert.Equal(t, []*Match{{Subs: "from", Start: 12, End: 16}}, match)
	match, err = iter.Next()
	require.NoError(t, err)
	assert.Equal(t, []*Match{{Subs: "Lua", Start: 17, End: 20}}, match)
	match, err = iter.Next()
	require.NoError(t, err)
	assert.Nil(t, match)
}
