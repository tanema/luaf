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
	assert.Equal(t, []*Local{}, fn.Locals)
	assert.Equal(t, []any{"class", "name", "foo", "bar"}, fn.Constants)
	assert.Equal(t, []Bytecode{
		iABCK(GETTABUP, 0, 0, false, 0, true),
		iABCK(GETTABLE, 1, 0, false, 1, true),
		iABCK(SELF, 2, 1, false, 2, true),
		iABCK(GETTABUP, 3, 0, false, 3, true),
		iABC(CALL, 2, 2, 2),
	}, fn.ByteCodes)
	assert.Equal(t, uint8(4), fn.stackPointer)
}

func TestParser_IndexAssign(t *testing.T) {
	p, fn := parser(`table.window = 23`)
	require.NoError(t, p.stat(fn))
	assert.Equal(t, []*Local{}, fn.Locals)
	assert.Equal(t, []any{"table", "window", int64(23)}, fn.Constants)
	assert.Equal(t, []Bytecode{
		iABCK(GETTABUP, 0, 0, false, 0, true),
		iABx(LOADK, 1, 2),
		iABCK(SETTABLE, 0, 1, true, 1, false),
	}, fn.ByteCodes)
	assert.Equal(t, uint8(1), fn.stackPointer)
}

func TestParser_LocalAssign(t *testing.T) {
	t.Run("multiple assignment", func(t *testing.T) {
		p, fn := parser(`local a, b, c = 1, true, "hello"`)
		require.NoError(t, p.stat(fn))
		assert.Equal(t, []*Local{{name: "a"}, {name: "b"}, {name: "c"}}, fn.Locals)
		assert.Equal(t, []any{int64(1), "hello"}, fn.Constants)
		assert.Equal(t, []Bytecode{
			iABx(LOADK, 0, 0),
			iAB(LOADBOOL, 1, 1),
			iABx(LOADK, 2, 1),
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
		assert.Equal(t, []*Local{{name: "hello", upvalRef: true}, {name: "testFn"}}, fn.Locals)
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
		assert.Len(t, testFn.Locals, 2)
		assert.Len(t, testFn.UpIndexes, 2)
		assert.Equal(t, []UpIndex{
			{fromStack: false, name: "_ENV", index: 0},
			{fromStack: true, name: "hello", index: 0},
		}, testFn.UpIndexes)
		assert.Equal(t, []*Local{{name: "a"}, {name: "b"}}, testFn.Locals)
		assert.Equal(t, []Bytecode{
			iABCK(GETTABUP, 2, 0, false, 0, true),
			iABC(GETUPVAL, 3, 1, 0),
			iABC(CALL, 2, 2, 2),
		}, testFn.ByteCodes)
		assert.Equal(t, uint8(3), fn.stackPointer)
	})

	t.Run("assignment attributes", func(t *testing.T) {
		p, fn := parser(`local a <const> = 42`)
		require.NoError(t, p.stat(fn))
		assert.Equal(t, []*Local{{name: "a", attrConst: true}}, fn.Locals)
		assert.Equal(t, []any{int64(42)}, fn.Constants)
		assert.Equal(t, []Bytecode{iABx(LOADK, 0, 0)}, fn.ByteCodes)
		assert.Equal(t, uint8(1), fn.stackPointer)
	})
}

func TestParser_Assign(t *testing.T) {
	t.Run("multiple assignment", func(t *testing.T) {
		p, fn := parser(`a, b, c = 1, true, "hello"`)
		require.NoError(t, p.stat(fn))
		assert.Len(t, fn.Locals, 0)
		assert.Equal(t, []any{"a", "b", "c", int64(1), "hello"}, fn.Constants)
		assert.Equal(t, []Bytecode{
			iABx(LOADK, 0, 3),
			iAB(LOADBOOL, 1, 1),
			iABx(LOADK, 2, 4),
			iABCK(SETTABUP, 0, 0, true, 0, false),
			iABCK(SETTABUP, 0, 1, true, 1, false),
			iABCK(SETTABUP, 0, 2, true, 2, false),
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
	assert.Equal(t, []*Local{{name: "hello", upvalRef: true}}, fn.Locals)
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
	assert.Len(t, fn.Locals, 0)
	assert.Equal(t, []any{"a", int64(42)}, fn.Constants)
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
	assert.Len(t, fn.Locals, 0)
	assert.Len(t, fn.Constants, 0)
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
	assert.Len(t, fn.Locals, 0)
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
	assert.Equal(t, []*Local{{name: "a"}}, fn.Locals)
	assert.Equal(t, []any{int64(1), int64(2), int64(3), "settings", "tim", int64(42), int64(54)}, fn.Constants)
	assert.Equal(t, []Bytecode{
		iABC(NEWTABLE, 0, 4, 2),
		iABx(LOADK, 1, 0),
		iABx(LOADK, 2, 1),
		iABx(LOADK, 3, 2),
		iAB(LOADBOOL, 4, 1),
		iABCK(SETTABLE, 0, 3, true, 4, false),
		iABCK(SETTABLE, 0, 4, true, 5, true),
		iABx(LOADK, 4, 6),
		iABC(SETLIST, 0, 5, 1),
	}, fn.ByteCodes)
	assert.Equal(t, fn.stackPointer, uint8(1))
}

func parser(src string) (*Parser, *FuncProto) {
	p := &Parser{
		rootfn: newFnProto("test", "env", nil, []string{"_ENV"}, false, 0),
		lex:    NewLexer(bytes.NewBufferString(src)),
	}
	return p, newFnProto("test", "main", p.rootfn, []string{}, false, 0)
}

// func debugBytecode(codes []Bytecode) {
//	for _, code := range codes {
//		println(code.String())
//	}
// }
