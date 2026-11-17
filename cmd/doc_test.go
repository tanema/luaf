package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func runDoc(args ...string) error {
	fs := pflag.NewFlagSet("doc", pflag.ContinueOnError)
	docCommand.Flags(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	return docCommand.Cmd(fs, fs.Args())
}

func TestDocCommandPathErrors(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	file := filepath.Join(dir, "mod.lua")
	require.NoError(t, os.WriteFile(file, []byte("local x = 1\n"), 0o600))

	t.Run("missing argument", func(t *testing.T) {
		t.Parallel()
		assert.EqualError(t, runDoc(), "missing path to document")
	})

	t.Run("missing file", func(t *testing.T) {
		t.Parallel()
		assert.ErrorIs(t, runDoc(filepath.Join(dir, "missing.lua")), os.ErrNotExist)
	})

	t.Run("directory", func(t *testing.T) {
		t.Parallel()
		assert.ErrorContains(t, runDoc(dir), "is a directory")
	})

	t.Run("file is accepted", func(t *testing.T) {
		t.Parallel()
		assert.EqualError(t, runDoc("-f", "bogus", file), "unknown output format bogus")
	})
}
