package luaf

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
)

var WarnEnabled = false

var envTable = &Table{
	hashtable: map[any]Value{
		"_VERSION":       &String{LUA_VERSION},
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
		"error":          &ExternFunc{stdError},
		"warn":           &ExternFunc{stdWarn},
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
		"require":        &ExternFunc{stdRequire},
		"package":        libPackage,
		"string":         libString,
		"table":          libTable,
		"math":           libMath,
		"utf8":           libUtf8,
		"os":             libOS,
		"io":             libIO,
		"coroutine":      libCoroutine,
		"debug":          libDebug,
	},
}

/*
TODO full functionality once GC is much more implemented though not all of this will apply because we do not need to do as much GC
collectgarbage ([opt [, arg]])
"collect": Performs a full garbage-collection cycle. This is the default option.
"stop": Stops automatic execution of the garbage collector. The collector will run only when explicitly invoked, until a call to restart it.
"restart": Restarts automatic execution of the garbage collector.
"count": Returns the total memory in use by Lua in Kbytes. The value has a fractional part, so that it multiplied by 1024 gives the exact number of bytes in use by Lua.
"step": Performs a garbage-collection step. The step "size" is controlled by arg. With a zero value, the collector will perform one basic (indivisible) step. For non-zero values, the collector will perform as if that amount of memory (in Kbytes) had been allocated by Lua. Returns true if the step finished a collection cycle.
"isrunning": Returns a boolean that tells whether the collector is running (i.e., not stopped).
"incremental": Change the collector mode to incremental. This option can be followed by three numbers: the garbage-collector pause, the step multiplier, and the step size. A zero means to not change that value.
"generational": Change the collector mode to generational. This option can be followed by two numbers: the garbage-collector minor multiplier and the major multiplier. A zero means to not change that value.
*/
func stdCollectgarbage(vm *VM, args []Value) ([]Value, error) {
	runtime.GC()
	return []Value{}, vm.collectGarbage(true)
}

func stdPrint(vm *VM, args []Value) ([]Value, error) {
	strParts := make([]string, len(args))
	for i, arg := range args {
		str, err := toString(vm, arg)
		if err != nil {
			return nil, err
		}
		strParts[i] = str.val
	}
	fmt.Println(strings.Join(strParts, "\t"))
	return nil, nil
}

func stdAssert(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(vm, args, "assert", "value", "~value"); err != nil {
		return nil, err
	}
	if toBool(args[0]).val {
		return []Value{args[0]}, nil
	}
	if len(args) > 1 {
		thisErr, err := toError(vm, args[1], 1)
		if err != nil {
			return nil, err
		}
		return nil, thisErr
	}
	return nil, vm.err("assertion failed")
}

func stdToString(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(vm, args, "tostring", "value"); err != nil {
		return nil, err
	}
	str, err := toString(vm, args[0])
	if err != nil {
		return nil, err
	}
	return []Value{str}, nil
}

func stdToNumber(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(vm, args, "tonumber", "value", "~number"); err != nil {
		return nil, err
	}
	base := 10
	if len(args) > 1 {
		switch baseVal := args[1].(type) {
		case *Integer, *Float:
			parsedBase, err := strconv.Atoi(args[1].String())
			if err != nil {
				return nil, argumentErr(vm, 2, "tonumber", fmt.Errorf("number has no integer representation"))
			}
			base = parsedBase
		default:
			return nil, argumentErr(vm, 2, "tonumber", fmt.Errorf("number expected, got %v", baseVal.Type()))
		}
	}
	return []Value{toNumber(args[0], base)}, nil
}

func stdType(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(vm, args, "type", "value"); err != nil {
		return nil, err
	}
	return []Value{&String{val: args[0].Type()}}, nil
}

func stdNext(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(vm, args, "next", "table", "~value"); err != nil {
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
	if err := assertArguments(vm, args, "pairs", "table"); err != nil {
		return nil, err
	}
	didDelegate, res, err := vm.delegateMetamethod(metaPairs, args[0])
	if err != nil {
		return nil, err
	} else if !didDelegate {
		return []Value{&ExternFunc{stdNext}, args[0], &Nil{}}, nil
	}
	if len(res) < 3 {
		return nil, vm.err("not enough return values from __pairs metamethod")
	}
	return args[:3], nil
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
	if err := assertArguments(vm, args, "ipairs", "table"); err != nil {
		return nil, err
	}
	return []Value{&ExternFunc{stdIPairsIterator}, args[0], &Integer{val: 0}}, nil
}

func stdSetMetatable(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(vm, args, "setmetatable", "table", "~table"); err != nil {
		return nil, err
	}
	didDelegate, res, err := vm.delegateMetamethod(metaMeta, args[0])
	if err != nil {
		return nil, err
	} else if didDelegate && len(res) > 0 {
		return nil, vm.err("cannot set a metatable on a table with the __metatable metamethod defined.")
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
	if err := assertArguments(vm, args, "getmetatable", "value"); err != nil {
		return nil, err
	}
	didDelegate, res, err := vm.delegateMetamethod(metaMeta, args[0])
	if err != nil {
		return nil, err
	} else if didDelegate && len(res) > 0 {
		return res[:1], nil
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
		return nil, argumentErr(vm, 1, "dofile", fmt.Errorf("string expected but found %v", args[0].Type()))
	} else if osfile, err := os.Open(str.val); err != nil {
		return nil, argumentErr(vm, 1, "dofile", fmt.Errorf("could not load file %v", str.val))
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

func stdError(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(vm, args, "error", "~value", "~number"); err != nil {
		return nil, err
	}
	var errObj Value = &Nil{}
	var err error
	if len(args) > 0 {
		errObj = args[0]
	}
	level := 1
	if len(args) > 1 {
		level = int(toInt(args[1]))
	}
	if level > len(vm.callStack) {
		level = 1
	}
	newError, err := toError(vm, errObj, level)
	if err != nil {
		return nil, err
	}
	return nil, newError
}

func stdWarn(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(vm, args, "error", "value"); err != nil {
		return nil, err
	}
	if msg := args[0].String(); strings.HasPrefix(msg, "@") {
		if msg == "@on" {
			WarnEnabled = true
		} else if msg == "@off" {
			WarnEnabled = false
		}
	} else if WarnEnabled {
		return stdPrint(vm, append([]Value{&String{val: "WARN:"}}, args...))
	}
	return []Value{}, nil
}

func stdPCall(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(vm, args, "pcall", "function"); err != nil {
		return nil, err
	}
	values, err := vm.Call("pcall", args[0].(callable), args[1:])
	if err != nil {
		newErr, err := toError(vm, &String{val: err.Error()}, 1)
		if err != nil {
			return nil, err
		}
		return []Value{&Boolean{false}, newErr}, nil
	}
	return append([]Value{&Boolean{true}}, values...), nil
}

func stdXPCall(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(vm, args, "xpcall", "function", "function"); err != nil {
		return nil, err
	}
	values, err := vm.Call("xpcall", args[0].(callable), args[2:])
	if err != nil {
		newErr, err := toError(vm, &String{val: err.Error()}, 1)
		if err != nil {
			return nil, err
		} else if _, err := vm.Call("xpcall", args[1].(callable), []Value{newErr}); err != nil {
			return nil, err
		}
		return []Value{&Boolean{false}, newErr}, nil
	}
	return append([]Value{&Boolean{true}}, values...), nil
}

func stdRawGet(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(vm, args, "rawget", "table", "value"); err != nil {
		return nil, err
	}
	res, err := args[0].(*Table).Index(args[1])
	if err != nil {
		return nil, err
	}
	return []Value{res}, nil
}

func stdRawSet(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(vm, args, "rawset", "table", "value", "value"); err != nil {
		return nil, err
	}
	return []Value{}, args[0].(*Table).SetIndex(args[1], args[2])
}

func stdRawEq(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(vm, args, "rawequal", "value", "value"); err != nil {
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
	if err := assertArguments(vm, args, "rawlen", "string|table"); err != nil {
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
	if err := assertArguments(vm, args, "select", "number"); err != nil {
		return nil, err
	}
	out := []Value{}
	rest := args[1:]
	if sel := toInt(args[0]); sel > 0 {
		out = rest[sel-1:]
	} else if sel < 0 {
		idx := len(rest) + int(sel)
		if idx < 0 {
			return nil, argumentErr(vm, 1, "select", fmt.Errorf("index out of range"))
		}
		out = rest[idx:]
	}
	return out, nil
}

// load (chunk [, chunkname [, mode [, env]]])
// chunk => string to be parsed or function to return parts to be concatted together
// chunkname => name for loaded func
// mode => b, t, bt
// env => table for env
func stdLoad(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(vm, args, "load", "string|function", "~string", "~string", "~table"); err != nil {
		return nil, err
	}
	var src string
	chunkname := "chunk"
	if args[0].Type() == string(typeString) {
		src = args[0].(*String).val
	} else if args[0].Type() == string(typeFunc) {
		fn := args[0].(callable)
		for {
			res, err := fn.Call(vm, 0)
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
	}
	return vm.LoadString(chunkname, src, mode, env)
}

// loadfile ([filename [, mode [, env]]])
func stdLoadFile(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(vm, args, "load", "~string", "~string", "~table"); err != nil {
		return nil, err
	}
	mode := ModeText & ModeBinary
	if len(args) == 0 {
		return vm.Load("chunk", os.Stdin, mode, nil)
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
	}
	return vm.LoadFile(filename, mode, env)
}

func assertArguments(vm *VM, args []Value, methodName string, assertions ...string) error {
	for i, assertion := range assertions {
		optional := strings.HasPrefix(assertion, "~")
		expectedTypes := strings.Split(strings.TrimPrefix(assertion, "~"), "|")
		if i >= len(args) && !optional {
			return argumentErr(vm, i+1, methodName, fmt.Errorf("%v expected", assertion))
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
			return argumentErr(vm, i+1, methodName, fmt.Errorf("%v expected but received %v", strings.Join(expectedTypes, ", "), valType))
		}
	}
	return nil
}

func argumentErr(vm *VM, nArg int, methodName string, err error) error {
	return vm.err("bad argument #%v to '%v' (%s)", nArg, methodName, err)
}
