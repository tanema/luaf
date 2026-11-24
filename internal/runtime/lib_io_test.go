package runtime

import (
	"io"
	"strings"
	"testing"

	"github.com/chzyer/readline"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//nolint:paralleltest // lineReader is shared package state.
func TestStdIOReadline(t *testing.T) {
	rl, err := readline.NewEx(&readline.Config{
		Stdin:          io.NopCloser(strings.NewReader("first\nsecond\n")),
		Stdout:         io.Discard,
		FuncIsTerminal: func() bool { return false },
	})
	require.NoError(t, err)
	lineReader = rl
	t.Cleanup(func() {
		_ = lineReader.Close()
		lineReader = nil
	})

	res, err := stdIOReadline(nil, []any{"> "})
	require.NoError(t, err)
	assert.Equal(t, []any{"first"}, res)

	res, err = stdIOReadline(nil, []any{})
	require.NoError(t, err)
	assert.Equal(t, []any{"second"}, res)

	res, err = stdIOReadline(nil, []any{})
	require.NoError(t, err)
	assert.Equal(t, []any{nil, "eof"}, res)

	_, err = stdIOReadline(nil, []any{int64(1)})
	require.Error(t, err)
}
