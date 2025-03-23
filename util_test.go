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
