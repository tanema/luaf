package luaf

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tanema/luaf/bytecode"
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
		bytecode.IABCK(bytecode.GETTABUP, 0, 0, false, 1, true),
		bytecode.IABCK(bytecode.GETTABLE, 0, 0, false, 0, true),
		bytecode.IABCK(bytecode.SELF, 0, 0, false, 2, true),
		bytecode.IABCK(bytecode.GETTABUP, 2, 0, false, 3, true),
		bytecode.IABC(bytecode.CALL, 0, 3, 2),
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
		bytecode.IABx(bytecode.LOADI, 0, 23),
		bytecode.IABCK(bytecode.GETTABUP, 1, 0, false, 1, true),
		bytecode.IABCK(bytecode.SETTABLE, 1, 0, true, 0, false),
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
			bytecode.IABx(bytecode.LOADI, 0, 1),
			bytecode.IAB(bytecode.LOADBOOL, 1, 1),
			bytecode.IABx(bytecode.LOADK, 2, 0),
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
			bytecode.IABx(bytecode.LOADK, 0, 0),
			bytecode.IABx(bytecode.CLOSURE, 1, 0),
			bytecode.IAB(bytecode.MOVE, 2, 1),
			bytecode.IABC(bytecode.CALL, 2, 1, 2),
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
			bytecode.IABCK(bytecode.GETTABUP, 2, 0, false, 0, true),
			bytecode.IABC(bytecode.GETUPVAL, 3, 1, 0),
			bytecode.IABC(bytecode.CALL, 2, 2, 2),
			bytecode.IAB(bytecode.RETURN, 0, 1),
		)
	})

	t.Run("assignment attributes", func(t *testing.T) {
		t.Parallel()
		p, fn := parser(`local a <const> = 42`)
		require.NoError(t, p.stat(fn))
		assert.Equal(t, []*local{{name: "a", attrConst: true}}, fn.locals)
		assertByteCodes(t, fn, bytecode.IABx(bytecode.LOADI, 0, 42))
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
			bytecode.IABx(bytecode.LOADI, 0, 1),
			bytecode.IAB(bytecode.LOADBOOL, 1, 1),
			bytecode.IABx(bytecode.LOADK, 2, 0),
			bytecode.IABCK(bytecode.SETTABUP, 0, 1, true, 0, false),
			bytecode.IABCK(bytecode.SETTABUP, 0, 2, true, 1, false),
			bytecode.IABCK(bytecode.SETTABUP, 0, 3, true, 2, false),
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
		bytecode.IABx(bytecode.LOADK, 0, 0),
		bytecode.IABx(bytecode.CLOSURE, 1, 0),
		bytecode.IABCK(bytecode.GETTABUP, 2, 0, false, 3, true),
		bytecode.IABCK(bytecode.GETTABLE, 2, 2, false, 2, true),
		bytecode.IABCK(bytecode.SETTABLE, 2, 1, true, 1, false),
		bytecode.IABCK(bytecode.GETTABUP, 1, 0, false, 1, true),
		bytecode.IABC(bytecode.CALL, 1, 1, 2),
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
			bytecode.IABx(bytecode.LOADI, 0, 42),
			bytecode.IABC(bytecode.RETURN, 0, 2, 0),
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
			bytecode.IABCK(bytecode.GETTABUP, 0, 0, false, 0, true),
			bytecode.IABx(bytecode.LOADI, 1, 42),
			bytecode.IAB(bytecode.VARARG, 2, 0),
			bytecode.IABC(bytecode.RETURN, 0, 0, 0),
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
			bytecode.IABC(bytecode.RETURN, 0, 1, 0),
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
			bytecode.IABCK(bytecode.GETTABUP, 0, 0, false, 0, true),
			bytecode.IAB(bytecode.LOADI, 1, 2),
			bytecode.IAB(bytecode.LOADI, 2, 1),
			bytecode.IABC(bytecode.TAILCALL, 0, 3, 0),
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
		bytecode.IAB(bytecode.LOADBOOL, 0, 1),
		bytecode.IABC(bytecode.TEST, 0, 0, 0),
		bytecode.IAsBx(bytecode.JMP, 1, -3),
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
		bytecode.IAB(bytecode.LOADBOOL, 0, 1),
		bytecode.IABC(bytecode.TEST, 0, 0, 0),
		bytecode.IAsBx(bytecode.JMP, 1, 1),
		bytecode.IAsBx(bytecode.JMP, 1, -4),
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
		bytecode.IAB(bytecode.LOADBOOL, 0, 1),
		bytecode.IABC(bytecode.TEST, 0, 0, 0),
		bytecode.IAsBx(bytecode.JMP, 1, 2),
		bytecode.IAsBx(bytecode.JMP, 1, 1),
		bytecode.IAsBx(bytecode.JMP, 1, -5),
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
		bytecode.IABC(bytecode.NEWTABLE, 0, 5, 2),
		bytecode.IABx(bytecode.LOADI, 1, 1),
		bytecode.IABx(bytecode.LOADI, 2, 2),
		bytecode.IABx(bytecode.LOADI, 3, 3),
		bytecode.IABx(bytecode.LOADI, 4, 54),
		bytecode.IABCK(bytecode.GETTABUP, 5, 0, false, 0, true),
		bytecode.IABC(bytecode.SETLIST, 0, 6, 1),
		bytecode.IAB(bytecode.LOADBOOL, 1, 1),
		bytecode.IABCK(bytecode.SETTABLE, 0, 1, true, 1, false),
		bytecode.IABCK(bytecode.SETTABLE, 0, 2, true, 3, true),
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
		bytecode.IAB(bytecode.LOADI, 0, 1),
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
		bytecode.IAB(bytecode.LOADBOOL, 0, 0),
		bytecode.IABC(bytecode.TEST, 0, 0, 0),
		bytecode.IABx(bytecode.JMP, 0, 1),
		bytecode.IABx(bytecode.JMP, 0, 6),
		bytecode.IAB(bytecode.LOADBOOL, 0, 0),
		bytecode.IABC(bytecode.TEST, 0, 0, 0),
		bytecode.IABx(bytecode.JMP, 0, 1),
		bytecode.IABx(bytecode.JMP, 0, 2),
		bytecode.IAB(bytecode.LOADI, 0, 1),
		bytecode.IABCK(bytecode.SETTABUP, 0, 0, true, 0, false),
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
			bytecode.IAB(bytecode.LOADI, 0, 1),
			bytecode.IAB(bytecode.LOADI, 1, 10),
			bytecode.IAB(bytecode.LOADI, 2, 2),
			bytecode.IAB(bytecode.FORPREP, 0, 2),
			bytecode.IABC(bytecode.MOVE, 3, 0, 0),
			bytecode.IABCK(bytecode.SETTABUP, 0, 0, true, 3, false),
			bytecode.IAsBx(bytecode.FORLOOP, 0, -3),
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
			bytecode.IABCK(bytecode.GETTABUP, 0, 0, false, 0, true),
			bytecode.IABCK(bytecode.GETTABUP, 1, 0, false, 1, true),
			bytecode.IABC(bytecode.CALL, 0, 2, 4),
			bytecode.IAsBx(bytecode.JMP, 0, 4),
			bytecode.IABC(bytecode.MOVE, 5, 3, 0),
			bytecode.IABC(bytecode.MOVE, 6, 4, 0),
			bytecode.IABCK(bytecode.GETTABUP, 6, 0, false, 1, true),
			bytecode.IABCK(bytecode.SETTABLE, 6, 6, false, 5, false),
			bytecode.IAB(bytecode.TFORCALL, 0, 2),
			bytecode.IAsBx(bytecode.TFORLOOP, 1, -6),
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
			bytecode.IAsBx(bytecode.JMP, 0, 0),
			bytecode.IAB(bytecode.LOADI, 0, 1),
			bytecode.IABCK(bytecode.SETTABUP, 0, 0, true, 0, false),
			bytecode.IAsBx(bytecode.JMP, 0, -3),
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

func assertByteCodes(t *testing.T, fn *FnProto, code ...uint32) {
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

func fmtBytecodes(codes []uint32) string {
	parts := make([]string, len(codes))
	for i, code := range codes {
		parts[i] = "\t" + bytecode.ToString(code)
	}
	return strings.Join(parts, "\n")
}
