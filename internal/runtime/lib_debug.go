package runtime

func createDebugLib() *Table {
	return &Table{
		hashtable: map[any]any{
			"debug":     Fn("debug.debug", stdDebug),
			"traceback": Fn("debug.traceback", stdDebugTraceback),
			"getinfo":   Fn("debug.getinfo", stdDebugGetInfo),
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

// stdDebugGetInfo implements a reduced debug.getinfo(level): only the level form
// is supported (not debug.getinfo(fn, ...)), and only the fields the callstack
// actually tracks are populated (currentline, source, short_src, name) - there's
// no record of where a function was originally *defined*.
func stdDebugGetInfo(vm *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "debug.getinfo", "number"); err != nil {
		return nil, err
	}
	level := toInt(args[0])
	idIdx := vm.callDepth - level
	lineIdx := idIdx + 1
	if level < 1 || idIdx < 0 || lineIdx > vm.callDepth {
		return []any{nil}, nil
	}
	id := vm.callStack[idIdx]
	line := vm.callStack[lineIdx]
	return []any{NewTable(nil, map[any]any{
		"currentline": line.Line,
		"source":      "@" + id.filename,
		"short_src":   id.filename,
		"name":        id.name,
	})}, nil
}
