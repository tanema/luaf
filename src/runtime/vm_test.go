package runtime

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tanema/luaf/src/bytecode"
	"github.com/tanema/luaf/src/parse"
)

func TestVM_Eval(t *testing.T) {
	t.Parallel()
	t.Run("MOVE", func(t *testing.T) {
		t.Parallel()
		fnproto := &parse.FnProto{
			Constants: []any{int64(23)},
			ByteCodes: []uint32{bytecode.IABx(bytecode.LOADK, 0, 0), bytecode.IAB(bytecode.MOVE, 1, 0)},
		}
		vm, value, err := tEval(fnproto)
		require.NoError(t, err)
		assert.Nil(t, value)
		assert.Equal(t, int64(23), vm.Stack[1])
		assert.Equal(t, int64(23), vm.Stack[2])
	})

	t.Run("LOADK", func(t *testing.T) {
		t.Parallel()
		fnproto := &parse.FnProto{
			Constants: []any{int64(23)},
			ByteCodes: []uint32{bytecode.IABx(bytecode.LOADK, 0, 0)},
		}
		vm, value, err := tEval(fnproto)
		require.NoError(t, err)
		assert.Nil(t, value)
		assert.Equal(t, int64(23), vm.Stack[1])
	})

	t.Run("LOADBOOL", func(t *testing.T) {
		t.Parallel()
		fnproto := &parse.FnProto{ByteCodes: []uint32{
			bytecode.IABx(bytecode.LOADBOOL, 0, 1),
			bytecode.IABC(bytecode.LOADBOOL, 1, 0, 1),
		}}
		vm, value, err := tEval(fnproto)
		require.NoError(t, err)
		assert.Nil(t, value)
		assert.Equal(t, true, vm.Stack[1])
		assert.Equal(t, false, vm.Stack[2])
	})

	t.Run("LOADI", func(t *testing.T) {
		t.Parallel()
		fnproto := &parse.FnProto{ByteCodes: []uint32{bytecode.IABx(bytecode.LOADI, 0, 1274)}}
		vm, value, err := tEval(fnproto)
		require.NoError(t, err)
		assert.Nil(t, value)
		assert.Equal(t, int64(1274), vm.Stack[1])
	})

	t.Run("LOADI EXTAARG", func(t *testing.T) {
		t.Parallel()
		t.Skip("TODO")
	})

	t.Run("LOADNil", func(t *testing.T) {
		t.Parallel()
		fnproto := &parse.FnProto{ByteCodes: []uint32{bytecode.IABx(bytecode.LOADNIL, 0, 8)}}
		vm, value, err := tEval(fnproto)
		require.NoError(t, err)
		assert.Nil(t, value)
		for i := 1; i < 9; i++ {
			assert.Nil(t, vm.Stack[i])
		}
	})

	t.Run("ADD", func(t *testing.T) {
		t.Parallel()
		fnproto := &parse.FnProto{
			Constants: []any{float64(32), float64(112), "Don't touch me"},
			ByteCodes: []uint32{
				bytecode.IABx(bytecode.LOADI, 0, 1274), bytecode.IABx(bytecode.LOADI, 1, 72), bytecode.IABC(bytecode.ADD, 0, 0, 1),
				bytecode.IABx(bytecode.LOADK, 1, 0), bytecode.IABx(bytecode.LOADK, 2, 1), bytecode.IABC(bytecode.ADD, 1, 1, 2),
				bytecode.IABx(bytecode.LOADI, 2, 42), bytecode.IABx(bytecode.LOADK, 3, 0), bytecode.IABC(bytecode.ADD, 2, 2, 3),
				bytecode.IABx(bytecode.LOADK, 3, 0), bytecode.IABx(bytecode.LOADI, 4, 99), bytecode.IABC(bytecode.ADD, 3, 3, 4),
				bytecode.IABx(bytecode.LOADK, 4, 2), bytecode.IABx(bytecode.LOADI, 5, 0), bytecode.IABC(bytecode.ADD, 4, 4, 5),
			},
		}
		vm, value, err := tEval(fnproto)
		require.Error(t, err)
		assert.Nil(t, value)
		assert.Equal(t, int64(1346), vm.Stack[1])
		assert.InEpsilon(t, float64(144), vm.Stack[2], 0)
		assert.InEpsilon(t, float64(74), vm.Stack[3], 0)
		assert.InEpsilon(t, float64(131), vm.Stack[4], 0)
	})

	t.Run("SUB", func(t *testing.T) {
		t.Parallel()
		fnproto := &parse.FnProto{
			Constants: []any{float64(32), float64(112), "Don't touch me"},
			ByteCodes: []uint32{
				bytecode.IABx(bytecode.LOADI, 0, 1274), bytecode.IABx(bytecode.LOADI, 1, 72), bytecode.IABC(bytecode.SUB, 0, 0, 1),
				bytecode.IABx(bytecode.LOADK, 1, 0), bytecode.IABx(bytecode.LOADK, 2, 1), bytecode.IABC(bytecode.SUB, 1, 1, 2),
				bytecode.IABx(bytecode.LOADI, 2, 42), bytecode.IABx(bytecode.LOADK, 3, 0), bytecode.IABC(bytecode.SUB, 2, 2, 3),
				bytecode.IABx(bytecode.LOADK, 3, 0), bytecode.IABx(bytecode.LOADI, 4, 99), bytecode.IABC(bytecode.SUB, 3, 3, 4),
				bytecode.IABx(bytecode.LOADK, 4, 2), bytecode.IABx(bytecode.LOADI, 5, 0), bytecode.IABC(bytecode.SUB, 4, 4, 5),
			},
		}
		vm, value, err := tEval(fnproto)
		require.Error(t, err)
		assert.Nil(t, value)
		assert.Equal(t, int64(1202), vm.Stack[1])
		assert.InEpsilon(t, float64(-80), vm.Stack[2], 0)
		assert.InEpsilon(t, float64(10), vm.Stack[3], 0)
		assert.InEpsilon(t, float64(-67), vm.Stack[4], 0)
	})

	t.Run("MUL", func(t *testing.T) {
		t.Parallel()
		fnproto := &parse.FnProto{
			Constants: []any{float64(32), float64(112), "Don't touch me"},
			ByteCodes: []uint32{
				bytecode.IABx(bytecode.LOADI, 0, 1274), bytecode.IABx(bytecode.LOADI, 1, 72), bytecode.IABC(bytecode.MUL, 0, 0, 1),
				bytecode.IABx(bytecode.LOADK, 1, 0), bytecode.IABx(bytecode.LOADK, 2, 1), bytecode.IABC(bytecode.MUL, 1, 1, 2),
				bytecode.IABx(bytecode.LOADI, 2, 42), bytecode.IABx(bytecode.LOADK, 3, 0), bytecode.IABC(bytecode.MUL, 2, 2, 3),
				bytecode.IABx(bytecode.LOADK, 3, 0), bytecode.IABx(bytecode.LOADI, 4, 99), bytecode.IABC(bytecode.MUL, 3, 3, 4),
				bytecode.IABx(bytecode.LOADK, 4, 2), bytecode.IABx(bytecode.LOADI, 5, 0), bytecode.IABC(bytecode.MUL, 4, 4, 5),
			},
		}
		vm, value, err := tEval(fnproto)
		require.Error(t, err)
		assert.Nil(t, value)
		assert.Equal(t, int64(91728), vm.Stack[1])
		assert.InEpsilon(t, float64(3584), vm.Stack[2], 0)
		assert.InEpsilon(t, float64(1344), vm.Stack[3], 0)
		assert.InEpsilon(t, float64(3168), vm.Stack[4], 0)
	})

	t.Run("DIV", func(t *testing.T) {
		t.Parallel()
		fnproto := &parse.FnProto{
			Constants: []any{float64(112), float64(32), "Don't touch me"},
			ByteCodes: []uint32{
				bytecode.IABx(bytecode.LOADI, 0, 1274), bytecode.IABx(bytecode.LOADI, 1, 10), bytecode.IABC(bytecode.DIV, 0, 0, 1),
				bytecode.IABx(bytecode.LOADK, 1, 0), bytecode.IABx(bytecode.LOADK, 2, 1), bytecode.IABC(bytecode.DIV, 1, 1, 2),
				bytecode.IABx(bytecode.LOADI, 2, 42), bytecode.IABx(bytecode.LOADK, 3, 0), bytecode.IABC(bytecode.DIV, 2, 2, 3),
				bytecode.IABx(bytecode.LOADK, 3, 0), bytecode.IABx(bytecode.LOADI, 4, 1), bytecode.IABC(bytecode.DIV, 3, 3, 4),
				bytecode.IABx(bytecode.LOADK, 4, 2), bytecode.IABx(bytecode.LOADI, 5, 0), bytecode.IABC(bytecode.DIV, 4, 4, 5),
			},
		}
		vm, value, err := tEval(fnproto)
		require.Error(t, err)
		assert.Nil(t, value)
		assert.InEpsilon(t, float64(127.4), vm.Stack[1], 0)
		assert.InEpsilon(t, float64(3.5), vm.Stack[2], 0)
		assert.InEpsilon(t, float64(0.375), vm.Stack[3], 0)
		assert.InEpsilon(t, float64(112), vm.Stack[4], 0)
	})

	t.Run("MOD", func(t *testing.T) {
		t.Parallel()
		fnproto := &parse.FnProto{
			Constants: []any{float64(112), float64(32), "Don't touch me"},
			ByteCodes: []uint32{
				bytecode.IABx(bytecode.LOADI, 0, 1274), bytecode.IABx(bytecode.LOADI, 1, 72), bytecode.IABC(bytecode.MOD, 0, 0, 1),
				bytecode.IABx(bytecode.LOADK, 1, 0), bytecode.IABx(bytecode.LOADK, 2, 1), bytecode.IABC(bytecode.MOD, 1, 1, 2),
				bytecode.IABx(bytecode.LOADI, 2, 42), bytecode.IABx(bytecode.LOADK, 3, 0), bytecode.IABC(bytecode.MOD, 2, 2, 3),
				bytecode.IABx(bytecode.LOADK, 3, 0), bytecode.IABx(bytecode.LOADI, 4, 1), bytecode.IABC(bytecode.MOD, 3, 3, 4),
				bytecode.IABx(bytecode.LOADK, 4, 2), bytecode.IABx(bytecode.LOADI, 5, 0), bytecode.IABC(bytecode.MOD, 4, 4, 5),
			},
		}
		vm, value, err := tEval(fnproto)
		require.Error(t, err)
		assert.Nil(t, value)
		assert.Equal(t, int64(50), vm.Stack[1])
		assert.InEpsilon(t, float64(16), vm.Stack[2], 0)
		assert.InEpsilon(t, float64(42), vm.Stack[3], 0)
		assert.Equal(t, float64(0), vm.Stack[4]) //nolint:testifylint
	})

	t.Run("POW", func(t *testing.T) {
		t.Parallel()
		fnproto := &parse.FnProto{
			Constants: []any{float64(2), float64(3), "Don't touch me"},
			ByteCodes: []uint32{
				bytecode.IABx(bytecode.LOADI, 0, 2),
				bytecode.IABx(bytecode.LOADI, 1, 4),
				bytecode.IABC(bytecode.POW, 0, 0, 1),
				bytecode.IABx(bytecode.LOADK, 1, 0),
				bytecode.IABx(bytecode.LOADK, 2, 1),
				bytecode.IABC(bytecode.POW, 1, 1, 2),
				bytecode.IABx(bytecode.LOADI, 2, 2),
				bytecode.IABx(bytecode.LOADK, 3, 0),
				bytecode.IABC(bytecode.POW, 2, 2, 3),
				bytecode.IABx(bytecode.LOADK, 3, 0),
				bytecode.IABx(bytecode.LOADI, 4, 1),
				bytecode.IABC(bytecode.POW, 3, 3, 4),
				bytecode.IABx(bytecode.LOADK, 4, 2),
				bytecode.IABx(bytecode.LOADI, 5, 0),
				bytecode.IABC(bytecode.POW, 4, 4, 5),
			},
		}
		vm, value, err := tEval(fnproto)
		require.Error(t, err)
		assert.Nil(t, value)
		assert.InEpsilon(t, float64(16), vm.Stack[1], 0)
		assert.InEpsilon(t, float64(8), vm.Stack[2], 0)
		assert.InEpsilon(t, float64(4), vm.Stack[3], 0)
		assert.InEpsilon(t, float64(2), vm.Stack[4], 0)
	})

	t.Run("IDIV", func(t *testing.T) {
		t.Parallel()
		fnproto := &parse.FnProto{
			Constants: []any{float64(112), float64(32), "Don't touch me"},
			ByteCodes: []uint32{
				bytecode.IABx(bytecode.LOADI, 0, 1274),
				bytecode.IABx(bytecode.LOADI, 1, 72),
				bytecode.IABC(bytecode.IDIV, 0, 0, 1),
				bytecode.IABx(bytecode.LOADK, 1, 0),
				bytecode.IABx(bytecode.LOADK, 2, 1),
				bytecode.IABC(bytecode.IDIV, 1, 1, 2),
				bytecode.IABx(bytecode.LOADI, 2, 42),
				bytecode.IABx(bytecode.LOADK, 3, 0),
				bytecode.IABC(bytecode.IDIV, 2, 2, 3),
				bytecode.IABx(bytecode.LOADK, 3, 0),
				bytecode.IABx(bytecode.LOADI, 4, 1),
				bytecode.IABC(bytecode.IDIV, 3, 3, 4),
				bytecode.IABx(bytecode.LOADK, 4, 2),
				bytecode.IABx(bytecode.LOADI, 5, 0),
				bytecode.IABC(bytecode.IDIV, 4, 4, 5),
			},
		}
		vm, value, err := tEval(fnproto)
		require.Error(t, err)
		assert.Nil(t, value)
		assert.Equal(t, int64(17), vm.Stack[1])
		assert.InEpsilon(t, float64(3), vm.Stack[2], 0)
		assert.Equal(t, float64(0), vm.Stack[3]) //nolint:testifylint
		assert.InEpsilon(t, float64(112), vm.Stack[4], 0)
	})

	t.Run("BAND", func(t *testing.T) {
		t.Parallel()
		fnproto := &parse.FnProto{
			Constants: []any{float64(2), float64(3), "Don't touch me"},
			ByteCodes: []uint32{
				bytecode.IABx(bytecode.LOADI, 0, 2),
				bytecode.IABx(bytecode.LOADI, 1, 4),
				bytecode.IABC(bytecode.BAND, 0, 0, 1),
				bytecode.IABx(bytecode.LOADK, 1, 0),
				bytecode.IABx(bytecode.LOADK, 2, 1),
				bytecode.IABC(bytecode.BAND, 1, 1, 2),
				bytecode.IABx(bytecode.LOADI, 2, 2),
				bytecode.IABx(bytecode.LOADK, 3, 0),
				bytecode.IABC(bytecode.BAND, 2, 2, 3),
				bytecode.IABx(bytecode.LOADK, 3, 0),
				bytecode.IABx(bytecode.LOADI, 4, 1),
				bytecode.IABC(bytecode.BAND, 3, 3, 4),
				bytecode.IABx(bytecode.LOADK, 4, 2),
				bytecode.IABx(bytecode.LOADI, 5, 0),
				bytecode.IABC(bytecode.BAND, 4, 4, 5),
			},
		}
		vm, value, err := tEval(fnproto)
		require.Error(t, err)
		assert.Nil(t, value)
		assert.Equal(t, int64(0), vm.Stack[1])
		assert.Equal(t, int64(2), vm.Stack[2])
		assert.Equal(t, int64(2), vm.Stack[3])
		assert.Equal(t, int64(0), vm.Stack[4])
	})

	t.Run("BOR", func(t *testing.T) {
		t.Parallel()
		fnproto := &parse.FnProto{
			Constants: []any{float64(2), float64(3), "Don't touch me"},
			ByteCodes: []uint32{
				bytecode.IABx(bytecode.LOADI, 0, 2), bytecode.IABx(bytecode.LOADI, 1, 4), bytecode.IABC(bytecode.BOR, 0, 0, 1),
				bytecode.IABx(bytecode.LOADK, 1, 0), bytecode.IABx(bytecode.LOADK, 2, 1), bytecode.IABC(bytecode.BOR, 1, 1, 2),
				bytecode.IABx(bytecode.LOADI, 2, 2), bytecode.IABx(bytecode.LOADK, 3, 0), bytecode.IABC(bytecode.BOR, 2, 2, 3),
				bytecode.IABx(bytecode.LOADK, 3, 0), bytecode.IABx(bytecode.LOADI, 4, 1), bytecode.IABC(bytecode.BOR, 3, 3, 4),
				bytecode.IABx(bytecode.LOADK, 4, 2), bytecode.IABx(bytecode.LOADI, 5, 0), bytecode.IABC(bytecode.BOR, 4, 4, 5),
			},
		}
		vm, value, err := tEval(fnproto)
		require.Error(t, err)
		assert.Nil(t, value)
		assert.Equal(t, int64(6), vm.Stack[1])
		assert.Equal(t, int64(3), vm.Stack[2])
		assert.Equal(t, int64(2), vm.Stack[3])
		assert.Equal(t, int64(3), vm.Stack[4])
	})

	t.Run("BXOR", func(t *testing.T) {
		t.Parallel()
		fnproto := &parse.FnProto{
			Constants: []any{float64(2), float64(3), "Don't touch me"},
			ByteCodes: []uint32{
				bytecode.IABx(bytecode.LOADI, 0, 2), bytecode.IABx(bytecode.LOADI, 1, 4), bytecode.IABC(bytecode.BXOR, 0, 0, 1),
				bytecode.IABx(bytecode.LOADK, 1, 0), bytecode.IABx(bytecode.LOADK, 2, 1), bytecode.IABC(bytecode.BXOR, 1, 1, 2),
				bytecode.IABx(bytecode.LOADI, 2, 2), bytecode.IABx(bytecode.LOADK, 3, 0), bytecode.IABC(bytecode.BXOR, 2, 2, 3),
				bytecode.IABx(bytecode.LOADK, 3, 0), bytecode.IABx(bytecode.LOADI, 4, 1), bytecode.IABC(bytecode.BXOR, 3, 3, 4),
				bytecode.IABx(bytecode.LOADK, 4, 2), bytecode.IABx(bytecode.LOADI, 5, 0), bytecode.IABC(bytecode.BXOR, 4, 4, 5),
			},
		}
		vm, value, err := tEval(fnproto)
		require.Error(t, err)
		assert.Nil(t, value)
		assert.Equal(t, int64(6), vm.Stack[1])
		assert.Equal(t, int64(1), vm.Stack[2])
		assert.Equal(t, int64(0), vm.Stack[3])
		assert.Equal(t, int64(3), vm.Stack[4])
	})

	t.Run("SHL", func(t *testing.T) {
		t.Parallel()
		fnproto := &parse.FnProto{
			Constants: []any{float64(2), float64(3), "Don't touch me"},
			ByteCodes: []uint32{
				bytecode.IABx(bytecode.LOADI, 0, 2), bytecode.IABx(bytecode.LOADI, 1, 4), bytecode.IABC(bytecode.SHL, 0, 0, 1),
				bytecode.IABx(bytecode.LOADK, 1, 0), bytecode.IABx(bytecode.LOADK, 2, 1), bytecode.IABC(bytecode.SHL, 1, 1, 2),
				bytecode.IABx(bytecode.LOADI, 2, 2), bytecode.IABx(bytecode.LOADK, 3, 0), bytecode.IABC(bytecode.SHL, 2, 2, 3),
				bytecode.IABx(bytecode.LOADK, 3, 0), bytecode.IABx(bytecode.LOADI, 4, 1), bytecode.IABC(bytecode.SHL, 3, 3, 4),
				bytecode.IABx(bytecode.LOADK, 4, 2), bytecode.IABx(bytecode.LOADI, 5, 0), bytecode.IABC(bytecode.SHL, 4, 4, 5),
			},
		}
		vm, value, err := tEval(fnproto)
		require.Error(t, err)
		assert.Nil(t, value)
		assert.Equal(t, int64(32), vm.Stack[1])
		assert.Equal(t, int64(16), vm.Stack[2])
		assert.Equal(t, int64(8), vm.Stack[3])
		assert.Equal(t, int64(4), vm.Stack[4])
	})

	t.Run("SHR", func(t *testing.T) {
		t.Parallel()
		fnproto := &parse.FnProto{
			Constants: []any{float64(100), float64(1), "Don't touch me"},
			ByteCodes: []uint32{
				bytecode.IABx(bytecode.LOADI, 0, 100), bytecode.IABx(bytecode.LOADI, 1, 1), bytecode.IABC(bytecode.SHR, 0, 0, 1),
				bytecode.IABx(bytecode.LOADK, 1, 0), bytecode.IABx(bytecode.LOADK, 2, 1), bytecode.IABC(bytecode.SHR, 1, 1, 2),
				bytecode.IABx(bytecode.LOADI, 2, 500), bytecode.IABx(bytecode.LOADK, 3, 1), bytecode.IABC(bytecode.SHR, 2, 2, 3),
				bytecode.IABx(bytecode.LOADK, 3, 0), bytecode.IABx(bytecode.LOADI, 4, 1), bytecode.IABC(bytecode.SHR, 3, 3, 4),
				bytecode.IABx(bytecode.LOADK, 4, 2), bytecode.IABx(bytecode.LOADI, 5, 0), bytecode.IABC(bytecode.SHR, 4, 4, 5),
			},
		}
		vm, value, err := tEval(fnproto)
		require.Error(t, err)
		assert.Nil(t, value)
		assert.Equal(t, int64(50), vm.Stack[1])
		assert.Equal(t, int64(50), vm.Stack[2])
		assert.Equal(t, int64(250), vm.Stack[3])
		assert.Equal(t, int64(50), vm.Stack[4])
	})

	t.Run("UNM", func(t *testing.T) {
		t.Parallel()
		fnproto := &parse.FnProto{
			Constants: []any{float64(200), "Don't touch me"},
			ByteCodes: []uint32{
				bytecode.IABx(bytecode.LOADI, 0, 100), bytecode.IAB(bytecode.UNM, 0, 0),
				bytecode.IABx(bytecode.LOADK, 1, 0), bytecode.IAB(bytecode.UNM, 1, 1),
				bytecode.IABx(bytecode.LOADK, 2, 1), bytecode.IAB(bytecode.UNM, 2, 2),
			},
		}
		vm, value, err := tEval(fnproto)
		require.Error(t, err)
		assert.Nil(t, value)
		assert.Equal(t, int64(-100), vm.Stack[1])
		assert.InEpsilon(t, float64(-200), vm.Stack[2], 0)
	})

	t.Run("BNOT", func(t *testing.T) {
		t.Parallel()
		fnproto := &parse.FnProto{
			Constants: []any{float64(100), "Don't touch me"},
			ByteCodes: []uint32{
				bytecode.IABx(bytecode.LOADI, 0, 100), bytecode.IAB(bytecode.BNOT, 0, 0),
				bytecode.IABx(bytecode.LOADK, 1, 0), bytecode.IAB(bytecode.BNOT, 1, 1),
			},
		}
		vm, value, err := tEval(fnproto)
		require.NoError(t, err)
		assert.Nil(t, value)
		assert.Equal(t, int64(-101), vm.Stack[1])
		assert.Equal(t, int64(-101), vm.Stack[2])
	})

	t.Run("NOT", func(t *testing.T) {
		t.Parallel()
		fnproto := &parse.FnProto{
			Constants: []any{float64(0), float64(1), "Don't touch me"},
			ByteCodes: []uint32{
				bytecode.IABx(bytecode.LOADI, 0, 0), bytecode.IAB(bytecode.NOT, 0, 0), // integer == 0
				bytecode.IABx(bytecode.LOADI, 1, 1), bytecode.IAB(bytecode.NOT, 1, 1), // integer != 0
				bytecode.IABx(bytecode.LOADK, 2, 0), bytecode.IAB(bytecode.NOT, 2, 2), // float == 0
				bytecode.IABx(bytecode.LOADK, 3, 1), bytecode.IAB(bytecode.NOT, 3, 3), // float != 0
				bytecode.IABx(bytecode.LOADNIL, 4, 1), bytecode.IAB(bytecode.NOT, 4, 4), // nil
				bytecode.IAB(bytecode.LOADBOOL, 5, 1), bytecode.IAB(bytecode.NOT, 5, 5), // true
				bytecode.IAB(bytecode.LOADBOOL, 6, 0), bytecode.IAB(bytecode.NOT, 6, 6), // false
				bytecode.IAB(bytecode.LOADK, 7, 2), bytecode.IAB(bytecode.NOT, 7, 7), // string
			},
		}
		vm, value, err := tEval(fnproto)
		require.NoError(t, err)
		assert.Nil(t, value)
		assert.Equal(t, false, vm.Stack[1])
		assert.Equal(t, false, vm.Stack[2])
		assert.Equal(t, false, vm.Stack[3])
		assert.Equal(t, false, vm.Stack[4])
		assert.Equal(t, true, vm.Stack[5])
		assert.Equal(t, false, vm.Stack[6])
		assert.Equal(t, true, vm.Stack[7])
		assert.Equal(t, false, vm.Stack[8])
	})

	t.Run("CONCAT", func(t *testing.T) {
		t.Parallel()
		fnproto := &parse.FnProto{
			Constants: []any{float64(200), "Don't touch me"},
			ByteCodes: []uint32{
				bytecode.IABx(bytecode.LOADI, 0, 100), bytecode.IABx(bytecode.LOADK, 1, 0), bytecode.IABx(bytecode.LOADK, 2, 1),
				bytecode.IABC(bytecode.CONCAT, 0, 0, 2),
			},
		}
		vm, value, err := tEval(fnproto)
		require.NoError(t, err)
		assert.Nil(t, value)
		assert.Equal(t, "100200Don't touch me", vm.Stack[1])
	})

	t.Run("JMP", func(t *testing.T) {
		t.Parallel()
		fnproto := &parse.FnProto{
			ByteCodes: []uint32{bytecode.IAsBx(bytecode.JMP, 0, 20)},
		}
		_, value, err := tEval(fnproto)
		require.NoError(t, err)
		assert.Nil(t, value)
	})

	t.Run("JMP close brokers", func(t *testing.T) {
		t.Parallel()
		t.Skip("TODO")
	})

	t.Run("EQ", func(t *testing.T) {
		t.Parallel()

		t.Run("is false expecting false should not increment pc", func(t *testing.T) {
			t.Parallel()
			fnproto := &parse.FnProto{ByteCodes: []uint32{
				bytecode.IABx(bytecode.LOADI, 0, 2),
				bytecode.IABx(bytecode.LOADI, 1, 1),
				bytecode.IABC(bytecode.EQ, 0, 0, 1),
			}}
			vm := New(context.Background(), nil)
			f, err := testEval(vm, fnproto)
			require.NoError(t, err)
			assert.Equal(t, int64(3), f.pc)
		})
		t.Run("is true expecting false should increment pc", func(t *testing.T) {
			t.Parallel()
			fnproto := &parse.FnProto{ByteCodes: []uint32{
				bytecode.IABx(bytecode.LOADI, 0, 1),
				bytecode.IABx(bytecode.LOADI, 1, 1),
				bytecode.IABC(bytecode.EQ, 0, 0, 1),
			}}
			vm := New(context.Background(), nil)
			f, err := testEval(vm, fnproto)
			require.NoError(t, err)
			assert.Equal(t, int64(4), f.pc)
		})
		t.Run("is true expecting true should not increment pc", func(t *testing.T) {
			t.Parallel()
			fnproto := &parse.FnProto{ByteCodes: []uint32{
				bytecode.IABx(bytecode.LOADI, 0, 1),
				bytecode.IABx(bytecode.LOADI, 1, 1),
				bytecode.IABC(bytecode.EQ, 1, 0, 1),
			}}
			vm := New(context.Background(), nil)
			f, err := testEval(vm, fnproto)
			require.NoError(t, err)
			assert.Equal(t, int64(3), f.pc)
		})
		t.Run("is false expecting true should increment pc", func(t *testing.T) {
			t.Parallel()
			fnproto := &parse.FnProto{ByteCodes: []uint32{
				bytecode.IABx(bytecode.LOADI, 0, 2),
				bytecode.IABx(bytecode.LOADI, 1, 1),
				bytecode.IABC(bytecode.EQ, 1, 0, 1),
			}}
			vm := New(context.Background(), nil)
			f, err := testEval(vm, fnproto)
			require.NoError(t, err)
			assert.Equal(t, int64(4), f.pc)
		})
	})

	t.Run("LT", func(t *testing.T) {
		t.Parallel()
		t.Run("is false expecting false should not increment pc", func(t *testing.T) {
			t.Parallel()
			fnproto := &parse.FnProto{ByteCodes: []uint32{
				bytecode.IABx(bytecode.LOADI, 0, 2),
				bytecode.IABx(bytecode.LOADI, 1, 1),
				bytecode.IABC(bytecode.LT, 0, 0, 1),
			}}
			vm := New(context.Background(), nil)
			f, err := testEval(vm, fnproto)
			require.NoError(t, err)
			assert.Equal(t, int64(3), f.pc)
		})
		t.Run("is true expecting false should increment pc", func(t *testing.T) {
			t.Parallel()
			fnproto := &parse.FnProto{ByteCodes: []uint32{
				bytecode.IABx(bytecode.LOADI, 0, 0),
				bytecode.IABx(bytecode.LOADI, 1, 1),
				bytecode.IABC(bytecode.LT, 0, 0, 1),
			}}
			vm := New(context.Background(), nil)
			f, err := testEval(vm, fnproto)
			require.NoError(t, err)
			assert.Equal(t, int64(4), f.pc)
		})
		t.Run("is true expecting true should not increment pc", func(t *testing.T) {
			t.Parallel()
			fnproto := &parse.FnProto{ByteCodes: []uint32{
				bytecode.IABx(bytecode.LOADI, 0, 0),
				bytecode.IABx(bytecode.LOADI, 1, 1),
				bytecode.IABC(bytecode.LT, 1, 0, 1),
			}}
			vm := New(context.Background(), nil)
			f, err := testEval(vm, fnproto)
			require.NoError(t, err)
			assert.Equal(t, int64(3), f.pc)
		})
		t.Run("is false expecting true should increment pc", func(t *testing.T) {
			t.Parallel()
			fnproto := &parse.FnProto{ByteCodes: []uint32{
				bytecode.IABx(bytecode.LOADI, 0, 2),
				bytecode.IABx(bytecode.LOADI, 1, 1),
				bytecode.IABC(bytecode.LT, 1, 0, 1),
			}}
			vm := New(context.Background(), nil)
			f, err := testEval(vm, fnproto)
			require.NoError(t, err)
			assert.Equal(t, int64(4), f.pc)
		})
		t.Run("compare non-number should err", func(t *testing.T) {
			t.Parallel()
			fnproto := &parse.FnProto{Constants: []any{"nope"}, ByteCodes: []uint32{
				bytecode.IABx(bytecode.LOADK, 0, 0),
				bytecode.IABx(bytecode.LOADI, 1, 1),
				bytecode.IABC(bytecode.LT, 1, 0, 1),
			}}
			vm := New(context.Background(), nil)
			_, err := testEval(vm, fnproto)
			require.Error(t, err)
		})
	})

	t.Run("LE", func(t *testing.T) {
		t.Parallel()
		t.Run("is false expecting false should not increment pc", func(t *testing.T) {
			t.Parallel()
			fnproto := &parse.FnProto{ByteCodes: []uint32{
				bytecode.IABx(bytecode.LOADI, 0, 2),
				bytecode.IABx(bytecode.LOADI, 1, 1),
				bytecode.IABC(bytecode.LE, 0, 0, 1),
			}}
			vm := New(context.Background(), nil)
			f, err := testEval(vm, fnproto)
			require.NoError(t, err)
			assert.Equal(t, int64(3), f.pc)
		})
		t.Run("is true expecting false should increment pc", func(t *testing.T) {
			t.Parallel()
			fnproto := &parse.FnProto{ByteCodes: []uint32{
				bytecode.IABx(bytecode.LOADI, 0, 0),
				bytecode.IABx(bytecode.LOADI, 1, 1),
				bytecode.IABC(bytecode.LE, 0, 0, 1),
			}}
			vm := New(context.Background(), nil)
			f, err := testEval(vm, fnproto)
			require.NoError(t, err)
			assert.Equal(t, int64(4), f.pc)
		})
		t.Run("is true expecting true should not increment pc", func(t *testing.T) {
			t.Parallel()
			fnproto := &parse.FnProto{ByteCodes: []uint32{
				bytecode.IABx(bytecode.LOADI, 0, 0),
				bytecode.IABx(bytecode.LOADI, 1, 1),
				bytecode.IABC(bytecode.LE, 1, 0, 1),
			}}
			vm := New(context.Background(), nil)
			f, err := testEval(vm, fnproto)
			require.NoError(t, err)
			assert.Equal(t, int64(3), f.pc)
		})
		t.Run("is false expecting true should increment pc", func(t *testing.T) {
			t.Parallel()
			fnproto := &parse.FnProto{ByteCodes: []uint32{
				bytecode.IABx(bytecode.LOADI, 0, 2),
				bytecode.IABx(bytecode.LOADI, 1, 1),
				bytecode.IABC(bytecode.LE, 1, 0, 1),
			}}
			vm := New(context.Background(), nil)
			f, err := testEval(vm, fnproto)
			require.NoError(t, err)
			assert.Equal(t, int64(4), f.pc)
		})
		t.Run("compare non-number should err", func(t *testing.T) {
			t.Parallel()
			fnproto := &parse.FnProto{Constants: []any{"nope"}, ByteCodes: []uint32{
				bytecode.IABx(bytecode.LOADK, 0, 0),
				bytecode.IABx(bytecode.LOADI, 1, 1),
				bytecode.IABC(bytecode.LE, 1, 0, 1),
			}}
			vm := New(context.Background(), nil)
			_, err := testEval(vm, fnproto)
			require.Error(t, err)
		})
	})

	t.Run("TEST", func(t *testing.T) {
		t.Parallel()
		t.Run("is false expecting false should not increment pc", func(t *testing.T) {
			t.Parallel()
			fnproto := &parse.FnProto{ByteCodes: []uint32{
				bytecode.IABx(bytecode.LOADBOOL, 0, 0),
				bytecode.IAB(bytecode.TEST, 0, 0),
			}}
			vm := New(context.Background(), nil)
			f, err := testEval(vm, fnproto)
			require.NoError(t, err)
			assert.Equal(t, int64(2), f.pc)
		})
		t.Run("is true expecting false should increment pc", func(t *testing.T) {
			t.Parallel()
			fnproto := &parse.FnProto{ByteCodes: []uint32{
				bytecode.IABx(bytecode.LOADBOOL, 0, 1),
				bytecode.IAB(bytecode.TEST, 0, 0),
			}}
			vm := New(context.Background(), nil)
			f, err := testEval(vm, fnproto)
			require.NoError(t, err)
			assert.Equal(t, int64(3), f.pc)
		})
		t.Run("is true expecting true should not increment pc", func(t *testing.T) {
			t.Parallel()
			fnproto := &parse.FnProto{ByteCodes: []uint32{
				bytecode.IABx(bytecode.LOADBOOL, 0, 1),
				bytecode.IAB(bytecode.TEST, 0, 1),
			}}
			vm := New(context.Background(), nil)
			f, err := testEval(vm, fnproto)
			require.NoError(t, err)
			assert.Equal(t, int64(2), f.pc)
		})
		t.Run("is false expecting true should increment pc", func(t *testing.T) {
			t.Parallel()
			fnproto := &parse.FnProto{ByteCodes: []uint32{
				bytecode.IABx(bytecode.LOADBOOL, 0, 0),
				bytecode.IAB(bytecode.TEST, 0, 1),
			}}
			vm := New(context.Background(), nil)
			f, err := testEval(vm, fnproto)
			require.NoError(t, err)
			assert.Equal(t, int64(3), f.pc)
		})
	})

	t.Run("LEN", func(t *testing.T) {
		t.Parallel()
		t.Run("String", func(t *testing.T) {
			t.Parallel()
			fnproto := &parse.FnProto{
				Constants: []any{"test string"},
				ByteCodes: []uint32{bytecode.IABCK(bytecode.LEN, 0, 0, true, 0, false)},
			}
			vm, _, err := tEval(fnproto)
			require.NoError(t, err)
			assert.Equal(t, int64(len("test string")), vm.Stack[1])
		})
		t.Run("Table", func(t *testing.T) {
			t.Parallel()
			fnproto := &parse.FnProto{
				ByteCodes: []uint32{
					bytecode.IABC(bytecode.NEWTABLE, 0, 3, 0),
					bytecode.IABx(bytecode.LOADI, 1, 21),
					bytecode.IABx(bytecode.LOADI, 2, 22),
					bytecode.IABx(bytecode.LOADI, 3, 23),
					bytecode.IABC(bytecode.SETLIST, 0, 4, 1),
					bytecode.IAB(bytecode.LEN, 1, 0),
				},
			}
			vm, _, err := tEval(fnproto)
			require.NoError(t, err)
			assert.Equal(t, int64(3), vm.Stack[2])
		})
		t.Run("Others", func(t *testing.T) {
			t.Parallel()
			fnproto := &parse.FnProto{
				Constants: []any{123.0},
				ByteCodes: []uint32{bytecode.IABCK(bytecode.LEN, 0, 0, true, 0, false)},
			}
			_, _, err := tEval(fnproto)
			require.Error(t, err)
		})
	})

	t.Run("SETTABLE", func(t *testing.T) {
		t.Parallel()
		fnproto := &parse.FnProto{
			Constants: []any{"hello", "world"},
			ByteCodes: []uint32{
				bytecode.IABC(bytecode.NEWTABLE, 0, 0, 1),
				bytecode.IABCK(bytecode.SETTABLE, 0, 0, true, 1, true),
			},
		}
		vm, _, err := tEval(fnproto)
		require.NoError(t, err)
		expectedTable := &Table{
			val:       []any{},
			hashtable: map[any]any{"hello": "world"},
			keyCache:  []any{"hello"},
		}
		assert.Equal(t, expectedTable, vm.Stack[1])
	})

	t.Run("GETTABLE", func(t *testing.T) {
		t.Parallel()
		fnproto := &parse.FnProto{
			Constants: []any{"hello", "world"},
			ByteCodes: []uint32{
				bytecode.IABC(bytecode.NEWTABLE, 0, 0, 1),
				bytecode.IABCK(bytecode.SETTABLE, 0, 0, true, 1, true),
				bytecode.IABCK(bytecode.GETTABLE, 1, 0, false, 0, true),
			},
		}
		vm, _, err := tEval(fnproto)
		require.NoError(t, err)
		expectedTable := &Table{
			val:       []any{},
			hashtable: map[any]any{"hello": "world"},
			keyCache:  []any{"hello"},
		}
		assert.Equal(t, expectedTable, vm.Stack[1])
		assert.Equal(t, "world", vm.Stack[2])
	})

	t.Run("SETLIST", func(t *testing.T) {
		t.Parallel()
		t.Run("with defined count at zero position", func(t *testing.T) {
			t.Parallel()
			fnproto := &parse.FnProto{
				ByteCodes: []uint32{
					bytecode.IABC(bytecode.NEWTABLE, 0, 3, 0),
					bytecode.IABx(bytecode.LOADI, 1, 20),
					bytecode.IABx(bytecode.LOADI, 2, 20),
					bytecode.IABx(bytecode.LOADI, 3, 20),
					bytecode.IABC(bytecode.SETLIST, 0, 4, 1),
				},
			}
			vm, _, err := tEval(fnproto)
			require.NoError(t, err)
			expectedTable := &Table{
				val:       []any{int64(20), int64(20), int64(20)},
				hashtable: map[any]any{},
			}
			assert.Equal(t, expectedTable, vm.Stack[1])
		})

		t.Run("with defined count at c position", func(t *testing.T) {
			t.Parallel()
			fnproto := &parse.FnProto{
				ByteCodes: []uint32{
					bytecode.IABC(bytecode.NEWTABLE, 0, 3, 0),
					bytecode.IABx(bytecode.LOADI, 1, 20),
					bytecode.IABx(bytecode.LOADI, 2, 20),
					bytecode.IABx(bytecode.LOADI, 3, 20),
					bytecode.IABC(bytecode.SETLIST, 0, 4, 3),
				},
			}
			vm, _, err := tEval(fnproto)
			require.NoError(t, err)
			expectedTable := &Table{
				val:       []any{nil, nil, int64(20), int64(20), int64(20)},
				hashtable: map[any]any{},
			}
			assert.Equal(t, expectedTable, vm.Stack[1])
		})
	})

	t.Run("GETUPVAL", func(t *testing.T) {
		t.Parallel()
		t.Run("open upval", func(t *testing.T) {
			t.Parallel()
			fnproto := &parse.FnProto{
				ByteCodes: []uint32{
					bytecode.IAB(bytecode.LOADI, 0, 42),
					bytecode.IAB(bytecode.GETUPVAL, 1, 0),
				},
			}
			vm := New(context.Background(), nil)
			err := testEvalUpvals(vm, fnproto, vm.newUpValueBroker("value", int64(42), 0))
			require.NoError(t, err)
			assert.Equal(t, int64(42), vm.Stack[0])
			assert.Equal(t, int64(42), vm.Stack[1])
		})
		t.Run("closed upval", func(t *testing.T) {
			t.Parallel()
			fnproto := &parse.FnProto{
				ByteCodes: []uint32{
					bytecode.IAB(bytecode.LOADI, 0, 42),
					bytecode.IAB(bytecode.GETUPVAL, 1, 0),
				},
			}
			vm := New(context.Background(), nil)
			err := testEvalUpvals(vm, fnproto, &upvalueBroker{name: "value", val: int64(77), open: false})
			require.NoError(t, err)
			assert.Equal(t, int64(42), vm.Stack[0])
			assert.Equal(t, int64(77), vm.Stack[1])
		})
	})

	t.Run("SETUPVAL", func(t *testing.T) {
		t.Parallel()
		t.Run("open upval", func(t *testing.T) {
			t.Parallel()
			fnproto := &parse.FnProto{
				ByteCodes: []uint32{
					bytecode.IAB(bytecode.LOADI, 0, 42),
					bytecode.IAB(bytecode.LOADI, 1, 77),
					bytecode.IAB(bytecode.SETUPVAL, 1, 0),
				},
			}
			vm := New(context.Background(), nil)
			err := testEvalUpvals(vm, fnproto, vm.newUpValueBroker("value", int64(42), 0))
			require.NoError(t, err)
			assert.Equal(t, int64(77), vm.Stack[1])
		})
		t.Run("closed upval", func(t *testing.T) {
			t.Parallel()
			fnproto := &parse.FnProto{
				ByteCodes: []uint32{
					bytecode.IAB(bytecode.LOADI, 0, 42),
					bytecode.IAB(bytecode.LOADI, 1, 77),
					bytecode.IAB(bytecode.SETUPVAL, 1, 0),
				},
			}
			vm := New(context.Background(), nil)
			upval := &upvalueBroker{name: "value", val: int64(42), open: false}
			err := testEvalUpvals(vm, fnproto, upval)
			require.NoError(t, err)
			assert.Equal(t, int64(42), vm.Stack[0])
			assert.Equal(t, int64(77), upval.val)
		})
	})

	t.Run("bytecode.GETTABUP", func(t *testing.T) {
		t.Parallel()
		t.Run("open upval", func(t *testing.T) {
			t.Parallel()
			fnproto := &parse.FnProto{
				ByteCodes: []uint32{
					bytecode.IABC(bytecode.NEWTABLE, 0, 3, 0),
					bytecode.IABx(bytecode.LOADI, 1, 20),
					bytecode.IABx(bytecode.LOADI, 2, 22),
					bytecode.IABx(bytecode.LOADI, 3, 24),
					bytecode.IABC(bytecode.SETLIST, 0, 4, 1),
					bytecode.IABx(bytecode.LOADI, 1, 1),
					bytecode.IABC(bytecode.GETTABUP, 1, 0, 1),
				},
			}
			vm := New(context.Background(), nil)
			err := testEvalUpvals(vm, fnproto, vm.newUpValueBroker("value", nil, 0))
			require.NoError(t, err)
			expectedTable := &Table{
				val:       []any{int64(20), int64(22), int64(24)},
				hashtable: map[any]any{},
			}
			assert.Equal(t, expectedTable, vm.Stack[0])
			assert.Equal(t, int64(20), vm.Stack[1])
		})
		t.Run("with key", func(t *testing.T) {
			t.Parallel()
			fnproto := &parse.FnProto{
				Constants: []any{"hello", "world"},
				ByteCodes: []uint32{
					bytecode.IABC(bytecode.NEWTABLE, 0, 0, 1),
					bytecode.IABCK(bytecode.SETTABLE, 0, 0, true, 1, true),
					bytecode.IABCK(bytecode.GETTABUP, 1, 0, false, 0, true),
				},
			}
			vm := New(context.Background(), nil)
			err := testEvalUpvals(vm, fnproto, vm.newUpValueBroker("value", nil, 0))
			require.NoError(t, err)
			expectedTable := &Table{
				val:       []any{},
				hashtable: map[any]any{"hello": "world"},
				keyCache:  []any{"hello"},
			}
			assert.Equal(t, expectedTable, vm.Stack[0])
			assert.Equal(t, "world", vm.Stack[1])
		})
		t.Run("closed upval", func(t *testing.T) {
			t.Parallel()
			fnproto := &parse.FnProto{
				ByteCodes: []uint32{
					bytecode.IABC(bytecode.NEWTABLE, 0, 3, 0),
					bytecode.IABx(bytecode.LOADI, 1, 20),
					bytecode.IABx(bytecode.LOADI, 2, 22),
					bytecode.IABx(bytecode.LOADI, 3, 24),
					bytecode.IABC(bytecode.SETLIST, 0, 4, 1),
					bytecode.IABx(bytecode.LOADI, 1, 1),
					bytecode.IABC(bytecode.GETTABUP, 1, 0, 1),
				},
			}
			table := &Table{
				val:       []any{int64(20), int64(22), int64(24)},
				hashtable: map[any]any{},
			}
			vm := New(context.Background(), nil)
			err := testEvalUpvals(vm, fnproto, &upvalueBroker{name: "value", val: table, open: false})
			require.NoError(t, err)
			expectedTable := &Table{
				val:       []any{int64(20), int64(22), int64(24)},
				hashtable: map[any]any{},
			}
			assert.Equal(t, expectedTable, vm.Stack[0])
			assert.Equal(t, int64(20), vm.Stack[1])
		})
	})

	t.Run("SETTABUP", func(t *testing.T) {
		t.Parallel()
		t.Run("open upval", func(t *testing.T) {
			t.Parallel()
			fnproto := &parse.FnProto{
				ByteCodes: []uint32{
					bytecode.IABC(bytecode.NEWTABLE, 0, 3, 0),
					bytecode.IABx(bytecode.LOADI, 1, 20),
					bytecode.IABx(bytecode.LOADI, 2, 22),
					bytecode.IABx(bytecode.LOADI, 3, 24),
					bytecode.IABC(bytecode.SETLIST, 0, 4, 1),
					bytecode.IABx(bytecode.LOADI, 1, 1),
					bytecode.IABx(bytecode.LOADI, 2, 55),
					bytecode.IABC(bytecode.SETTABUP, 0, 1, 2),
				},
			}
			vm := New(context.Background(), nil)
			err := testEvalUpvals(vm, fnproto, vm.newUpValueBroker("value", nil, 0))
			require.NoError(t, err)
			expectedTable := &Table{
				val:       []any{int64(55), int64(22), int64(24)},
				hashtable: map[any]any{},
			}
			assert.Equal(t, expectedTable, vm.Stack[0])
		})
		t.Run("with key", func(t *testing.T) {
			t.Parallel()
			fnproto := &parse.FnProto{
				Constants: []any{"hello", "world", "tim"},
				ByteCodes: []uint32{
					bytecode.IABC(bytecode.NEWTABLE, 0, 0, 1),
					bytecode.IABCK(bytecode.SETTABLE, 0, 0, true, 1, true),
					bytecode.IABCK(bytecode.SETTABUP, 0, 0, true, 2, true),
				},
			}
			vm := New(context.Background(), nil)
			err := testEvalUpvals(vm, fnproto, vm.newUpValueBroker("value", nil, 0))
			require.NoError(t, err)
			expectedTable := &Table{
				val:       []any{},
				hashtable: map[any]any{"hello": "tim"},
				keyCache:  []any{"hello"},
			}
			assert.Equal(t, expectedTable, vm.Stack[0])
		})
		t.Run("closed upval", func(t *testing.T) {
			t.Parallel()
			fnproto := &parse.FnProto{
				ByteCodes: []uint32{
					bytecode.IABC(bytecode.NEWTABLE, 0, 3, 0),
					bytecode.IABx(bytecode.LOADI, 1, 20),
					bytecode.IABx(bytecode.LOADI, 2, 22),
					bytecode.IABx(bytecode.LOADI, 3, 24),
					bytecode.IABC(bytecode.SETLIST, 0, 4, 1),
					bytecode.IABx(bytecode.LOADI, 1, 1),
					bytecode.IABx(bytecode.LOADI, 2, 99),
					bytecode.IABC(bytecode.SETTABUP, 0, 1, 2),
				},
			}
			table := &Table{
				val:       []any{int64(20), int64(22), int64(24)},
				hashtable: map[any]any{},
			}
			vm := New(context.Background(), nil)
			err := testEvalUpvals(vm, fnproto, &upvalueBroker{name: "value", val: table, open: false})
			require.NoError(t, err)
			expectedTable := &Table{
				val:       []any{int64(99), int64(22), int64(24)},
				hashtable: map[any]any{},
			}
			assert.Equal(t, expectedTable, table)
		})
	})

	t.Run("RETURN", func(t *testing.T) {
		t.Parallel()
		t.Run("All return values", func(t *testing.T) {
			t.Parallel()
			fnproto := &parse.FnProto{
				Constants: []any{"don't touch me", "hello", "world"},
				ByteCodes: []uint32{
					bytecode.IABx(bytecode.LOADK, 0, 0),
					bytecode.IABx(bytecode.LOADK, 1, 1),
					bytecode.IABx(bytecode.LOADK, 2, 2),
					bytecode.IAB(bytecode.RETURN, 1, 0),
				},
			}
			_, values, err := tEval(fnproto)
			require.NoError(t, err)
			assert.Equal(t, []any{"hello", "world"}, values)
		})

		t.Run("specified return vals", func(t *testing.T) {
			t.Parallel()
			fnproto := &parse.FnProto{
				Constants: []any{"don't touch me", "hello", "world"},
				ByteCodes: []uint32{
					bytecode.IABx(bytecode.LOADK, 0, 0),
					bytecode.IABx(bytecode.LOADK, 1, 1),
					bytecode.IABx(bytecode.LOADK, 2, 2),
					bytecode.IAB(bytecode.RETURN, 1, 3),
				},
			}
			_, values, err := tEval(fnproto)
			require.NoError(t, err)
			assert.Equal(t, []any{"hello", "world"}, values)
		})

		t.Run("specified return vals more than provided", func(t *testing.T) {
			t.Parallel()
			fnproto := &parse.FnProto{
				Constants: []any{"don't touch me", "hello", "world"},
				ByteCodes: []uint32{
					bytecode.IABx(bytecode.LOADK, 0, 0),
					bytecode.IABx(bytecode.LOADK, 1, 1),
					bytecode.IABx(bytecode.LOADK, 2, 2),
					bytecode.IAB(bytecode.RETURN, 1, 5),
				},
			}
			_, values, err := tEval(fnproto)
			require.NoError(t, err)
			assert.Equal(t, []any{"hello", "world", nil, nil}, values)
		})

		t.Run("no return values", func(t *testing.T) {
			t.Parallel()
			fnproto := &parse.FnProto{
				Constants: []any{"don't touch me", "hello", "world"},
				ByteCodes: []uint32{
					bytecode.IAB(bytecode.RETURN, 0, 1),
				},
			}
			vm, values, err := tEval(fnproto)
			require.NoError(t, err)
			assert.Equal(t, []any{}, values)
			assert.Equal(t, []any{}, vm.Stack[:vm.top])
		})
	})

	t.Run("VARARG", func(t *testing.T) {
		t.Parallel()
		t.Run("All xargs", func(t *testing.T) {
			t.Parallel()
			fnproto := &parse.FnProto{
				Constants: []any{"don't touch me", "hello", "world"},
				ByteCodes: []uint32{bytecode.IAB(bytecode.VARARG, 0, 0)},
			}
			vm := New(context.Background(), nil)
			vm.Stack = []any{int64(11), float64(42), "hello"}
			vm.top = 3
			_, err := vm.Eval(fnproto)
			require.NoError(t, err)
			assert.Equal(t, int64(11), vm.Stack[0])
			assert.InEpsilon(t, float64(42), vm.Stack[1], 0)
			assert.Equal(t, "hello", vm.Stack[2])
		})
		t.Run("nargs", func(t *testing.T) {
			t.Parallel()
			fnproto := &parse.FnProto{
				Constants: []any{"don't touch me", "hello", "world"},
				ByteCodes: []uint32{bytecode.IAB(bytecode.VARARG, 0, 2)},
			}
			vm := New(context.Background(), nil)
			vm.Stack = []any{int64(11), float64(42), "hello"}
			vm.top = 3
			_, err := vm.Eval(fnproto)
			require.NoError(t, err)
			assert.Equal(t, int64(11), vm.Stack[0])
		})
		t.Run("nargs with offset", func(t *testing.T) {
			t.Parallel()
			fnproto := &parse.FnProto{
				Constants: []any{"don't touch me", "hello", "world"},
				ByteCodes: []uint32{bytecode.IAB(bytecode.VARARG, 0, 2)},
			}
			vm := New(context.Background(), nil)
			vm.Stack = []any{int64(11), float64(42), "hello"}
			vm.top = 3
			_, err := vm.Eval(fnproto)
			require.NoError(t, err)
			assert.Equal(t, int64(11), vm.Stack[0])
		})
	})

	t.Run("bytecode.CALL", func(t *testing.T) {
		t.Parallel()
		called := false
		env := &Table{
			hashtable: map[any]any{
				"foo": Fn("foo", func(*VM, []any) ([]any, error) {
					called = true
					return []any{int64(42)}, nil
				}),
			},
		}

		fnproto := &parse.FnProto{
			Constants: []any{"foo", "./tmp/out"},
			ByteCodes: []uint32{
				bytecode.IABCK(bytecode.GETTABUP, 0, 0, false, 0, true),
				bytecode.IAB(bytecode.LOADK, 1, 1),
				bytecode.IAB(bytecode.LOADI, 2, 1),
				bytecode.IABC(bytecode.CALL, 0, 3, 2),
			},
		}

		vm := New(context.Background(), nil)
		envUpval := &upvalueBroker{name: "_ENV", val: env}
		f := vm.newFrame(fnproto, vm.top, 0, []*upvalueBroker{envUpval})
		_, err := vm.eval(f)
		require.NoError(t, err)
		assert.True(t, called)
		assert.Equal(t, int64(0), f.framePointer)
		assert.Equal(t, int64(1), vm.top)
		assert.Equal(t, int64(42), vm.Stack[0])
	})

	t.Run("CLOSURE", func(t *testing.T) {
		t.Parallel()
		t.Skip("TODO")
	})

	t.Run("SELF", func(t *testing.T) {
		t.Parallel()
		t.Skip("TODO")
	})

	t.Run("TAILbytecode.CALL", func(t *testing.T) {
		t.Parallel()
		t.Skip("TODO")
	})

	t.Run("FORLOOP", func(t *testing.T) {
		t.Parallel()
		t.Skip("TODO")
	})

	t.Run("FORPREP", func(t *testing.T) {
		t.Parallel()
		t.Skip("TODO")
	})

	t.Run("TFORLOOP", func(t *testing.T) {
		t.Parallel()
		t.Skip("TODO")
	})

	t.Run("TFORbytecode.CALL", func(t *testing.T) {
		t.Parallel()
		t.Skip("TODO")
	})
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

func testEval(vm *VM, fn *parse.FnProto) (*frame, error) {
	f := vm.newEnvFrame(fn, vm.top, nil)
	_, err := vm.eval(f)
	return f, err
}

func tEval(fn *parse.FnProto) (*VM, []any, error) {
	vm := New(context.Background(), nil)
	val, err := vm.Eval(fn)
	return vm, val, err
}

func testEvalUpvals(vm *VM, fn *parse.FnProto, upvals ...*upvalueBroker) error {
	_, err := vm.eval(vm.newFrame(fn, vm.top, 0, upvals))
	return err
}

func TestEnsureSize(t *testing.T) {
	t.Parallel()
	a := []string{}
	assert.Empty(t, a)
	ensureSize(&a, 5)
	assert.Len(t, a, 6)
	a[5] = "did it"
}
