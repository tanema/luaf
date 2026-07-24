package runtime

import (
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/tanema/luaf/internal/conf"
	"github.com/tanema/luaf/internal/lerrors"
	"github.com/tanema/luaf/internal/parse"
)

var (
	// WarnEnabled is the flag that will toggle warn messages, it can be toggled with the Warn() function.
	WarnEnabled  = false
	libsLoaded   = false
	libLoaderMux sync.Mutex
	_ENVName     = "_ENV"
)

func createDefaultEnv(withLibs bool) *Table {
	env := &Table{
		hashtable: map[any]any{
			"_LUAF_ENV":      true, // a variable to help check compatibility.
			"HOST_OS":        runtime.GOOS,
			"HOST_ARCH":      runtime.GOARCH,
			"_VERSION":       conf.LUAVERSION,
			"collectgarbage": Fn("collectgarbage", stdCollectgarbage),
			"error":          Fn("error", stdError),
			"getmetatable":   Fn("getmetatable", stdGetMetatable),
			"load":           Fn("load", stdLoad),
			"next":           Fn("next", stdNext),
			"rawequal":       Fn("rawequal", stdRawEq),
			"rawget":         Fn("rawget", stdRawGet),
			"rawlen":         Fn("rawlen", stdRawLen),
			"rawset":         Fn("rawset", stdRawSet),
			"require":        Fn("require", stdRequire),
			"setmetatable":   Fn("setmetatable", stdSetMetatable),
			"tonumber":       Fn("tonumber", stdToNumber),
			"tostring":       Fn("tostring", stdToString),
			"type":           Fn("type", stdType),
			"warn":           Fn("warn", stdWarn),
			"xpcall":         Fn("xpcall", stdXPCall),
			"package":        stdPackageLib,
		},
	}
	libLoaderMux.Lock()
	if withLibs && !libsLoaded {
		stdPkgFactories := map[string]func() *Table{
			"coroutine": createCoroutineLib,
			"debug":     createDebugLib,
			"io":        createIOLib,
			"math":      createMathLib,
			"os":        createOSLib,
			"string":    createStringLib,
			"table":     createTableLib,
			"utf8":      createUtf8Lib,
		}
		for name, fact := range stdPkgFactories {
			lib := fact()
			env.hashtable[name] = lib
			loadedPackages.hashtable[name] = lib
		}
		libsLoaded = true
	} else if withLibs && libsLoaded {
		stdPkgs := []string{"coroutine", "debug", "io", "math", "os", "string", "table", "utf8"}
		for _, name := range stdPkgs {
			env.hashtable[name] = loadedPackages.hashtable[name]
		}
	}
	libLoaderMux.Unlock()
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
	_, err := fmt.Fprintln(out, strings.Join(strParts, split))
	return nil, err
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
			return nil, argumentErr(2, "tonumber", fmt.Errorf("number expected, got %v", typeName(baseVal)))
		}
	}
	return []any{toNumber(args[0], base)}, nil
}

func stdType(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "type", "value"); err != nil {
		return nil, err
	}
	return []any{typeName(args[0])}, nil
}

func stdNext(vm *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "next", "table", "~value"); err != nil {
		return nil, err
	}

	table := args[0].(*Table)
	hashKeys := table.Keys()

	allKeys := make([]any, 0, len(table.val)+len(hashKeys))
	for i := 1; i <= len(table.val); i++ {
		if table.val[i-1] != nil {
			allKeys = append(allKeys, int64(i))
		}
	}
	allKeys = append(allKeys, hashKeys...)

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

func stdSetMetatable(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "setmetatable", "table", "~table"); err != nil {
		return nil, err
	}
	if method := findMetavalue(parse.MetaMeta, args[0]); method != nil {
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
	if method := findMetavalue(parse.MetaMeta, args[0]); method != nil {
		return []any{method}, nil
	}
	metatable := getMetatable(args[0])
	if metatable == nil {
		return []any{nil}, nil
	}
	return []any{metatable}, nil
}

func stdError(vm *VM, args []any) ([]any, error) {
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
	return nil, newUserErr(vm, level, errObj)
}

func stdWarn(vm *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "warn", "value"); err != nil {
		return nil, err
	}
	return warn(vm, args...)
}

func warn(vm *VM, args ...any) ([]any, error) {
	if len(args) == 1 && strings.HasPrefix(ToString(args[0]), "@") {
		switch args[0] {
		case "@on":
			WarnEnabled = true
		case "@off":
			WarnEnabled = false
		}
		return []any{}, nil
	}
	if !WarnEnabled {
		return []any{}, nil
	}
	return stdprintaux(vm, append([]any{"Lua warning: "}, args...), os.Stderr, "")
}

func stdXPCall(vm *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "xpcall", "function|table", "function|table"); err != nil {
		return nil, err
	}
	values, err := vm.call(args[0], args[2:])
	if err != nil {
		res, err := vm.call(args[1], []any{getErrVal(err)})
		return append([]any{false}, res...), err
	}
	return append([]any{true}, values...), nil
}

func stdRawGet(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "rawget", "table", "value"); err != nil {
		return nil, err
	}
	res, err := args[0].(*Table).Get(args[1])
	if err != nil {
		return nil, err
	}
	return []any{res}, nil
}

func stdRawSet(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "rawset", "table", "value", "value"); err != nil {
		return nil, err
	}
	return []any{}, args[0].(*Table).Set(args[1], args[2])
}

func stdRawEq(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "rawequal", "value", "value"); err != nil {
		return nil, err
	}
	lVal, rVal := args[0], args[1]

	typeA, typeB := typeName(lVal), typeName(rVal)
	if typeA != typeB {
		return []any{false}, nil
	}

	switch tval := lVal.(type) {
	case string:
		return []any{tval == rVal.(string)}, nil
	case int64, float64:
		vA, vB := toFloat(lVal), toFloat(rVal)
		return []any{vA == vB}, nil
	case bool:
		return []any{tval == rVal.(bool)}, nil
	case nil:
		return []any{true}, nil
	default:
		return []any{lVal == rVal}, nil
	}
}

func stdRawLen(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "rawlen", "string|table"); err != nil {
		return nil, err
	}
	switch tval := args[0].(type) {
	case string:
		return []any{int64(len(tval))}, nil
	case *Table:
		return []any{int64(len(tval.val))}, nil
	}
	return []any{}, nil
}

func stdLoad(vm *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "load", "string|function", "~string", "~string", "~value"); err != nil {
		return nil, err
	}
	var src string
	chunkname := "chunk"
	if str, isStr := args[0].(string); isStr {
		src = str
	} else if typeName(args[0]) == "function" {
		var buf strings.Builder
		for {
			res, err := vm.call(args[0], []any{})
			if err != nil {
				return []any{nil, getErrVal(err)}, nil
			} else if len(res) == 0 || res[0] == nil {
				break
			}
			str, isString := res[0].(string)
			if !isString {
				return []any{nil, "reader function must return a string"}, nil
			} else if str == "" {
				break
			}
			if _, err := buf.WriteString(str); err != nil {
				return nil, err
			}
		}

		src += buf.String()
	}
	mode := parse.ModeText | parse.ModeBinary
	if len(args) > 1 && args[1] != nil {
		chunkname = args[1].(string)
	}
	if len(args) > 2 && args[2] != nil {
		switch args[2].(string) {
		case "b":
			mode = parse.ModeBinary
		case "t":
			mode = parse.ModeText
		}
	}
	var env any = vm.env
	if len(args) > 3 && args[3] != nil {
		env = args[3]
	}

	fn, err := parse.Parse(chunkname, strings.NewReader(src), mode)
	var retVals []any
	if err != nil {
		retVals = []any{nil, err.Error()}
	} else {
		retVals = []any{&Closure{val: fn, upvalues: loadedChunkUpvalues(fn, env)}}
	}
	return retVals, nil
}

func loadedChunkUpvalues(fn *parse.FnProto, env any) []*upvalueBroker {
	upvalues := make([]*upvalueBroker, len(fn.UpIndexes))
	for i, idx := range fn.UpIndexes {
		if idx.Name == _ENVName {
			upvalues[i] = &upvalueBroker{name: _ENVName, val: env}
		} else {
			upvalues[i] = &upvalueBroker{name: idx.Name}
		}
	}
	return upvalues
}

func assertArguments(args []any, methodName string, assertions ...string) error {
	for i, assertion := range assertions {
		optional := strings.HasPrefix(assertion, "~")
		expectedTypes := strings.Split(strings.TrimPrefix(assertion, "~"), "|")
		if i >= len(args) || (optional && args[i] == nil) {
			if !optional {
				return argumentErr(i+1, methodName, fmt.Errorf("%v expected", assertion))
			}
			continue
		} else if strings.TrimPrefix(assertion, "~") == "value" {
			continue
		} else if valType := typeName(args[i]); !slices.Contains(expectedTypes, valType) {
			return argumentErr(
				i+1,
				methodName,
				fmt.Errorf("%v expected but received %v", strings.Join(expectedTypes, ", "), valType),
			)
		}
	}
	return nil
}

func argumentErr(nArg int, methodName string, err error) error {
	return fmt.Errorf("bad argument #%v to '%v' (%w)", nArg, methodName, err)
}

func getErrVal(err error) any {
	var luaErr *lerrors.Error
	if errors.As(err, &luaErr) {
		if luaErr.Value != nil {
			return luaErr.Value
		}
		return err.Error()
	}
	return err.Error()
}
