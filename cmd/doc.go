package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/pflag"

	"github.com/tanema/luaf/internal/ldoc"
	"github.com/tanema/luaf/internal/parse"
)

var docCommand = Command{
	Name:        "doc",
	Usage:       "luaf doc [options] <path>",
	Description: "Generate documentation for project",
	Flags: func(fs *pflag.FlagSet) {
		fs.StringP("format", "f", "text", "what format to generate documentation (text, markdown, html, json)")
		fs.BoolP("notes", "n", false, "show todo notes in output document")
		fs.BoolP("private", "p", false, "show documentation on private variables as well.")
		fs.BoolP("package", "k", false, "show documentation on package variables as well.")
		fs.BoolP("todos", "t", false, "list todo's and warnings marked in comments")
		fs.BoolP("serve", "s", false, "run doc server to view the docs")
		fs.StringP("output", "o", "./doc", "where to output generated documentation")
	},
	Cmd: func(flags *pflag.FlagSet, args []string) error {
		if len(args) == 0 {
			return errors.New("missing path to document")
		}
		info, err := os.Stat(args[0])
		if err != nil {
			return err
		} else if info.IsDir() {
			return fmt.Errorf("%s is a directory, only single files can be documented", args[0])
		}
		src, err := os.Open(args[0])
		if err != nil {
			return err
		}
		defer func() { _ = src.Close() }()

		showAnchors := mustFlag(flags.GetBool("notes"))
		showPrivate := mustFlag(flags.GetBool("private"))
		showPackage := mustFlag(flags.GetBool("package"))
		listTodos := mustFlag(flags.GetBool("todos"))
		doc := ldoc.New(ldoc.Config{
			ShowPrivate: showPrivate,
			ShowPackage: showPackage,
			ShowAnchors: showAnchors || listTodos,
		})
		cfg, err := systemConf()
		if err != nil {
			return err
		}
		if _, err = parse.ParseDoc(args[0], src, parse.ModeText, cfg, doc); err != nil {
			return err
		}

		format := mustFlag(flags.GetString("format"))
		if err != nil {
			return err
		}

		if listTodos {
			for _, a := range doc.Anchors {
				if _, err := fmt.Fprintf(os.Stdout, "● [%s] %s %s\n", a.Label, a.Address, a.Message); err != nil {
					return err
				}
			}
			return nil
		}

		var out string
		switch format {
		case "text":
			out = doc.String()
		case "markdown":
			out = doc.GenerateMarkdown()
		case "html":
			out = doc.GenerateHTML()
		case "json":
			var outBytes []byte
			outBytes, err = json.Marshal(doc)
			if err != nil {
				return err
			}
			out = string(outBytes)
		default:
			return fmt.Errorf("unknown output format %s", format)
		}

		_, err = fmt.Fprint(os.Stdout, out)
		return err
	},
}
