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
		hashtable: map[any]any{
			"_VERSION":       LUAVERSION,
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

func stdCollectgarbage(vm *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "collectgarbage", "~string"); err != nil {
		return nil, err
	}
	mode := "collect"
	if len(args) > 0 {
		mode = args[0].(string)
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
		return []any{int64(m.TotalAlloc / 1024)}, nil
	case "isrunning":
		return []any{!vm.gcOff}, nil
	case "incremental", "generational":
	}
	return []any{}, nil
}

func stdprintaux(vm *VM, args []any, out io.Writer, split string) ([]any, error) {
	strParts := make([]string, len(args))
	for i, arg := range args {
		str, err := vm.toString(arg)
		if err != nil {
			return nil, err
		}
		strParts[i] = str
	}
	fmt.Fprintln(out, strings.Join(strParts, split))
	return nil, nil
}

func stdPrint(vm *VM, args []any) ([]any, error) {
	return stdprintaux(vm, args, os.Stdout, "\t")
}

func stdAssert(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "assert", "value", "~value"); err != nil {
		return nil, err
	} else if toBool(args[0]) {
		return args, nil
	} else if len(args) > 1 {
		return nil, &UserError{val: args[1], level: 1}
	}
	return nil, errors.New("assertion failed")
}

func stdToString(vm *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "tostring", "value"); err != nil {
		return nil, err
	}
	str, err := vm.toString(args[0])
	if err != nil {
		return nil, err
	}
	return []any{str}, nil
}

func stdToNumber(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "tonumber", "value", "~number"); err != nil {
		return nil, err
	}
	base := 10
	if len(args) > 1 {
		switch baseVal := args[1].(type) {
		case int64, float64:
			parsedBase, err := strconv.Atoi(ToString(args[1]))
			if err != nil {
				return nil, argumentErr(2, "tonumber", errors.New("number has no integer representation"))
			}
			base = parsedBase
		default:
			return nil, argumentErr(2, "tonumber", fmt.Errorf("number expected, got %v", TypeName(baseVal)))
		}
	}
	return []any{toNumber(args[0], base)}, nil
}

func stdType(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "type", "value"); err != nil {
		return nil, err
	}
	return []any{TypeName(args[0])}, nil
}

func stdNext(vm *VM, args []any) ([]any, error) {
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
		return []any{nil}, nil
	}
	var toFind any
	if len(args) > 1 {
		toFind = args[1]
	}
	if toFind == nil {
		key := allKeys[0]
		val, _ := vm.index(table, nil, key)
		return []any{key, val}, nil
	}
	for i, key := range allKeys {
		if key == toKey(toFind) {
			if i < len(allKeys)-1 {
				tkey := allKeys[i+1]
				val, _ := vm.index(table, nil, tkey)
				return []any{tkey, val}, nil
			}
			break
		}
	}
	return []any{nil}, nil
}

func stdPairs(vm *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "pairs", "table"); err != nil {
		return nil, err
	}
	if method := findMetavalue(metaPairs, args[0]); method != nil {
		res, err := vm.call(method, []any{args[0]})
		if err != nil {
			return nil, err
		} else if len(res) < 3 {
			return nil, errors.New("not enough return values from __pairs metamethod")
		}
		return res, nil
	}
	return []any{Fn("pairs.next", stdNext), args[0], nil}, nil
}

func stdIPairsIterator(vm *VM, args []any) ([]any, error) {
	table := args[0].(*Table)
	i := args[1].(int64) + 1
	val, err := vm.index(table, nil, i)
	if err != nil {
		return nil, err
	} else if val == nil {
		return []any{nil}, nil
	}
	return []any{i, val}, nil
}

func stdIPairs(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "ipairs", "table"); err != nil {
		return nil, err
	}
	return []any{Fn("ipairs.next", stdIPairsIterator), args[0], int64(0)}, nil
}

func stdSetMetatable(_ *VM, args []any) ([]any, error) {
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
	return []any{table}, nil
}

func stdGetMetatable(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "getmetatable", "value"); err != nil {
		return nil, err
	}
	if method := findMetavalue(metaMeta, args[0]); method != nil {
		return []any{method}, nil
	}
	metatable := getMetatable(args[0])
	if metatable == nil {
		return []any{nil}, nil
	}
	return []any{metatable}, nil
}

func stdDoFile(vm *VM, args []any) ([]any, error) {
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

	str := args[0].(string)
	if _, err := os.Open(str); err != nil {
		return nil, argumentErr(1, "dofile", fmt.Errorf("could not load file %v", str))
	}

	fn, err := ParseFile(str, ModeText)
	if err != nil {
		return nil, err
	}
	return vm.Eval(fn)
}

func stdError(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "error", "~value", "~number"); err != nil {
		return nil, err
	}
	var errObj any
	if len(args) > 0 {
		errObj = args[0]
	}
	level := 1
	if len(args) > 1 {
		level = int(toInt(args[1]))
	}
	return nil, &UserError{val: errObj, level: level}
}

func stdWarn(vm *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "warn", "value"); err != nil {
		return nil, err
	}
	if msg := ToString(args[0]); strings.HasPrefix(msg, "@") && len(args) == 1 {
		if msg == "@on" {
			WarnEnabled = true
		} else if msg == "@off" {
			WarnEnabled = false
		}
	} else if WarnEnabled {
		return stdprintaux(vm, append([]any{"Lua warning: "}, args...), os.Stderr, "")
	}
	return []any{}, nil
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

func stdPCall(vm *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "pcall", "function"); err != nil {
		return nil, err
	}
	values, err := vm.call(args[0], args[1:])
	var retValues []any
	if err != nil {
		retValues = []any{false, &UserError{val: err.Error(), level: 1}}
	} else {
		retValues = append([]any{true}, values...)
	}
	return retValues, nil
}

func stdXPCall(vm *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "xpcall", "function", "function"); err != nil {
		return nil, err
	}
	values, err := vm.call(args[0], args[2:])
	if err != nil {
		newErr := &UserError{val: err.Error(), level: 1}
		if _, err := vm.call(args[1], []any{newErr}); err != nil {
			return nil, err
		}
		return []any{false, newErr}, nil
	}
	return append([]any{true}, values...), nil
}

func stdRawGet(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "rawget", "table", "value"); err != nil {
		return nil, err
	}
	res, err := args[0].(*Table).Index(args[1])
	if err != nil {
		return nil, err
	}
	return []any{res}, nil
}

func stdRawSet(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "rawset", "table", "value", "value"); err != nil {
		return nil, err
	}
	return []any{}, args[0].(*Table).SetIndex(args[1], args[2])
}

func stdRawEq(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "rawequal", "value", "value"); err != nil {
		return nil, err
	}
	lVal, rVal := args[0], args[1]

	typeA, typeB := TypeName(lVal), TypeName(rVal)
	if typeA != typeB {
		return []any{false}, nil
	}

	switch typeA {
	case "string":
		return []any{lVal.(string) == rVal.(string)}, nil
	case "number":
		vA, vB := toFloat(lVal), toFloat(rVal)
		return []any{vA == vB}, nil
	case "boolean":
		return []any{lVal.(bool) == rVal.(bool)}, nil
	case "nil":
		return []any{true}, nil
	case "table", "function", "closure":
		return []any{lVal == rVal}, nil
	default:
		return []any{false}, nil
	}
}

func stdRawLen(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "rawlen", "string|table"); err != nil {
		return nil, err
	}
	switch TypeName(args[0]) {
	case "string":
		str := args[0].(string)
		return []any{int64(len(str))}, nil
	case "table":
		tbl := args[0].(*Table)
		return []any{int64(len(tbl.val))}, nil
	}
	return nil, nil
}

func stdSelect(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "select", "number|string"); err != nil {
		return nil, err
	}
	if isString(args[0]) {
		strArg := args[0].(string)
		if strArg != "#" {
			return nil, argumentErr(1, "select", errors.New("(number expected, got string)"))
		}
		return []any{int64(len(args) - 1)}, nil
	}

	out := []any{}
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
func stdLoad(vm *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "load", "string|function", "~string", "~string", "~table"); err != nil {
		return nil, err
	}
	var src string
	chunkname := "chunk"
	if TypeName(args[0]) == string(typeString) {
		src = args[0].(string)
	} else if TypeName(args[0]) == string(typeFunc) {
		for {
			res, err := vm.call(args[0], []any{})
			if err != nil {
				return nil, err
			} else if len(res) == 0 || res[0] == nil {
				break
			}
			retVal := res[0]
			str, isString := retVal.(string)
			if retVal == nil || (isString && str == "") {
				break
			}
			src += str
		}
	}
	mode := ModeText & ModeBinary
	if len(args) > 1 {
		chunkname = args[1].(string)
	}
	if len(args) > 2 {
		modeStr := args[2].(string)
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
	var retVals []any
	if err != nil {
		retVals = []any{nil, err.Error()}
	} else {
		retVals = []any{&Closure{
			val:      fn,
			upvalues: []*upvalueBroker{{name: "_ENV", val: env}},
		}}
	}
	return retVals, nil
}

// loadfile ([filename [, mode [, env]]]).
func stdLoadFile(vm *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "load", "~string", "~string", "~table"); err != nil {
		return nil, err
	}
	mode := ModeText & ModeBinary
	if len(args) == 0 {
		fn, err := Parse("chunk", os.Stdin, mode)
		if err != nil {
			return nil, err
		}
		return []any{&Closure{
			val:      fn,
			upvalues: []*upvalueBroker{{name: "_ENV", val: vm.env}},
		}}, nil
	}
	filename := args[0].(string)
	if len(args) > 1 {
		modeStr := args[1].(string)
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

	return []any{&Closure{
		val:      fn,
		upvalues: []*upvalueBroker{{name: "_ENV", val: env}},
	}}, nil
}

func assertArguments(args []any, methodName string, assertions ...string) error {
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
		valType := TypeName(args[i])
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
