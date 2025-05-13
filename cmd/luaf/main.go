// Package main is the main entrypoint to the luaf application
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime/pprof"
	"slices"
	"strings"

	"github.com/tanema/luaf"
)

var (
	vm          *luaf.VM
	listOpcodes bool
	parseOnly   bool
	showVersion bool
	executeStat string
	interactive bool
	warningsOn  bool
)

const execDefaultVal = "not_valid_empty_value_that_isnt_empty"

func init() {
	flag.BoolVar(&listOpcodes, "l", false, "list opcodes")
	flag.BoolVar(&parseOnly, "p", false, "parse only")
	flag.BoolVar(&showVersion, "v", false, "show version information")
	flag.StringVar(&executeStat, "e", execDefaultVal, "execute string 'stat'")
	flag.BoolVar(&interactive, "i", false, "enter interactive mode after executing a script")
	flag.BoolVar(&warningsOn, "W", false, "turn warnings on")
}

func main() {
	if os.Getenv("LUAF_PROFILE") != "" {
		defer runProfiling(os.Getenv("LUAF_PROFILE"))()
	}
	flag.Parse()

	luaf.WarnEnabled = warningsOn
	vm = luaf.NewVM(context.Background(), os.Args...)
	defer func() { _ = vm.Close() }()

	args := flag.Args()
	if slices.Contains(os.Args, "--") {
		if idx := slices.Index(args, "--"); idx >= 0 {
			args = args[0:idx]
		} else {
			args = []string{}
		}
	}

	if showVersion {
		printVersion()
	}
	if stat, _ := os.Stdin.Stat(); (stat.Mode() & os.ModeCharDevice) == 0 {
		data, err := io.ReadAll(os.Stdin)
		checkErr(err)
		parse("<stdin>", strings.NewReader(string(data)))
	} else if executeStat != execDefaultVal {
		parse("<string>", strings.NewReader(executeStat))
	} else if len(args) == 0 && !showVersion {
		runREPL()
	} else if len(args) > 0 {
		if info, err := os.Stat(args[0]); err == nil && !info.IsDir() {
			src, err := os.Open(args[0])
			checkErr(err)
			defer func() { _ = src.Close() }()
			parse(args[0], src)
		}
	} else if !showVersion {
		printUsage()
	}
}

func printVersion() {
	fmt.Fprintf(os.Stderr, "%v %v\n", luaf.LUAVERSION, luaf.LUACOPYRIGHT)
}

func printUsage() {
	printVersion()
	fmt.Fprint(os.Stderr, "usage: luaf [options] [script [args]]\n")
	flag.PrintDefaults()
}

func checkErr(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func parse(path string, src io.ReadSeeker) {
	fn, err := luaf.Parse(path, src, luaf.ModeText)
	checkErr(err)
	if !parseOnly {
		_, err = vm.Eval(fn)
		checkErr(err)
	}
	if listOpcodes {
		fmt.Fprintln(os.Stderr, fn.String())
	}
	if interactive {
		runREPL()
	}
}

func runREPL() {
	printVersion()
	fmt.Fprint(os.Stderr, "Press ctrl-c to quit or clear current buffer.\n")
	checkErr(vm.REPL())
}

func runProfiling(filename string) func() {
	f, err := os.Create(filename)
	checkErr(err)
	fmt.Fprintf(os.Stderr, "Started Profiling: %v", f.Name())
	checkErr(pprof.StartCPUProfile(f))
	return pprof.StopCPUProfile
}
