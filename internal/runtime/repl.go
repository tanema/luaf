package runtime

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/chzyer/readline"
	"github.com/tanema/luaf/internal/parse"
)

// REPL will start an interactive repl parsing and running lua code.
func (vm *VM) REPL() error {
	fn := parse.NewEmptyFnProto("<repl>", nil)
	ifn, err := vm.push(&Closure{val: fn})
	if err != nil {
		return err
	}
	f := vm.newEnvFrame(fn, ifn+1, nil)
	return vm.repl(f)
}

func (vm *VM) repl(f *frame) error {
	rl, err := readline.New("> ")
	if err != nil {
		return err
	}
	buf := bytes.NewBuffer(nil)
	for {
		src, err := rl.Readline()
		if err != nil {
			if errors.Is(err, readline.ErrInterrupt) {
				if buf.Len() > 0 {
					rl.SetPrompt("> ")
					buf.Reset()
					fmt.Fprint(os.Stderr, "Press ctrl-c again to quit.\n")
					continue
				}
				break
			}
			if errors.Is(err, io.EOF) {
				break
			}
			fmt.Fprintln(os.Stderr, err)
			continue
		}

		if _, err = buf.WriteString(src + "\n"); err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}

		replFn, err := parse.TryStat(buf.String(), f.fn)
		if err != nil {
			if errors.Is(err, io.EOF) {
				rl.SetPrompt("...> ")
				continue
			}
			rl.SetPrompt("> ")
			buf.Reset()
			fmt.Fprintln(os.Stderr, err)
			continue
		}

		rl.SetPrompt("> ")
		buf.Reset()
		if res, err := vm.evalInContext(replFn, f); err != nil {
			fmt.Fprintln(os.Stderr, err)
		} else if len(res) > 0 {
			strParts := make([]string, len(res))
			for i, arg := range res {
				strParts[i] = ToString(arg)
			}
			fmt.Fprintln(os.Stderr, strings.Join(strParts, "\t"))
		}
	}
	return nil
}

// evalInContext evaluates replFn in the context of an existing frame, building
// upvalue brokers from ctx so that locals and upvalues of the calling scope are
// accessible. This is the same broker construction the CLOSURE opcode uses.
func (vm *VM) evalInContext(replFn *parse.FnProto, ctx *frame) ([]any, error) {
	ifn, err := vm.push(&Closure{val: replFn})
	if err != nil {
		return nil, err
	}
	upvals := make([]*upvalueBroker, len(replFn.UpIndexes))
	for i, idx := range replFn.UpIndexes {
		if idx.FromStack {
			stackIdx := uint64(ctx.framePointer) + uint64(idx.Index)
			if j, ok := search(ctx.openBrokers, stackIdx, findBroker); ok {
				upvals[i] = ctx.openBrokers[j]
			} else {
				newBroker := vm.newUpValueBroker(idx.Name, vm.get(ctx, int64(idx.Index), false), stackIdx)
				ctx.openBrokers = append(ctx.openBrokers, newBroker)
				upvals[i] = newBroker
			}
		} else if int(idx.Index) < len(ctx.upvals) {
			upvals[i] = ctx.upvals[idx.Index]
		}
	}
	return vm.eval(vm.newFrame(replFn, ifn+1, 0, upvals, vm.vmargs...), true)
}
