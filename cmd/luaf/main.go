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

	"github.com/tanema/luaf/src/conf"
	"github.com/tanema/luaf/src/parse"
	"github.com/tanema/luaf/src/runtime"
)

var (
	vm          *runtime.VM
	listOpcodes bool
	parseOnly   bool
	showVersion bool
	executeStat string
	interactive bool
	warningsOn  bool
)

func init() {
	flag.BoolVar(&listOpcodes, "l", false, "list opcodes")
	flag.BoolVar(&parseOnly, "p", false, "parse only")
	flag.BoolVar(&showVersion, "v", false, "show version information")
	flag.StringVar(&executeStat, "e", "", "execute string 'stat'")
	flag.BoolVar(&interactive, "i", false, "enter interactive mode after executing a script")
	flag.BoolVar(&warningsOn, "W", false, "turn warnings on")
}

func main() {
	if os.Getenv("LUAF_PROFILE") != "" {
		defer runProfiling(os.Getenv("LUAF_PROFILE"))()
	}
	flag.Usage = printUsage
	flag.Parse()

	runtime.WarnEnabled = warningsOn
	vm = runtime.New(context.Background(), nil, os.Args...)
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
		parseSrc("<stdin>", strings.NewReader(string(data)))
	} else if executeStat != "" {
		parseSrc("<string>", strings.NewReader(executeStat))
	} else if len(args) == 0 && !showVersion {
		runREPL()
	} else if len(args) > 0 {
		if info, err := os.Stat(args[0]); err == nil && !info.IsDir() {
			src, err := os.Open(args[0])
			checkErr(err)
			defer func() { _ = src.Close() }()
			parseSrc(args[0], src)
		}
	} else if !showVersion {
		printUsage()
	}
}

func printVersion() {
	fmt.Fprintf(os.Stderr, "%v\n", conf.FullVersion())
}

func printUsage() {
	printVersion()
	fmt.Fprint(os.Stderr, "\nUsage: luaf [options] [script [args]]\n")
	flag.PrintDefaults()
}

func checkErr(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func parseSrc(path string, src io.ReadSeeker) {
	fn, err := parse.Parse(path, src, parse.ModeText)
	checkErr(err)
	if listOpcodes {
		fmt.Fprintln(os.Stderr, fn.String())
	}
	if !parseOnly {
		_, err = vm.Eval(fn)
		checkErr(err)
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
	checkErr(pprof.StartCPUProfile(f))
	return pprof.StopCPUProfile
}
