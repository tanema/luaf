package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/chzyer/readline"
	"github.com/tanema/luaf"
)

const (
	VERSION = "0.0.1"
	YEAR    = "2024"
)

var (
	listOpcodes bool
	parseOnly   bool
	showVersion bool
	executeStat string
	interactive bool

	vm = luaf.NewVM()
)

func init() {
	flag.BoolVar(&listOpcodes, "l", false, "list opcodes")
	flag.BoolVar(&parseOnly, "p", false, "parse only")
	flag.BoolVar(&showVersion, "v", false, "show version information")
	flag.StringVar(&executeStat, "e", "", "execute string 'stat'")
	flag.BoolVar(&interactive, "i", false, "enter interactive mode after executing a script")
}

func main() {
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
	fmt.Fprintf(os.Stderr, "luaf %v Copyright (C) %v\n", VERSION, YEAR)
}

func printUsage() {
	printVersion()
	fmt.Fprint(os.Stderr, "usage: luaf [options] [script [args]]\n")
	flag.PrintDefaults()
}

func checkErr(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
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
			fmt.Fprintln(os.Stderr, value)
		}
	}
}
