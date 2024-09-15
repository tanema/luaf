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
