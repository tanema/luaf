package cmd

import (
	"github.com/spf13/pflag"
	"github.com/tanema/luaf/internal/lsp"
)

var lspCommand = Command{
	Name:        "lsp",
	Usage:       "luaf lsp [options]",
	Description: "Run a lua lsp server. Usually used by your text editor.",
	Cmd: func(_ *pflag.FlagSet, _ []string) error {
		return lsp.NewServer().Listen()
	},
}
