package main

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/chzyer/readline"
	"github.com/tanema/shine"
)

func main() {
	if stat, _ := os.Stdin.Stat(); (stat.Mode() & os.ModeCharDevice) == 0 {
		mustparse("<stdin>", os.Stdin)
	} else if len(os.Args) == 1 {
		runREPL()
	} else if info, err := os.Stat(os.Args[1]); err == nil && !info.IsDir() {
		parseFile(os.Args[1])
	} else {
		mustparse("<string>", bytes.NewBufferString(os.Args[1]))
	}
}

func fatal(msg string, args ...any) {
	fmt.Printf(msg, args...)
	os.Exit(1)
}

func parseFile(path string) {
	src, err := os.Open(path)
	if err != nil {
		fatal("File Error: %v", err)
	}
	mustparse(path, src)
}

func mustparse(path string, src io.Reader) {
	if err := parse(shine.NewVM(), path, src); err != nil {
		fatal("Error: %v", err)
	}
}

func parse(vm *shine.VM, path string, src io.Reader) error {
	fn, err := shine.Parse(path, src)
	if err != nil {
		return err
	}
	fmt.Println(fn.String())
	err = vm.Eval(fn)
	fmt.Println(vm.Stack, len(vm.Stack))
	return err
}

func runREPL() {
	vm := shine.NewVM()
	rl, err := readline.New("> ")
	if err != nil {
		fatal("Readline Error: %v", err)
	}
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
		if err := parse(vm, "<repl>", bytes.NewBufferString(src)); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}
}
