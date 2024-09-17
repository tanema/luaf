package shine

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBytecodeABC(t *testing.T) {
	code := IABC(MOVE, 12, 22, 33)
	a, b, c := code.ABC()
	assert.Equal(t, MOVE, code.Op())
	assert.Equal(t, int64(12), a)
	assert.Equal(t, int64(22), b)
	assert.Equal(t, int64(33), c)

	code = IABx(MOVE, 12, 300)
	a, x := code.ABx()
	assert.Equal(t, MOVE, code.Op())
	assert.Equal(t, int64(12), a)
	assert.Equal(t, int64(300), x)

	code = IAsBx(MOVE, 12, -300)
	a, xs := code.AsBx()
	assert.Equal(t, MOVE, code.Op())
	assert.Equal(t, int64(12), a)
	assert.Equal(t, int64(-300), xs)
}

func TestParseBytecode(t *testing.T) {
	opcode, err := ParseOpcode("MOVE 2 1 0")
	assert.Nil(t, err)
	assert.Equal(t, Bytecode(0x10203), opcode)
	assert.Equal(t, MOVE, opcode.Op())
	a, b, c := opcode.ABC()
	assert.Equal(t, int64(2), a)
	assert.Equal(t, int64(1), b)
	assert.Equal(t, int64(0), c)

	opcode, err = ParseOpcode("LOADK 2 255")
	assert.Nil(t, err)
	assert.Equal(t, Bytecode(0xFF0204), opcode)
	assert.Equal(t, LOADK, opcode.Op())
	a, b = opcode.ABx()
	assert.Equal(t, int64(2), a)
	assert.Equal(t, int64(255), b)

	opcode, err = ParseOpcode("JMP 2 -20")
	assert.Nil(t, err)
	assert.Equal(t, Bytecode(0xFFEC0221), opcode)
	assert.Equal(t, JMP, opcode.Op())
	a, b = opcode.AsBx()
	assert.Equal(t, int64(2), a)
	assert.Equal(t, int64(-20), b)

	_, err = ParseOpcode("LOADK 2 100000")
	assert.NotNil(t, err)
}
