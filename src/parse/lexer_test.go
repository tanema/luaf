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
		{`--this is a comment
			`, &token{Kind: tokenComment, StringVal: "this is a comment", LineInfo: linfo}},
		{`--!this is a comment
			`, &token{Kind: tokenComment, StringVal: "!this is a comment", LineInfo: linfo}},
		{`--[===[this is a comment]===]`, &token{Kind: tokenComment, StringVal: "this is a comment", LineInfo: linfo}},
		{"[[this is a string]]", &token{Kind: tokenString, StringVal: "this is a string", LineInfo: linfo}},
		{"[=[[this is a string]]=]", &token{Kind: tokenString, StringVal: "[this is a string]", LineInfo: linfo}},
		{"[[\n\n" + longstr + "]]", &token{Kind: tokenString, StringVal: longstr, LineInfo: linfo}},
		{`'[%z\1-\31\\"]'`, &token{Kind: tokenString, StringVal: `[%z\1-\31\"]`, LineInfo: linfo}},
		{"\"this is a string\"", &token{Kind: tokenString, StringVal: "this is a string", LineInfo: linfo}},
		{"'this is a string'", &token{Kind: tokenString, StringVal: "this is a string", LineInfo: linfo}},
		{"22", &token{Kind: tokenInteger, IntVal: 22, LineInfo: linfo}},
		{"23.43", &token{Kind: tokenFloat, FloatVal: 23.43, LineInfo: linfo}},
		{"23.43e-12", &token{Kind: tokenFloat, FloatVal: 23.43e-12, LineInfo: linfo}},
		{"23.43e5", &token{Kind: tokenFloat, FloatVal: 23.43e5, LineInfo: linfo}},
		{"0xAF2", &token{Kind: tokenInteger, IntVal: 2802, LineInfo: linfo}},
		{"0xAF2p2", &token{Kind: tokenFloat, FloatVal: 11208, LineInfo: linfo}},
		{"0xAF2p-12", &token{Kind: tokenFloat, FloatVal: 0.68408203125, LineInfo: linfo}},
		{"foobar", &token{Kind: tokenIdentifier, StringVal: "foobar", LineInfo: linfo}},
		{"foobar42", &token{Kind: tokenIdentifier, StringVal: "foobar42", LineInfo: linfo}},
		{"_foo_bar42", &token{Kind: tokenIdentifier, StringVal: "_foo_bar42", LineInfo: linfo}},
		{"0x0.1", &token{Kind: tokenFloat, FloatVal: 0.0625, LineInfo: linfo}},
		{"0x4.1e2p3", &token{Kind: tokenFloat, FloatVal: 32.94140625, LineInfo: linfo}},
		{"0x1.13aP3", &token{Kind: tokenFloat, FloatVal: 8.61328125, LineInfo: linfo}},
		{"2.E-1", &token{Kind: tokenFloat, FloatVal: 0.2, LineInfo: linfo}},
		{"2.E+1", &token{Kind: tokenFloat, FloatVal: 20, LineInfo: linfo}},
		{"08", &token{Kind: tokenInteger, IntVal: 8, LineInfo: linfo}},
		{"0", &token{Kind: tokenInteger, IntVal: 0, LineInfo: linfo}},
		{".0", &token{Kind: tokenFloat, FloatVal: 0, LineInfo: linfo}},
	}

	operators := []tokenType{
		tokenEq, tokenLe, tokenShiftLeft, tokenGe, tokenShiftRight, tokenNe, tokenFloorDivide,
		tokenDots, tokenDoubleColon,
	}

	linfo = LineInfo{Line: 1, Column: 0}
	for _, op := range operators {
		tests = append(tests, parseTokenTest{string(op), &token{Kind: op, LineInfo: linfo}})
	}

	for key, kw := range keywords {
		tests = append(tests, parseTokenTest{key, &token{Kind: kw, LineInfo: linfo}})
	}

	for _, test := range tests {
		out, err := lex(test.src)
		require.NoError(t, err)
		assert.Equal(t, test.token, out)
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

	lexer := newLexer(bytes.NewBufferString(luaSource))
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
	lexer := newLexer(bytes.NewBufferString(luaSource))
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
	return newLexer(bytes.NewBufferString(str)).Next()
}
