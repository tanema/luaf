package luaf

import (
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
)

var WarnEnabled = false

func createDefaultEnv(withLibs bool) *Table {
	env := &Table{
		hashtable: map[any]Value{
			"_VERSION":       &String{LUAVERSION},
			"assert":         Fn("assert", stdAssert),
			"collectgarbage": Fn("collectgarbage", stdCollectgarbage),
			"dofile":         Fn("dofile", stdDoFile),
			"error":          Fn("error", stdError),
			"getmetatable":   Fn("getmetatable", stdGetMetatable),
			"ipairs":         Fn("ipairs", stdIPairs),
			"load":           Fn("load", stdLoad),
			"loadfile":       Fn("loadfile", stdLoadFile),
			"next":           Fn("next", stdNext),
			"pairs":          Fn("pairs", stdPairs),
			"pcall":          Fn("pcall", stdPCall),
			"print":          Fn("print", stdPrint),
			"rawequal":       Fn("rawequal", stdRawEq),
			"rawget":         Fn("rawget", stdRawGet),
			"rawlen":         Fn("rawlen", stdRawLen),
			"rawset":         Fn("rawset", stdRawSet),
			"require":        Fn("require", stdRequire),
			"select":         Fn("select", stdSelect),
			"setmetatable":   Fn("setmetatable", stdSetMetatable),
			"tonumber":       Fn("tonumber", stdToNumber),
			"tostring":       Fn("tostring", stdToString),
			"type":           Fn("type", stdType),
			"warn":           Fn("warn", stdWarn),
			"xpcall":         Fn("xpcall", stdXPCall),
		},
	}
	if withLibs {
		env.hashtable["coroutine"] = createCoroutineLib()
		env.hashtable["debug"] = createDebugLib()
		env.hashtable["io"] = createIOLib()
		env.hashtable["math"] = createMathLib()
		env.hashtable["os"] = createOSLib()
		env.hashtable["string"] = createStringLib()
		env.hashtable["table"] = createTableLib()
		env.hashtable["utf8"] = createUtf8Lib()
		env.hashtable["package"] = libPackage
	}
	return env
}

func stdCollectgarbage(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "collectgarbage", "~string"); err != nil {
		return nil, err
	}
	mode := "collect"
	if len(args) > 0 {
		mode = args[0].(*String).val
	}
	switch mode {
	case "collect", "step":
		runtime.GC()
		vm.collectGarbage(true)
	case "stop":
		vm.gcOff = true
	case "restart":
		vm.gcOff = false
	case "count":
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		return []Value{&Integer{val: int64(m.TotalAlloc / 1024)}}, nil
	case "isrunning":
		return []Value{&Boolean{val: !vm.gcOff}}, nil
	case "incremental", "generational":
	}
	return []Value{}, nil
}

func stdprintaux(vm *VM, args []Value, out io.Writer, split string) ([]Value, error) {
	strParts := make([]string, len(args))
	for i, arg := range args {
		str, err := toString(vm, arg)
		if err != nil {
			return nil, err
		}
		strParts[i] = str.val
	}
	fmt.Fprintln(out, strings.Join(strParts, split))
	return nil, nil
}

func stdPrint(vm *VM, args []Value) ([]Value, error) {
	return stdprintaux(vm, args, os.Stdout, "\t")
}

func stdAssert(_ *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "assert", "value", "~value"); err != nil {
		return nil, err
	} else if toBool(args[0]).val {
		return args, nil
	} else if len(args) > 1 {
		return nil, &UserError{val: args[1], level: 1}
	}
	return nil, errors.New("assertion failed")
}

func stdToString(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "tostring", "value"); err != nil {
		return nil, err
	}
	str, err := toString(vm, args[0])
	if err != nil {
		return nil, err
	}
	return []Value{str}, nil
}

func stdToNumber(_ *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "tonumber", "value", "~number"); err != nil {
		return nil, err
	}
	base := 10
	if len(args) > 1 {
		switch baseVal := args[1].(type) {
		case *Integer, *Float:
			parsedBase, err := strconv.Atoi(args[1].String())
			if err != nil {
				return nil, argumentErr(2, "tonumber", errors.New("number has no integer representation"))
			}
			base = parsedBase
		default:
			return nil, argumentErr(2, "tonumber", fmt.Errorf("number expected, got %v", baseVal.Type()))
		}
	}
	return []Value{toNumber(args[0], base)}, nil
}

func stdType(_ *VM, args []Value) ([]Value, error) {
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
	allKeys := make([]any, len(table.val)+len(keys))
	for i := 1; i <= len(table.val); i++ {
		allKeys[i-1] = int64(i)
	}
	copy(allKeys[len(table.val):], keys)

	if len(allKeys) == 0 {
		return []Value{&Nil{}}, nil
	}
	var toFind Value = &Nil{}
	if len(args) > 1 {
		toFind = args[1]
	}
	if _, isNil := toFind.(*Nil); isNil {
		key := ToValue(allKeys[0])
		val, _ := vm.index(table, nil, key)
		return []Value{key, val}, nil
	}
	for i, key := range allKeys {
		if key == toKey(toFind) {
			if i < len(allKeys)-1 {
				tkey := ToValue(allKeys[i+1])
				val, _ := vm.index(table, nil, tkey)
				return []Value{tkey, val}, nil
			}
			break
		}
	}
	return []Value{&Nil{}}, nil
}

func stdPairs(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "pairs", "table"); err != nil {
		return nil, err
	}
	if method := findMetavalue(metaPairs, args[0]); method != nil {
		res, err := vm.call(method, []Value{args[0]})
		if err != nil {
			return nil, err
		} else if len(res) < 3 {
			return nil, errors.New("not enough return values from __pairs metamethod")
		}
		return res, nil
	}
	return []Value{Fn("pairs.next", stdNext), args[0], &Nil{}}, nil
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

func stdIPairs(_ *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "ipairs", "table"); err != nil {
		return nil, err
	}
	return []Value{Fn("ipairs.next", stdIPairsIterator), args[0], &Integer{val: 0}}, nil
}

func stdSetMetatable(_ *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "setmetatable", "table", "~table"); err != nil {
		return nil, err
	}
	if method := findMetavalue(metaMeta, args[0]); method != nil {
		return nil, errors.New("cannot set a metatable on a table with the __metatable metamethod defined")
	}
	table := args[0].(*Table)
	if len(args) > 1 {
		table.metatable = args[1].(*Table)
	} else {
		table.metatable = nil
	}
	return []Value{table}, nil
}

func stdGetMetatable(_ *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "getmetatable", "value"); err != nil {
		return nil, err
	}
	if method := findMetavalue(metaMeta, args[0]); method != nil {
		return []Value{method}, nil
	}
	metatable := args[0].Meta()
	if metatable == nil {
		return []Value{&Nil{}}, nil
	}
	return []Value{metatable}, nil
}

func stdDoFile(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "dofile", "~string"); err != nil {
		return nil, err
	}

	if len(args) < 1 {
		fn, err := Parse("stdin", os.Stdin, ModeText)
		if err != nil {
			return nil, err
		}
		return vm.Eval(fn)
	}

	str := args[0].(*String)
	if _, err := os.Open(str.val); err != nil {
		return nil, argumentErr(1, "dofile", fmt.Errorf("could not load file %v", str.val))
	}

	fn, err := ParseFile(str.val, ModeText)
	if err != nil {
		return nil, err
	}
	return vm.Eval(fn)
}

func stdError(_ *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "error", "~value", "~number"); err != nil {
		return nil, err
	}
	var errObj Value = &Nil{}
	if len(args) > 0 {
		errObj = args[0]
	}
	level := 1
	if len(args) > 1 {
		level = int(toInt(args[1]))
	}
	return nil, &UserError{val: errObj, level: level}
}

func stdWarn(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "warn", "value"); err != nil {
		return nil, err
	}
	if msg := args[0].String(); strings.HasPrefix(msg, "@") && len(args) == 1 {
		if msg == "@on" {
			WarnEnabled = true
		} else if msg == "@off" {
			WarnEnabled = false
		}
	} else if WarnEnabled {
		return stdprintaux(vm, append([]Value{&String{val: "Lua warning: "}}, args...), os.Stderr, "")
	}
	return []Value{}, nil
}

func Warn(args ...string) {
	if len(args) == 1 && strings.HasPrefix(args[0], "@") {
		if args[0] == "@on" {
			WarnEnabled = true
		} else if args[0] == "@off" {
			WarnEnabled = false
		}
		return
	}
	if !WarnEnabled {
		return
	}
	fmt.Fprintln(os.Stderr, strings.Join(args, ""))
}

func stdPCall(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "pcall", "function"); err != nil {
		return nil, err
	}
	values, err := vm.call(args[0], args[1:])
	var retValues []Value
	if err != nil {
		retValues = []Value{&Boolean{false}, &UserError{val: &String{val: err.Error()}, level: 1}}
	} else {
		retValues = append([]Value{&Boolean{true}}, values...)
	}
	return retValues, nil
}

func stdXPCall(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "xpcall", "function", "function"); err != nil {
		return nil, err
	}
	values, err := vm.call(args[0], args[2:])
	if err != nil {
		newErr := &UserError{val: &String{val: err.Error()}, level: 1}
		if _, err := vm.call(args[1], []Value{newErr}); err != nil {
			return nil, err
		}
		return []Value{&Boolean{false}, newErr}, nil
	}
	return append([]Value{&Boolean{true}}, values...), nil
}

func stdRawGet(_ *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "rawget", "table", "value"); err != nil {
		return nil, err
	}
	res, err := args[0].(*Table).Index(args[1])
	if err != nil {
		return nil, err
	}
	return []Value{res}, nil
}

func stdRawSet(_ *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "rawset", "table", "value", "value"); err != nil {
		return nil, err
	}
	return []Value{}, args[0].(*Table).SetIndex(args[1], args[2])
}

func stdRawEq(_ *VM, args []Value) ([]Value, error) {
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
		strA, _ := lVal.(*String)
		strB, _ := rVal.(*String)
		return []Value{&Boolean{val: strA.val == strB.val}}, nil
	case "number":
		vA, vB := toFloat(lVal), toFloat(rVal)
		return []Value{&Boolean{val: vA == vB}}, nil
	case "boolean":
		strA, _ := lVal.(*Boolean)
		strB, _ := rVal.(*Boolean)
		return []Value{&Boolean{val: strA.val == strB.val}}, nil
	case "nil":
		return []Value{&Boolean{val: true}}, nil
	case "table", "function", "closure":
		return []Value{&Boolean{val: lVal == rVal}}, nil
	default:
		return []Value{&Boolean{val: false}}, nil
	}
}

func stdRawLen(_ *VM, args []Value) ([]Value, error) {
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

func stdSelect(_ *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "select", "number|string"); err != nil {
		return nil, err
	}
	if isString(args[0]) {
		strArg := args[0].(*String).val
		if strArg != "#" {
			return nil, argumentErr(1, "select", errors.New("(number expected, got string)"))
		}
		return []Value{&Integer{val: int64(len(args) - 1)}}, nil
	}

	out := []Value{}
	rest := args[1:]
	if sel := toInt(args[0]); sel > 0 {
		out = rest[sel-1:]
	} else if sel < 0 {
		idx := len(rest) + int(sel)
		if idx < 0 {
			return nil, argumentErr(1, "select", errors.New("index out of range"))
		}
		out = rest[idx:]
	}
	return out, nil
}

// env => table for env.
func stdLoad(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "load", "string|function", "~string", "~string", "~table"); err != nil {
		return nil, err
	}
	var src string
	chunkname := "chunk"
	if args[0].Type() == string(typeString) {
		src = args[0].(*String).val
	} else if args[0].Type() == string(typeFunc) {
		for {
			res, err := vm.call(args[0], []Value{})
			if err != nil {
				return nil, err
			} else if len(res) == 0 || res[0] == nil {
				break
			}
			retVal := res[0]
			_, isNil := retVal.(*Nil)
			str, isString := retVal.(*String)
			if isNil || (isString && str.val == "") {
				break
			}
			src += str.val
		}
	}
	mode := ModeText & ModeBinary
	if len(args) > 1 {
		chunkname = args[1].(*String).val
	}
	if len(args) > 2 {
		modeStr := args[2].(*String).val
		if modeStr == "b" {
			mode = ModeBinary
		} else if modeStr == "t" {
			mode = ModeText
		}
	}
	var env *Table
	if len(args) > 3 {
		env = args[3].(*Table)
	} else {
		env = vm.env
	}

	fn, err := Parse(chunkname, strings.NewReader(src), mode)
	var retVals []Value
	if err != nil {
		retVals = []Value{&Nil{}, &String{val: err.Error()}}
	} else {
		retVals = []Value{&Closure{
			val:      fn,
			upvalues: []*upvalueBroker{{name: "_ENV", val: env}},
		}}
	}
	return retVals, nil
}

// loadfile ([filename [, mode [, env]]]).
func stdLoadFile(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "load", "~string", "~string", "~table"); err != nil {
		return nil, err
	}
	mode := ModeText & ModeBinary
	if len(args) == 0 {
		fn, err := Parse("chunk", os.Stdin, mode)
		if err != nil {
			return nil, err
		}
		return []Value{&Closure{
			val:      fn,
			upvalues: []*upvalueBroker{{name: "_ENV", val: vm.env}},
		}}, nil
	}
	filename := args[0].(*String).val
	if len(args) > 1 {
		modeStr := args[1].(*String).val
		if modeStr == "b" {
			mode = ModeBinary
		} else if modeStr == "t" {
			mode = ModeText
		}
	}

	var env *Table
	if len(args) > 2 {
		env = args[2].(*Table)
	} else {
		env = vm.env
	}

	fn, err := ParseFile(filename, mode)
	if err != nil {
		return nil, err
	}

	return []Value{&Closure{
		val:      fn,
		upvalues: []*upvalueBroker{{name: "_ENV", val: env}},
	}}, nil
}

func assertArguments(args []Value, methodName string, assertions ...string) error {
	for i, assertion := range assertions {
		optional := strings.HasPrefix(assertion, "~")
		expectedTypes := strings.Split(strings.TrimPrefix(assertion, "~"), "|")
		if i >= len(args) && !optional {
			return argumentErr(i+1, methodName, fmt.Errorf("%v expected", assertion))
		} else if i >= len(args) && optional {
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
			return argumentErr(
				i+1,
				methodName,
				fmt.Errorf(
					"%v expected but received %v",
					strings.Join(expectedTypes, ", "),
					valType,
				))
		}
	}
	return nil
}

func argumentErr(nArg int, methodName string, err error) error {
	return fmt.Errorf("bad argument #%v to '%v' (%w)", nArg, methodName, err)
}
