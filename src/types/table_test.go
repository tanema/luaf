package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewTable(t *testing.T) {
	tbl := NewTable()
	assert.Equal(t, TblFree, tbl.Hint)
	assert.Equal(t, map[string]Definition{}, tbl.FieldDefn)
	assert.Equal(t, Any, tbl.KeyDefn)
	assert.Equal(t, Any, tbl.ValDefn)
}

func TestTableCheck(t *testing.T) {
}
