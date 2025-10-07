package parse

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type parseTokenTest struct {
	src   string
	token *token
	err   *string
}

const longstr = `return function(_params)
  assert(type(_params) == 'table', 'params to template render should be a table')
  for name, val in pairs(_params) do
    _ENV[name] = val
  end
  local _tmpl_output = ''
`

func TestNextToken(t *testing.T) {
	t.Parallel()
	linfo := LineInfo{Line: 1, Column: 1}
	tests := []parseTokenTest{
		{
			src: `--this is a comment
		`,
			token: &token{Kind: tokenComment, StringVal: "this is a comment", LineInfo: linfo},
		},
		{
			src: `--[[
		this is a comment]]`,
			token: &token{Kind: tokenComment, StringVal: "this is a comment", LineInfo: linfo},
		},
		{
			src: `--!this is a comment
		`,
			token: &token{Kind: tokenComment, StringVal: "!this is a comment", LineInfo: linfo},
		},
		{
			src:   `--[===[this is a comment]===]`,
			token: &token{Kind: tokenComment, StringVal: "this is a comment", LineInfo: linfo},
		},
		{
			src:   "[[this is a string]]",
			token: &token{Kind: tokenString, StringVal: "this is a string", LineInfo: linfo},
		},
		{
			src:   `"this is a \x01 string"`,
			token: &token{Kind: tokenString, StringVal: "this is a \x01 string", LineInfo: linfo},
		},
		{
			src:   `"this is a \x012 string"`,
			token: &token{Kind: tokenString, StringVal: "this is a \x012 string", LineInfo: linfo},
		},
		{
			src: `"this is a \x0 string"`,
			err: ptr("hexadecimal digit expected near"),
		},
		{
			src: `"this is a \x string"`,
			err: ptr("hexadecimal digit expected near"),
		},
		{
			src:   `"this is a \255 string"`,
			token: &token{Kind: tokenString, StringVal: "this is a ÿ string", LineInfo: linfo},
		},
		{
			src:   `"this is a \2555 string"`,
			token: &token{Kind: tokenString, StringVal: "this is a ÿ5 string", LineInfo: linfo},
		},
		{
			src:   `"this is \97 string"`,
			token: &token{Kind: tokenString, StringVal: "this is a string", LineInfo: linfo},
		},
		{
			src:   `"this is a \0 string"`,
			token: &token{Kind: tokenString, StringVal: "this is a \x00 string", LineInfo: linfo},
		},
		{
			src: `"this is a \s"`,
			err: ptr("unexpected escape"),
		},
		{
			src:   `"this is a \z       string"`,
			token: &token{Kind: tokenString, StringVal: "this is a string", LineInfo: linfo},
		},
		{
			src:   `"this is a \u{255} string"`,
			token: &token{Kind: tokenString, StringVal: "this is a \u0255 string", LineInfo: linfo},
		},
		{
			src:   "[=[[this is a string]]=]",
			token: &token{Kind: tokenString, StringVal: "[this is a string]", LineInfo: linfo},
		},
		{
			src:   "[[\n\n" + longstr + "]]",
			token: &token{Kind: tokenString, StringVal: longstr, LineInfo: linfo},
		},
		{
			src:   `'[%z\1-\31\\"]'`,
			token: &token{Kind: tokenString, StringVal: "[%z\x01-\x1f\\\"]", LineInfo: linfo},
		},
		{
			src:   "\"this is a string\"",
			token: &token{Kind: tokenString, StringVal: "this is a string", LineInfo: linfo},
		},
		{
			src:   "'this is a string'",
			token: &token{Kind: tokenString, StringVal: "this is a string", LineInfo: linfo},
		},
		{
			src:   "22",
			token: &token{Kind: tokenInteger, IntVal: 22, LineInfo: linfo},
		},
		{
			src:   "23.43",
			token: &token{Kind: tokenFloat, FloatVal: 23.43, LineInfo: linfo},
		},
		{
			src:   "23.43e-12",
			token: &token{Kind: tokenFloat, FloatVal: 23.43e-12, LineInfo: linfo},
		},
		{
			src:   "23.43e5",
			token: &token{Kind: tokenFloat, FloatVal: 23.43e5, LineInfo: linfo},
		},
		{
			src:   "0xAF2",
			token: &token{Kind: tokenInteger, IntVal: 2802, LineInfo: linfo},
		},
		{
			src:   "0xAF2p2",
			token: &token{Kind: tokenFloat, FloatVal: 11208, LineInfo: linfo},
		},
		{
			src:   "0xAF2p-12",
			token: &token{Kind: tokenFloat, FloatVal: 0.68408203125, LineInfo: linfo},
		},
		{
			src:   "foobar",
			token: &token{Kind: tokenIdentifier, StringVal: "foobar", LineInfo: linfo},
		},
		{
			src:   "foobar42",
			token: &token{Kind: tokenIdentifier, StringVal: "foobar42", LineInfo: linfo},
		},
		{
			src:   "::foobar::",
			token: &token{Kind: tokenLabel, StringVal: "foobar", LineInfo: linfo},
		},
		{
			src: "::foobar",
			err: ptr("unexpected character while parsing label"),
		},
		{
			src:   "_foo_bar42",
			token: &token{Kind: tokenIdentifier, StringVal: "_foo_bar42", LineInfo: linfo},
		},
		{
			src:   "0x0.1",
			token: &token{Kind: tokenFloat, FloatVal: 0.0625, LineInfo: linfo},
		},
		{
			src: "0xG",
			err: ptr("invalid syntax"),
		},
		{
			src:   "0x4.1e2p3",
			token: &token{Kind: tokenFloat, FloatVal: 32.94140625, LineInfo: linfo},
		},
		{
			src:   "0x1.13aP3",
			token: &token{Kind: tokenFloat, FloatVal: 8.61328125, LineInfo: linfo},
		},
		{
			src:   "2.E-1",
			token: &token{Kind: tokenFloat, FloatVal: 0.2, LineInfo: linfo},
		},
		{
			src:   "2.E+1",
			token: &token{Kind: tokenFloat, FloatVal: 20, LineInfo: linfo},
		},
		{
			src:   "08",
			token: &token{Kind: tokenInteger, IntVal: 8, LineInfo: linfo},
		},
		{
			src:   "0",
			token: &token{Kind: tokenInteger, IntVal: 0, LineInfo: linfo},
		},
		{
			src:   ".0",
			token: &token{Kind: tokenFloat, FloatVal: 0, LineInfo: linfo},
		},
	}

	operators := []tokenType{
		tokenEq, tokenLe, tokenShiftLeft, tokenGe, tokenShiftRight, tokenNe, tokenFloorDivide,
		tokenDots,
	}

	linfo = LineInfo{Line: 1, Column: 0}
	for _, op := range operators {
		tests = append(tests, parseTokenTest{src: string(op), token: &token{Kind: op, LineInfo: linfo}})
	}

	for key, kw := range keywords {
		tests = append(tests, parseTokenTest{src: key, token: &token{Kind: kw, LineInfo: linfo}})
	}

	for _, test := range tests {
		out, err := lex(test.src)
		if test.err != nil {
			require.ErrorContains(t, err, *test.err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.token, out)
		}
	}
}

func TestLexSource(t *testing.T) {
	t.Parallel()
	luaSource := `
require('lib')

function foo:bar(self, name)
	self:println(name)
end

foo:bar("tim")
`

	lexer := newLexer("test", bytes.NewBufferString(luaSource))
	tokens := []*token{}
	var tk *token
	var err error
	for {
		tk, err = lexer.Next()
		if err != nil {
			break
		}
		tokens = append(tokens, tk)
	}
	assert.Len(t, tokens, 26)
	assert.Equal(t, io.EOF, err)
}

func TestLexPeek(t *testing.T) {
	t.Parallel()
	luaSource := `local a = 1`
	lexer := newLexer("test", bytes.NewBufferString(luaSource))
	tk, err := lexer.Peek()
	require.NoError(t, err)
	assert.Equal(t, tokenLocal, tk.Kind)
	tk, err = lexer.Peek()
	require.NoError(t, err)
	assert.Equal(t, tokenLocal, tk.Kind)
	tk, err = lexer.Next()
	require.NoError(t, err)
	assert.Equal(t, tokenLocal, tk.Kind)

	tk, err = lexer.Next()
	require.NoError(t, err)
	assert.Equal(t, tokenIdentifier, tk.Kind)

	tk, err = lexer.Next()
	require.NoError(t, err)
	assert.Equal(t, tokenAssign, tk.Kind)

	tk, err = lexer.Next()
	require.NoError(t, err)
	assert.Equal(t, tokenInteger, tk.Kind)

	tk, err = lexer.Peek()
	require.NoError(t, err)
	assert.Equal(t, tokenEOS, tk.Kind)
}

func lex(str string) (*token, error) {
	return newLexer("test", bytes.NewBufferString(str)).Next()
}

func ptr[T any](value T) *T {
	return &value
}
