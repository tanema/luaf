package luaf

import (
	"fmt"
	"strconv"
	"strings"
)

var stdlib = map[any]Value{
	"_VERSION": &String{"Luaf 0.0.1"},
	"print":    &ExternFunc{stdPrint},
	"assert":   &ExternFunc{stdAssert},
	"type":     &ExternFunc{stdType},
	"tostring": &ExternFunc{stdToString},
	"tonumber": &ExternFunc{stdToNumber},
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
	return nil, fmt.Errorf("assertion failed!")
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
