package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/chzyer/readline"
	"github.com/tanema/lauf"
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

	vm = lauf.NewVM()
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
	if showVersion {
		printVersion()
	} else if stat, _ := os.Stdin.Stat(); (stat.Mode() & os.ModeCharDevice) == 0 {
		checkErr(parse("<stdin>", os.Stdin))
	} else if executeStat != "" {
		checkErr(parse("<string>", bytes.NewBufferString(executeStat)))
	} else if len(os.Args) == 1 {
		runREPL()
	} else if info, err := os.Stat(os.Args[1]); err == nil && !info.IsDir() {
		checkErr(parse(os.Args[1], openFile(os.Args[1])))
	} else {
		printUsage()
	}
}

func printVersion() {
	fmt.Fprintf(os.Stderr, "Lauf %v Copyright (C) %v", VERSION, YEAR)
}

func printUsage() {
	printVersion()
	fmt.Fprint(os.Stderr, "\nusage: lauf [options] [script [args]]\n")
	flag.PrintDefaults()
}

func checkErr(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v", err)
		os.Exit(1)
	}
}

func openFile(path string) io.Reader {
	src, err := os.Open(path)
	checkErr(err)
	return src
}

func parse(path string, src io.Reader) error {
	fn, err := lauf.Parse(path, src)
	if err != nil {
		return err
	}
	if !parseOnly {
		if err := vm.Eval(fn); err != nil {
			return err
		}
	}
	if listOpcodes {
		fmt.Fprint(os.Stderr, fn.String())
	}
	if interactive {
		runREPL()
	}
	return nil
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
		if fn, err := lauf.Parse("<repl>", bytes.NewBufferString(src)); err != nil {
			fmt.Fprintln(os.Stderr, err)
		} else if err := vm.Eval(fn); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}
}
