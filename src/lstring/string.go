// Package lstring is a small collection of string utilities.
package lstring

import (
	"strings"
)

// Substring will get the substring of a string with a start and end index.
// Indexes can be negative, and if they are they will be subtracted from the length.
func Substring(str string, start, end int64) string {
	i := substringIndex(start, int64(len(str)))
	if i > int64(len(str)) || i < 0 {
		return ""
	}

	j := substringIndex(end, int64(len(str)))
	if j < i {
		return ""
	}
	return str[i-1 : clamp(j, i, int64(len(str)))]
}

// Reverse will reverse the order of the string.
func Reverse(str string) string {
	rstr := []rune(str)
	for i, j := 0, len(str)-1; i < j; i, j = i+1, j-1 {
		rstr[i], rstr[j] = rstr[j], rstr[i]
	}
	return string(rstr)
}

// Repeat will repeat a string *count* number of times and join them with the
// provided separator.
func Repeat(str, sep string, count int64) string {
	parts := make([]string, count)
	for i := range count {
		parts[i] = str
	}
	return strings.Join(parts, sep)
}

func substringIndex(i, strLen int64) int64 {
	if i < 0 {
		return strLen + i
	} else if i == 0 {
		return 1
	}
	return i
}

func clamp(f, low, high int64) int64 {
	return min(max(f, low), high)
}
