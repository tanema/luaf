package runtime

import (
	"errors"

	"github.com/tanema/luaf/internal/parse"
)

var threadMetatable *Table

func createCoroutineLib() *Table {
	threadMetatable = &Table{
		hashtable: map[any]any{
			string(parse.MetaName):     "THREAD",
			string(parse.MetaClose):    Fn("coroutine.close", stdThreadClose),
			string(parse.MetaToString): Fn("thread:__tostring", stdThreadToString),
			"RUNNING":                  threadStateRunning,
			"SUSPENDED":                threadStateSuspended,
			"DEAD":                     threadStateDead,
			string(parse.MetaIndex): &Table{
				hashtable: map[any]any{
					"close":   Fn("coroutine.close", stdThreadClose),
					"running": Fn("coroutine.running", stdThreadRunning),
					"status":  Fn("coroutine.status", stdThreadStatus),
				},
			},
		},
	}

	return &Table{
		hashtable: map[any]any{
			"close":       Fn("coroutine.close", stdThreadClose),
			"create":      Fn("coroutine.create", stdThreadCreate),
			"isyieldable": Fn("coroutine.isyeildable", stdThreadIsYieldable),
			"running":     Fn("coroutine.running", stdThreadRunning),
			"status":      Fn("coroutine.status", stdThreadStatus),
			"resume":      Fn("coroutine.resume", stdThreadResume),
			"yield":       Fn("coroutine.yield", stdThreadYield),
			"wrap":        Fn("coroutine.wrap", stdThreadWrap),
		},
	}
}

func stdThreadCreate(vm *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "coroutine.create", "function"); err != nil {
		return nil, err
	}
	newVM, err := vm.newYieldable(args[0])
	if err != nil {
		return nil, err
	}
	return []any{newVM}, err
}

func stdThreadIsYieldable(vm *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "coroutine.isyieldable", "~thread"); err != nil {
		return nil, err
	}
	if len(args) > 0 {
		return []any{args[0].(*VM).yieldable}, nil
	}
	return []any{vm.yieldable}, nil
}

func stdThreadRunning(vm *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "coroutine.running", "~thread"); err != nil {
		return nil, err
	}
	status := vm.status
	if len(args) > 0 {
		status = args[0].(*VM).status
	}
	return []any{vm, status == threadStateRunning}, nil
}

func stdThreadStatus(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "coroutine.status", "thread"); err != nil {
		return nil, err
	}
	return []any{string(args[0].(*VM).status)}, nil
}

func stdThreadClose(vm *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "coroutine.close", "thread"); err != nil {
		return nil, err
	}
	thread := vm
	if len(args) > 0 {
		thread = args[0].(*VM)
	}

	if !thread.yieldable {
		return nil, errors.New("cannot close main thread")
	}

	thread.cancel()
	return []any{}, nil
}

func stdThreadResume(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "coroutine.resume", "thread"); err != nil {
		return nil, err
	}
	return args[0].(*VM).resume(args[1:])
}

func stdThreadYield(vm *VM, args []any) ([]any, error) {
	if !vm.yieldable {
		return nil, errors.New("cannot yield from outside a coroutine")
	}
	return args, &Interrupt{kind: InterruptYield}
}

func stdThreadWrap(vm *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "coroutine.wrap", "function"); err != nil {
		return nil, err
	}

	newVM, err := vm.newYieldable(args[0])
	resume := func(_ *VM, args []any) ([]any, error) { return newVM.resume(args) }
	return []any{Fn("coroutine.resume", resume)}, err
}

func stdThreadToString(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "thread:__tostring", "thread"); err != nil {
		return nil, err
	}
	return []any{ToString(args[0])}, nil
}
