package conf

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFullVersion(t *testing.T) {
	t.Parallel()
	version := FullVersion()
	assert.Equal(t, fmt.Sprintf("%v Copyright (C) %v", LUAVERSION, time.Now().Year()), version)
}

func TestCopyright(t *testing.T) {
	t.Parallel()
	copyright := Copyright()
	assert.Equal(t, fmt.Sprintf("Copyright (C) %v", time.Now().Year()), copyright)
}
