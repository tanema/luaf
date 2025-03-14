package luaf

import (
	"context"
	"fmt"
)

type (
	ThreadState string
	Thread      struct {
		vm     *VM
		fn     Value
		cancel func()
		status ThreadState
	}
)

const (
	threadStateRunning   ThreadState = "running"
	threadStateSuspended ThreadState = "suspended"
	threadStateDead      ThreadState = "dead"
)

var threadMetatable *Table

func createCoroutineLib() *Table {
	threadMetatable = &Table{
		hashtable: map[any]Value{
			"__name":     &String{val: "THREAD"},
			"__close":    Fn("coroutine.close", stdThreadClose),
			"__tostring": Fn("thread:__tostring", stdThreadToString),
			"__index": &Table{
				hashtable: map[any]Value{
					"close":   Fn("coroutine.close", stdThreadClose),
					"running": Fn("coroutine.running", stdThreadRunning),
					"status":  Fn("coroutine.status", stdThreadStatus),
				},
			},
		},
	}

	return &Table{
		hashtable: map[any]Value{
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

func newThread(vm *VM, fn Value) (*Thread, error) {
	_, isCls := fn.(*Closure)
	_, isFn := fn.(*GoFunc)
	if !isCls && !isFn {
		return nil, fmt.Errorf("cannot create a thread from a %s", fn.Type())
	}
	ctx, cancel := context.WithCancel(vm.ctx)
	newVM := NewEnvVM(ctx, vm.env)
	newVM.yieldable = true
	return &Thread{
		vm:     newVM,
		fn:     fn,
		cancel: cancel,
		status: threadStateSuspended,
	}, nil
}

func (t *Thread) Type() string   { return "thread" }
func (t *Thread) Val() any       { return t }
func (t *Thread) String() string { return fmt.Sprintf("thread %p", t) }
func (t *Thread) Meta() *Table   { return threadMetatable }
func (t *Thread) resume(args []Value) ([]Value, error) {
	wasYielded := t.status == threadStateSuspended && t.vm.yielded
	t.status = threadStateRunning

	var res []Value
	var err error
	if wasYielded {
		res, err = t.vm.resume()
	} else {
		res, err = t.vm.call(t.fn, args)
	}

	if err != nil {
		if intr, isInterrupt := err.(*Interrupt); isInterrupt && intr.kind == InterruptYield {
			t.status = threadStateSuspended
			return res, nil
		}
		return nil, err
	}
	t.status = threadStateDead
	if len(res) == 0 {
		return []Value{&Nil{}}, nil
	}
	return res, nil
}

func stdThreadCreate(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "coroutine.create", "function"); err != nil {
		return nil, err
	}
	thr, err := newThread(vm, args[0])
	return []Value{thr}, err
}

func stdThreadIsYieldable(vm *VM, args []Value) ([]Value, error) {
	return []Value{&Boolean{val: vm.yieldable}}, nil
}

func stdThreadRunning(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "coroutine.running", "thread"); err != nil {
		return nil, err
	}
	return []Value{&Boolean{val: args[0].(*Thread).status == threadStateRunning}}, nil
}

func stdThreadStatus(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "coroutine.status", "thread"); err != nil {
		return nil, err
	}
	return []Value{&String{val: string(args[0].(*Thread).status)}}, nil
}

func stdThreadClose(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "coroutine.close", "thread"); err != nil {
		return nil, err
	}
	args[0].(*Thread).cancel()
	return []Value{}, nil
}

func stdThreadResume(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "coroutine.resume", "thread"); err != nil {
		return nil, err
	}
	thread := args[0].(*Thread)
	return thread.resume(args[1:])
}

func stdThreadYield(vm *VM, args []Value) ([]Value, error) {
	return args, &Interrupt{kind: InterruptYield}
}

func stdThreadWrap(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "coroutine.wrap", "function"); err != nil {
		return nil, err
	}
	thread, err := newThread(vm, args[0])
	if err != nil {
		return nil, err
	}
	resume := func(vm *VM, args []Value) ([]Value, error) {
		return thread.resume(args[1:])
	}
	return []Value{Fn("coroutine.resume", resume)}, nil
}

func stdThreadToString(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "thread:__tostring", "thread"); err != nil {
		return nil, err
	}
	return []Value{&String{val: args[0].String()}}, nil
}
