package luaf

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnsureSize(t *testing.T) {
	t.Parallel()
	a := []string{}
	assert.Empty(t, a)
	ensureSize(&a, 5)
	assert.Len(t, a, 6)
	a[5] = "did it"
}
