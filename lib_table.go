package luaf

import (
	"slices"
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
	if err := assertArguments(vm, args, "table.insert", "table", "value"); err != nil {
		return nil, err
	}
	tbl := args[0].(*Table)
	i := len(tbl.val)
	var val Value
	if isNumber(args[1]) && len(args) > 2 {
		i = int(toInt(args[1]))
		val = args[2]
	} else {
		val = args[1]
	}
	ensureSize(&tbl.val, i-1)
	tbl.val = slices.Insert(tbl.val, i-1, val)
	return []Value{}, nil
}

func stdTableMove(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(vm, args, "table.move", "table", "number", "number", "number", "~table"); err != nil {
		return nil, err
	}
	tbl1 := args[0].(*Table)
	tbl2 := tbl1
	from := toInt(args[1])
	nelem := toInt(args[2])
	to := toInt(args[3])
	if len(args) > 4 {
		tbl2 = args[4].(*Table)
	}
	if int(from) > len(tbl1.val) || from < 0 {
		return []Value{}, nil
	}
	chunk := make([]Value, nelem)
	copy(chunk, tbl1.val[from-1:from+nelem-1])
	tbl1.val = slices.Delete(tbl1.val, int(from-1), int(from+nelem-1))
	ensureSize(&tbl2.val, int(to-1))
	tbl2.val = slices.Insert(tbl2.val, int(to-1), chunk...)
	return []Value{}, nil
}

func stdTableRemove(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(vm, args, "table.remove", "table", "~number"); err != nil {
		return nil, err
	}
	tbl := args[0].(*Table)
	i := len(tbl.val) - 1
	if len(args) > 1 {
		i = int(toInt(args[1]))
	}
	if i > len(tbl.val) || i < 0 {
		return []Value{}, nil
	}
	tbl.val = slices.Delete(tbl.val, i-1, i)
	return []Value{}, nil
}

func stdTablePack(vm *VM, args []Value) ([]Value, error) {
	return []Value{NewTable(args, nil)}, nil
}

func stdTableSort(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(vm, args, "table.sort", "table", "~function"); err != nil {
		return nil, err
	}

	var sortErr error

	tbl := args[0].(*Table)
	if len(args) > 1 {
		fn := args[1].(callable)
		slices.SortFunc(tbl.val, func(l, r Value) int {
			if sortErr != nil {
				return 0
			}
			res, err := vm.Call("table.sort", fn, []Value{l, r})
			if err != nil {
				sortErr = err
			}
			if len(res) > 0 {
				if toBool(res[0]).val {
					return -1
				}
				return 1
			}
			return 0
		})
		return nil, sortErr
	}

	slices.SortFunc(tbl.val, func(l, r Value) int {
		if sortErr != nil {
			return 0
		}
		i, err := vm.compareVal(metaLe, l, r)
		if err != nil {
			sortErr = err
		}
		return i
	})

	return nil, sortErr
}

func stdTableUnpack(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(vm, args, "table.unpack", "table"); err != nil {
		return nil, err
	}
	tbl := args[0].(*Table)
	return tbl.val, nil
}
