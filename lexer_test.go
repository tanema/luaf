package luaf

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

type parseTokenTest struct {
	src   string
	token *Token
}

const longstr = `return function(_params)
  assert(type(_params) == 'table', 'params to template render should be a table')
  for name, val in pairs(_params) do
    _ENV[name] = val
  end
  local _tmpl_output = ''
`

func TestNextToken(t *testing.T) {
	linfo := LineInfo{Line: 1, Column: 1}
	tests := []parseTokenTest{
		{`--this is a comment
			`, &Token{Kind: TokenComment, StringVal: "this is a comment", LineInfo: linfo}},
		{`--[===[this is a comment]===]`, &Token{Kind: TokenComment, StringVal: "this is a comment", LineInfo: linfo}},
		{"[[this is a string]]", &Token{Kind: TokenString, StringVal: "this is a string", LineInfo: linfo}},
		{"[=[[this is a string]]=]", &Token{Kind: TokenString, StringVal: "[this is a string]", LineInfo: linfo}},
		{"[[\n\n" + longstr + "]]", &Token{Kind: TokenString, StringVal: longstr, LineInfo: linfo}},
		{`'[%z\1-\31\\"]'`, &Token{Kind: TokenString, StringVal: `[%z\1-\31\"]`, LineInfo: linfo}},
		{"\"this is a string\"", &Token{Kind: TokenString, StringVal: "this is a string", LineInfo: linfo}},
		{"'this is a string'", &Token{Kind: TokenString, StringVal: "this is a string", LineInfo: linfo}},
		{"22", &Token{Kind: TokenInteger, IntVal: 22, LineInfo: linfo}},
		{"23.43", &Token{Kind: TokenFloat, FloatVal: 23.43, LineInfo: linfo}},
		{"23.43e-12", &Token{Kind: TokenFloat, FloatVal: 23.43e-12, LineInfo: linfo}},
		{"23.43e5", &Token{Kind: TokenFloat, FloatVal: 23.43e5, LineInfo: linfo}},
		{"0xAF2", &Token{Kind: TokenInteger, IntVal: 2802, LineInfo: linfo}},
		{"0xAF2p2", &Token{Kind: TokenFloat, FloatVal: 11208, LineInfo: linfo}},
		{"0xAF2p-12", &Token{Kind: TokenFloat, FloatVal: 0.68408203125, LineInfo: linfo}},
		{"foobar", &Token{Kind: TokenIdentifier, StringVal: "foobar", LineInfo: linfo}},
		{"foobar42", &Token{Kind: TokenIdentifier, StringVal: "foobar42", LineInfo: linfo}},
		{"_foo_bar42", &Token{Kind: TokenIdentifier, StringVal: "_foo_bar42", LineInfo: linfo}},
		{"0x0.1", &Token{Kind: TokenFloat, FloatVal: 0.0625, LineInfo: linfo}},
		{"0x4.1e2p3", &Token{Kind: TokenFloat, FloatVal: 32.94140625, LineInfo: linfo}},
		{"0x1.13aP3", &Token{Kind: TokenFloat, FloatVal: 8.61328125, LineInfo: linfo}},
		{"2.E-1", &Token{Kind: TokenFloat, FloatVal: 0.2, LineInfo: linfo}},
		{"2.E+1", &Token{Kind: TokenFloat, FloatVal: 20, LineInfo: linfo}},
		{"08", &Token{Kind: TokenInteger, IntVal: 8, LineInfo: linfo}},
		{"0", &Token{Kind: TokenInteger, IntVal: 0, LineInfo: linfo}},
		{".0", &Token{Kind: TokenFloat, FloatVal: 0, LineInfo: linfo}},
	}

	operators := []TokenType{
		TokenEq, TokenLe, TokenShiftLeft, TokenGe, TokenShiftRight, TokenNe, TokenFloorDivide,
		TokenDots, TokenDoubleColon, TokenLength,
	}

	linfo = LineInfo{Line: 1, Column: 0}
	for _, op := range operators {
		tests = append(tests, parseTokenTest{string(op), &Token{Kind: op, LineInfo: linfo}})
	}

	for key, kw := range keywords {
		tests = append(tests, parseTokenTest{key, &Token{Kind: kw, LineInfo: linfo}})
	}

	for _, test := range tests {
		out, err := lex(test.src)
		if !assert.Nil(t, err) {
			fmt.Println(err.Error())
		}
		assert.Equal(t, test.token, out)
	}
}

func TestLexSource(t *testing.T) {
	luaSource := `
require('lib')

function foo:bar(self, name)
	self:println(name)
end

foo:bar("tim")
`

	lexer := NewLexer(bytes.NewBufferString(luaSource))
	tokens := []*Token{}
	var tk *Token
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
	luaSource := `local a = 1`
	lexer := NewLexer(bytes.NewBufferString(luaSource))
	tk := lexer.Peek()
	assert.Equal(t, TokenLocal, tk.Kind)
	tk = lexer.Peek()
	assert.Equal(t, TokenLocal, tk.Kind)
	tk, err := lexer.Next()
	assert.Nil(t, err)
	assert.Equal(t, TokenLocal, tk.Kind)

	tk, err = lexer.Next()
	assert.Nil(t, err)
	assert.Equal(t, TokenIdentifier, tk.Kind)

	tk, err = lexer.Next()
	assert.Nil(t, err)
	assert.Equal(t, TokenAssign, tk.Kind)

	tk, err = lexer.Next()
	assert.Nil(t, err)
	assert.Equal(t, TokenInteger, tk.Kind)

	assert.Equal(t, TokenEOS, lexer.Peek().Kind)
}

func lex(str string) (*Token, error) {
	return NewLexer(bytes.NewBufferString(str)).Next()
}
