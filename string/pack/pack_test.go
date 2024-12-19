package pack

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPack(t *testing.T) {
	out, err := Pack("<zj", "test", int64(12))
	assert.NoError(t, err)
	assert.Equal(t, "test\x00\f\x00\x00\x00\x00\x00\x00\x00", out)

	format := "iii"
	data := []any{int64(12), int64(12), int64(12)}
	out, err = Pack(format, data...)
	assert.NoError(t, err)
	unpackeddata, err := Unpack(format, out)
	assert.NoError(t, err)
	assert.Equal(t, data, unpackeddata)
}
