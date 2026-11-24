package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/pflag"
)

var envCommand = Command{
	Name:        "env",
	Usage:       "luaf env",
	Description: "Dump config values for the luaf system config.",
	Hidden:      true,
	Cmd: func(*pflag.FlagSet, []string) error {
		cfg, err := systemConf()
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(os.Stdout, cfg.String())
		return err
	},
}
