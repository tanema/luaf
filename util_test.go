package luaf

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnsureSizeGrow(t *testing.T) {
	a := []string{}
	assert.Len(t, a, 0)
	assert.Equal(t, 0, cap(a))
	ensureSizeGrow(&a, 5)
	assert.Len(t, a, 12)
	assert.Equal(t, 12, cap(a))
	a[5] = "did it"
}

func TestEnsureSize(t *testing.T) {
	a := []string{}
	assert.Len(t, a, 0)
	ensureSize(&a, 5)
	assert.Len(t, a, 6)
	a[5] = "did it"
}

func TestTruncate(t *testing.T) {
	a := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	assert.Len(t, a, 10)
	assert.Equal(t, 10, cap(a))
	truncate(&a, 5)
	assert.Len(t, a, 5)
	assert.Equal(t, 5, cap(a))
}
