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
	fn := newFnProto("<repl>", "<main>", newFnProto("", "env", nil, []string{"_ENV"}, false, LineInfo{}), []string{}, true, LineInfo{})
	ifn, err := vm.push(&Closure{val: fn})
	if err != nil {
		return err
	}
	f := vm.newEnvFrame(fn, ifn+1, 0, nil)
	return vm.repl(f)
}

func (vm *VM) repl(f *frame) error {
	p := NewParser()
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

		p.lex = newLexer(strings.NewReader(buf.String()))
		if err = p.stat(f.fn); err != nil {
			if err == io.EOF {
				rl.SetPrompt("...> ")
				continue
			}
			p.lex = newLexer(strings.NewReader(buf.String()))
			if err = p.stat(f.fn); err != nil {
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
		if res, err := vm.eval(f); err != nil {
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
