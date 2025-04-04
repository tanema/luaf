package luaf

func createDebugLib() *Table {
	return &Table{
		hashtable: map[any]Value{
			"debug":     Fn("debug.debug", stdDebug),
			"traceback": Fn("debug.traceback", stdDebugTraceback),
		},
	}
}

func stdDebug(*VM, []Value) ([]Value, error) {
	return nil, &Interrupt{kind: InterruptDebug}
}

func stdDebugTraceback(vm *VM, _ []Value) ([]Value, error) {
	tbl := NewTable(nil, nil)
	for i := range vm.callStack.top {
		tbl.val = append(tbl.val, &String{val: vm.callStack.data[i].String()})
	}
	return []Value{tbl}, nil
}
