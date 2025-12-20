package runtime

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tanema/luaf/src/bytecode"
	"github.com/tanema/luaf/src/parse"
)

func TestVM_Eval(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		desc      string
		constants []any
		code      []uint32
		fntbl     []*parse.FnProto
		result    []any
		err       error
	}{
		{
			desc:      "MOV",
			constants: []any{int64(23)},
			code: []uint32{
				bytecode.IABx(bytecode.LOADK, 0, 0), bytecode.IAB(bytecode.MOVE, 1, 0), bytecode.IAB(bytecode.RETURN, 0, 3),
			},
			result: []any{int64(23), int64(23)},
		},
		{
			desc:      "LOADK",
			constants: []any{int64(23)},
			code:      []uint32{bytecode.IABx(bytecode.LOADK, 0, 0), bytecode.IAB(bytecode.RETURN, 0, 2)},
			result:    []any{int64(23)},
		},
		{
			desc: "LOADBOOL",
			code: []uint32{
				bytecode.IABx(bytecode.LOADBOOL, 0, 1), bytecode.IABC(bytecode.LOADBOOL, 1, 0, 1),
				bytecode.IAB(bytecode.RETURN, 0, 0), bytecode.IAB(bytecode.RETURN, 0, 3),
			},
			result: []any{true, false},
		},
		{
			desc:   "LOADI",
			code:   []uint32{bytecode.IABx(bytecode.LOADI, 0, 1274), bytecode.IAB(bytecode.RETURN, 0, 2)},
			result: []any{int64(1274)},
		},
		{
			desc: "LOADNil",
			code: []uint32{
				bytecode.IABx(bytecode.LOADI, 0, 0), bytecode.IABx(bytecode.LOADI, 1, 1),
				bytecode.IABx(bytecode.LOADI, 2, 2), bytecode.IABx(bytecode.LOADI, 3, 3),
				bytecode.IABx(bytecode.LOADI, 4, 4), bytecode.IABx(bytecode.LOADI, 5, 5),
				bytecode.IABx(bytecode.LOADNIL, 0, 4), bytecode.IAB(bytecode.RETURN, 0, 7),
			},
			result: []any{nil, nil, nil, nil, nil, int64(5)},
		},
		{
			desc:      "ADD",
			constants: []any{float64(32), float64(112)},
			code: []uint32{
				bytecode.IABx(bytecode.LOADI, 0, 1274), bytecode.IABx(bytecode.LOADI, 1, 72), bytecode.IABC(bytecode.ADD, 0, 0, 1),
				bytecode.IABx(bytecode.LOADK, 1, 0), bytecode.IABx(bytecode.LOADK, 2, 1), bytecode.IABC(bytecode.ADD, 1, 1, 2),
				bytecode.IABx(bytecode.LOADI, 2, 42), bytecode.IABx(bytecode.LOADK, 3, 0), bytecode.IABC(bytecode.ADD, 2, 2, 3),
				bytecode.IABx(bytecode.LOADK, 3, 0), bytecode.IABx(bytecode.LOADI, 4, 99), bytecode.IABC(bytecode.ADD, 3, 3, 4),
				bytecode.IAB(bytecode.RETURN, 0, 5),
			},
			result: []any{int64(1346), float64(144), float64(74), float64(131)},
		},
		{
			desc:      "ADD incompatible types",
			constants: []any{"Don't touch me"},
			code: []uint32{
				bytecode.IABx(bytecode.LOADK, 0, 0), bytecode.IABx(bytecode.LOADI, 1, 0), bytecode.IABC(bytecode.ADD, 0, 0, 1),
			},
			err: errors.New("cannot __add string with number"),
		},
		{
			desc:      "SUB",
			constants: []any{float64(32), float64(112)},
			code: []uint32{
				bytecode.IABx(bytecode.LOADI, 0, 1274), bytecode.IABx(bytecode.LOADI, 1, 72), bytecode.IABC(bytecode.SUB, 0, 0, 1),
				bytecode.IABx(bytecode.LOADK, 1, 0), bytecode.IABx(bytecode.LOADK, 2, 1), bytecode.IABC(bytecode.SUB, 1, 1, 2),
				bytecode.IABx(bytecode.LOADI, 2, 42), bytecode.IABx(bytecode.LOADK, 3, 0), bytecode.IABC(bytecode.SUB, 2, 2, 3),
				bytecode.IABx(bytecode.LOADK, 3, 0), bytecode.IABx(bytecode.LOADI, 4, 99), bytecode.IABC(bytecode.SUB, 3, 3, 4),
				bytecode.IAB(bytecode.RETURN, 0, 5),
			},
			result: []any{int64(1202), float64(-80), float64(10), float64(-67)},
		},
		{
			desc:      "SUB incompatible types",
			constants: []any{"Don't touch me"},
			code: []uint32{
				bytecode.IABx(bytecode.LOADK, 0, 0), bytecode.IABx(bytecode.LOADI, 1, 0), bytecode.IABC(bytecode.SUB, 0, 0, 1),
			},
			err: errors.New("cannot __sub string with number"),
		},
		{
			desc:      "MUL",
			constants: []any{float64(32), float64(112)},
			code: []uint32{
				bytecode.IABx(bytecode.LOADI, 0, 1274), bytecode.IABx(bytecode.LOADI, 1, 72), bytecode.IABC(bytecode.MUL, 0, 0, 1),
				bytecode.IABx(bytecode.LOADK, 1, 0), bytecode.IABx(bytecode.LOADK, 2, 1), bytecode.IABC(bytecode.MUL, 1, 1, 2),
				bytecode.IABx(bytecode.LOADI, 2, 42), bytecode.IABx(bytecode.LOADK, 3, 0), bytecode.IABC(bytecode.MUL, 2, 2, 3),
				bytecode.IABx(bytecode.LOADK, 3, 0), bytecode.IABx(bytecode.LOADI, 4, 99), bytecode.IABC(bytecode.MUL, 3, 3, 4),
				bytecode.IAB(bytecode.RETURN, 0, 5),
			},
			result: []any{int64(91728), float64(3584), float64(1344), float64(3168)},
		},
		{
			desc:      "MUL incompatible types",
			constants: []any{"Don't touch me"},
			code: []uint32{
				bytecode.IABx(bytecode.LOADK, 0, 0), bytecode.IABx(bytecode.LOADI, 1, 0), bytecode.IABC(bytecode.MUL, 0, 0, 1),
			},
			err: errors.New("cannot __mul string with number"),
		},
		{
			desc:      "DIV",
			constants: []any{float64(112), float64(32)},
			code: []uint32{
				bytecode.IABx(bytecode.LOADI, 0, 1274), bytecode.IABx(bytecode.LOADI, 1, 10), bytecode.IABC(bytecode.DIV, 0, 0, 1),
				bytecode.IABx(bytecode.LOADK, 1, 0), bytecode.IABx(bytecode.LOADK, 2, 1), bytecode.IABC(bytecode.DIV, 1, 1, 2),
				bytecode.IABx(bytecode.LOADI, 2, 42), bytecode.IABx(bytecode.LOADK, 3, 0), bytecode.IABC(bytecode.DIV, 2, 2, 3),
				bytecode.IABx(bytecode.LOADK, 3, 0), bytecode.IABx(bytecode.LOADI, 4, 1), bytecode.IABC(bytecode.DIV, 3, 3, 4),
				bytecode.IAB(bytecode.RETURN, 0, 5),
			},
			result: []any{float64(127.4), float64(3.5), float64(0.375), float64(112)},
		},
		{
			desc:      "DIV incompatible types",
			constants: []any{"Don't touch me"},
			code: []uint32{
				bytecode.IABx(bytecode.LOADK, 0, 0), bytecode.IABx(bytecode.LOADI, 1, 0), bytecode.IABC(bytecode.DIV, 0, 0, 1),
			},
			err: errors.New("cannot __div string with number"),
		},
		{
			desc:      "MOD",
			constants: []any{float64(112), float64(32), "Don't touch me"},
			code: []uint32{
				bytecode.IABx(bytecode.LOADI, 0, 1274), bytecode.IABx(bytecode.LOADI, 1, 72), bytecode.IABC(bytecode.MOD, 0, 0, 1),
				bytecode.IABx(bytecode.LOADK, 1, 0), bytecode.IABx(bytecode.LOADK, 2, 1), bytecode.IABC(bytecode.MOD, 1, 1, 2),
				bytecode.IABx(bytecode.LOADI, 2, 42), bytecode.IABx(bytecode.LOADK, 3, 0), bytecode.IABC(bytecode.MOD, 2, 2, 3),
				bytecode.IABx(bytecode.LOADK, 3, 0), bytecode.IABx(bytecode.LOADI, 4, 1), bytecode.IABC(bytecode.MOD, 3, 3, 4),
				bytecode.IAB(bytecode.RETURN, 0, 5),
			},
			result: []any{int64(50), float64(16), float64(42), float64(0)},
		},
		{
			desc:      "MOD incompatible types",
			constants: []any{"Don't touch me"},
			code: []uint32{
				bytecode.IABx(bytecode.LOADK, 0, 0), bytecode.IABx(bytecode.LOADI, 1, 0), bytecode.IABC(bytecode.MOD, 0, 0, 1),
			},
			err: errors.New("cannot __mod string with number"),
		},
		{
			desc:      "POW",
			constants: []any{float64(2), float64(3)},
			code: []uint32{
				bytecode.IABx(bytecode.LOADI, 0, 2), bytecode.IABx(bytecode.LOADI, 1, 4), bytecode.IABC(bytecode.POW, 0, 0, 1),
				bytecode.IABx(bytecode.LOADK, 1, 0), bytecode.IABx(bytecode.LOADK, 2, 1), bytecode.IABC(bytecode.POW, 1, 1, 2),
				bytecode.IABx(bytecode.LOADI, 2, 2), bytecode.IABx(bytecode.LOADK, 3, 0), bytecode.IABC(bytecode.POW, 2, 2, 3),
				bytecode.IABx(bytecode.LOADK, 3, 0), bytecode.IABx(bytecode.LOADI, 4, 1), bytecode.IABC(bytecode.POW, 3, 3, 4),
				bytecode.IAB(bytecode.RETURN, 0, 5),
			},
			result: []any{float64(16), float64(8), float64(4), float64(2)},
		},
		{
			desc:      "POW incompatible types",
			constants: []any{"Don't touch me"},
			code: []uint32{
				bytecode.IABx(bytecode.LOADK, 0, 0), bytecode.IABx(bytecode.LOADI, 1, 0), bytecode.IABC(bytecode.POW, 0, 0, 1),
			},
			err: errors.New("cannot __pow string with number"),
		},
		{
			desc:      "IDIV",
			constants: []any{float64(112), float64(32)},
			code: []uint32{
				bytecode.IABx(bytecode.LOADI, 0, 98), bytecode.IABx(bytecode.LOADI, 1, 2), bytecode.IABC(bytecode.IDIV, 0, 0, 1),
				bytecode.IABx(bytecode.LOADK, 1, 0), bytecode.IABx(bytecode.LOADK, 2, 1), bytecode.IABC(bytecode.IDIV, 1, 1, 2),
				bytecode.IABx(bytecode.LOADI, 2, 42), bytecode.IABx(bytecode.LOADK, 3, 0), bytecode.IABC(bytecode.IDIV, 2, 2, 3),
				bytecode.IABx(bytecode.LOADK, 3, 0), bytecode.IABx(bytecode.LOADI, 4, 1), bytecode.IABC(bytecode.IDIV, 3, 3, 4),
				bytecode.IAB(bytecode.RETURN, 0, 5),
			},
			result: []any{int64(49), float64(3), float64(0), float64(112)},
		},
		{
			desc:      "IDIV incompatible types",
			constants: []any{"Don't touch me"},
			code: []uint32{
				bytecode.IABx(bytecode.LOADK, 0, 0), bytecode.IABx(bytecode.LOADI, 1, 0), bytecode.IABC(bytecode.IDIV, 0, 0, 1),
			},
			err: errors.New("cannot __idiv string with number"),
		},
		{
			desc:      "BAND",
			constants: []any{float64(2), float64(3)},
			code: []uint32{
				bytecode.IABx(bytecode.LOADI, 0, 2), bytecode.IABx(bytecode.LOADI, 1, 4), bytecode.IABC(bytecode.BAND, 0, 0, 1),
				bytecode.IABx(bytecode.LOADK, 1, 0), bytecode.IABx(bytecode.LOADK, 2, 1), bytecode.IABC(bytecode.BAND, 1, 1, 2),
				bytecode.IABx(bytecode.LOADI, 2, 2), bytecode.IABx(bytecode.LOADK, 3, 0), bytecode.IABC(bytecode.BAND, 2, 2, 3),
				bytecode.IABx(bytecode.LOADK, 3, 0), bytecode.IABx(bytecode.LOADI, 4, 1), bytecode.IABC(bytecode.BAND, 3, 3, 4),
				bytecode.IAB(bytecode.RETURN, 0, 5),
			},
			result: []any{int64(0), int64(2), int64(2), int64(0)},
		},
		{
			desc:      "BAND incompatible types",
			constants: []any{"Don't touch me"},
			code: []uint32{
				bytecode.IABx(bytecode.LOADK, 0, 0), bytecode.IABx(bytecode.LOADI, 1, 0), bytecode.IABC(bytecode.BAND, 0, 0, 1),
			},
			err: errors.New("cannot __band string and number"),
		},
		{
			desc:      "BOR",
			constants: []any{float64(2), float64(3)},
			code: []uint32{
				bytecode.IABx(bytecode.LOADI, 0, 2), bytecode.IABx(bytecode.LOADI, 1, 4), bytecode.IABC(bytecode.BOR, 0, 0, 1),
				bytecode.IABx(bytecode.LOADK, 1, 0), bytecode.IABx(bytecode.LOADK, 2, 1), bytecode.IABC(bytecode.BOR, 1, 1, 2),
				bytecode.IABx(bytecode.LOADI, 2, 2), bytecode.IABx(bytecode.LOADK, 3, 0), bytecode.IABC(bytecode.BOR, 2, 2, 3),
				bytecode.IABx(bytecode.LOADK, 3, 0), bytecode.IABx(bytecode.LOADI, 4, 1), bytecode.IABC(bytecode.BOR, 3, 3, 4),
				bytecode.IAB(bytecode.RETURN, 0, 5),
			},
			result: []any{int64(6), int64(3), int64(2), int64(3)},
		},
		{
			desc:      "BOR incompatible types",
			constants: []any{"Don't touch me"},
			code: []uint32{
				bytecode.IABx(bytecode.LOADK, 0, 0), bytecode.IABx(bytecode.LOADI, 1, 0), bytecode.IABC(bytecode.BOR, 0, 0, 1),
			},
			err: errors.New("cannot __bor string and number"),
		},
		{
			desc:      "BXOR",
			constants: []any{float64(2), float64(3)},
			code: []uint32{
				bytecode.IABx(bytecode.LOADI, 0, 2), bytecode.IABx(bytecode.LOADI, 1, 4), bytecode.IABC(bytecode.BXOR, 0, 0, 1),
				bytecode.IABx(bytecode.LOADK, 1, 0), bytecode.IABx(bytecode.LOADK, 2, 1), bytecode.IABC(bytecode.BXOR, 1, 1, 2),
				bytecode.IABx(bytecode.LOADI, 2, 2), bytecode.IABx(bytecode.LOADK, 3, 0), bytecode.IABC(bytecode.BXOR, 2, 2, 3),
				bytecode.IABx(bytecode.LOADK, 3, 0), bytecode.IABx(bytecode.LOADI, 4, 1), bytecode.IABC(bytecode.BXOR, 3, 3, 4),
				bytecode.IAB(bytecode.RETURN, 0, 5),
			},
			result: []any{int64(6), int64(1), int64(0), int64(3)},
		},
		{
			desc:      "BXOR incompatible types",
			constants: []any{"Don't touch me"},
			code: []uint32{
				bytecode.IABx(bytecode.LOADK, 0, 0), bytecode.IABx(bytecode.LOADI, 1, 0), bytecode.IABC(bytecode.BXOR, 0, 0, 1),
			},
			err: errors.New("cannot __bxor string and number"),
		},
		{
			desc:      "SHL",
			constants: []any{float64(2), float64(3)},
			code: []uint32{
				bytecode.IABx(bytecode.LOADI, 0, 2), bytecode.IABx(bytecode.LOADI, 1, 4), bytecode.IABC(bytecode.SHL, 0, 0, 1),
				bytecode.IABx(bytecode.LOADK, 1, 0), bytecode.IABx(bytecode.LOADK, 2, 1), bytecode.IABC(bytecode.SHL, 1, 1, 2),
				bytecode.IABx(bytecode.LOADI, 2, 2), bytecode.IABx(bytecode.LOADK, 3, 0), bytecode.IABC(bytecode.SHL, 2, 2, 3),
				bytecode.IABx(bytecode.LOADK, 3, 0), bytecode.IABx(bytecode.LOADI, 4, 1), bytecode.IABC(bytecode.SHL, 3, 3, 4),
				bytecode.IAB(bytecode.RETURN, 0, 5),
			},
			result: []any{int64(32), int64(16), int64(8), int64(4)},
		},
		{
			desc:      "SHL incompatible types",
			constants: []any{"Don't touch me"},
			code: []uint32{
				bytecode.IABx(bytecode.LOADK, 0, 0), bytecode.IABx(bytecode.LOADI, 1, 0), bytecode.IABC(bytecode.SHL, 0, 0, 1),
			},
			err: errors.New("cannot __shl string and number"),
		},
		{
			desc:      "SHR",
			constants: []any{float64(100), float64(1)},
			code: []uint32{
				bytecode.IABx(bytecode.LOADI, 0, 100), bytecode.IABx(bytecode.LOADI, 1, 1), bytecode.IABC(bytecode.SHR, 0, 0, 1),
				bytecode.IABx(bytecode.LOADK, 1, 0), bytecode.IABx(bytecode.LOADK, 2, 1), bytecode.IABC(bytecode.SHR, 1, 1, 2),
				bytecode.IABx(bytecode.LOADI, 2, 500), bytecode.IABx(bytecode.LOADK, 3, 1), bytecode.IABC(bytecode.SHR, 2, 2, 3),
				bytecode.IABx(bytecode.LOADK, 3, 0), bytecode.IABx(bytecode.LOADI, 4, 1), bytecode.IABC(bytecode.SHR, 3, 3, 4),
				bytecode.IAB(bytecode.RETURN, 0, 5),
			},
			result: []any{int64(50), int64(50), int64(250), int64(50)},
		},
		{
			desc:      "SHR incompatible types",
			constants: []any{"Don't touch me"},
			code: []uint32{
				bytecode.IABx(bytecode.LOADK, 0, 0), bytecode.IABx(bytecode.LOADI, 1, 0), bytecode.IABC(bytecode.SHR, 0, 0, 1),
			},
			err: errors.New("cannot __shr string and number"),
		},
		{
			desc:      "UNM",
			constants: []any{float64(200)},
			code: []uint32{
				bytecode.IABx(bytecode.LOADI, 0, 100), bytecode.IAB(bytecode.UNM, 0, 0),
				bytecode.IABx(bytecode.LOADK, 1, 0), bytecode.IAB(bytecode.UNM, 1, 1),
				bytecode.IAB(bytecode.RETURN, 0, 3),
			},
			result: []any{int64(-100), float64(-200)},
		},
		{
			desc:      "UNM incompatible types",
			constants: []any{"Don't touch me"},
			code:      []uint32{bytecode.IABx(bytecode.LOADK, 0, 0), bytecode.IAB(bytecode.UNM, 0, 0)},
			err:       errors.New("cannot __unm string with number"),
		},
		{
			desc:      "BNOT",
			constants: []any{float64(100)},
			code: []uint32{
				bytecode.IABx(bytecode.LOADI, 0, 100), bytecode.IAB(bytecode.BNOT, 0, 0),
				bytecode.IABx(bytecode.LOADK, 1, 0), bytecode.IAB(bytecode.BNOT, 1, 1),
				bytecode.IAB(bytecode.RETURN, 0, 3),
			},
			result: []any{int64(-101), int64(-101)},
		},
		{
			desc:      "BNOT incompatible types",
			constants: []any{"Don't touch me"},
			code: []uint32{
				bytecode.IABx(bytecode.LOADK, 0, 0), bytecode.IAB(bytecode.BNOT, 0, 0),
			},
			err: errors.New("cannot __bnot string"),
		},
		{
			desc:      "NOT",
			constants: []any{float64(0), float64(1), "Don't touch me"},
			code: []uint32{
				bytecode.IABx(bytecode.LOADI, 0, 0), bytecode.IAB(bytecode.NOT, 0, 0), // integer == 0
				bytecode.IABx(bytecode.LOADI, 1, 1), bytecode.IAB(bytecode.NOT, 1, 1), // integer != 0
				bytecode.IABx(bytecode.LOADK, 2, 0), bytecode.IAB(bytecode.NOT, 2, 2), // float == 0
				bytecode.IABx(bytecode.LOADK, 3, 1), bytecode.IAB(bytecode.NOT, 3, 3), // float != 0
				bytecode.IABx(bytecode.LOADNIL, 4, 1), bytecode.IAB(bytecode.NOT, 4, 4), // nil
				bytecode.IAB(bytecode.LOADBOOL, 5, 1), bytecode.IAB(bytecode.NOT, 5, 5), // true
				bytecode.IAB(bytecode.LOADBOOL, 6, 0), bytecode.IAB(bytecode.NOT, 6, 6), // false
				bytecode.IAB(bytecode.LOADK, 7, 2), bytecode.IAB(bytecode.NOT, 7, 7), // string
				bytecode.IAB(bytecode.RETURN, 0, 9),
			},
			result: []any{false, false, false, false, true, false, true, false},
		},
		{
			desc:      "CONCAT",
			constants: []any{float64(200), "Don't touch me"},
			code: []uint32{
				bytecode.IABx(bytecode.LOADI, 0, 100), bytecode.IABx(bytecode.LOADK, 1, 0), bytecode.IABx(bytecode.LOADK, 2, 1),
				bytecode.IABC(bytecode.CONCAT, 0, 0, 2),
				bytecode.IAB(bytecode.RETURN, 0, 2),
			},
			result: []any{"100200Don't touch me"},
		},
		{
			desc: "EQ is false expecting false should not increment pc",
			code: []uint32{
				bytecode.IABx(bytecode.LOADI, 0, 2), bytecode.IABx(bytecode.LOADI, 1, 1), bytecode.IABC(bytecode.EQ, 0, 0, 1),
				bytecode.IAB(bytecode.RETURN, 0, 3), 0xFFFFFFFF,
			},
			result: []any{int64(2), int64(1)},
		},
		{
			desc: "EQ is true expecting false should increment pc",
			code: []uint32{
				bytecode.IABx(bytecode.LOADI, 0, 1), bytecode.IABx(bytecode.LOADI, 1, 1), bytecode.IABC(bytecode.EQ, 0, 0, 1),
				0xFFFFFFFF, bytecode.IAB(bytecode.RETURN, 0, 3),
			},
			result: []any{int64(1), int64(1)},
		},
		{
			desc: "EQ is true expecting true should not increment pc",
			code: []uint32{
				bytecode.IABx(bytecode.LOADI, 0, 1), bytecode.IABx(bytecode.LOADI, 1, 1), bytecode.IABC(bytecode.EQ, 1, 0, 1),
				bytecode.IAB(bytecode.RETURN, 0, 3), 0xFFFFFFFF,
			},
			result: []any{int64(1), int64(1)},
		},
		{
			desc: "EQ is false expecting true should increment pc",
			code: []uint32{
				bytecode.IABx(bytecode.LOADI, 0, 2), bytecode.IABx(bytecode.LOADI, 1, 1), bytecode.IABC(bytecode.EQ, 1, 0, 1),
				0xFFFFFFFF, bytecode.IAB(bytecode.RETURN, 0, 3),
			},
			result: []any{int64(2), int64(1)},
		},
		{
			desc: "LT is false expecting false should not increment pc",
			code: []uint32{
				bytecode.IABx(bytecode.LOADI, 0, 2), bytecode.IABx(bytecode.LOADI, 1, 1), bytecode.IABC(bytecode.LT, 0, 0, 1),
				bytecode.IAB(bytecode.RETURN, 0, 3), 0xFFFFFFFF,
			},
			result: []any{int64(2), int64(1)},
		},
		{
			desc: "LT is true expecting false should increment pc",
			code: []uint32{
				bytecode.IABx(bytecode.LOADI, 0, 0), bytecode.IABx(bytecode.LOADI, 1, 1), bytecode.IABC(bytecode.LT, 0, 0, 1),
				0xFFFFFFFF, bytecode.IAB(bytecode.RETURN, 0, 3),
			},
			result: []any{int64(0), int64(1)},
		},
		{
			desc: "LT is true expecting true should not increment pc",
			code: []uint32{
				bytecode.IABx(bytecode.LOADI, 0, 0), bytecode.IABx(bytecode.LOADI, 1, 1), bytecode.IABC(bytecode.LT, 1, 0, 1),
				bytecode.IAB(bytecode.RETURN, 0, 3), 0xFFFFFFFF,
			},
			result: []any{int64(0), int64(1)},
		},
		{
			desc: "LT is false expecting true should increment pc",
			code: []uint32{
				bytecode.IABx(bytecode.LOADI, 0, 2), bytecode.IABx(bytecode.LOADI, 1, 1), bytecode.IABC(bytecode.LT, 1, 0, 1),
				0xFFFFFFFF, bytecode.IAB(bytecode.RETURN, 0, 3),
			},
			result: []any{int64(2), int64(1)},
		},
		{
			desc:      "LT compare non-number should err",
			constants: []any{"nope"},
			code: []uint32{
				bytecode.IABx(bytecode.LOADK, 0, 0), bytecode.IABx(bytecode.LOADI, 1, 1), bytecode.IABC(bytecode.LT, 1, 0, 1),
			},
			err: errors.New("cannot __lt string and number"),
		},
		{
			desc: "LE is false expecting false should not increment pc",
			code: []uint32{
				bytecode.IABx(bytecode.LOADI, 0, 2), bytecode.IABx(bytecode.LOADI, 1, 1), bytecode.IABC(bytecode.LE, 0, 0, 1),
				bytecode.IAB(bytecode.RETURN, 0, 3), 0xFFFFFFFF,
			},
			result: []any{int64(2), int64(1)},
		},
		{
			desc: "LE is true expecting false should increment pc",
			code: []uint32{
				bytecode.IABx(bytecode.LOADI, 0, 0), bytecode.IABx(bytecode.LOADI, 1, 1), bytecode.IABC(bytecode.LE, 0, 0, 1),
				0xFFFFFFFF, bytecode.IAB(bytecode.RETURN, 0, 3),
			},
			result: []any{int64(0), int64(1)},
		},
		{
			desc: "LE is true expecting true should not increment pc",
			code: []uint32{
				bytecode.IABx(bytecode.LOADI, 0, 0), bytecode.IABx(bytecode.LOADI, 1, 1), bytecode.IABC(bytecode.LE, 1, 0, 1),
				bytecode.IAB(bytecode.RETURN, 0, 3), 0xFFFFFFFF,
			},
			result: []any{int64(0), int64(1)},
		},
		{
			desc: "LE is false expecting true should increment pc",
			code: []uint32{
				bytecode.IABx(bytecode.LOADI, 0, 2), bytecode.IABx(bytecode.LOADI, 1, 1), bytecode.IABC(bytecode.LE, 1, 0, 1),
				0xFFFFFFFF, bytecode.IAB(bytecode.RETURN, 0, 3),
			},
			result: []any{int64(2), int64(1)},
		},
		{
			desc:      "LE compare non-number should err",
			constants: []any{"nope"},
			code: []uint32{
				bytecode.IABx(bytecode.LOADK, 0, 0), bytecode.IABx(bytecode.LOADI, 1, 1), bytecode.IABC(bytecode.LE, 1, 0, 1),
			},
			err: errors.New("cannot __le string and number"),
		},
		{
			desc: "TEST is false expecting false should not increment pc",
			code: []uint32{
				bytecode.IABx(bytecode.LOADBOOL, 0, 0), bytecode.IAB(bytecode.TEST, 0, 0),
				bytecode.IAB(bytecode.RETURN, 0, 2), 0xFFFFFFFF,
			},
			result: []any{false},
		},
		{
			desc: "TEST is true expecting false should increment pc",
			code: []uint32{
				bytecode.IABx(bytecode.LOADBOOL, 0, 1), bytecode.IAB(bytecode.TEST, 0, 0),
				0xFFFFFFFF, bytecode.IAB(bytecode.RETURN, 0, 2),
			},
			result: []any{true},
		},
		{
			desc: "TEST is true expecting true should not increment pc",
			code: []uint32{
				bytecode.IABx(bytecode.LOADBOOL, 0, 1), bytecode.IAB(bytecode.TEST, 0, 1),
				bytecode.IAB(bytecode.RETURN, 0, 2), 0xFFFFFFFF,
			},
			result: []any{true},
		},
		{
			desc: "TEST is false expecting true should increment pc",
			code: []uint32{
				bytecode.IABx(bytecode.LOADBOOL, 0, 0), bytecode.IAB(bytecode.TEST, 0, 1),
				0xFFFFFFFF, bytecode.IAB(bytecode.RETURN, 0, 2),
			},
			result: []any{false},
		},
		{
			desc:      "LEN string",
			constants: []any{"test string"},
			code:      []uint32{bytecode.IABCK(bytecode.LEN, 0, 0, true, 0, false), bytecode.IAB(bytecode.RETURN, 0, 2)},
			result:    []any{int64(11)},
		},
		{
			desc: "LEN table",
			code: []uint32{
				bytecode.IABC(bytecode.NEWTABLE, 0, 3, 0),
				bytecode.IABx(bytecode.LOADI, 1, 21), bytecode.IABx(bytecode.LOADI, 2, 22), bytecode.IABx(bytecode.LOADI, 3, 23),
				bytecode.IABC(bytecode.SETLIST, 0, 4, 1),
				bytecode.IAB(bytecode.LEN, 0, 0),
				bytecode.IAB(bytecode.RETURN, 0, 2),
			},
			result: []any{int64(3)},
		},
		{
			desc:      "LEN others",
			constants: []any{123.0},
			code:      []uint32{bytecode.IABCK(bytecode.LEN, 0, 0, true, 0, false)},
			err:       errors.New("attempt to get length of a number"),
		},
		{
			desc:      "SETTABLE",
			constants: []any{"hello", "world"},
			code: []uint32{
				bytecode.IABC(bytecode.NEWTABLE, 0, 0, 1),
				bytecode.IABCK(bytecode.SETTABLE, 0, 0, true, 1, true),
				bytecode.IAB(bytecode.RETURN, 0, 2),
			},
			result: []any{&Table{val: []any{}, hashtable: map[any]any{"hello": "world"}, keyCache: []any{"hello"}}},
		},
		{
			desc:      "GETTABLE",
			constants: []any{"hello", "world"},
			code: []uint32{
				bytecode.IABC(bytecode.NEWTABLE, 0, 0, 1),
				bytecode.IABCK(bytecode.SETTABLE, 0, 0, true, 1, true),
				bytecode.IABCK(bytecode.GETTABLE, 0, 0, false, 0, true),
				bytecode.IAB(bytecode.RETURN, 0, 2),
			},
			result: []any{"world"},
		},
		{
			desc: "SETLIST with defined count at zero position",
			code: []uint32{
				bytecode.IABC(bytecode.NEWTABLE, 0, 3, 0),
				bytecode.IABx(bytecode.LOADI, 1, 20),
				bytecode.IABx(bytecode.LOADI, 2, 20),
				bytecode.IABx(bytecode.LOADI, 3, 20),
				bytecode.IABC(bytecode.SETLIST, 0, 4, 1),
				bytecode.IAB(bytecode.RETURN, 0, 2),
			},
			result: []any{&Table{val: []any{int64(20), int64(20), int64(20)}, hashtable: map[any]any{}}},
		},
		{
			desc: "SETLIST with defined count at c position",
			code: []uint32{
				bytecode.IABC(bytecode.NEWTABLE, 0, 3, 0),
				bytecode.IABx(bytecode.LOADI, 1, 20),
				bytecode.IABx(bytecode.LOADI, 2, 20),
				bytecode.IABx(bytecode.LOADI, 3, 20),
				bytecode.IABC(bytecode.SETLIST, 0, 4, 3),
				bytecode.IAB(bytecode.RETURN, 0, 2),
			},
			result: []any{&Table{val: []any{nil, nil, int64(20), int64(20), int64(20)}, hashtable: map[any]any{}}},
		},
		{
			desc:      "RETURN all values",
			constants: []any{"don't touch me", "hello", "world"},
			code: []uint32{
				bytecode.IABx(bytecode.LOADK, 0, 0),
				bytecode.IABx(bytecode.LOADK, 1, 1),
				bytecode.IABx(bytecode.LOADK, 2, 2),
				bytecode.IAB(bytecode.RETURN, 1, 0),
			},
			result: []any{"hello", "world"},
		},
		{
			desc:      "RETURN specified return vals",
			constants: []any{"don't touch me", "hello", "world"},
			code: []uint32{
				bytecode.IABx(bytecode.LOADK, 0, 0),
				bytecode.IABx(bytecode.LOADK, 1, 1),
				bytecode.IABx(bytecode.LOADK, 2, 2),
				bytecode.IAB(bytecode.RETURN, 1, 3),
			},
			result: []any{"hello", "world"},
		},
		{
			desc:      "RETURN specified return vals more than provided",
			constants: []any{"don't touch me", "hello", "world"},
			code: []uint32{
				bytecode.IABx(bytecode.LOADK, 0, 0),
				bytecode.IABx(bytecode.LOADK, 1, 1),
				bytecode.IABx(bytecode.LOADK, 2, 2),
				bytecode.IAB(bytecode.RETURN, 1, 5),
			},
			result: []any{"hello", "world", nil, nil},
		},
		{
			desc:      "RETURN no return values",
			constants: []any{"don't touch me", "hello", "world"},
			code:      []uint32{bytecode.IAB(bytecode.RETURN, 0, 1)},
			result:    []any{},
		},
		{
			desc:      "RETURN  return more values than available",
			constants: []any{"don't touch me", "hello", "world"},
			code:      []uint32{bytecode.IAB(bytecode.RETURN, 0, 5)},
			result:    []any{nil, nil, nil, nil},
		},
		{
			desc: "JMP",
			code: []uint32{
				bytecode.IAsBx(bytecode.JMP, 0, 1),
				0xFFFFFFFF, bytecode.IAB(bytecode.RETURN, 0, 1),
			},
			result: []any{},
		},
		{
			desc: "FOR",
			code: []uint32{
				bytecode.IABx(bytecode.LOADI, 0, 0),
				bytecode.IABx(bytecode.LOADI, 1, 1),
				bytecode.IABx(bytecode.LOADI, 2, 10),
				bytecode.IABx(bytecode.LOADI, 3, 1),
				bytecode.IAsBx(bytecode.FORPREP, 1, 2),
				bytecode.IABx(bytecode.LOADI, 4, 1),
				bytecode.IABC(bytecode.ADD, 0, 0, 4),
				bytecode.IAsBx(bytecode.FORLOOP, 1, -3),
				bytecode.IAB(bytecode.RETURN, 0, 2),
			},
			result: []any{int64(10)},
		},
		{
			desc:      "FOR with float",
			constants: []any{float64(0), float64(1), float64(10)},
			code: []uint32{
				bytecode.IABx(bytecode.LOADK, 0, 0),
				bytecode.IABx(bytecode.LOADK, 1, 1),
				bytecode.IABx(bytecode.LOADK, 2, 2),
				bytecode.IABx(bytecode.LOADK, 3, 1),
				bytecode.IAsBx(bytecode.FORPREP, 1, 2),
				bytecode.IABx(bytecode.LOADI, 4, 1),
				bytecode.IABC(bytecode.ADD, 0, 0, 4),
				bytecode.IAsBx(bytecode.FORLOOP, 1, -3),
				bytecode.IAB(bytecode.RETURN, 0, 2),
			},
			result: []any{float64(10)},
		},
		{
			desc: "FOR With zero step",
			code: []uint32{
				bytecode.IABx(bytecode.LOADI, 0, 0),
				bytecode.IABx(bytecode.LOADI, 1, 1),
				bytecode.IABx(bytecode.LOADI, 2, 10),
				bytecode.IABx(bytecode.LOADI, 3, 0),
				bytecode.IAsBx(bytecode.FORPREP, 1, 2),
				bytecode.IABx(bytecode.LOADI, 4, 1),
				bytecode.IABC(bytecode.ADD, 0, 0, 4),
				bytecode.IAsBx(bytecode.FORLOOP, 1, -3),
				bytecode.IAB(bytecode.RETURN, 0, 2),
			},
			err: errors.New("0 step in numerical for"),
		},
		{
			desc: "FOR bad var",
			code: []uint32{
				bytecode.IABx(bytecode.LOADI, 0, 0),
				bytecode.IABx(bytecode.LOADBOOL, 1, 1),
				bytecode.IABx(bytecode.LOADI, 2, 10),
				bytecode.IABx(bytecode.LOADI, 3, 0),
				bytecode.IAsBx(bytecode.FORPREP, 1, 2),
				bytecode.IABx(bytecode.LOADI, 4, 1),
				bytecode.IABC(bytecode.ADD, 0, 0, 4),
				bytecode.IAsBx(bytecode.FORLOOP, 1, -3),
				bytecode.IAB(bytecode.RETURN, 0, 2),
			},
			err: errors.New("non-numeric limit value"),
		},
		{
			desc:      "TFOR",
			constants: []any{"ipairs"},
			code: []uint32{
				bytecode.IABC(bytecode.NEWTABLE, 0, 4, 0),
				bytecode.IABx(bytecode.LOADI, 1, 4),
				bytecode.IABx(bytecode.LOADI, 2, 3),
				bytecode.IABx(bytecode.LOADI, 3, 2),
				bytecode.IABx(bytecode.LOADI, 4, 1),
				bytecode.IABC(bytecode.SETLIST, 0, 5, 1),
				bytecode.IABx(bytecode.LOADI, 1, 0),
				bytecode.IABCK(bytecode.GETTABUP, 2, 0, false, 0, true),
				bytecode.IAB(bytecode.MOVE, 3, 0),
				bytecode.IABC(bytecode.CALL, 2, 2, 4),
				bytecode.IAsBx(bytecode.JMP, 0, 4),
				bytecode.IAB(bytecode.MOVE, 7, 1),
				bytecode.IAB(bytecode.MOVE, 8, 6),
				bytecode.IABC(bytecode.ADD, 7, 7, 8),
				bytecode.IAB(bytecode.MOVE, 1, 7),
				bytecode.IAB(bytecode.TFORCALL, 2, 2),
				bytecode.IAsBx(bytecode.TFORLOOP, 3, -6),
				bytecode.IAB(bytecode.MOVE, 2, 1),
				bytecode.IAB(bytecode.RETURN, 2, 2),
			},
			result: []any{int64(10)},
		},
		{
			desc:      "SELF and CLOSURE and CALL",
			constants: []any{"test"},
			code: []uint32{
				bytecode.IABC(bytecode.NEWTABLE, 0, 0, 1),
				bytecode.IABx(bytecode.CLOSURE, 1, 0),
				bytecode.IABCK(bytecode.SETTABLE, 0, 0, true, 1, false),
				bytecode.IAB(bytecode.MOVE, 1, 0),
				bytecode.IABCK(bytecode.SELF, 1, 1, false, 0, true),
				bytecode.IABC(bytecode.TAILCALL, 1, 2, 0),
				bytecode.IAB(bytecode.RETURN, 0, 1),
			},
			fntbl: []*parse.FnProto{{ByteCodes: []uint32{
				bytecode.IABx(bytecode.LOADI, 0, 42),
				bytecode.IAB(bytecode.RETURN, 0, 2),
			}}},
			result: []any{int64(42)},
		},
		{
			desc:      "GETTABUP and SETTABUP",
			constants: []any{"test"},
			code: []uint32{
				bytecode.IAB(bytecode.LOADI, 0, 42),
				bytecode.IABx(bytecode.CLOSURE, 1, 0),
				bytecode.IABCK(bytecode.SETTABUP, 0, 0, true, 1, false),
				bytecode.IABCK(bytecode.GETTABUP, 1, 0, false, 0, true),
				bytecode.IABC(bytecode.CALL, 1, 1, 2),
				bytecode.IAB(bytecode.MOVE, 2, 1),
				bytecode.IAB(bytecode.RETURN, 2, 2),
			},
			fntbl: []*parse.FnProto{{
				ByteCodes: []uint32{
					bytecode.IAB(bytecode.GETUPVAL, 0, 0),
					bytecode.IAB(bytecode.RETURN, 0, 2),
				},
				UpIndexes: []parse.Upindex{{Name: "a", FromStack: true, Index: 0}},
			}},
			result: []any{int64(42)},
		},
		{
			desc: "GETUPVAL and SETUPVAL closed upval",
			code: []uint32{
				bytecode.IABx(bytecode.CLOSURE, 0, 0),
				bytecode.IABC(bytecode.CALL, 0, 1, 2),
				bytecode.IABC(bytecode.CALL, 0, 1, 2),
				bytecode.IAB(bytecode.RETURN, 0, 2),
			},
			fntbl: []*parse.FnProto{{
				ByteCodes: []uint32{
					bytecode.IAB(bytecode.LOADI, 0, 42),
					bytecode.IABx(bytecode.CLOSURE, 1, 0),
					bytecode.IAB(bytecode.CLOSE, 0, 0),
					bytecode.IAB(bytecode.RETURN, 1, 2),
				},
				FnTable: []*parse.FnProto{{
					ByteCodes: []uint32{
						bytecode.IAB(bytecode.LOADI, 0, 32),
						bytecode.IAB(bytecode.SETUPVAL, 0, 0),
						bytecode.IAB(bytecode.GETUPVAL, 0, 0),
						bytecode.IAB(bytecode.RETURN, 0, 2),
					},
					UpIndexes: []parse.Upindex{{Name: "a", FromStack: true, Index: 0}},
				}},
			}},
			result: []any{int64(32)},
		},
		{
			desc: "VARARG all args",
			code: []uint32{
				bytecode.IABx(bytecode.CLOSURE, 0, 0),
				bytecode.IABx(bytecode.LOADI, 1, 1),
				bytecode.IABx(bytecode.LOADI, 2, 2),
				bytecode.IABx(bytecode.LOADI, 3, 3),
				bytecode.IABC(bytecode.CALL, 0, 4, 2),
				bytecode.IAB(bytecode.RETURN, 0, 2),
			},
			fntbl: []*parse.FnProto{{
				ByteCodes: []uint32{
					bytecode.IABC(bytecode.NEWTABLE, 0, 1, 0),
					bytecode.IAB(bytecode.VARARG, 1, 0),
					bytecode.IABC(bytecode.SETLIST, 0, 0, 1),
					bytecode.IAB(bytecode.RETURN, 0, 2),
				},
			}},
			result: []any{&Table{
				val:       []any{int64(1), int64(2), int64(3)},
				hashtable: map[any]any{},
			}},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()

			vm := New(context.Background(), nil)
			value, err := vm.Eval(&parse.FnProto{
				Constants: tc.constants,
				ByteCodes: tc.code,
				FnTable:   tc.fntbl,
			})
			if tc.err == nil {
				require.NoError(t, err)
				assert.Equal(t, tc.result, value, "result value not equal")
			} else {
				require.ErrorContains(t, err, tc.err.Error())
				require.Nil(t, value)
			}
		})
	}
}

func TestVM_call(t *testing.T) {
	t.Parallel()
	t.Run("Go Func call", func(t *testing.T) {
		t.Parallel()
		fn := Fn("testFn", func(_ *VM, params []any) ([]any, error) {
			assert.Len(t, params, 3)
			return append(params, int64(42)), nil
		})

		vm := New(context.Background(), nil)
		vm.callDepth = 20
		vm.top = 32

		res, err := vm.call(fn, []any{"one", int64(2), float64(3)})
		require.NoError(t, err)
		assert.Equal(t, []any{"one", int64(2), float64(3), int64(42)}, res)
		assert.Equal(t, int64(20), vm.callDepth)
		assert.Equal(t, int64(32), vm.top)
	})

	t.Run("closure call", func(t *testing.T) {
		t.Parallel()

		fn := &Closure{
			val: &parse.FnProto{
				Constants: []any{float64(32), float64(112), "Don't touch me"},
				ByteCodes: []uint32{
					bytecode.IABx(bytecode.LOADI, 0, 1274), bytecode.IABx(bytecode.LOADI, 1, 72), bytecode.IABC(bytecode.ADD, 0, 0, 1),
					bytecode.IABx(bytecode.LOADK, 1, 0), bytecode.IABx(bytecode.LOADK, 2, 1), bytecode.IABC(bytecode.ADD, 1, 1, 2),
					bytecode.IABx(bytecode.LOADI, 2, 42), bytecode.IABx(bytecode.LOADK, 3, 0), bytecode.IABC(bytecode.ADD, 2, 2, 3),
					bytecode.IABx(bytecode.LOADK, 3, 0), bytecode.IABx(bytecode.LOADI, 4, 99), bytecode.IABC(bytecode.ADD, 3, 3, 4),
					bytecode.IAB(bytecode.RETURN, 0, 0),
				},
			},
		}

		vm := New(context.Background(), nil)
		vm.callDepth = 20
		vm.top = 32

		res, err := vm.call(fn, []any{})
		require.NoError(t, err)
		assert.Equal(t, []any{int64(1346), float64(144), float64(74), float64(131), int64(99)}, res)
		assert.Equal(t, int64(20), vm.callDepth)
		assert.Equal(t, int64(32), vm.top)
	})

	t.Run("Trying to call something not callable", func(t *testing.T) {
		t.Parallel()
		vm := New(context.Background(), nil)
		_, err := vm.call(int64(22), []any{})
		assert.Error(t, err)
	})
}

func TestEnsureSize(t *testing.T) {
	t.Parallel()
	a := []string{}
	assert.Empty(t, a)
	ensureSize(&a, 5)
	assert.Len(t, a, 6)
	a[5] = "did it"
}
