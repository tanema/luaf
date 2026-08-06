// Package main is the main entrypoint to the luaf application
package main

import (
	"fmt"
	"os"
	"runtime/pprof"

	"github.com/tanema/luaf/cmd"
)

func main() {
	if err := runCommand(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func runCommand(args []string) error {
	if os.Getenv("LUAF_PROFILE") != "" {
		f, err := os.Create(os.Getenv("LUAF_PROFILE"))
		if err != nil {
			return err
		} else if err = pprof.StartCPUProfile(f); err != nil {
			return err
		}
		defer pprof.StopCPUProfile()
	}
	return cmd.Exec(args)
}
