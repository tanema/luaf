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
	if err := assertArguments(args, "assert", "value", "~value"); err != nil {
		return nil, err
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
	if err := assertArguments(args, "tostring", "value"); err != nil {
		return nil, err
	}
	return []Value{&String{val: args[0].String()}}, nil
}

func stdToNumber(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "tonumber", "value", "~number"); err != nil {
		return nil, err
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
	if err := assertArguments(args, "type", "value"); err != nil {
		return nil, err
	}
	return []Value{&String{val: args[0].Type()}}, nil
}

func stdNext(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "next", "table", "~value"); err != nil {
		return nil, err
	}

	table := args[0].(*Table)
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
	if err := assertArguments(args, "pairs", "table"); err != nil {
		return nil, err
	}
	return []Value{&ExternFunc{stdNext}, args[0].(*Table), &Nil{}}, nil
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
	if err := assertArguments(args, "ipairs", "table"); err != nil {
		return nil, err
	}
	return []Value{&ExternFunc{stdIPairsIterator}, args[0], &Integer{val: 0}}, nil
}

func stdSetMetatable(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "setmetatable", "table", "~table"); err != nil {
		return nil, err
	}
	table := args[0].(*Table)
	if len(args) > 1 {
		table.metatable = args[1].(*Table)
	} else {
		table.metatable = nil
	}
	return []Value{table}, nil
}

func stdGetMetatable(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "getmetatable", "value"); err != nil {
		return nil, err
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
	if err := assertArguments(args, "pcall", "function"); err != nil {
		return nil, err
	}
	values, err := vm.Call(args[0].(callable), args[1:])
	if err != nil {
		return []Value{&Boolean{false}, &Error{val: &String{val: err.Error()}}}, nil
	}
	return append([]Value{&Boolean{true}}, values...), nil
}

func stdXPCall(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "xpcall", "function", "function"); err != nil {
		return nil, err
	}
	values, err := vm.Call(args[0].(callable), args[2:])
	if err != nil {
		if _, err := vm.Call(args[1].(callable), []Value{&Error{&String{err.Error()}}}); err != nil {
			return nil, err
		}
		return []Value{&Boolean{false}, &Error{val: &String{val: err.Error()}}}, nil
	}
	return append([]Value{&Boolean{true}}, values...), nil
}

func stdRawGet(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "rawget", "table", "value"); err != nil {
		return nil, err
	}
	res, err := args[0].(*Table).Index(args[1])
	if err != nil {
		return nil, err
	}
	return []Value{res}, nil
}

func stdRawSet(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "rawset", "table", "value", "value"); err != nil {
		return nil, err
	}
	return []Value{}, args[0].(*Table).SetIndex(args[1], args[2])
}

func stdRawEq(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "rawequal", "value", "value"); err != nil {
		return nil, err
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
	if err := assertArguments(args, "rawlen", "string|table"); err != nil {
		return nil, err
	}
	switch args[0].Type() {
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
	if err := assertArguments(args, "select", "number"); err != nil {
		return nil, err
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

func assertArguments(args []Value, methodName string, assertions ...string) error {
	for i, assertion := range assertions {
		optional := strings.HasPrefix(assertion, "~")
		expectedTypes := strings.Split(strings.TrimPrefix(assertion, "~"), "|")

		if i >= len(args) && !optional {
			return fmt.Errorf("bad argument #%v to '%v' (%v expected)", i+1, methodName, assertion)
		} else if i >= len(args) && !strings.HasPrefix(assertion, "~") {
			return nil
		} else if strings.TrimPrefix(assertion, "~") == "value" {
			continue
		}

		typeFound := false
		valType := args[i].Type()
		for _, expected := range expectedTypes {
			if expected == valType {
				typeFound = true
				break
			}
		}
		if !typeFound {
			return fmt.Errorf("bad argument #%v to '%v' (%v expected but received %v)", i+1, methodName, strings.Join(expectedTypes, ", "), valType)
		}
	}
	return nil
}
