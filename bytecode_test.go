package luaf

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBytecodeABC(t *testing.T) {
	t.Run("iAB", func(t *testing.T) {
		code := iAB(MOVE, 12, 22)
		assert.Equal(t, MOVE, code.op())
		assert.Equal(t, int64(12), code.getA())
		b, bK := code.getBK()
		assert.Equal(t, int64(22), b)
		assert.False(t, bK)
		c, cK := code.getCK()
		assert.Equal(t, int64(0), c)
		assert.False(t, cK)
		assert.Equal(t, BytecodeTypeABC, code.Kind())
	})

	t.Run("iABC", func(t *testing.T) {
		code := iABC(MOVE, 12, 22, 33)
		assert.Equal(t, MOVE, code.op())
		assert.Equal(t, int64(12), code.getA())
		b, bK := code.getBK()
		assert.Equal(t, int64(22), b)
		assert.False(t, bK)
		c, cK := code.getCK()
		assert.Equal(t, int64(33), c)
		assert.False(t, cK)
		assert.Equal(t, BytecodeTypeABC, code.Kind())
	})

	t.Run("iABCK", func(t *testing.T) {
		code := iABCK(MOVE, 12, 22, true, 33, false)
		assert.Equal(t, MOVE, code.op())
		assert.Equal(t, int64(12), code.getA())
		b, bK := code.getBK()
		assert.Equal(t, int64(22), b)
		assert.True(t, bK)
		c, cK := code.getCK()
		assert.Equal(t, int64(33), c)
		assert.False(t, cK)
		assert.Equal(t, BytecodeTypeABC, code.Kind())
	})

	t.Run("iABx", func(t *testing.T) {
		code := iABx(LOADK, 12, 300)
		a, x := code.getA(), code.getBx()
		assert.Equal(t, LOADK, code.op())
		assert.Equal(t, int64(12), a)
		assert.Equal(t, int64(300), x)
		assert.Equal(t, BytecodeTypeABx, code.Kind())
	})

	t.Run("iAsBx", func(t *testing.T) {
		code := iAsBx(JMP, 12, -300)
		a, xs := code.getA(), code.getsBx()
		assert.Equal(t, JMP, code.op())
		assert.Equal(t, int64(12), a)
		assert.Equal(t, int64(-300), xs)
		assert.Equal(t, BytecodeTypeAsBx, code.Kind())
	})

	t.Run("exarg", func(t *testing.T) {
		code := exarg(-300)
		assert.Equal(t, EXARG, code.op())
		assert.Equal(t, int64(-300), code.getExarg())
		assert.Equal(t, BytecodeTypesBx, code.Kind())
	})
}
