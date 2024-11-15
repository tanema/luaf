package luaf

import (
	"fmt"
	"strconv"
	"strings"
)

var stdlib = map[any]Value{
	"_VERSION":     &String{"Luaf 0.0.1"},
	"print":        &ExternFunc{stdPrint},
	"assert":       &ExternFunc{stdAssert},
	"type":         &ExternFunc{stdType},
	"tostring":     &ExternFunc{stdToString},
	"tonumber":     &ExternFunc{stdToNumber},
	"next":         &ExternFunc{stdNext},
	"pairs":        &ExternFunc{stdPairs},
	"ipairs":       &ExternFunc{stdIPairs},
	"setmetatable": &ExternFunc{stdSetMetatable},
	"getmetatable": &ExternFunc{stdGetMetatable},
}

func stdPrint(args []Value) ([]Value, error) {
	strParts := make([]string, len(args))
	for i, arg := range args {
		strParts[i] = arg.String()
	}
	fmt.Println(strings.Join(strParts, "\t"))
	return nil, nil
}

func stdAssert(args []Value) ([]Value, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("bad argument #1 to 'assert' (value expected)")
	}
	if args[0].Bool().val {
		return []Value{args[0]}, nil
	}
	if len(args) > 1 {
		return nil, &Error{val: args[1]}
	}
	return nil, fmt.Errorf("assertion failed")
}

func stdToString(args []Value) ([]Value, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("bad argument #1 to 'tostring' (value expected)")
	}
	return []Value{&String{val: args[0].String()}}, nil
}

func stdToNumber(args []Value) ([]Value, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("bad argument #1 to 'tonumber' (value expected)")
	}
	val := args[0].String()
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
	intVal, err := strconv.ParseInt(val, base, 64)
	return []Value{&Integer{val: intVal}}, err
}

func stdType(args []Value) ([]Value, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("bad argument #1 to 'type' (value expected)")
	}
	return []Value{&String{val: args[0].Type()}}, nil
}

func stdNext(args []Value) ([]Value, error) {
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
		val, _ := table.Index(key)
		return []Value{key, val}, nil
	}
	for i, key := range keys {
		if key == toFind.ToKey() {
			if i < len(keys)-1 {
				tkey := ToValue(keys[i+1])
				val, _ := table.Index(tkey)
				return []Value{tkey, val}, nil
			} else {
				break
			}
		}
	}
	return []Value{&Nil{}}, nil
}

func stdPairs(args []Value) ([]Value, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("bad argument #1 to 'pairs' (table expected)")
	}
	table, isTable := args[0].(*Table)
	if !isTable {
		return nil, fmt.Errorf("bad argument #1 to 'pairs' (table expected but found %v)", args[0].Type())
	}
	return []Value{&ExternFunc{stdNext}, table, &Nil{}}, nil
}

func stdIPairsIterator(args []Value) ([]Value, error) {
	table := args[0].(*Table)
	i := &Integer{val: args[1].(*Integer).val + 1}
	val, err := table.Index(i)
	if err != nil {
		return nil, err
	} else if _, isNil := val.(*Nil); isNil {
		return []Value{&Nil{}}, nil
	}
	return []Value{i, val}, nil
}

func stdIPairs(args []Value) ([]Value, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("bad argument #1 to 'pairs' (table expected)")
	}
	table, isTable := args[0].(*Table)
	if !isTable {
		return nil, fmt.Errorf("bad argument #1 to 'pairs' (table expected but found %v)", args[0].Type())
	}
	return []Value{&ExternFunc{stdIPairsIterator}, table, &Integer{val: 0}}, nil
}

func stdSetMetatable(args []Value) ([]Value, error) {
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
	return nil, nil

}

func stdGetMetatable(args []Value) ([]Value, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("bad argument #1 to 'getmetatable' (table expected)")
	}
	table, isTable := args[0].(*Table)
	if !isTable {
		return nil, fmt.Errorf("bad argument #1 to 'setmetatable' (table expected but found %v)", args[0].Type())
	}
	return []Value{table.metatable}, nil
}
