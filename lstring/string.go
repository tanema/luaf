package lstring

import (
	"strings"
)

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

func Reverse(str string) string {
	rstr := []rune(str)
	for i, j := 0, len(str)-1; i < j; i, j = i+1, j-1 {
		rstr[i], rstr[j] = rstr[j], rstr[i]
	}
	return string(rstr)
}

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
