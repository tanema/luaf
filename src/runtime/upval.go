package runtime

import (
	"fmt"
	"sync"
)

type upvalueBroker struct {
	val       any
	stackLock *sync.Mutex
	stack     *[]any
	name      string
	index     uint64
	open      bool
}

func (vm *VM) newUpValueBroker(name string, val any, index uint64) *upvalueBroker {
	return &upvalueBroker{
		stackLock: &vm.stackLock,
		stack:     &vm.Stack,
		name:      name,
		val:       val,
		index:     index,
		open:      true,
	}
}

func (b *upvalueBroker) String() string {
	return fmt.Sprintf("<-id: %v name: %v open: %v->", b.index, b.name, b.open)
}

func (b *upvalueBroker) Get() any {
	if b.open {
		b.stackLock.Lock()
		defer b.stackLock.Unlock()
		return (*b.stack)[b.index]
	}
	return b.val
}

func (b *upvalueBroker) Set(val any) {
	if b.open {
		b.stackLock.Lock()
		defer b.stackLock.Unlock()
		(*b.stack)[b.index] = val
	}
	b.val = val
}

func (b *upvalueBroker) Close() {
	if !b.open {
		return
	}
	b.stackLock.Lock()
	defer b.stackLock.Unlock()
	b.val = (*b.stack)[b.index]
	b.open = false
	b.stack = nil
}
