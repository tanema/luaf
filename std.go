package luaf

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

var stdlib = map[any]Value{
	"_VERSION":       &String{"Luaf 0.0.1"},
	"print":          &ExternFunc{stdPrint},
	"assert":         &ExternFunc{stdAssert},
	"type":           &ExternFunc{stdType},
	"tostring":       &ExternFunc{stdToString},
	"tonumber":       &ExternFunc{stdToNumber},
	"next":           &ExternFunc{stdNext},
	"pairs":          &ExternFunc{stdPairs},
	"ipairs":         &ExternFunc{stdIPairs},
	"setmetatable":   &ExternFunc{stdSetMetatable},
	"getmetatable":   &ExternFunc{stdGetMetatable},
	"dofile":         &ExternFunc{stdDoFile},
	"pcall":          &ExternFunc{stdPCall},
	"xpcall":         &ExternFunc{stdXPCall},
	"rawget":         &ExternFunc{stdRawGet},
	"rawset":         &ExternFunc{stdRawSet},
	"rawequal":       &ExternFunc{stdRawEq},
	"rawlen":         &ExternFunc{stdRawLen},
	"collectgarbage": &ExternFunc{stdCollectgarbage},
	"select":         &ExternFunc{stdSelect},
	"load":           &ExternFunc{stdLoad},
	"loadfile":       &ExternFunc{stdLoadFile},
}

func stdCollectgarbage(vm *VM, args []Value) ([]Value, error) {
	//noop
	return []Value{}, nil
}

func stdPrint(vm *VM, args []Value) ([]Value, error) {
	strParts := make([]string, len(args))
	for i, arg := range args {
		strParts[i] = arg.String()
	}
	fmt.Println(strings.Join(strParts, "\t"))
	return nil, nil
}

func stdAssert(vm *VM, args []Value) ([]Value, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("bad argument #1 to 'assert' (value expected)")
	}
	if toBool(args[0]).val {
		return []Value{args[0]}, nil
	}
	if len(args) > 1 {
		return nil, &Error{val: args[1]}
	}
	return nil, fmt.Errorf("assertion failed")
}

func stdToString(vm *VM, args []Value) ([]Value, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("bad argument #1 to 'tostring' (value expected)")
	}
	return []Value{&String{val: args[0].String()}}, nil
}

func stdToNumber(vm *VM, args []Value) ([]Value, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("bad argument #1 to 'tonumber' (value expected)")
	}
	base := 10
	if len(args) > 1 {
		switch baseVal := args[1].(type) {
		case *Integer, *Float:
			parsedBase, err := strconv.Atoi(args[1].String())
			if err != nil {
				return nil, fmt.Errorf("bad argument #2 to 'tonumber' (number has no integer representation)")
			}
			base = parsedBase
		default:
			return nil, fmt.Errorf("bad argument #2 to 'tonumber' (number expected, got %v)", baseVal.Type())
		}
	}
	return []Value{toNumber(args[0], base)}, nil
}

func stdType(vm *VM, args []Value) ([]Value, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("bad argument #1 to 'type' (value expected)")
	}
	return []Value{&String{val: args[0].Type()}}, nil
}

func stdNext(vm *VM, args []Value) ([]Value, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("bad argument #1 to 'next' (table expected)")
	}
	table, isTable := args[0].(*Table)
	if !isTable {
		return nil, fmt.Errorf("bad argument #1 to 'next' (table expected but found %v)", args[0].Type())
	}
	keys := table.Keys()
	if len(keys) == 0 {
		return []Value{&Nil{}}, nil
	}
	var toFind Value = &Nil{}
	if len(args) > 1 {
		toFind = args[1]
	}
	if _, isNil := toFind.(*Nil); isNil {
		key := ToValue(keys[0])
		val, _ := vm.index(table, nil, key)
		return []Value{key, val}, nil
	}
	for i, key := range keys {
		if key == toKey(toFind) {
			if i < len(keys)-1 {
				tkey := ToValue(keys[i+1])
				val, _ := vm.index(table, nil, tkey)
				return []Value{tkey, val}, nil
			} else {
				break
			}
		}
	}
	return []Value{&Nil{}}, nil
}

func stdPairs(vm *VM, args []Value) ([]Value, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("bad argument #1 to 'pairs' (table expected)")
	}
	table, isTable := args[0].(*Table)
	if !isTable {
		return nil, fmt.Errorf("bad argument #1 to 'pairs' (table expected but found %v)", args[0].Type())
	}
	return []Value{&ExternFunc{stdNext}, table, &Nil{}}, nil
}

func stdIPairsIterator(vm *VM, args []Value) ([]Value, error) {
	table := args[0].(*Table)
	i := &Integer{val: args[1].(*Integer).val + 1}
	val, err := vm.index(table, nil, i)
	if err != nil {
		return nil, err
	} else if _, isNil := val.(*Nil); isNil {
		return []Value{&Nil{}}, nil
	}
	return []Value{i, val}, nil
}

func stdIPairs(vm *VM, args []Value) ([]Value, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("bad argument #1 to 'pairs' (table expected)")
	}
	table, isTable := args[0].(*Table)
	if !isTable {
		return nil, fmt.Errorf("bad argument #1 to 'pairs' (table expected but found %v)", args[0].Type())
	}
	return []Value{&ExternFunc{stdIPairsIterator}, table, &Integer{val: 0}}, nil
}

func stdSetMetatable(vm *VM, args []Value) ([]Value, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("bad argument #1 to 'setmetatable' (table expected)")
	}
	table, isTable := args[0].(*Table)
	if !isTable {
		return nil, fmt.Errorf("bad argument #1 to 'setmetatable' (table expected but found %v)", args[0].Type())
	}
	if len(args) > 1 {
		metatable, isTable := args[1].(*Table)
		if !isTable {
			return nil, fmt.Errorf("bad argument #2 to 'setmetatable' (table expected but found %v)", args[1].Type())
		}
		table.metatable = metatable
	} else {
		table.metatable = nil
	}
	return []Value{table}, nil
}

func stdGetMetatable(vm *VM, args []Value) ([]Value, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("bad argument #1 to 'getmetatable' (value expected)")
	}
	metatable := args[0].Meta()
	if metatable == nil {
		return []Value{&Nil{}}, nil
	}
	return []Value{metatable}, nil
}

func stdDoFile(vm *VM, args []Value) ([]Value, error) {
	var file io.Reader
	filename := "stdin"
	if len(args) < 1 {
		file = os.Stdin
	} else if str, isString := args[0].(*String); !isString {
		return nil, fmt.Errorf("bad argument #1 to 'dofile' (string expected but found %v)", args[0].Type())
	} else if osfile, err := os.Open(str.val); err != nil {
		return nil, fmt.Errorf("bad argument #1 to 'dofile' could not load file %v", str.val)
	} else {
		filename = str.val
		file = osfile
	}
	fn, err := Parse(filename, file)
	if err != nil {
		return nil, err
	}
	return vm.Eval(fn)
}

func stdPCall(vm *VM, args []Value) ([]Value, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("bad argument #1 to 'pcall' (function expected)")
	}
	fn, isCallable := args[0].(callable)
	if !isCallable {
		return nil, fmt.Errorf("bad argument #1 to 'pcall' (function expected but %v found)", args[0].Type())
	}
	values, err := vm.Call(fn, args[1:])
	if err != nil {
		return []Value{&Boolean{false}, &Error{val: &String{val: err.Error()}}}, nil
	}
	return append([]Value{&Boolean{true}}, values...), nil
}

func stdXPCall(vm *VM, args []Value) ([]Value, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("bad argument #1 to 'xpcall' (function expected)")
	}
	fn, isCallable := args[0].(callable)
	if !isCallable {
		return nil, fmt.Errorf("bad argument #1 to 'xpcall' (function expected but %v found)", args[0].Type())
	}
	if len(args) < 2 {
		return nil, fmt.Errorf("bad argument #2 to 'xpcall' (function expected)")
	}
	msgh, isString := args[1].(callable)
	if !isString {
		return nil, fmt.Errorf("bad argument #1 to 'xpcall' (function expected but %v found)", args[0].Type())
	}
	values, err := vm.Call(fn, args[2:])
	if err != nil {
		if _, err := vm.Call(msgh, []Value{&Error{&String{err.Error()}}}); err != nil {
			return nil, err
		}
		return []Value{&Boolean{false}, &Error{val: &String{val: err.Error()}}}, nil
	}
	return append([]Value{&Boolean{true}}, values...), nil
}

func stdRawGet(vm *VM, args []Value) ([]Value, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("bad argument #1 to 'rawget' (table expected)")
	}
	table, isTable := args[0].(*Table)
	if !isTable {
		return nil, fmt.Errorf("bad argument #1 to 'rawget' (table expected but found %v)", args[0].Type())
	}
	if len(args) < 2 {
		return nil, fmt.Errorf("bad argument #2 to 'rawget' (value expected)")
	}
	res, err := table.Index(args[1])
	if err != nil {
		return nil, err
	}
	return []Value{res}, nil
}

func stdRawSet(vm *VM, args []Value) ([]Value, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("bad argument #1 to 'rawset' (table expected)")
	}
	table, isTable := args[0].(*Table)
	if !isTable {
		return nil, fmt.Errorf("bad argument #1 to 'rawset' (table expected but found %v)", args[0].Type())
	}
	if len(args) < 2 {
		return nil, fmt.Errorf("bad argument #2 to 'rawset' (value expected)")
	}
	if len(args) < 3 {
		return nil, fmt.Errorf("bad argument #3 to 'rawset' (value expected)")
	}
	return []Value{}, table.SetIndex(args[1], args[2])
}

func stdRawEq(vm *VM, args []Value) ([]Value, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("bad argument #1 to 'rawequal' (value expected)")
	}
	if len(args) < 2 {
		return nil, fmt.Errorf("bad argument #2 to 'rawequal' (value expected)")
	}
	lVal, rVal := args[0], args[1]

	typeA, typeB := lVal.Type(), rVal.Type()
	if typeA != typeB {
		return []Value{&Boolean{val: false}}, nil
	}

	switch typeA {
	case "string":
		strA, strB := lVal.(*String), rVal.(*String)
		return []Value{&Boolean{val: strA.val == strB.val}}, nil
	case "number":
		vA, vB := toFloat(lVal), toFloat(rVal)
		return []Value{&Boolean{val: vA == vB}}, nil
	case "boolean":
		strA, strB := lVal.(*Boolean), rVal.(*Boolean)
		return []Value{&Boolean{val: strA.val == strB.val}}, nil
	case "nil":
		return []Value{&Boolean{val: true}}, nil
	case "table", "function", "closure":
		return []Value{&Boolean{val: lVal == rVal}}, nil
	default:
		return []Value{&Boolean{val: false}}, nil
	}
}

func stdRawLen(vm *VM, args []Value) ([]Value, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("bad argument #1 to 'rawlen' (value expected)")
	}
	argType := args[0].Type()
	if argType != "string" && argType != "table" {
		return nil, fmt.Errorf("bad argument #1 to 'rawlen' (string or table expected but found %v)", argType)
	}

	switch argType {
	case "string":
		str := args[0].(*String)
		return []Value{&Integer{val: int64(len(str.val))}}, nil
	case "table":
		tbl := args[0].(*Table)
		return []Value{&Integer{val: int64(len(tbl.val))}}, nil
	}
	return nil, nil
}

func stdSelect(vm *VM, args []Value) ([]Value, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("bad argument #1 to 'select' (number expected)")
	} else if !isNumber(args[0]) {
		return nil, fmt.Errorf("bad argument #1 to 'select' (number expected)")
	}
	out := []Value{}
	rest := args[1:]
	if sel := toInt(args[0]); sel > 0 {
		out = rest[sel-1:]
	} else if sel < 0 {
		idx := len(rest) + int(sel)
		if idx < 0 {
			return nil, fmt.Errorf("bad argument #1 to 'select' (index out of range)")
		}
		out = rest[idx:]
	}
	return out, nil
}

func stdLoad(vm *VM, args []Value) ([]Value, error) {
	return nil, nil
}

func stdLoadFile(vm *VM, args []Value) ([]Value, error) {
	return nil, nil
}
