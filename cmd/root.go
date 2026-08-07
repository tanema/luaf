// Package cmd is the package that defines command and subcommands for the luaf
// application.
package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/pflag"

	"github.com/tanema/luaf/internal/conf"
	"github.com/tanema/luaf/internal/parse"
	"github.com/tanema/luaf/internal/runtime"
)

type (
	command interface {
		flags() error
		run() error
	}
	rootCmd struct {
		vm          *runtime.VM
		listOpcodes bool
		parseOnly   bool
		showVersion bool
		executeStat string
		interactive bool
		warningsOn  bool
		flagSet     *pflag.FlagSet
	}
)

var subcommands = map[string]command{
	"test": &testCmd{},
	"doc":  &docCmd{},
}

// Exec is the main entrypoint that parses the command line args to decide how
// the application should react.
func Exec(args []string) error {
	var cmd command = &rootCmd{}
	if len(args) > 0 {
		if sub, found := subcommands[args[0]]; found {
			cmd = sub
		}
	}
	if err := cmd.flags(); err != nil {
		return fmt.Errorf("error while parsing command line arguments: %w", err)
	}
	return cmd.run()
}

func (cmd *rootCmd) flags() error {
	cmd.flagSet = pflag.NewFlagSet("luaf", pflag.ExitOnError)
	cmd.flagSet.BoolVarP(&cmd.listOpcodes, "list", "l", false, "list opcodes parsed from the code.")
	cmd.flagSet.BoolVarP(&cmd.parseOnly, "parse-only", "p", false, "only parse the lua code, do not execute it.")
	cmd.flagSet.BoolVarP(&cmd.showVersion, "version", "v", false, "show version information")
	cmd.flagSet.StringVarP(&cmd.executeStat, "execute", "e", "", "execute string 'stat'")
	cmd.flagSet.BoolVarP(&cmd.interactive, "interactive", "i", false, "enter interactive mode after executing a script")
	cmd.flagSet.BoolVarP(&cmd.warningsOn, "warnings-on", "W", false, "turn warnings on")
	cmd.flagSet.Usage = cmd.usage
	return cmd.flagSet.Parse(os.Args[1:])
}

func (cmd *rootCmd) usage() {
	fmt.Fprint(os.Stderr, "usage: luaf [options] [script [args]]\n")
	fmt.Fprint(os.Stderr, "\nFlags:\n")
	cmd.flagSet.PrintDefaults()
	fmt.Fprint(os.Stderr, "\nSubcommands:\n")
	fmt.Fprint(os.Stderr, "  test\tRun automated tests at specified paths\n")
	fmt.Fprint(os.Stderr, "  doc \tGenerate documentation for project\n")
	fmt.Fprint(os.Stderr, "\n")
}

func (cmd *rootCmd) run() error {
	var err error
	runtime.WarnEnabled = cmd.warningsOn

	cmd.vm, err = runtime.New(context.Background(), nil, fmtCLIArgs(cmd.flagSet)...)
	if err != nil {
		return err
	}

	defer func() { _ = cmd.vm.Close() }()

	args := cmd.flagSet.Args()
	if cmd.showVersion {
		cmd.printVersion()
	}
	if stat, _ := os.Stdin.Stat(); (stat.Mode() & os.ModeCharDevice) == 0 {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		return cmd.parseSrc("<stdin>", strings.NewReader(string(data)))
	} else if cmd.executeStat != "" {
		return cmd.parseSrc("<string>", strings.NewReader(cmd.executeStat))
	} else if len(args) == 0 && !cmd.showVersion {
		return cmd.runREPL()
	} else if len(args) > 0 {
		if info, err := os.Stat(args[0]); err == nil && !info.IsDir() {
			src, err := os.Open(args[0])
			if err != nil {
				return err
			}
			defer func() { _ = src.Close() }()
			return cmd.parseSrc(args[0], src)
		}
	} else if !cmd.showVersion {
		cmd.printVersion()
	}
	return nil
}

func (cmd *rootCmd) printVersion() {
	fmt.Fprintf(os.Stderr, " ⦿  %v\n", conf.FullVersion())
}

func (cmd *rootCmd) parseSrc(path string, src io.ReadSeeker) error {
	fn, err := parse.Parse(path, src, parse.ModeText)
	if err != nil {
		return err
	}
	if cmd.listOpcodes {
		fmt.Fprintln(os.Stderr, fn.String())
	}
	if !cmd.parseOnly {
		_, err = cmd.vm.Eval(fn)
		if err != nil {
			return err
		}
	}
	if cmd.interactive {
		return cmd.runREPL()
	}
	return nil
}

func (cmd *rootCmd) runREPL() error {
	cmd.printVersion()
	fmt.Fprint(os.Stderr, "Press ctrl-c to quit or clear current buffer.\n")
	return cmd.vm.REPL()
}

func fmtCLIArgs(flagSet *pflag.FlagSet) []string {
	flagArgs := flagSet.Args()
	args := make([]string, len(flagArgs))
	copy(args, flagArgs)

	dashIdx := flagSet.ArgsLenAtDash()
	if dashIdx < 0 {
		return args
	}
	return append(args[:dashIdx], append([]string{"--"}, args[dashIdx:]...)...)
}
