package luaf

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParser_LocalAssign(t *testing.T) {
	t.Run("multiple assignment", func(t *testing.T) {
		p, fn := parser(`local a, b, c = 1, true, "hello"`)
		require.NoError(t, p.stat(fn))
		require.Len(t, fn.Locals, 3)
		assert.Equal(t, []*Local{{name: "a"}, {name: "b"}, {name: "c"}}, fn.Locals)
		require.Len(t, fn.Constants, 2)
		assert.Equal(t, []any{int64(1), "hello"}, fn.Constants)
		require.Len(t, fn.ByteCodes, 3)
		assert.Equal(t, []Bytecode{
			iABx(LOADK, 0, 0),
			iAB(LOADBOOL, 1, 1),
			iABx(LOADK, 2, 1),
		}, fn.ByteCodes)
		assert.Equal(t, fn.stackPointer, uint8(3))
	})

	t.Run("function assignment", func(t *testing.T) {
		p, fn := parser(`
local hello = "hello world"
local function testFn(a, b, ...)
	print(hello)
end
testFn()
`)
		require.NoError(t, p.statList(fn))
		require.Len(t, fn.Locals, 2)
		assert.Equal(t, []*Local{{name: "hello", upvalRef: true}, {name: "testFn"}}, fn.Locals)
		require.Len(t, fn.Constants, 1)
		assert.Equal(t, []any{"hello world"}, fn.Constants)
		require.Len(t, fn.ByteCodes, 4)
		require.Len(t, fn.FnTable, 1)
		assert.Equal(t, []Bytecode{
			iABx(LOADK, 0, 0),
			iABx(CLOSURE, 1, 0),
			iABC(MOVE, 2, 1, 0),
			iABC(CALL, 2, 1, 0),
		}, fn.ByteCodes)
		assert.Equal(t, fn.stackPointer, uint8(2))

		testFn := fn.FnTable[0]
		assert.Equal(t, 2, testFn.Arity)
		assert.True(t, testFn.Varargs)
		require.Len(t, testFn.Constants, 1)
		assert.Equal(t, []any{"print"}, testFn.Constants)
		require.Len(t, testFn.Locals, 2)
		require.Len(t, testFn.UpIndexes, 2)
		assert.Equal(t, []UpIndex{
			{fromStack: false, name: "_ENV", index: 0},
			{fromStack: true, name: "hello", index: 0},
		}, testFn.UpIndexes)
		assert.Equal(t, []*Local{{name: "a"}, {name: "b"}}, testFn.Locals)
		require.Len(t, testFn.ByteCodes, 3)
		assert.Equal(t, []Bytecode{
			iABCK(GETTABUP, 2, 0, false, 0, true),
			iABC(GETUPVAL, 3, 1, 0),
			iABC(CALL, 2, 2, 0),
		}, testFn.ByteCodes)
		assert.Equal(t, testFn.stackPointer, uint8(2))
	})

	t.Run("assignment attributes", func(t *testing.T) {
		p, fn := parser(`local a <const> = 42`)
		require.NoError(t, p.stat(fn))
		require.Len(t, fn.Locals, 1)
		assert.Equal(t, []*Local{{name: "a"}}, fn.Locals)
		require.Len(t, fn.Constants, 1)
		assert.Equal(t, []any{int64(42)}, fn.Constants)
		require.Len(t, fn.ByteCodes, 1)
		assert.Equal(t, []Bytecode{iABx(LOADK, 0, 0)}, fn.ByteCodes)
		assert.Equal(t, fn.stackPointer, uint8(1))
	})
}

func TestParser_Assign(t *testing.T) {
	t.Run("multiple assignment", func(t *testing.T) {
		p, fn := parser(`a, b, c = 1, true, "hello"`)
		require.NoError(t, p.stat(fn))
		require.Len(t, fn.Locals, 0)
		require.Len(t, fn.Constants, 5)
		assert.Equal(t, []any{"a", "b", "c", int64(1), "hello"}, fn.Constants)
		require.Len(t, fn.ByteCodes, 6)
		assert.Equal(t, []Bytecode{
			iABx(LOADK, 0, 3),
			iAB(LOADBOOL, 1, 1),
			iABx(LOADK, 2, 4),
			iABCK(SETTABUP, 0, 0, true, 0, false),
			iABCK(SETTABUP, 0, 1, true, 1, false),
			iABCK(SETTABUP, 0, 2, true, 2, false),
		}, fn.ByteCodes)
		assert.Equal(t, fn.stackPointer, uint8(3))
	})
}

func TestParser_FuncStat(t *testing.T) {
	p, fn := parser(`
local hello = "hello world"
function testFn()
	print(hello)
end
testFn()
`)
	require.NoError(t, p.statList(fn))
	require.Len(t, fn.Locals, 1)
	assert.Equal(t, []*Local{{name: "hello", upvalRef: true}}, fn.Locals)
	require.Len(t, fn.Constants, 2)
	assert.Equal(t, []any{"hello world", "testFn"}, fn.Constants)
	require.Len(t, fn.ByteCodes, 5)
	assert.Equal(t, []Bytecode{
		iABx(LOADK, 0, 0),
		iABx(CLOSURE, 1, 0),
		iABCK(SETTABUP, 0, 1, true, 1, false),
		iABCK(GETTABUP, 1, 0, false, 1, true),
		iABC(CALL, 1, 1, 0),
	}, fn.ByteCodes)
	assert.Equal(t, fn.stackPointer, uint8(1))
}

func TestParser_ReturnStat(t *testing.T) {
	p, fn := parser(`return a, 42, ...`)
	require.NoError(t, p.stat(fn))
	require.Len(t, fn.Locals, 0)
	require.Len(t, fn.Constants, 2)
	assert.Equal(t, []any{"a", int64(42)}, fn.Constants)
	require.Len(t, fn.ByteCodes, 4)
	assert.Equal(t, []Bytecode{
		iABCK(GETTABUP, 0, 0, false, 0, true),
		iABx(LOADK, 1, 1),
		iAB(VARARG, 2, 0),
		iABC(RETURN, 0, 0, 0),
	}, fn.ByteCodes)
	assert.Equal(t, uint8(3), fn.stackPointer)
}

func TestParser_RepeatStat(t *testing.T) {
	p, fn := parser(`repeat until true`)
	require.NoError(t, p.stat(fn))
	require.Len(t, fn.Locals, 0)
	require.Len(t, fn.Constants, 0)
	require.Len(t, fn.ByteCodes, 3)
	assert.Equal(t, []Bytecode{
		iAB(LOADBOOL, 0, 1),
		iABC(TEST, 0, 0, 0),
		iAsBx(JMP, 1, -2),
	}, fn.ByteCodes)
	assert.Equal(t, uint8(0), fn.stackPointer)
}

func TestParser_WhileStat(t *testing.T) {
	p, fn := parser(`while true do end`)
	require.NoError(t, p.stat(fn))
	require.Len(t, fn.Locals, 0)
	require.Len(t, fn.Constants, 0)
	require.Len(t, fn.ByteCodes, 4)
	assert.Equal(t, []Bytecode{
		iAB(LOADBOOL, 0, 1),
		iABC(TEST, 0, 0, 0),
		iAsBx(JMP, 1, 1),
		iAsBx(JMP, 1, -4),
	}, fn.ByteCodes)
	assert.Equal(t, uint8(0), fn.stackPointer)
}

func TestParser_TableConstructor(t *testing.T) {
	p, fn := parser(`local a = {1, 2, 3, settings = true, ["tim"] = 42, 54}`)
	require.NoError(t, p.stat(fn))
	require.Len(t, fn.Locals, 1)
	assert.Equal(t, []*Local{{name: "a"}}, fn.Locals)
	require.Len(t, fn.Constants, 7)
	assert.Equal(t, []any{int64(1), int64(2), int64(3), "settings", "tim", int64(42), int64(54)}, fn.Constants)
	require.Len(t, fn.ByteCodes, 11)
	assert.Equal(t, []Bytecode{
		iABC(NEWTABLE, 0, 4, 2),
		iABx(LOADK, 1, 0),
		iABx(LOADK, 2, 1),
		iABx(LOADK, 3, 2),
		iAB(LOADBOOL, 4, 1),
		iABCK(SETTABLE, 0, 4, false, 3, true),
		iABx(LOADK, 4, 4),
		iABx(LOADK, 5, 5),
		iABC(SETTABLE, 0, 4, 5),
		iABx(LOADK, 4, 6),
		iABC(SETLIST, 0, 5, 1),
	}, fn.ByteCodes)
	assert.Equal(t, fn.stackPointer, uint8(1))
}

func parser(src string) (*Parser, *FuncProto) {
	p := &Parser{
		rootfn: newFnProto(nil, []string{"_ENV"}, false),
		lex:    NewLexer(bytes.NewBufferString(src)),
	}
	return p, newFnProto(p.rootfn, []string{}, false)
}
