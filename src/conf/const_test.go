package conf

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCopyright(t *testing.T) {
	t.Parallel()
	assert.Equal(t, LUACOPYRIGHT, fmt.Sprintf("Copyright (C) %v", time.Now().Year()))
}
