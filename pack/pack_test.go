package pack

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPack(t *testing.T) {
	out, err := Pack("<zj", "test", int64(12))
	assert.NoError(t, err)
	assert.Equal(t, "test\x00\f\x00\x00\x00\x00\x00\x00\x00", out)
}
