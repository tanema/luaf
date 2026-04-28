package bytecode

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAllCodesCovered(t *testing.T) {
	t.Parallel()
	// Did we add too many instructions?
	require.LessOrEqual(t, int(MAXCODES), int(^uint8(0)>>1))
	// Do all opcodes have a string defined.
	assert.Equal(t, len(opcodeToString), int(MAXCODES))
	for i := MOVE; i < MAXCODES; i++ {
		if _, found := opcodeToString[i]; !found {
			t.Errorf("Bytecode %v not found", int(i))
		} else if kind := Kind(uint32(i)); kind == TypeEx {
			t.Errorf("unexpected extra arg at index %v %v", int(i), opcodeToString[i])
		}
		ToString(uint32(i)) // smoke test
	}
}

func TestBytecodeABC(t *testing.T) {
	t.Parallel()
	t.Run("iAB", func(t *testing.T) {
		t.Parallel()
		code := IAB(MOVE, 12, 22)
		assert.Equal(t, MOVE, GetOp(code))
		assert.Equal(t, int64(12), GetA(code))
		b, bK := GetBK(code)
		assert.Equal(t, int64(22), b)
		assert.False(t, bK)
		c, cK := GetCK(code)
		assert.Equal(t, int64(0), c)
		assert.False(t, cK)
		assert.Equal(t, TypeABC, Kind(code))
	})

	t.Run("iABC", func(t *testing.T) {
		t.Parallel()
		code := IABC(MOVE, 12, 22, 33)
		assert.Equal(t, MOVE, GetOp(code))
		assert.Equal(t, int64(12), GetA(code))
		b, bK := GetBK(code)
		assert.Equal(t, int64(22), b)
		assert.False(t, bK)
		c, cK := GetCK(code)
		assert.Equal(t, int64(33), c)
		assert.False(t, cK)
		assert.Equal(t, TypeABC, Kind(code))
	})

	t.Run("iABCK", func(t *testing.T) {
		t.Parallel()
		code := IABCK(MOVE, 12, 22, true, 33, false)
		assert.Equal(t, MOVE, GetOp(code))
		assert.Equal(t, int64(12), GetA(code))
		b, bK := GetBK(code)
		assert.Equal(t, int64(22), b)
		assert.True(t, bK)
		c, cK := GetCK(code)
		assert.Equal(t, int64(33), c)
		assert.False(t, cK)
		assert.Equal(t, TypeABC, Kind(code))
	})

	t.Run("iABx", func(t *testing.T) {
		t.Parallel()
		code := IABx(LOADK, 12, 300)
		a, x := GetA(code), GetBx(code)
		assert.Equal(t, LOADK, GetOp(code))
		assert.Equal(t, int64(12), a)
		assert.Equal(t, int64(300), x)
		assert.Equal(t, TypeABx, Kind(code))
	})

	t.Run("iAsBx", func(t *testing.T) {
		t.Parallel()
		code := IAsBx(JMP, 12, -300)
		a, xs := GetA(code), GetsBx(code)
		assert.Equal(t, JMP, GetOp(code))
		assert.Equal(t, int64(12), a)
		assert.Equal(t, int64(-300), xs)
		assert.Equal(t, TypeAsBx, Kind(code))
	})
}
