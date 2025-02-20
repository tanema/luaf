package luaf

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/chzyer/readline"
)

func (vm *VM) REPL() error {
	rl, err := readline.New("> ")
	if err != nil {
		return err
	}
	buf := bytes.NewBuffer(nil)
	for {
		src, err := rl.Readline()
		if err != nil {
			if err == readline.ErrInterrupt {
				if buf.Len() > 0 {
					rl.SetPrompt("> ")
					buf.Reset()
					fmt.Fprint(os.Stderr, "Press ctrl-c again to quit.\n")
					continue
				} else {
					break
				}
			} else {
				fmt.Fprintln(os.Stderr, err)
			}
			continue
		}

		if _, err := buf.WriteString(src + " "); err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}

		fn, err := Parse("<repl>", strings.NewReader(buf.String()), ModeText)
		if err != nil {
			if err == io.EOF {
				rl.SetPrompt("...> ")
				continue
			}
			fn, err = Parse("<repl>", strings.NewReader("return "+buf.String()), ModeText)
			if err != nil {
				if err == io.EOF {
					rl.SetPrompt("...> ")
					continue
				}
				rl.SetPrompt("> ")
				buf.Reset()
				fmt.Fprintln(os.Stderr, err)
				continue
			}
		}

		rl.SetPrompt("> ")
		buf.Reset()
		if res, err := vm.Eval(fn); err != nil {
			fmt.Fprintln(os.Stderr, err)
		} else if res != nil {
			strParts := []string{}
			for _, arg := range res {
				if arg != nil {
					strParts = append(strParts, arg.String())
				}
			}
			if len(strParts) > 0 && len(strings.Join(strParts, "\t")) > 0 {
				fmt.Fprintln(os.Stderr, strings.Join(strParts, "\t"))
			}
		}
	}
	return nil
}
