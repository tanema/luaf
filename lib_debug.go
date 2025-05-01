package luaf

func createDebugLib() *Table {
	return &Table{
		hashtable: map[any]any{
			"debug":     Fn("debug.debug", stdDebug),
			"traceback": Fn("debug.traceback", stdDebugTraceback),
		},
	}
}

func stdDebug(*VM, []any) ([]any, error) {
	return nil, &Interrupt{kind: InterruptDebug}
}

func stdDebugTraceback(vm *VM, _ []any) ([]any, error) {
	tbl := NewTable(nil, nil)
	for i := range vm.callDepth {
		tbl.val = append(tbl.val, vm.callStack[i])
	}
	return []any{tbl}, nil
}
