package luaf

import "sync"

type UpvalueBroker struct {
	index     uint64
	open      bool
	name      string
	stackLock *sync.Mutex
	stack     *[]Value
	val       Value
}

func (vm *VM) newUpValueBroker(name string, val Value, index uint64) *UpvalueBroker {
	return &UpvalueBroker{
		stackLock: &vm.stackLock,
		stack:     &vm.Stack,
		name:      name,
		val:       val,
		index:     index,
		open:      true,
	}
}

func (b *UpvalueBroker) Get() Value {
	if b.open {
		b.stackLock.Lock()
		defer b.stackLock.Unlock()
		return (*b.stack)[b.index]
	}
	return b.val
}

func (b *UpvalueBroker) Set(val Value) {
	if b.open {
		b.stackLock.Lock()
		defer b.stackLock.Unlock()
		(*b.stack)[b.index] = val
	}
	b.val = val
}

func (b *UpvalueBroker) Close() {
	if !b.open {
		return
	}
	b.stackLock.Lock()
	defer b.stackLock.Unlock()
	b.val = (*b.stack)[b.index]
	b.open = false
	b.stack = nil
}
