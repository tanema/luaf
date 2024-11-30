package luaf

import (
	"strings"
)

var libTable = &Table{
	hashtable: map[any]Value{
		"concat": &ExternFunc{stdTableConcat},
		"insert": &ExternFunc{stdTableInsert},
		"move":   &ExternFunc{stdTableMove},
		"pack":   &ExternFunc{stdTablePack},
		"remove": &ExternFunc{stdTableRemove},
		"sort":   &ExternFunc{stdTableSort},
		"unpack": &ExternFunc{stdTableUnpack},
	},
}

func stdTableConcat(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(vm, args, "concat", "table", "~string", "~number", "~number"); err != nil {
		return nil, err
	}
	tbl := args[0].(*Table).val
	sep := ""
	i, j := int64(0), int64(len(tbl))
	if len(args) > 1 {
		sep = args[1].(*String).val
	}
	if len(args) > 2 {
		i = toInt(args[2])
	}
	if len(args) > 3 {
		j = toInt(args[3])
	}
	strParts := make([]string, len(args))
	for i := i; i < j; i++ {
		strParts[i] = tbl[i].String()
	}
	return []Value{&String{val: strings.Join(strParts, sep)}}, nil
}

func stdTableInsert(vm *VM, args []Value) ([]Value, error) {
	return nil, nil
}

func stdTableMove(vm *VM, args []Value) ([]Value, error) {
	return nil, nil
}

func stdTableRemove(vm *VM, args []Value) ([]Value, error) {
	return nil, nil
}

func stdTablePack(vm *VM, args []Value) ([]Value, error) {
	return []Value{NewTable(args, nil)}, nil
}

func stdTableSort(vm *VM, args []Value) ([]Value, error) {
	return nil, nil
}

func stdTableUnpack(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(vm, args, "unpack", "table"); err != nil {
		return nil, err
	}
	tbl := args[0].(*Table)
	return tbl.val, nil
}
