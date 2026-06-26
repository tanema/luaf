package parse

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tanema/luaf/src/bytecode"
	"github.com/tanema/luaf/src/types"
)

type TestFn struct {
	description  string
	input        string
	locals       []*Local
	upindexes    []Upindex
	constants    []any
	bytecodes    []uint32
	stackpointer uint8
	afterAssert  func(t *testing.T, p *Parser, fn *FnProto)
}

func TestParser(t *testing.T) {
	_envUpIndex := Upindex{
		FromStack: true,
		Name:      _ENVName,
		Index:     0,
		typeDefn: &types.Table{
			KeyDefn:   types.Any,
			ValDefn:   types.Any,
			FieldDefn: map[string]types.Definition{},
		},
	}

	t.Parallel()
	testcases := []TestFn{
		{
			description: "parser config",
			input:       `--!nostringCoers,requireOnly,envReadonly,localOnly,strict`,
			afterAssert: func(t *testing.T, p *Parser, _ *FnProto) {
				t.Helper()
				assert.False(t, p.config.StringCoers)
				assert.True(t, p.config.EnvReadonly)
				assert.True(t, p.config.RequireOnly)
				assert.True(t, p.config.LocalOnly)
				assert.True(t, p.config.Strict)
			},
		},
		{
			description: "comments",
			input: `
			;
			-- just a plain comment
			;
			`,
			afterAssert: func(t *testing.T, p *Parser, _ *FnProto) {
				t.Helper()
				assert.Equal(t, " just a plain comment", p.lastComment)
			},
		},
		{
			description: "suffix expression",
			input:       `class.name:foo(bar)`,
			constants:   []any{"name", "class", "foo", "bar"},
			upindexes:   []Upindex{_envUpIndex},
			bytecodes: []uint32{
				bytecode.IABC(bytecode.GETTABUP, 0, 0, 1, true),
				bytecode.IABC(bytecode.GETTABLE, 0, 0, 0, true),
				bytecode.IABC(bytecode.SELF, 0, 0, 2, true),
				bytecode.IABC(bytecode.GETTABUP, 2, 0, 3, true),
				bytecode.IABC(bytecode.CALL, 0, 3, 2, false),
			},
			stackpointer: 1,
		},
		{
			description: "index assign",
			input:       `table.window = 23`,
			constants:   []any{"table", "window"},
			upindexes:   []Upindex{_envUpIndex},
			bytecodes: []uint32{
				bytecode.IAsBx(bytecode.LOADI, 0, 23),
				bytecode.IABC(bytecode.GETTABUP, 1, 0, 0, true),
				bytecode.IABx(bytecode.LOADK, 2, 1),
				bytecode.IABC(bytecode.SETTABLE, 1, 2, 0, false),
			},
			stackpointer: 3,
		},
		{
			description:  "assignment attributes",
			input:        `local a <const> = 42`,
			locals:       []*Local{{name: "a", attrConst: true, typeDefn: types.Number}},
			bytecodes:    []uint32{bytecode.IAsBx(bytecode.LOADI, 0, 42)},
			stackpointer: 1,
		},
		{
			description: "local multiple assignment",
			input:       `local a, b, c = 1, true, "abcd"`,
			locals: []*Local{
				{name: "a", typeDefn: types.Number},
				{name: "b", typeDefn: types.Bool},
				{name: "c", typeDefn: types.String},
			},
			constants: []any{"abcd"},
			bytecodes: []uint32{
				bytecode.IAsBx(bytecode.LOADI, 0, 1),
				bytecode.IAB(bytecode.LOADTRUE, 1, 0),
				bytecode.IABx(bytecode.LOADK, 2, 0),
			},
			stackpointer: 3,
		},
		{
			description: "multiple assignment",
			input:       `a, b, c = 1, true, "defg"`,
			constants:   []any{"defg", "a", "b", "c"},
			upindexes:   []Upindex{_envUpIndex},
			bytecodes: []uint32{
				bytecode.IAsBx(bytecode.LOADI, 0, 1),             // 1
				bytecode.IAB(bytecode.LOADTRUE, 1, 0),            // true
				bytecode.IABx(bytecode.LOADK, 2, 0),              // "defg"
				bytecode.IABC(bytecode.GETUPVAL, 3, 0, 0, false), // ENV
				bytecode.IABC(bytecode.GETUPVAL, 4, 0, 0, false), // ENV
				bytecode.IABC(bytecode.GETUPVAL, 5, 0, 0, false), // ENV
				bytecode.IABx(bytecode.LOADK, 6, 1),              // a
				bytecode.IABC(bytecode.SETTABLE, 3, 6, 0, false), // ENV[a] = 1
				bytecode.IABx(bytecode.LOADK, 7, 2),              // b
				bytecode.IABC(bytecode.SETTABLE, 4, 7, 1, false), // ENV[b] = true
				bytecode.IABx(bytecode.LOADK, 8, 3),              // c
				bytecode.IABC(bytecode.SETTABLE, 5, 8, 2, false), // ENV[c] = "defg"
			},
			stackpointer: 9,
		},
		{
			description: "local function assignment",
			input: `local greeting = "hello world"
local function testFn(a, b, ...)
	print(greeting)
end
testFn()
`,
			locals: []*Local{
				{name: "greeting", upvalRef: true, typeDefn: types.String},
				{name: "testFn", typeDefn: &types.Function{}},
			},
			constants: []any{"hello world"},
			upindexes: []Upindex{_envUpIndex},
			bytecodes: []uint32{
				bytecode.IABx(bytecode.LOADK, 0, 0),
				bytecode.IABx(bytecode.CLOSURE, 1, 0),
				bytecode.IAB(bytecode.MOVE, 2, 1),
				bytecode.IABC(bytecode.CALL, 2, 1, 2, false),
			},
			stackpointer: 3,
			afterAssert: func(t *testing.T, _ *Parser, fn *FnProto) {
				t.Helper()
				testFn := fn.FnTable[0]
				assert.Equal(t, int64(2), testFn.Arity)
				assert.True(t, testFn.Varargs)
				assert.Equal(t, []any{"print"}, testFn.Constants)
				compareFn(t, TestFn{
					locals:    []*Local{{name: "a", typeDefn: types.Any}, {name: "b", typeDefn: types.Any}},
					constants: []any{"print"},
					upindexes: []Upindex{
						{FromStack: false, Name: _ENVName, Index: 0, typeDefn: types.NewTable()},
						{FromStack: true, Name: "greeting", Index: 0, typeDefn: types.String},
					},
					bytecodes: []uint32{
						bytecode.IABC(bytecode.GETTABUP, 2, 0, 0, true),
						bytecode.IABC(bytecode.GETUPVAL, 3, 1, 0, false),
						bytecode.IABC(bytecode.CALL, 2, 2, 2, false),
						bytecode.Return(0, 0),
					},
					stackpointer: 2,
				}, testFn)
			},
		},
		{
			description: "func stat",
			input: `local greet = "hello world"
function tbl.robot:testFn()
	print(hello)
end
testFn()
`,
			locals:    []*Local{{name: "greet", upvalRef: false, typeDefn: types.String}},
			constants: []any{"hello world", "robot", "tbl", "testFn"},
			upindexes: []Upindex{_envUpIndex},
			bytecodes: []uint32{
				bytecode.IABx(bytecode.LOADK, 0, 0),              // "hello world"
				bytecode.IABx(bytecode.CLOSURE, 1, 0),            // function
				bytecode.IABC(bytecode.GETTABUP, 2, 0, 2, true),  // ENV[tbl]
				bytecode.IABC(bytecode.GETTABLE, 2, 2, 1, true),  // tbl["robot"]
				bytecode.IABx(bytecode.LOADK, 3, 3),              // testFn
				bytecode.IABC(bytecode.SETTABLE, 2, 3, 1, false), // tbl["robot"]["testFn"] = function
				bytecode.IABC(bytecode.GETTABUP, 1, 0, 3, true),  // ENV["testFn"] # this is bad lua but accurate bytecode
				bytecode.IABC(bytecode.CALL, 1, 1, 2, false),     // ENV["testFn"]()
			},
			stackpointer: 2,
		},
		{
			description: "plain return",
			input:       `return 42`,
			bytecodes: []uint32{
				bytecode.IAsBx(bytecode.LOADI, 0, 42),
				bytecode.Return(0, 1),
			},
			stackpointer: 1,
		},
		{
			description: "multiple return",
			input:       `return a, 42, ...`,
			constants:   []any{"a"},
			upindexes:   []Upindex{_envUpIndex},
			bytecodes: []uint32{
				bytecode.IABC(bytecode.GETTABUP, 0, 0, 0, true),
				bytecode.IAsBx(bytecode.LOADI, 1, 42),
				bytecode.IAB(bytecode.VARARG, 2, 0),
				bytecode.Return(0, -1),
			},
			stackpointer: 3,
		},
		{
			description: "empty return",
			input:       `return`,
			bytecodes: []uint32{
				bytecode.Return(0, 0),
			},
		},
		{
			description: "tailcall",
			input:       `return min(2, 1)`,
			constants:   []any{"min"},
			upindexes:   []Upindex{_envUpIndex},
			bytecodes: []uint32{
				bytecode.IABC(bytecode.GETTABUP, 0, 0, 0, true),
				bytecode.IAsBx(bytecode.LOADI, 1, 2),
				bytecode.IAsBx(bytecode.LOADI, 2, 1),
				bytecode.IABC(bytecode.TAILCALL, 0, 3, 0, false),
			},
			stackpointer: 1,
		},
		{
			description: "repeat stat",
			input:       `repeat until true`,
			bytecodes: []uint32{
				bytecode.IAB(bytecode.LOADTRUE, 0, 0),
				bytecode.IABC(bytecode.TEST, 0, 0, 0, false),
				bytecode.Jump(-3),
			},
		},
		{
			description: "while stat",
			input: `while true do 
				print(a)
			end`,
			upindexes: []Upindex{_envUpIndex},
			constants: []any{"print", "a"},
			bytecodes: []uint32{
				bytecode.IAB(bytecode.LOADTRUE, 0, 0),
				bytecode.IABC(bytecode.TEST, 0, 0, 0, false),
				bytecode.Jump(4),
				bytecode.IABC(bytecode.GETTABUP, 0, 0, 0, true),
				bytecode.IABC(bytecode.GETTABUP, 1, 0, 1, true),
				bytecode.IABC(bytecode.CALL, 0, 2, 2, false),
				bytecode.Jump(-7),
			},
		},
		{
			description: "break stat",
			input:       `while true do break end`,
			bytecodes: []uint32{
				bytecode.IAB(bytecode.LOADTRUE, 0, 0),
				bytecode.IABC(bytecode.TEST, 0, 0, 0, false),
				bytecode.Jump(2),
				bytecode.Jump(1),
				bytecode.Jump(-5),
			},
		},
		{
			description: "table constructor",
			input: `local a = {
				1,
				2,
				3,
				-- throw a comment in there
				settings = true,
				["tim"] = 42,
				54,
				othertable,
			}`,
			locals:    []*Local{{name: "a", typeDefn: types.NewTable()}},
			constants: []any{"othertable", "settings", "tim", int64(42)},
			upindexes: []Upindex{_envUpIndex},
			bytecodes: []uint32{
				bytecode.IvABC(bytecode.NEWTABLE, 0, 5, 2, false),
				bytecode.IAsBx(bytecode.LOADI, 1, 1),
				bytecode.IAsBx(bytecode.LOADI, 2, 2),
				bytecode.IAsBx(bytecode.LOADI, 3, 3),
				bytecode.IAsBx(bytecode.LOADI, 4, 54),
				bytecode.IABC(bytecode.GETTABUP, 5, 0, 0, true),
				bytecode.IvABC(bytecode.SETLIST, 0, 6, 1, false),
				bytecode.IABx(bytecode.LOADK, 1, 1),
				bytecode.IAB(bytecode.LOADTRUE, 2, 0),
				bytecode.IABC(bytecode.SETTABLE, 0, 1, 2, false),
				bytecode.IABx(bytecode.LOADK, 1, 2),
				bytecode.IABC(bytecode.SETTABLE, 0, 1, 3, true),
			},
			stackpointer: 1,
		},
		{
			description: "do stat",
			input: `do
				local a = 1
			end`,
			bytecodes: []uint32{
				bytecode.IAsBx(bytecode.LOADI, 0, 1),
			},
		},
		{
			description: "If Stat",
			input: `if 2 == 1 then
				a = 44
			elseif 1 == 2 then
				a = 22
			else
				a = 1
			end`,
			constants: []any{"a"},
			upindexes: []Upindex{_envUpIndex},
			bytecodes: []uint32{
				// Simple because the other branches were optimized out.
				bytecode.IAsBx(bytecode.LOADI, 0, 1),             // 1
				bytecode.IABx(bytecode.LOADK, 1, 0),              // a
				bytecode.IABC(bytecode.SETTABUP, 0, 1, 0, false), // ENV[a] = 1
			},
		},
		{
			description: "for num",
			input: `for i = 1, 10, 2 do
				a = i
			end`,
			constants: []any{"a"},
			upindexes: []Upindex{_envUpIndex},
			bytecodes: []uint32{
				bytecode.IAsBx(bytecode.LOADI, 0, 1),             // 1
				bytecode.IAsBx(bytecode.LOADI, 1, 10),            // 10
				bytecode.IAsBx(bytecode.LOADI, 2, 2),             // 2
				bytecode.IABx(bytecode.FORPREP, 0, 3),            // Start for loop jump 3
				bytecode.IABC(bytecode.MOVE, 3, 0, 0, false),     // Move i to 3
				bytecode.IABx(bytecode.LOADK, 4, 0),              // a
				bytecode.IABC(bytecode.SETTABUP, 0, 4, 3, false), // ENV[a] = i
				bytecode.IABx(bytecode.FORLOOP, 0, 4),            // jump back 3
			},
		},
		{
			description: "for num reverse",
			input: `local forNumSum = 0
			for i = 10, 1, -1 do
				forNumSum = forNumSum + i
			end`,
			locals: []*Local{{name: "forNumSum", typeDefn: types.Number}},
			bytecodes: []uint32{
				bytecode.IAsBx(bytecode.LOADI, 0, 0),        // 0 [forNumSum]
				bytecode.IAsBx(bytecode.LOADI, 1, 10),       // 1
				bytecode.IAsBx(bytecode.LOADI, 2, 1),        // 10
				bytecode.IAsBx(bytecode.LOADI, 3, -1),       // 2
				bytecode.IABx(bytecode.FORPREP, 1, 4),       // Start for loop jump 4
				bytecode.IAB(bytecode.MOVE, 4, 0),           // Move forNumSum to 4
				bytecode.IAB(bytecode.MOVE, 5, 1),           // Move i to 5
				bytecode.IABC(bytecode.ADD, 4, 4, 5, false), // forNumSum + i
				bytecode.IAB(bytecode.MOVE, 0, 4),           // forNumSum = (forNumSum + i)
				bytecode.IABx(bytecode.FORLOOP, 1, 5),       // Jump back 3
			},
			stackpointer: 1,
		},
		{
			description: "for in",
			input: `for k, v in pairs(tbl) do
				tbl[v] = k
			end`,
			constants: []any{"pairs", "tbl"},
			upindexes: []Upindex{_envUpIndex},
			bytecodes: []uint32{
				bytecode.IABC(bytecode.GETTABUP, 0, 0, 0, true), // pairs
				bytecode.IABC(bytecode.GETTABUP, 1, 0, 1, true), // tbl
				bytecode.IABC(bytecode.CALL, 0, 2, 4, false),    // pairs(tbl)
				bytecode.Jump(4), // Jump to TFORCALL
				bytecode.IABC(bytecode.MOVE, 5, 3, 0, false),     // k
				bytecode.IABC(bytecode.GETTABUP, 6, 0, 1, true),  // ENV[tbl]
				bytecode.IABC(bytecode.MOVE, 7, 4, 0, false),     // v
				bytecode.IABC(bytecode.SETTABLE, 6, 7, 5, false), // tbl[v] = k
				bytecode.IAsBx(bytecode.TFORCALL, 0, 2),
				bytecode.IABx(bytecode.TFORLOOP, 1, 6),
			},
		},
		{
			description: "GOTO",
			input: `goto first
			::first::
			::comehere::
			a = 1
			goto comehere`,
			constants: []any{"a"},
			upindexes: []Upindex{_envUpIndex},
			bytecodes: []uint32{
				bytecode.Jump(0),
				bytecode.IAsBx(bytecode.LOADI, 0, 1),             // 1
				bytecode.IABx(bytecode.LOADK, 1, 0),              // a
				bytecode.IABC(bytecode.SETTABUP, 0, 1, 0, false), // ENV[a] = 1
				bytecode.Jump(-4),
			},
			stackpointer: 0,
		},
		{
			description: "close leaked locals",
			//nolint:dupword
			input: `local function test()
				local a = 42

				return function()
					return a
				end
			end

			local a = test()()
			return a`,
			locals: []*Local{{name: "test", typeDefn: &types.Function{}}, {name: "a", typeDefn: types.Any}},
			bytecodes: []uint32{
				bytecode.IABx(bytecode.CLOSURE, 0, 0),
				bytecode.IABC(bytecode.MOVE, 1, 0, 0, false),
				bytecode.IABC(bytecode.CALL, 1, 1, 2, false),
				bytecode.IABC(bytecode.CALL, 1, 1, 2, false),
				bytecode.IABC(bytecode.MOVE, 2, 1, 0, false),
				bytecode.Return(2, 1),
			},
			stackpointer: 3,
			afterAssert: func(t *testing.T, _ *Parser, fn *FnProto) {
				t.Helper()
				require.Len(t, fn.FnTable, 1)
				compareFn(t, TestFn{
					bytecodes: []uint32{
						bytecode.IAsBx(bytecode.LOADI, 0, 42),
						bytecode.IABx(bytecode.CLOSURE, 1, 0),
						bytecode.IABC(bytecode.CLOSE, 0, 0, 0, false),
						bytecode.Return(1, 1),
					},
				}, fn.FnTable[0])
				compareFn(t, TestFn{
					upindexes: []Upindex{{Name: "a", FromStack: true, typeDefn: &types.Union{Defn: []types.Definition{
						&types.Simple{Name: "int"}, &types.Simple{Name: "float"},
					}}}},
					bytecodes: []uint32{
						bytecode.IAB(bytecode.GETUPVAL, 0, 0),
						bytecode.Return(0, 1),
					},
				}, fn.FnTable[0].FnTable[0])
			},
		},
	}

	for _, tc := range testcases {
		assert.NotEmpty(t, tc.description, "no description on testcase")
		assert.NotEmpty(t, tc.input, "no parse input")
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()

			p := &Parser{
				rootfn: newRootFn(),
				lex:    newLexer("test", bytes.NewBufferString(tc.input)),
			}
			fn := NewFnProto(
				"test",
				"main",
				p.rootfn,
				[]*Local{},
				false,
				&types.Function{Params: []types.NamedPair{}, Return: []types.Definition{types.Any}},
				LineInfo{},
			)

			require.NoError(t, p.chunk(fn))
			compareFn(t, tc, fn)
			if tc.afterAssert != nil {
				tc.afterAssert(t, p, fn)
			}
		})
	}
}

func compareFn(t *testing.T, tc TestFn, fn *FnProto) {
	t.Helper()
	if len(tc.locals) == 0 {
		assert.Empty(t, fn.Locals)
	} else {
		assert.Equal(t, tc.locals, fn.Locals)
	}
	if len(tc.constants) == 0 {
		assert.Empty(t, fn.Constants)
	} else {
		assert.Equal(t, tc.constants, fn.Constants)
	}
	assert.Equal(t, tc.bytecodes, fn.ByteCodes, fmtBytecodeDiff(tc.bytecodes, fn.ByteCodes))
	assert.Equal(t, tc.stackpointer, fn.stackPointer)
	assert.Equal(t, tc.upindexes, fn.UpIndexes)
}

func fmtBytecodeDiff(expected, actual []uint32) string {
	parts := []string{}
	for i := range max(len(expected), len(actual)) {
		if i < len(expected) && i < len(actual) {
			exp, act := expected[i], actual[i]
			if exp == act {
				parts = append(parts, "  "+bytecode.ToString(exp))
			} else {
				parts = append(parts, "╔-"+bytecode.ToString(exp))
				parts = append(parts, "╚+"+bytecode.ToString(act))
			}
		} else if i < len(expected) && i >= len(actual) {
			parts = append(parts, " -", bytecode.ToString(expected[i]))
		} else if i >= len(expected) && i < len(actual) {
			parts = append(parts, " +", bytecode.ToString(actual[i]))
		}
	}
	return `
Bytcodes are not equal.
` + strings.Join(parts, "\n")
}
