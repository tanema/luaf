package luaf

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParser_SuffixExpr(t *testing.T) {
	p, fn := parser(`class.name:foo(bar)`)
	require.NoError(t, p.stat(fn))
	assert.Equal(t, []*local{}, fn.locals)
	assert.Equal(t, []any{"name", "class", "foo", "bar"}, fn.Constants)
	assert.Equal(t, []Bytecode{
		iABCK(GETTABUP, 0, 0, false, 1, true),
		iABCK(GETTABLE, 0, 0, false, 0, true),
		iABCK(SELF, 0, 0, false, 2, true),
		iABCK(GETTABUP, 2, 0, false, 3, true),
		iABC(CALL, 0, 2, 2),
	}, fn.ByteCodes)
	assert.Equal(t, uint8(1), fn.stackPointer)
}

func TestParser_IndexAssign(t *testing.T) {
	p, fn := parser(`table.window = 23`)
	require.NoError(t, p.stat(fn))
	assert.Equal(t, []*local{}, fn.locals)
	assert.Equal(t, []any{"window", "table"}, fn.Constants)
	assert.Equal(t, []Bytecode{
		iABx(LOADI, 0, 23),
		iABCK(GETTABUP, 1, 0, false, 1, true),
		iABCK(SETTABLE, 1, 0, true, 0, false),
	}, fn.ByteCodes)
	assert.Equal(t, uint8(1), fn.stackPointer)
}

func TestParser_LocalAssign(t *testing.T) {
	t.Run("multiple assignment", func(t *testing.T) {
		p, fn := parser(`local a, b, c = 1, true, "hello"`)
		require.NoError(t, p.stat(fn))
		assert.Equal(t, []*local{{name: "a"}, {name: "b"}, {name: "c"}}, fn.locals)
		assert.Equal(t, []any{"hello"}, fn.Constants)
		assert.Equal(t, []Bytecode{
			iABx(LOADI, 0, 1),
			iAB(LOADBOOL, 1, 1),
			iABx(LOADK, 2, 0),
		}, fn.ByteCodes)
		assert.Equal(t, uint8(3), fn.stackPointer)
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
		assert.Equal(t, []*local{{name: "hello", upvalRef: true}, {name: "testFn"}}, fn.locals)
		assert.Equal(t, []any{"hello world"}, fn.Constants)
		assert.Len(t, fn.FnTable, 1)
		assert.Equal(t, []Bytecode{
			iABx(LOADK, 0, 0),
			iABx(CLOSURE, 1, 0),
			iAB(MOVE, 2, 1),
			iABC(CALL, 2, 1, 2),
		}, fn.ByteCodes)
		assert.Equal(t, uint8(3), fn.stackPointer)

		testFn := fn.FnTable[0]
		assert.Equal(t, 2, testFn.Arity)
		assert.True(t, testFn.Varargs)
		assert.Equal(t, []any{"print"}, testFn.Constants)
		assert.Len(t, testFn.locals, 2)
		assert.Len(t, testFn.UpIndexes, 2)
		assert.Equal(t, []UpIndex{
			{FromStack: false, Name: "_ENV", Index: 0},
			{FromStack: true, Name: "hello", Index: 0},
		}, testFn.UpIndexes)
		assert.Equal(t, []*local{{name: "a"}, {name: "b"}}, testFn.locals)
		assert.Equal(t, []Bytecode{
			iABCK(GETTABUP, 2, 0, false, 0, true),
			iABC(GETUPVAL, 3, 1, 0),
			iABC(CALL, 2, 2, 2),
			iAB(RETURN, 0, 1),
		}, testFn.ByteCodes)
	})

	t.Run("assignment attributes", func(t *testing.T) {
		p, fn := parser(`local a <const> = 42`)
		require.NoError(t, p.stat(fn))
		assert.Equal(t, []*local{{name: "a", attrConst: true}}, fn.locals)
		assert.Equal(t, []Bytecode{iABx(LOADI, 0, 42)}, fn.ByteCodes)
		assert.Equal(t, uint8(1), fn.stackPointer)
	})
}

func TestParser_Assign(t *testing.T) {
	t.Run("multiple assignment", func(t *testing.T) {
		p, fn := parser(`a, b, c = 1, true, "hello"`)
		require.NoError(t, p.stat(fn))
		assert.Len(t, fn.locals, 0)
		assert.Equal(t, []any{"hello", "a", "b", "c"}, fn.Constants)
		assert.Equal(t, []Bytecode{
			iABx(LOADI, 0, 1),
			iAB(LOADBOOL, 1, 1),
			iABx(LOADK, 2, 0),
			iABCK(SETTABUP, 0, 1, true, 0, false),
			iABCK(SETTABUP, 0, 2, true, 1, false),
			iABCK(SETTABUP, 0, 3, true, 2, false),
		}, fn.ByteCodes)
		assert.Equal(t, uint8(3), fn.stackPointer)
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
	assert.Equal(t, []*local{{name: "hello", upvalRef: true}}, fn.locals)
	assert.Equal(t, []any{"hello world", "testFn"}, fn.Constants)
	assert.Equal(t, []Bytecode{
		iABx(LOADK, 0, 0),
		iABx(CLOSURE, 1, 0),
		iABCK(SETTABUP, 0, 1, true, 1, false),
		iABCK(GETTABUP, 1, 0, false, 1, true),
		iABC(CALL, 1, 1, 2),
	}, fn.ByteCodes)
	assert.Equal(t, uint8(2), fn.stackPointer)
}

func TestParser_ReturnStat(t *testing.T) {
	p, fn := parser(`return a, 42, ...`)
	require.NoError(t, p.stat(fn))
	assert.Len(t, fn.locals, 0)
	assert.Equal(t, []any{"a"}, fn.Constants)
	assert.Equal(t, []Bytecode{
		iABCK(GETTABUP, 0, 0, false, 0, true),
		iABx(LOADI, 1, 42),
		iAB(VARARG, 2, 0),
		iABC(RETURN, 0, 0, 0),
	}, fn.ByteCodes)
	assert.Equal(t, uint8(3), fn.stackPointer)
}

func TestParser_RepeatStat(t *testing.T) {
	p, fn := parser(`repeat until true`)
	require.NoError(t, p.stat(fn))
	assert.Len(t, fn.locals, 0)
	assert.Len(t, fn.Constants, 0)
	assert.Equal(t, []Bytecode{
		iAB(LOADBOOL, 0, 1),
		iABC(TEST, 0, 0, 0),
		iAsBx(JMP, 1, -3),
	}, fn.ByteCodes)
	assert.Equal(t, uint8(0), fn.stackPointer)
}

func TestParser_WhileStat(t *testing.T) {
	p, fn := parser(`while true do end`)
	require.NoError(t, p.stat(fn))
	assert.Len(t, fn.locals, 0)
	assert.Len(t, fn.Constants, 0)
	assert.Len(t, fn.ByteCodes, 4)
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
	assert.Equal(t, []*local{{name: "a"}}, fn.locals)
	assert.Equal(t, []any{"settings", "tim", int64(42)}, fn.Constants)
	assert.Equal(t, []Bytecode{
		iABC(NEWTABLE, 0, 4, 2),
		iABx(LOADI, 1, 1),
		iABx(LOADI, 2, 2),
		iABx(LOADI, 3, 3),
		iABx(LOADI, 4, 54),
		iABC(SETLIST, 0, 5, 1),
		iAB(LOADBOOL, 2, 1),
		iABCK(SETTABLE, 0, 0, true, 2, false),
		iABCK(SETTABLE, 0, 1, true, 2, true),
	}, fn.ByteCodes)
	assert.Equal(t, fn.stackPointer, uint8(1))
}

func parser(src string) (*Parser, *FnProto) {
	p := &Parser{
		rootfn: newFnProto("test", "env", nil, []string{"_ENV"}, false, LineInfo{}),
		lex:    NewLexer(bytes.NewBufferString(src)),
	}
	return p, newFnProto("test", "main", p.rootfn, []string{}, false, LineInfo{})
}
