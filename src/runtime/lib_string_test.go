package runtime

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatString(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		pattern string
		val     any
		output  string
	}{
		{pattern: "%%", val: int64(42), output: "%"},
		{pattern: "%d", val: int64(42), output: "42"},
		{pattern: "%i", val: int64(42), output: "42"},
		{pattern: "%u", val: int64(42), output: "42"},
		{pattern: "%o", val: int64(42), output: "52"},
		{pattern: "%x", val: int64(42), output: "2a"},
		{pattern: "%X", val: int64(42), output: "2A"},
		{pattern: "%c", val: int64(42), output: "*"},
		{pattern: "%f", val: float64(42), output: "42"},
		{pattern: "%e", val: float64(42), output: "4.200000e+01"},
		{pattern: "%E", val: float64(42), output: "4.200000E+01"},
		{pattern: "%g", val: float64(42), output: "42"},
		{pattern: "%G", val: float64(42), output: "42"},
		{pattern: "%a", val: float64(42), output: "0x1.5p+05"},
		{pattern: "%A", val: float64(42), output: "0X1.5P+05"},
		{pattern: "%s", val: "test this", output: "test this"},
		{pattern: "%02x", val: float64(0.0), output: "00"},
		{pattern: "%08X", val: float64(0xFFFFFFFF), output: "FFFFFFFF"},
		{pattern: "%+08d", val: int64(31501), output: "+0031501"},
		{pattern: "%u", val: int64(0xffffffff), output: "4294967295"},
		{pattern: "%o", val: int64(0xABCD), output: "125715"},
		{pattern: "0x%8X", val: int64(0x8f000003), output: "0x8F000003"},
		{pattern: "%d", val: int64(9007199254740992), output: "9007199254740992"},
		{pattern: "%i", val: int64(-9007199254740992), output: "-9007199254740992"},
		{pattern: "%#12o", val: int64(10), output: "         012"},
		{pattern: "%#10x", val: int64(10), output: "       0xa"},
		{pattern: "%#-17X", val: int64(100), output: "0X64             "},
		{pattern: "%013i", val: int64(-100), output: "-000000000100"},
		{pattern: "%2.5d", val: int64(-100), output: "-00100"},
		{pattern: "%.u", val: int64(0), output: "0"},
		{pattern: "%+#014.0f", val: int64(100), output: "+000000000100."},
		{pattern: "%-16c", val: int64(97), output: "a               "},
		{pattern: "%+.3G", val: float64(1.5), output: "+1.5"},
		{pattern: "%.0s", val: "alo", output: ""},
		{pattern: "%.s", val: "alo", output: ""},
		{pattern: "% 1.0E", val: float64(100), output: " 1E+02"},
		{pattern: "% .1g", val: float64(1024), output: " 1e+03"},
		{pattern: "%q", val: nil, output: "nil"},
		{pattern: "%s", val: nil, output: "nil"},
	}

	vm := New(context.Background(), nil)
	for _, tc := range testcases {
		out, err := formatString(vm, tc.pattern, tc.val)
		require.NoError(t, err)
		assert.Equal(t, tc.output, out, tc.pattern)
	}
}

func TestSubstring(t *testing.T) {
	t.Parallel()

	l := int64(len("123456789"))
	testcases := []struct {
		input      string
		start, end int64
		expected   string
	}{
		{"123456789", 2, 4, "234"},
		{"123456789", 7, l, "789"},
		{"123456789", 7, 6, ""},
		{"123456789", 7, 7, "7"},
		{"123456789", 0, 0, ""},
		{"123456789", -10, 10, "123456789"},
		{"123456789", 1, 9, "123456789"},
		{"123456789", -10, -20, ""},
		{"123456789", -1, l, "9"},
		{"123456789", -4, l, "6789"},
		{"123456789", -6, -4, "456"},
		{"\000123456789", 3, 5, "234"},
		{"\000123456789", 8, int64(len("\000123456789")), "789"},
		{" \n isto e assim", 12, 16, "assim"},
		{" \n isto é assim", 12, 16, "assim"},
	}

	for _, tc := range testcases {
		out := substring(tc.input, tc.start, tc.end)
		assert.Equal(t, tc.expected, out)
	}
}
