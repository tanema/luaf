package cmd

import (
	"github.com/spf13/pflag"
	"github.com/tanema/luaf/internal/lsp"
)

type lspCmd struct{}

func (cmd *lspCmd) flags() *pflag.FlagSet {
	flagSet := pflag.NewFlagSet("lsp", pflag.ExitOnError)
	return flagSet
}

func (cmd *lspCmd) usage() string {
	return "usage: luaf lsp [options]"
}

func (cmd *lspCmd) run(_ []string) error {
	lspServer := lsp.NewServer()

	return lspServer.Listen()
}
