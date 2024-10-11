package lauf

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBytecodeABC(t *testing.T) {
	code := iABCK(MOVE, 12, 22, true, 33, false)
	assert.Equal(t, MOVE, code.op())
	assert.Equal(t, int64(12), code.getA())
	b, bK := code.getBK()
	assert.Equal(t, int64(22), b)
	assert.True(t, bK)
	c, cK := code.getCK()
	assert.Equal(t, int64(33), c)
	assert.False(t, cK)

	code = iABx(MOVE, 12, 300)
	a, x := code.getA(), code.getBx()
	assert.Equal(t, MOVE, code.op())
	assert.Equal(t, int64(12), a)
	assert.Equal(t, int64(300), x)

	code = iAsBx(MOVE, 12, -300)
	a, xs := code.getA(), code.getsBx()
	assert.Equal(t, MOVE, code.op())
	assert.Equal(t, int64(12), a)
	assert.Equal(t, int64(-300), xs)
}

func TestParseBytecode(t *testing.T) {
	opcode, err := ParseOpcode("MOVE 2 1k 5k")
	assert.Nil(t, err)
	assert.Equal(t, iABCK(MOVE, 2, 1, true, 5, true), opcode)
	assert.Equal(t, MOVE, opcode.op())
	assert.Equal(t, int64(2), opcode.getA())
	b, bK := opcode.getBK()
	assert.Equal(t, int64(1), b)
	assert.True(t, bK)
	c, cK := opcode.getCK()
	assert.Equal(t, int64(5), c)
	assert.True(t, cK)

	opcode, err = ParseOpcode("LOADK 2 255")
	assert.Nil(t, err)
	assert.Equal(t, iABx(LOADK, 2, 255), opcode)
	assert.Equal(t, LOADK, opcode.op())
	a, b := opcode.getA(), opcode.getBx()
	assert.Equal(t, int64(2), a)
	assert.Equal(t, int64(255), b)

	opcode, err = ParseOpcode("JMP 2 -20")
	assert.Nil(t, err)
	assert.Equal(t, iAsBx(JMP, 2, -20), opcode)
	assert.Equal(t, JMP, opcode.op())
	a, b = opcode.getA(), opcode.getsBx()
	assert.Equal(t, int64(2), a)
	assert.Equal(t, int64(-20), b)

	_, err = ParseOpcode("LOADK 2 100000")
	assert.NotNil(t, err)
}
