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
		parse("<stdin>", os.Stdin)
	} else if len(os.Args) == 1 {
		runREPL()
	} else if info, err := os.Stat(os.Args[1]); err == nil && !info.IsDir() {
		parseFile(os.Args[1])
	} else {
		parse("<strin>", bytes.NewBufferString(os.Args[1]))
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
	parse(path, src)
}

func parse(path string, src io.Reader) {
	if _, err := shine.Parse(path, src); err != nil {
		fatal("Error: %v", err)
	}
}

func runREPL() {
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
		if _, err := shine.Parse("<repl>", bytes.NewBufferString(src)); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}
}
