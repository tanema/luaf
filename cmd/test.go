package cmd

import (
	"errors"

	"github.com/spf13/pflag"
)

type testCmd struct {
	verbose bool
}

func (cmd *testCmd) flags() *pflag.FlagSet {
	flagSet := pflag.NewFlagSet("test", pflag.ExitOnError)
	flagSet.BoolVarP(&cmd.verbose, "verbose", "v", false, "show verbose output")
	return flagSet
}

func (cmd *testCmd) usage() string {
	return "usage: luaf test [options] <path>"
}

func (cmd *testCmd) run(_ []string) error {
	return errors.New("not implemented yet")
}
