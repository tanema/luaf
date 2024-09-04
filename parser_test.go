package shine

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseNameList(t *testing.T) {
	testCases := []struct {
		input string
		names []string
		err   error
	}{
		{"a, b, c", []string{"a", "b", "c"}, nil},
		{"a,b,c =", []string{"a", "b", "c"}, nil},
		{"a,b,c", []string{"a", "b", "c"}, nil},
		{"a", []string{"a"}, nil},
		{"a =", []string{"a"}, nil},
		{"a23", []string{"a23"}, nil},
		{"a, 23 =", nil, errors.New("expected Name but found integer")},
	}

	for _, testCase := range testCases {
		lex := NewLexer(bytes.NewBufferString(testCase.input))
		names, err := parseNameList(lex)
		assert.Equal(t, testCase.names, names)
		assert.Equal(t, testCase.err, err)
	}
}
