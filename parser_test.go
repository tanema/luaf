package luaf

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalAssign(t *testing.T) {
	p, fn := parser(`a, b, c = 1, true, "hello"`)
	require.NoError(t, p.localassign(fn))
	require.Len(t, fn.Locals, 3)
	assert.Equal(t, []string{"a", "b", "c"}, fn.Locals)
	require.Len(t, fn.Constants, 2)
	assert.Equal(t, []any{int64(1), "hello"}, fn.Constants)
	require.Len(t, fn.ByteCodes, 3)
	assert.Equal(t, fn.ByteCodes[0], iABx(LOADK, 0, 0))
	assert.Equal(t, fn.ByteCodes[1], iAB(LOADBOOL, 1, 1))
	assert.Equal(t, fn.ByteCodes[2], iABx(LOADK, 2, 1))
}

func TestAssign(t *testing.T) {
	p, fn := parser(`a, b, c = 1, true, "hello"`)
	require.NoError(t, p.stat(fn))
	require.Len(t, fn.Locals, 0)
	require.Len(t, fn.Constants, 5)
	assert.Equal(t, []any{"a", "b", "c", int64(1), "hello"}, fn.Constants)
	require.Len(t, fn.ByteCodes, 6)
	assert.Equal(t, fn.ByteCodes[0], iABx(LOADK, 0, 3))
	assert.Equal(t, fn.ByteCodes[1], iAB(LOADBOOL, 1, 1))
	assert.Equal(t, fn.ByteCodes[2], iABx(LOADK, 2, 4))
	assert.Equal(t, fn.ByteCodes[3], iABCK(SETTABUP, 0, 0, true, 0, false))
	assert.Equal(t, fn.ByteCodes[4], iABCK(SETTABUP, 0, 1, true, 1, false))
	assert.Equal(t, fn.ByteCodes[5], iABCK(SETTABUP, 0, 2, true, 2, false))
}

func parser(src string) (*Parser, *FuncProto) {
	p := &Parser{
		rootfn: newFnProto(nil, []string{"_ENV"}, false),
		lex:    NewLexer(bytes.NewBufferString(src)),
	}
	return p, newFnProto(p.rootfn, []string{}, false)
}
