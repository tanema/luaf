package luaf

func createDebugLib() *Table {
	return &Table{
		hashtable: map[any]Value{
			"debug":     Fn("debug.debug", stdDebug),
			"traceback": Fn("debug.traceback", stdDebugTraceback),
		},
	}
}

func stdDebug(vm *VM, args []Value) ([]Value, error) {
	return nil, &Interrupt{kind: InterruptDebug}
}

func stdDebugTraceback(vm *VM, args []Value) ([]Value, error) {
	tbl := NewTable(nil, nil)
	for i := 0; i < vm.callStack.top; i++ {
		tbl.val = append(tbl.val, &String{val: vm.callStack.data[i].String()})
	}
	return []Value{tbl}, nil
}
