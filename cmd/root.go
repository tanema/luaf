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
		flags() *pflag.FlagSet
		run(args []string) error
		usage() string
	}
	rootCmd struct {
		vm          *runtime.VM
		listOpcodes bool
		parseOnly   bool
		showVersion bool
		executeStat string
		interactive bool
		warningsOn  bool
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
	isRoot := true
	idx := 1
	if len(args) > 0 {
		if sub, found := subcommands[args[0]]; found {
			isRoot = false
			cmd = sub
			idx = 2
		}
	}
	flagSet := cmd.flags()
	flagSet.Usage = usageFn(cmd, flagSet, isRoot)
	if err := flagSet.Parse(os.Args[idx:]); err != nil {
		return fmt.Errorf("error while parsing command line arguments: %w", err)
	}

	flagArgs := flagSet.Args()
	cmdargs := make([]string, len(flagArgs))
	copy(cmdargs, flagArgs)

	dashIdx := flagSet.ArgsLenAtDash()
	if dashIdx >= 0 {
		cmdargs = append(cmdargs[:dashIdx], append([]string{"--"}, cmdargs[dashIdx:]...)...)
	}

	return cmd.run(cmdargs)
}

func usageFn(cmd command, flagSet *pflag.FlagSet, isRoot bool) func() {
	return func() {
		fmt.Fprint(os.Stderr, cmd.usage())
		fmt.Fprint(os.Stderr, "\n")
		if flagSet.HasFlags() {
			fmt.Fprint(os.Stderr, "\nFlags:\n")
			flagSet.PrintDefaults()
		}
		if isRoot {
			fmt.Fprint(os.Stderr, "\nSubcommands:\n")
			fmt.Fprint(os.Stderr, "  test\tRun automated tests at specified paths\n")
			fmt.Fprint(os.Stderr, "  doc \tGenerate documentation for project\n")
			fmt.Fprint(os.Stderr, "\n")
		}
	}
}

func (cmd *rootCmd) flags() *pflag.FlagSet {
	flagSet := pflag.NewFlagSet("luaf", pflag.ExitOnError)
	flagSet.BoolVarP(&cmd.listOpcodes, "list", "l", false, "list opcodes parsed from the code.")
	flagSet.BoolVarP(&cmd.parseOnly, "parse-only", "p", false, "only parse the lua code, do not execute it.")
	flagSet.BoolVarP(&cmd.showVersion, "version", "v", false, "show version information")
	flagSet.StringVarP(&cmd.executeStat, "execute", "e", "", "execute string 'stat'")
	flagSet.BoolVarP(&cmd.interactive, "interactive", "i", false, "enter interactive mode after executing a script")
	flagSet.BoolVarP(&cmd.warningsOn, "warnings-on", "W", false, "turn warnings on")
	return flagSet
}

func (cmd *rootCmd) usage() string {
	return "usage: luaf [options] [script [args]]"
}

func (cmd *rootCmd) run(args []string) error {
	var err error
	runtime.WarnEnabled = cmd.warningsOn

	cmd.vm, err = runtime.New(context.Background(), nil, args...)
	if err != nil {
		return err
	}

	defer func() { _ = cmd.vm.Close() }()

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
	fn, _, err := parse.Parse(path, src, parse.ModeText)
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
