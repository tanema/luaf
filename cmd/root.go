// Package cmd is the package that defines command and subcommands for the luaf
// application.
package cmd

import (
	"context"
	"errors"
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
	Command struct {
		Name        string
		Usage       string
		Description string
		Flags       func(*pflag.FlagSet)
		Cmd         func(*pflag.FlagSet, []string) error
		SubCommands []Command
	}
)

var rootCommand = Command{
	Name:        "luaf",
	Usage:       "luaf [options] [script [args]]",
	Description: "Luaf is an implementation of PUC lua",
	Flags: func(flagSet *pflag.FlagSet) {
		flagSet.BoolP("list", "l", false, "list opcodes parsed from the code.")
		flagSet.BoolP("parse-only", "p", false, "only parse the lua code, do not execute it.")
		flagSet.BoolP("version", "v", false, "show version information")
		flagSet.StringP("execute", "e", "", "execute string 'stat'")
		flagSet.BoolP("interactive", "i", false, "enter interactive mode after executing a script")
		flagSet.BoolP("warnings-on", "W", false, "turn warnings on")
	},
	Cmd:         runRootCmd,
	SubCommands: []Command{testCommand, docCommand, fmtCommand, lspCommand},
}

// Exec is the main entrypoint that parses the command line args to decide how
// the application should react.
func Exec(args []string) error {
	cmd := rootCommand
	idx := 1
	if len(args) > 0 {
		name := args[0]
		for _, scmd := range cmd.SubCommands {
			if scmd.Name == name {
				cmd = scmd
				idx = 2
				break
			}
		}
	}
	flagSet := pflag.NewFlagSet(cmd.Name, pflag.ExitOnError)
	if cmd.Flags != nil {
		cmd.Flags(flagSet)
	}
	flagSet.Usage = usageFn(cmd, flagSet)
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

	if cmd.Cmd == nil {
		return errors.New("not implemented yet")
	}

	return cmd.Cmd(flagSet, cmdargs)
}

func usageFn(cmd Command, flagSet *pflag.FlagSet) func() {
	return func() {
		fmt.Fprintf(os.Stderr, "%s\n\n", cmd.Usage)
		fmt.Fprintf(os.Stderr, "%s\n\n", cmd.Description)
		if flagSet.HasFlags() {
			fmt.Fprint(os.Stderr, "Flags:\n")
			flagSet.PrintDefaults()
			fmt.Fprint(os.Stderr, "\n")
		}
		if len(cmd.SubCommands) > 0 {
			fmt.Fprint(os.Stderr, "Subcommands:\n")
			for _, subcommand := range cmd.SubCommands {
				fmt.Fprintf(os.Stderr, "  %s\t%s\n", subcommand.Name, subcommand.Description)
			}
			fmt.Fprint(os.Stderr, "\n")
		}
	}
}

func runRootCmd(fs *pflag.FlagSet, args []string) error {
	showVersion := mustFlag(fs.GetBool("version"))
	listOpcodes := mustFlag(fs.GetBool("list"))
	parseOnly := mustFlag(fs.GetBool("parse-only"))
	interactive := mustFlag(fs.GetBool("interactive"))
	runtime.WarnEnabled = mustFlag(fs.GetBool("warnings-on"))
	var src io.ReadSeeker
	var filename string

	if stat, _ := os.Stdin.Stat(); (stat.Mode() & os.ModeCharDevice) == 0 {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		filename = "<stdin>"
		src = strings.NewReader(string(data))
	} else if execStat := mustFlag(fs.GetString("execute")); execStat != "" {
		filename = "<string>"
		src = strings.NewReader(execStat)
	} else if len(args) == 0 && !showVersion {
		interactive = true
		showVersion = true
	} else if len(args) > 0 {
		filename = args[0]
		info, err := os.Stat(args[0])
		if err != nil {
			return err
		} else if info.IsDir() {
			return fmt.Errorf("cannot evaluate a directory %s", info.Name())
		}
		file, err := os.Open(filename)
		if err != nil {
			return err
		}
		defer func() { _ = file.Close() }()
		src = file
	}

	vm, err := runtime.New(context.Background(), nil, args...)
	if err != nil {
		return err
	}
	defer func() { _ = vm.Close() }()

	if showVersion {
		fmt.Fprintf(os.Stderr, " ⦿  %v\n", conf.FullVersion())
	}

	if src != nil {
		fn, _, err := parse.Parse(filename, src, parse.ModeText)
		if err != nil {
			return err
		}
		if listOpcodes {
			fmt.Fprintln(os.Stderr, fn.String())
		}
		if !parseOnly {
			_, err = vm.Eval(fn)
			if err != nil {
				return err
			}
		}
	}

	if interactive {
		fmt.Fprint(os.Stderr, "Press ctrl-c to quit or clear current buffer.\n")
		return vm.REPL()
	}
	return nil
}

func mustFlag[T any](val T, err error) T {
	if err != nil {
		panic(fmt.Sprintf("failed to get flag: %v", err))
	}
	return val
}
