package luaf

var libDebug = &Table{
	hashtable: map[any]Value{
		"debug": &ExternFunc{stdDebug},
	},
}

func stdDebug(vm *VM, args []Value) ([]Value, error) {
	return []Value{}, vm.REPL()
}
