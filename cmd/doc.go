package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/pflag"
)

type docCmd struct {
	verbose   bool
	markdown  bool
	text      bool
	html      bool
	http      bool
	outputDir string
	flagSet   *pflag.FlagSet
}

// DocCmd is the command that will generate documentation for a lua project.
func DocCmd() error {
	cmd := docCmd{}
	if err := cmd.flags(); err != nil {
		return err
	}
	return cmd.run()
}

func (cmd *docCmd) flags() error {
	cmd.flagSet = pflag.NewFlagSet("doc", pflag.ExitOnError)
	cmd.flagSet.BoolVar(&cmd.verbose, "v", false, "show verbose output")
	cmd.flagSet.BoolVar(&cmd.text, "t", false, "output text only formatting")
	cmd.flagSet.BoolVar(&cmd.markdown, "m", false, "output markdown formatting")
	cmd.flagSet.BoolVar(&cmd.html, "h", false, "output html formatting")
	cmd.flagSet.BoolVar(&cmd.http, "s", false, "run doc server to view the docs")
	cmd.flagSet.StringVar(&cmd.outputDir, "o", "./doc", "where to output generated documentation")
	cmd.flagSet.Usage = cmd.usage
	return cmd.flagSet.Parse(os.Args[2:])
}

func (cmd *docCmd) usage() {
	fmt.Fprint(os.Stderr, "usage: luaf doc [options] <path>\n")
	cmd.flagSet.PrintDefaults()
}

func (cmd *docCmd) run() error {
	return errors.New("not implemented yet")
}
