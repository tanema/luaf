package luaf

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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

func TestCutout(t *testing.T) {
	a := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	assert.Len(t, a, 10)
	assert.Equal(t, 10, cap(a))
	cutout(&a, 5, 8)
	assert.Len(t, a, 10)
	assert.Equal(t, []int{1, 2, 3, 4, 5, 9, 10, 0, 0, 0}, a)

	b := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	cutout(&b, 5, 20)
	assert.Len(t, b, 10)
	assert.Equal(t, []int{1, 2, 3, 4, 5, 0, 0, 0, 0, 0}, b)
}
