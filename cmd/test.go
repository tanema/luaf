package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/pflag"
)

type testCmd struct {
	verbose bool
	flagSet *pflag.FlagSet
}

func (cmd *testCmd) flags() error {
	cmd.flagSet = pflag.NewFlagSet("test", pflag.ExitOnError)
	cmd.flagSet.BoolVarP(&cmd.verbose, "verbose", "v", false, "show verbose output")
	cmd.flagSet.Usage = cmd.usage
	return cmd.flagSet.Parse(os.Args[2:])
}

func (cmd *testCmd) usage() {
	fmt.Fprint(os.Stderr, "usage: luaf test [options] <path>\n")
	cmd.flagSet.PrintDefaults()
}

func (cmd *testCmd) run() error {
	return errors.New("not implemented yet")
}
