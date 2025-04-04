package pack

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPack(t *testing.T) {
	t.Parallel()
	out, err := Pack("<zj", "test", int64(12))
	require.NoError(t, err)
	assert.Equal(t, "test\x00\f\x00\x00\x00\x00\x00\x00\x00", out)

	format := "iii"
	data := []any{int64(12), int64(12), int64(12)}
	out, err = Pack(format, data...)
	require.NoError(t, err)
	unpackeddata, err := Unpack(format, out)
	require.NoError(t, err)
	assert.Equal(t, data, unpackeddata)
}
