package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/pflag"

	"github.com/tanema/luaf/internal/parse"
)

type docCmd struct {
	verbose   bool
	markdown  bool
	text      bool
	html      bool
	http      bool
	outputDir string
}

func (cmd *docCmd) flags() *pflag.FlagSet {
	flagSet := pflag.NewFlagSet("doc", pflag.ExitOnError)
	flagSet.BoolVar(&cmd.verbose, "v", false, "show verbose output")
	flagSet.BoolVar(&cmd.text, "t", false, "output text only formatting")
	flagSet.BoolVar(&cmd.markdown, "m", false, "output markdown formatting")
	flagSet.BoolVar(&cmd.html, "h", false, "output html formatting")
	flagSet.BoolVar(&cmd.http, "s", false, "run doc server to view the docs")
	flagSet.StringVar(&cmd.outputDir, "o", "./doc", "where to output generated documentation")
	return flagSet
}

func (cmd *docCmd) usage() string {
	return "usage: luaf doc [options] <path>"
}

func (cmd *docCmd) run(args []string) error {
	if len(args) > 0 {
		if info, err := os.Stat(args[0]); err == nil && !info.IsDir() {
			src, err := os.Open(args[0])
			if err != nil {
				return err
			}
			defer func() { _ = src.Close() }()

			_, doc, err := parse.Parse(args[0], src, parse.ModeText)
			if err != nil {
				return err
			}
			fmt.Fprint(os.Stderr, doc.String())
		}
	}
	return nil
}
