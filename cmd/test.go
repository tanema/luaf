package cmd

import (
	"context"
	"strings"

	"github.com/spf13/pflag"

	"github.com/tanema/luaf/internal/parse"
	"github.com/tanema/luaf/internal/runtime"
)

var testCommand = Command{
	Name:        "test",
	Usage:       "luaf test [options] <path>",
	Description: "Run automated tests at specified paths",
	Flags: func(flagSet *pflag.FlagSet) {
		flagSet.BoolP("verbose", "v", false, "verbose test output.")
	},
	Cmd: runTest,
}

func runTest(flags *pflag.FlagSet, args []string) error {
	cfg, err := systemConf()
	if err != nil {
		return err
	}

	fn, err := parse.Parse("<builtin.testing>", strings.NewReader("__LUAF__BuiltinSupport__.test()"), parse.ModeText, cfg)
	if err != nil {
		return err
	}

	vm, err := runtime.New(context.Background(), cfg, args...)
	if err != nil {
		return err
	}
	defer func() { _ = vm.Close() }()

	if mustFlag(flags.GetBool("verbose")) {
		if err = vm.SetGlobal("verbose", true); err != nil {
			return err
		}
	}
	_, err = vm.Eval(fn)
	return err
}
