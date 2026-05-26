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
	cmpMap := map[Op]string{}
	for i := MOVE; i < MAXCODES; i++ {
		if _, found := opcodeToString[i]; !found {
			t.Errorf("Bytecode %v not found", int(i))
		} else if kind := Kind(uint32(i)); kind == TypeUNKNOWN {
			t.Errorf("unexpected extra arg at index %v %v", int(i), opcodeToString[i])
		}
		ToString(uint32(i)) // smoke test
		cmpMap[i] = opcodeToString[i]
	}
	assert.Equal(t, opcodeToString, cmpMap)
}

func TestBytecodeABC(t *testing.T) {
	t.Parallel()
	t.Run("iAB", func(t *testing.T) {
		t.Parallel()
		code := IAB(MOVE, 12, 22)
		assert.Equal(t, MOVE, GetOp(code))
		assert.Equal(t, int64(12), GetA(code))
		assert.Equal(t, int64(22), GetB(code))
		assert.Equal(t, int64(0), GetC(code))
		assert.Equal(t, TypeABC, Kind(code))
	})

	t.Run("iABC true const", func(t *testing.T) {
		t.Parallel()
		code := IABC(MOVE, 12, 22, 33, true)
		assert.Equal(t, MOVE, GetOp(code))
		assert.Equal(t, int64(12), GetA(code))
		assert.Equal(t, int64(22), GetB(code))
		assert.Equal(t, int64(33), GetC(code))
		assert.True(t, GetK(code))
		assert.Equal(t, TypeABC, Kind(code))
	})

	t.Run("iABC false const", func(t *testing.T) {
		t.Parallel()
		code := IABC(MOVE, 12, 22, 33, false)
		assert.Equal(t, MOVE, GetOp(code))
		assert.Equal(t, int64(12), GetA(code))
		assert.Equal(t, int64(22), GetB(code))
		assert.Equal(t, int64(33), GetC(code))
		assert.False(t, GetK(code))
		assert.Equal(t, TypeABC, Kind(code))
	})

	t.Run("ivABC", func(t *testing.T) {
		t.Parallel()
		code := IvABC(NEWTABLE, 12, 22, 33, true)
		assert.Equal(t, NEWTABLE, GetOp(code))
		assert.Equal(t, int64(12), GetA(code))
		assert.Equal(t, int64(22), GetvB(code))
		assert.Equal(t, int64(33), GetvC(code))
		assert.True(t, GetK(code))
		assert.Equal(t, TypevABC, Kind(code))
	})

	t.Run("iABx", func(t *testing.T) {
		t.Parallel()
		code := IABx(LOADK, 12, 300)
		assert.Equal(t, LOADK, GetOp(code))
		assert.Equal(t, int64(12), GetA(code))
		assert.Equal(t, int64(300), GetBx(code))
		assert.Equal(t, TypeABx, Kind(code))
	})

	t.Run("iAsBx", func(t *testing.T) {
		t.Parallel()
		code := IAsBx(ADDI, 12, -300)
		assert.Equal(t, ADDI, GetOp(code))
		assert.Equal(t, int64(12), GetA(code))
		assert.Equal(t, int64(-300), GetsBx(code))
		assert.Equal(t, TypeAsBx, Kind(code))
	})

	t.Run("iAx", func(t *testing.T) {
		t.Parallel()
		code := ExArg(325)
		assert.Equal(t, EXARG, GetOp(code))
		assert.Equal(t, uint64(325), GetAx(code))
		assert.Equal(t, TypeAx, Kind(code))
	})

	t.Run("isJ", func(t *testing.T) {
		t.Parallel()
		code := Jump(-300)
		assert.Equal(t, JMP, GetOp(code))
		assert.Equal(t, int64(-300), GetJump(code))
		assert.Equal(t, TypesJ, Kind(code))
	})
}
