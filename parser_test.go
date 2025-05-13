package luaf

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParserConfig(t *testing.T) {
	t.Parallel()
	p, fn := parser(`--!nostringCoers,requireOnly,envReadonly,localOnly,strict`)
	require.NoError(t, p.stat(fn))
	assert.False(t, p.config.StringCoers)
	assert.True(t, p.config.EnvReadonly)
	assert.True(t, p.config.RequireOnly)
	assert.True(t, p.config.LocalOnly)
	assert.True(t, p.config.Strict)
}

func TestParser_Comment(t *testing.T) {
	t.Parallel()
	p, fn := parser(`
	;
	-- just a plain comment
	;
	`)
	require.NoError(t, p.statList(fn))
	assert.Equal(t, " just a plain comment", p.lastComment)
}

func TestParser_SuffixExpr(t *testing.T) {
	t.Parallel()
	p, fn := parser(`class.name:foo(bar)`)
	require.NoError(t, p.stat(fn))
	assert.Equal(t, []*local{}, fn.locals)
	assert.Equal(t, []any{"name", "class", "foo", "bar"}, fn.Constants)
	assertByteCodes(t, fn,
		iABCK(GETTABUP, 0, 0, false, 1, true),
		iABCK(GETTABLE, 0, 0, false, 0, true),
		iABCK(SELF, 0, 0, false, 2, true),
		iABCK(GETTABUP, 2, 0, false, 3, true),
		iABC(CALL, 0, 3, 2),
	)
	assert.Equal(t, uint8(1), fn.stackPointer)
}

func TestParser_IndexAssign(t *testing.T) {
	t.Parallel()
	p, fn := parser(`table.window = 23`)
	require.NoError(t, p.stat(fn))
	assert.Equal(t, []*local{}, fn.locals)
	assert.Equal(t, []any{"window", "table"}, fn.Constants)
	assertByteCodes(t, fn,
		iABx(LOADI, 0, 23),
		iABCK(GETTABUP, 1, 0, false, 1, true),
		iABCK(SETTABLE, 1, 0, true, 0, false),
	)
	assert.Equal(t, uint8(2), fn.stackPointer)
}

func TestParser_LocalAssign(t *testing.T) {
	t.Parallel()

	t.Run("multiple assignment", func(t *testing.T) {
		t.Parallel()
		p, fn := parser(`local a, b, c = 1, true, "hello"`)
		require.NoError(t, p.stat(fn))
		assert.Equal(t, []*local{{name: "a"}, {name: "b"}, {name: "c"}}, fn.locals)
		assert.Equal(t, []any{"hello"}, fn.Constants)
		assertByteCodes(t, fn,
			iABx(LOADI, 0, 1),
			iAB(LOADBOOL, 1, 1),
			iABx(LOADK, 2, 0),
		)
		assert.Equal(t, uint8(3), fn.stackPointer)
	})

	t.Run("function assignment", func(t *testing.T) {
		t.Parallel()
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
		assertByteCodes(t, fn,
			iABx(LOADK, 0, 0),
			iABx(CLOSURE, 1, 0),
			iAB(MOVE, 2, 1),
			iABC(CALL, 2, 1, 2),
		)
		assert.Equal(t, uint8(3), fn.stackPointer)

		testFn := fn.FnTable[0]
		assert.Equal(t, int64(2), testFn.Arity)
		assert.True(t, testFn.Varargs)
		assert.Equal(t, []any{"print"}, testFn.Constants)
		assert.Len(t, testFn.locals, 2)
		assert.Len(t, testFn.UpIndexes, 2)
		assert.Equal(t, []upindex{
			{FromStack: false, Name: "_ENV", Index: 0},
			{FromStack: true, Name: "hello", Index: 0},
		}, testFn.UpIndexes)
		assert.Equal(t, []*local{{name: "a"}, {name: "b"}}, testFn.locals)
		assertByteCodes(t, testFn,
			iABCK(GETTABUP, 2, 0, false, 0, true),
			iABC(GETUPVAL, 3, 1, 0),
			iABC(CALL, 2, 2, 2),
			iAB(RETURN, 0, 1),
		)
	})

	t.Run("assignment attributes", func(t *testing.T) {
		t.Parallel()
		p, fn := parser(`local a <const> = 42`)
		require.NoError(t, p.stat(fn))
		assert.Equal(t, []*local{{name: "a", attrConst: true}}, fn.locals)
		assertByteCodes(t, fn, iABx(LOADI, 0, 42))
		assert.Equal(t, uint8(1), fn.stackPointer)
	})
}

func TestParser_Assign(t *testing.T) {
	t.Parallel()
	t.Run("multiple assignment", func(t *testing.T) {
		t.Parallel()
		p, fn := parser(`a, b, c = 1, true, "hello"`)
		require.NoError(t, p.stat(fn))
		assert.Empty(t, fn.locals)
		assert.Equal(t, []any{"hello", "a", "b", "c"}, fn.Constants)
		assertByteCodes(t, fn,
			iABx(LOADI, 0, 1),
			iAB(LOADBOOL, 1, 1),
			iABx(LOADK, 2, 0),
			iABCK(SETTABUP, 0, 1, true, 0, false),
			iABCK(SETTABUP, 0, 2, true, 1, false),
			iABCK(SETTABUP, 0, 3, true, 2, false),
		)
		assert.Equal(t, uint8(3), fn.stackPointer)
	})
}

func TestParser_FuncStat(t *testing.T) {
	t.Parallel()
	p, fn := parser(`
local hello = "hello world"
function tbl.robot:testFn()
	print(hello)
end
testFn()
`)
	require.NoError(t, p.statList(fn))
	assert.Equal(t, []*local{{name: "hello", upvalRef: true}}, fn.locals)
	assert.Equal(t, []any{"hello world", "testFn", "robot", "tbl"}, fn.Constants)
	assertByteCodes(t, fn,
		iABx(LOADK, 0, 0),
		iABx(CLOSURE, 1, 0),
		iABCK(GETTABUP, 2, 0, false, 3, true),
		iABCK(GETTABLE, 2, 2, false, 2, true),
		iABCK(SETTABLE, 2, 1, true, 1, false),
		iABCK(GETTABUP, 1, 0, false, 1, true),
		iABC(CALL, 1, 1, 2),
	)
	assert.Equal(t, uint8(2), fn.stackPointer)
}

func TestParser_ReturnStat(t *testing.T) {
	t.Parallel()
	t.Run("plain return", func(t *testing.T) {
		t.Parallel()
		p, fn := parser(`return 42`)
		require.NoError(t, p.stat(fn))
		assert.Empty(t, fn.locals)
		assert.Empty(t, fn.Constants)
		assertByteCodes(t, fn,
			iABx(LOADI, 0, 42),
			iABC(RETURN, 0, 2, 0),
		)
		assert.Equal(t, uint8(1), fn.stackPointer)
	})
	t.Run("multiple return", func(t *testing.T) {
		t.Parallel()
		p, fn := parser(`return a, 42, ...`)
		require.NoError(t, p.stat(fn))
		assert.Empty(t, fn.locals)
		assert.Equal(t, []any{"a"}, fn.Constants)
		assertByteCodes(t, fn,
			iABCK(GETTABUP, 0, 0, false, 0, true),
			iABx(LOADI, 1, 42),
			iAB(VARARG, 2, 0),
			iABC(RETURN, 0, 0, 0),
		)
		assert.Equal(t, uint8(3), fn.stackPointer)
	})
	t.Run("empty return", func(t *testing.T) {
		t.Parallel()
		p, fn := parser(`return`)
		require.NoError(t, p.stat(fn))
		assert.Empty(t, fn.locals)
		assert.Empty(t, fn.Constants)
		assertByteCodes(t, fn,
			iABC(RETURN, 0, 1, 0),
		)
		assert.Equal(t, uint8(0), fn.stackPointer)
	})
	t.Run("tailcall", func(t *testing.T) {
		t.Parallel()
		p, fn := parser(`return min(2, 1)`)
		require.NoError(t, p.stat(fn))
		assert.Empty(t, fn.locals)
		assert.Len(t, fn.Constants, 1)
		assertByteCodes(t, fn,
			iABCK(GETTABUP, 0, 0, false, 0, true),
			iAB(LOADI, 1, 2),
			iAB(LOADI, 2, 1),
			iABC(TAILCALL, 0, 3, 0),
		)
		assert.Equal(t, uint8(1), fn.stackPointer)
	})
}

func TestParser_RepeatStat(t *testing.T) {
	t.Parallel()
	p, fn := parser(`repeat until true`)
	require.NoError(t, p.stat(fn))
	assert.Empty(t, fn.locals)
	assert.Empty(t, fn.Constants)
	assertByteCodes(t, fn,
		iAB(LOADBOOL, 0, 1),
		iABC(TEST, 0, 0, 0),
		iAsBx(JMP, 1, -3),
	)
	assert.Equal(t, uint8(0), fn.stackPointer)
}

func TestParser_WhileStat(t *testing.T) {
	t.Parallel()
	p, fn := parser(`while true do end`)
	require.NoError(t, p.stat(fn))
	assert.Empty(t, fn.locals)
	assert.Empty(t, fn.Constants)
	assertByteCodes(t, fn,
		iAB(LOADBOOL, 0, 1),
		iABC(TEST, 0, 0, 0),
		iAsBx(JMP, 1, 1),
		iAsBx(JMP, 1, -4),
	)
	assert.Equal(t, uint8(0), fn.stackPointer)
}

func TestParser_BreakStat(t *testing.T) {
	t.Parallel()
	p, fn := parser(`while true do break end`)
	require.NoError(t, p.stat(fn))
	assert.Empty(t, fn.locals)
	assert.Empty(t, fn.Constants)
	assertByteCodes(t, fn,
		iAB(LOADBOOL, 0, 1),
		iABC(TEST, 0, 0, 0),
		iAsBx(JMP, 1, 2),
		iAsBx(JMP, 1, 1),
		iAsBx(JMP, 1, -5),
	)
	assert.Equal(t, uint8(0), fn.stackPointer)
}

func TestParser_TableConstructor(t *testing.T) {
	t.Parallel()
	p, fn := parser(`local a = {
		1,
		2,
		3,
		-- throw a comment in there
		settings = true,
		["tim"] = 42,
		54,
		othertable,
	}`)
	require.NoError(t, p.stat(fn))
	assert.Equal(t, []*local{{name: "a"}}, fn.locals)
	assert.Equal(t, []any{"othertable", "settings", "tim", int64(42)}, fn.Constants)
	assertByteCodes(t, fn,
		iABC(NEWTABLE, 0, 5, 2),
		iABx(LOADI, 1, 1),
		iABx(LOADI, 2, 2),
		iABx(LOADI, 3, 3),
		iABx(LOADI, 4, 54),
		iABCK(GETTABUP, 5, 0, false, 0, true),
		iABC(SETLIST, 0, 6, 1),
		iAB(LOADBOOL, 1, 1),
		iABCK(SETTABLE, 0, 1, true, 1, false),
		iABCK(SETTABLE, 0, 2, true, 3, true),
	)
	assert.Equal(t, uint8(1), fn.stackPointer)
}

func TestParser_DoStat(t *testing.T) {
	t.Parallel()
	p, fn := parser(`
	do
		local a = 1
	end`)
	require.NoError(t, p.stat(fn))
	assert.Empty(t, fn.locals)
	assert.Empty(t, fn.Constants)
	assertByteCodes(t, fn,
		iAB(LOADI, 0, 1),
	)
	assert.Equal(t, uint8(0), fn.stackPointer)
}

func TestParser_IfStat(t *testing.T) {
	t.Parallel()
	p, fn := parser(`
	if 2 == 1 then
	elseif 1 == 2 then
	else
		a = 1
	end
	`)
	require.NoError(t, p.stat(fn))
	assert.Empty(t, fn.locals)
	assert.Len(t, fn.Constants, 1)
	assertByteCodes(t, fn,
		iAB(LOADBOOL, 0, 0),
		iABC(TEST, 0, 0, 0),
		iABx(JMP, 0, 1),
		iABx(JMP, 0, 6),
		iAB(LOADBOOL, 0, 0),
		iABC(TEST, 0, 0, 0),
		iABx(JMP, 0, 1),
		iABx(JMP, 0, 2),
		iAB(LOADI, 0, 1),
		iABCK(SETTABUP, 0, 0, true, 0, false),
	)
	assert.Equal(t, uint8(0), fn.stackPointer)
}

func TestParser_ForStat(t *testing.T) {
	t.Parallel()
	t.Run("for num", func(t *testing.T) {
		t.Parallel()
		p, fn := parser(`
		for i = 1, 10, 2 do
			a = i
		end
		`)
		require.NoError(t, p.stat(fn))
		assert.Empty(t, fn.locals)
		assert.Len(t, fn.Constants, 1)
		assertByteCodes(t, fn,
			iAB(LOADI, 0, 1),
			iAB(LOADI, 1, 10),
			iAB(LOADI, 2, 2),
			iAB(FORPREP, 0, 2),
			iABC(MOVE, 3, 0, 0),
			iABCK(SETTABUP, 0, 0, true, 3, false),
			iAsBx(FORLOOP, 0, -3),
		)
		assert.Equal(t, uint8(0), fn.stackPointer)
	})

	t.Run("for in", func(t *testing.T) {
		t.Parallel()
		p, fn := parser(`
		for k, v in pairs(tbl) do
			tbl[v] = k
		end
		`)
		require.NoError(t, p.stat(fn))
		assert.Empty(t, fn.locals)
		assert.Len(t, fn.Constants, 2)
		assertByteCodes(t, fn,
			iABCK(GETTABUP, 0, 0, false, 0, true),
			iABCK(GETTABUP, 1, 0, false, 1, true),
			iABC(CALL, 0, 2, 4),
			iAsBx(JMP, 0, 4),
			iABC(MOVE, 5, 3, 0),
			iABC(MOVE, 6, 4, 0),
			iABCK(GETTABUP, 6, 0, false, 1, true),
			iABCK(SETTABLE, 6, 6, false, 5, false),
			iAB(TFORCALL, 0, 2),
			iAsBx(TFORLOOP, 1, -6),
		)
		assert.Equal(t, uint8(0), fn.stackPointer)
	})
}

func TestParser_GOTO(t *testing.T) {
	t.Parallel()
	t.Run("for num", func(t *testing.T) {
		t.Parallel()
		p, fn := parser(`
		goto first
		::first::
		::comehere::
		a = 1
		goto comehere
		`)
		require.NoError(t, p.block(fn, false))
		assert.Empty(t, fn.locals)
		assert.Len(t, fn.Constants, 1)
		assertByteCodes(t, fn,
			iAsBx(JMP, 0, 0),
			iAB(LOADI, 0, 1),
			iABCK(SETTABUP, 0, 0, true, 0, false),
			iAsBx(JMP, 0, -3),
		)
		assert.Equal(t, uint8(0), fn.stackPointer)
	})
}

func parser(src string) (*Parser, *FnProto) {
	p := &Parser{
		rootfn: newFnProto("test", "env", nil, []string{"_ENV"}, false, lineInfo{}),
		lex:    newLexer(bytes.NewBufferString(src)),
	}
	return p, newFnProto("test", "main", p.rootfn, []string{}, false, lineInfo{})
}

func assertByteCodes(t *testing.T, fn *FnProto, code ...Bytecode) {
	t.Helper()
	assert.Equal(t, code, fn.ByteCodes, `
Bytcodes are not equal.
expected:
%s
actual:
%s`,
		fmtBytecodes(code),
		fmtBytecodes(fn.ByteCodes),
	)
}

func fmtBytecodes(codes []Bytecode) string {
	parts := make([]string, len(codes))
	for i, code := range codes {
		parts[i] = "\t" + code.String()
	}
	return strings.Join(parts, "\n")
}
