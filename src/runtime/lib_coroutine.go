package runtime

import (
	"context"
	"errors"
	"fmt"
)

type (
	threadstate string
	// Thread is a lua coroutine.
	Thread struct {
		vm     *VM
		fn     any
		cancel func()
		status threadstate
	}
)

const (
	threadStateRunning   threadstate = "running"
	threadStateSuspended threadstate = "suspended"
	threadStateDead      threadstate = "dead"
)

var threadMetatable *Table

func createCoroutineLib() *Table {
	threadMetatable = &Table{
		hashtable: map[any]any{
			"__name":     "THREAD",
			"__close":    Fn("coroutine.close", stdThreadClose),
			"__tostring": Fn("thread:__tostring", stdThreadToString),
			"__index": &Table{
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

func (t *Thread) String() string {
	return fmt.Sprintf("thread %p", t)
}

func newThread(vm *VM, fn any) (*Thread, error) {
	_, isCls := fn.(*Closure)
	_, isFn := fn.(*GoFunc)
	if !isCls && !isFn {
		return nil, fmt.Errorf("cannot create a thread from a %s", typeName(fn))
	}
	ctx, cancel := context.WithCancel(vm.ctx)
	newVM := New(ctx, vm.env)
	newVM.yieldable = true
	return &Thread{
		vm:     newVM,
		fn:     fn,
		cancel: cancel,
		status: threadStateSuspended,
	}, nil
}

func (t *Thread) resume(args []any) ([]any, error) {
	wasYielded := t.status == threadStateSuspended && t.vm.yielded
	t.status = threadStateRunning

	var res []any
	var err error
	if wasYielded {
		res, err = t.vm.resume()
	} else {
		res, err = t.vm.call(t.fn, args)
	}

	if err != nil {
		var intr *Interrupt
		if errors.As(err, &intr) && intr.kind == InterruptYield {
			t.status = threadStateSuspended
			return res, nil
		}
		return nil, err
	}
	t.status = threadStateDead
	if len(res) == 0 {
		return []any{nil}, nil
	}
	return res, nil
}

func stdThreadCreate(vm *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "coroutine.create", "function"); err != nil {
		return nil, err
	}
	thr, err := newThread(vm, args[0])
	return []any{thr}, err
}

func stdThreadIsYieldable(vm *VM, _ []any) ([]any, error) {
	return []any{vm.yieldable}, nil
}

func stdThreadRunning(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "coroutine.running", "thread"); err != nil {
		return nil, err
	}
	thread, _ := args[0].(*Thread)
	return []any{thread.status == threadStateRunning}, nil
}

func stdThreadStatus(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "coroutine.status", "thread"); err != nil {
		return nil, err
	}
	thread, _ := args[0].(*Thread)
	return []any{string(thread.status)}, nil
}

func stdThreadClose(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "coroutine.close", "thread"); err != nil {
		return nil, err
	}
	thread, _ := args[0].(*Thread)
	thread.cancel()
	return []any{}, nil
}

func stdThreadResume(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "coroutine.resume", "thread"); err != nil {
		return nil, err
	}
	thread, _ := args[0].(*Thread)
	return thread.resume(args[1:])
}

func stdThreadYield(_ *VM, args []any) ([]any, error) {
	return args, &Interrupt{kind: InterruptYield}
}

func stdThreadWrap(vm *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "coroutine.wrap", "function"); err != nil {
		return nil, err
	}
	thread, err := newThread(vm, args[0])
	if err != nil {
		return nil, err
	}
	resume := func(_ *VM, args []any) ([]any, error) {
		return thread.resume(args)
	}
	return []any{Fn("coroutine.resume", resume)}, nil
}

func stdThreadToString(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "thread:__tostring", "thread"); err != nil {
		return nil, err
	}
	return []any{ToString(args[0])}, nil
}
