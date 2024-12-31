package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime/pprof"
	"strings"

	"github.com/chzyer/readline"
	"github.com/tanema/luaf"
)

var (
	vm          *luaf.VM
	listOpcodes bool
	parseOnly   bool
	showVersion bool
	executeStat string
	interactive bool
)

func init() {
	flag.BoolVar(&listOpcodes, "l", false, "list opcodes")
	flag.BoolVar(&parseOnly, "p", false, "parse only")
	flag.BoolVar(&showVersion, "v", false, "show version information")
	flag.StringVar(&executeStat, "e", "", "execute string 'stat'")
	flag.BoolVar(&interactive, "i", false, "enter interactive mode after executing a script")
}

func main() {
	vm = luaf.NewVM(context.Background())
	if os.Getenv("LUAF_PROFILE") != "" {
		defer runProfiling()()
	}
	flag.Parse()
	args := flag.Args()
	if showVersion {
		printVersion()
	} else if stat, _ := os.Stdin.Stat(); (stat.Mode() & os.ModeCharDevice) == 0 {
		checkErr(parse("<stdin>", os.Stdin))
	} else if executeStat != "" {
		checkErr(parse("<string>", bytes.NewBufferString(executeStat)))
	} else if len(args) == 0 {
		runREPL()
	} else if info, err := os.Stat(args[0]); err == nil && !info.IsDir() {
		checkErr(parse(args[0], openFile(args[0])))
	} else {
		printUsage()
	}
}

func printVersion() {
	fmt.Fprintf(os.Stderr, "%v %v\n", luaf.LUA_VERSION, luaf.LUA_COPYWRITE)
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

func openFile(path string) io.Reader {
	src, err := os.Open(path)
	checkErr(err)
	return src
}

func parse(path string, src io.Reader) error {
	fn, err := luaf.Parse(path, src)
	if err != nil {
		return err
	}
	if !parseOnly {
		_, err = vm.Eval(fn)
	}
	if listOpcodes {
		fmt.Fprintln(os.Stderr, fn.String())
	}
	if interactive {
		runREPL()
	}
	return err
}

func runREPL() {
	printVersion()
	rl, err := readline.New("> ")
	checkErr(err)
	for {
		src, err := rl.Readline()
		if err != nil {
			if err == readline.ErrInterrupt {
				break
			} else {
				fmt.Fprintln(os.Stderr, err)
			}
			continue
		}
		if fn, err := luaf.Parse("<repl>", bytes.NewBufferString(src)); err != nil {
			fmt.Fprintln(os.Stderr, err)
		} else if value, err := vm.Eval(fn); err != nil {
			fmt.Fprintln(os.Stderr, err)
		} else if value != nil {
			printResults(value)
		}
	}
}

func printResults(args []luaf.Value) {
	strParts := make([]string, len(args))
	for i, arg := range args {
		strParts[i] = arg.String()
	}
	fmt.Fprintln(os.Stderr, strings.Join(strParts, "\t"))
}

func runProfiling() func() {
	f, err := os.CreateTemp("", "luaf-*.pprof")
	checkErr(err)
	fmt.Fprintf(os.Stderr, "Started Profiling: %v", f.Name())
	checkErr(pprof.StartCPUProfile(f))
	return pprof.StopCPUProfile
}
