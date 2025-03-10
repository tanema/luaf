package luaf

import (
	"context"
	"fmt"
)

type (
	ThreadState string
	Thread      struct {
		vm     *VM
		fn     *Closure
		cancel func()
		status ThreadState
	}
)

const (
	threadStateRunning   ThreadState = "running"
	threadStateSuspended ThreadState = "suspended"
	threadStateNormal    ThreadState = "normal"
	threadStateDead      ThreadState = "dead"
)

var threadMetatable = &Table{
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

var libCoroutine = &Table{
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

func newThread(vm *VM, fn *Closure) *Thread {
	ctx, cancel := context.WithCancel(vm.ctx)
	newVM := NewEnvVM(ctx, vm.env)
	newVM.yieldable = true
	return &Thread{
		vm:     newVM,
		fn:     fn,
		cancel: cancel,
		status: threadStateNormal,
	}
}

func (t *Thread) Type() string   { return "thread" }
func (t *Thread) Val() any       { return t }
func (t *Thread) String() string { return fmt.Sprintf("thread %p", t) }
func (t *Thread) Meta() *Table   { return threadMetatable }

func stdThreadCreate(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(vm, args, "coroutine.create", "function"); err != nil {
		return nil, err
	}
	cls, isCls := args[0].(*Closure)
	if !isCls {
		return nil, argumentErr(vm, 1, "coroutine.create", fmt.Errorf("cannot create coroutine from builtin function"))
	}
	return []Value{newThread(vm, cls)}, nil
}

func stdThreadIsYieldable(vm *VM, args []Value) ([]Value, error) {
	return []Value{&Boolean{val: vm.yieldable}}, nil
}

func stdThreadRunning(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(vm, args, "coroutine.running", "thread"); err != nil {
		return nil, err
	}
	return []Value{&Boolean{val: args[0].(*Thread).status == threadStateRunning}}, nil
}

func stdThreadStatus(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(vm, args, "coroutine.status", "thread"); err != nil {
		return nil, err
	}
	return []Value{&String{val: string(args[0].(*Thread).status)}}, nil
}

func stdThreadClose(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(vm, args, "coroutine.close", "thread"); err != nil {
		return nil, err
	}
	args[0].(*Thread).cancel()
	return []Value{}, nil
}

func stdThreadResume(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(vm, args, "coroutine.resume", "thread"); err != nil {
		return nil, err
	}
	thread := args[0].(*Thread)
	return thread.vm.Call(thread.fn, args[1:])
}

func stdThreadYield(vm *VM, args []Value) ([]Value, error) {
	return nil, &Interrupt{kind: InterruptYield}
}

func stdThreadWrap(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(vm, args, "coroutine.wrap", "function"); err != nil {
		return nil, err
	}
	cls, isCls := args[0].(*Closure)
	if !isCls {
		return nil, argumentErr(vm, 1, "coroutine.wrap", fmt.Errorf("cannot create coroutine from builtin function"))
	}
	resume := func(vm *VM, args []Value) ([]Value, error) {
		thread := newThread(vm, cls)
		thread.status = threadStateRunning
		defer func() { thread.status = threadStateDead }()
		return thread.vm.Call(thread.fn, args)
	}
	return []Value{Fn("coroutine.resume", resume)}, nil
}

func stdThreadToString(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(vm, args, "thread:__tostring", "thread"); err != nil {
		return nil, err
	}
	return []Value{&String{val: args[0].String()}}, nil
}
