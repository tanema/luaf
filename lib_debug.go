package luaf

var libDebug = &Table{
	hashtable: map[any]Value{
		"debug": Fn("debug.debug", stdDebug),
	},
}

func stdDebug(vm *VM, args []Value) ([]Value, error) {
	return nil, &Interrupt{kind: InterruptDebug}
}
