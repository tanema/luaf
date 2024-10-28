package luaf

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLocalAssign(t *testing.T) {
	p, fn := parser(`a, b, c = 1, true, "hello"`)
	require.NoError(t, p.localassign(fn))
}

func parser(src string) (*Parser, *FuncProto) {
	p := &Parser{
		rootfn: newFnProto(nil, []string{"_ENV"}, false),
		lex:    NewLexer(bytes.NewBufferString(src)),
	}
	return p, newFnProto(p.rootfn, []string{}, false)
}

func parse(src string) (*FuncProto, error) {
	return Parse("test_run", bytes.NewBufferString(src))
}

func mustparse(t *testing.T, src string) *FuncProto {
	fn, err := Parse("test_run", bytes.NewBufferString(src))
	require.NoError(t, err)
	return fn
}
