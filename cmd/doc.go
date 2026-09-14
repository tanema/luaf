package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/pflag"

	"github.com/tanema/luaf/internal/parse"
)

var docCommand = Command{
	Name:        "doc",
	Usage:       "luaf doc [options] <path>",
	Description: "Generate documentation for project",
	Flags: func(fs *pflag.FlagSet) {
		fs.Bool("v", false, "show verbose output")
		fs.Bool("t", false, "output text only formatting")
		fs.Bool("m", false, "output markdown formatting")
		fs.Bool("h", false, "output html formatting")
		fs.Bool("s", false, "run doc server to view the docs")
		fs.String("o", "./doc", "where to output generated documentation")
	},
	Cmd: func(flags *pflag.FlagSet, args []string) error {
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
	},
}
