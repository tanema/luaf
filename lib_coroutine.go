package luaf

import (
	"context"
	"fmt"
)

type (
	ThreadState string
	Thread      struct {
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
		"__close":    &ExternFunc{stdThreadClose},
		"__tostring": &ExternFunc{stdThreadToString},
		"__index": &Table{
			hashtable: map[any]Value{
				"close":   &ExternFunc{stdThreadClose},
				"running": &ExternFunc{stdThreadRunning},
				"status":  &ExternFunc{stdThreadStatus},
			},
		},
	},
}

var libCoroutine = &Table{
	hashtable: map[any]Value{
		"close":       &ExternFunc{stdThreadClose},
		"create":      &ExternFunc{stdThreadCreate},
		"isyieldable": &ExternFunc{stdThreadIsYieldable},
		"running":     &ExternFunc{stdThreadRunning},
		"status":      &ExternFunc{stdThreadStatus},
		"resume":      &ExternFunc{},
		"yield":       &ExternFunc{},
		"wrap":        &ExternFunc{},
	},
}

func (t *Thread) Type() string   { return "thread" }
func (t *Thread) Val() any       { return t }
func (t *Thread) String() string { return fmt.Sprintf("thread %p", t) }
func (t *Thread) Meta() *Table   { return threadMetatable }

func stdThreadCreate(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(vm, args, "coroutine.create", "function"); err != nil {
		return nil, err
	}
	_, cancel := context.WithCancel(vm.ctx)
	return []Value{&Thread{
		cancel: cancel,
		status: threadStateNormal,
	}}, nil
}

func stdThreadIsYieldable(*VM, []Value) ([]Value, error) {
	return []Value{&Boolean{val: true}}, nil
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

func stdThreadToString(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(vm, args, "thread:__tostring", "thread"); err != nil {
		return nil, err
	}
	return []Value{&String{val: args[0].String()}}, nil
}
