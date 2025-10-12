package runtime

import (
	"fmt"
	"strings"

	"github.com/tanema/luaf/src/lstring"
	"github.com/tanema/luaf/src/lstring/pack"
	"github.com/tanema/luaf/src/lstring/pattern"
	"github.com/tanema/luaf/src/parse"
)

var stringMetaTable *Table

func createStringLib() *Table {
	strLib := &Table{
		hashtable: map[any]any{
			"byte":     Fn("string.byte", stdStringByte),
			"char":     Fn("string.char", stdStringChar),
			"dump":     Fn("string.dump", stdStringDump),
			"find":     Fn("string.find", stdStringFind),
			"match":    Fn("string.match", stdStringMatch),
			"gmatch":   Fn("string.gmatch", stdStringGMatch),
			"gsub":     Fn("string.gsub", stdStringGSub),
			"format":   Fn("string.format", stdStringFormat),
			"len":      Fn("string.len", stdStringLen),
			"lower":    Fn("string.lower", stdStringLower),
			"rep":      Fn("string.rep", stdStringRep),
			"reverse":  Fn("string.reverse", stdStringReverse),
			"upper":    Fn("string.upper", stdStringUpper),
			"sub":      Fn("string.sub", stdStringSub),
			"pack":     Fn("string.pack", stdStringPack),
			"packsize": Fn("string.packsize", stdStringPacksize),
			"unpack":   Fn("string.unpack", stdStringUnpack),
		},
	}

	// if the strings are convertable into numbers.
	stringMetaTable = &Table{
		hashtable: map[any]any{
			"__name":  "STRING",
			"__add":   strArith(parse.MetaAdd),
			"__sub":   strArith(parse.MetaSub),
			"__mul":   strArith(parse.MetaMul),
			"__mod":   strArith(parse.MetaMod),
			"__pow":   strArith(parse.MetaPow),
			"__div":   strArith(parse.MetaDiv),
			"__idiv":  strArith(parse.MetaIDiv),
			"__unm":   strArith(parse.MetaUNM),
			"__index": strLib,
		},
	}
	return strLib
}

func strArith(op parse.MetaMethod) *GoFunc {
	return &GoFunc{
		name: fmt.Sprintf("string:%s", op),
		val: func(vm *VM, args []any) ([]any, error) {
			var lval, rval any
			if len(args) < 1 {
				return nil, fmt.Errorf("bad argument #1 to 'string:%v' (value expected)", op)
			}
			lval = args[0]
			if op == parse.MetaUNM || op == parse.MetaBNot {
				rval = int64(0) // mock second value for unary
			} else if len(args) < 2 {
				return nil, fmt.Errorf("bad argument #2 to 'string:%v' (value expected)", op)
			} else {
				rval = args[1]
			}
			lnum := toNumber(lval, 10)
			rnum := toNumber(rval, 10)
			if lnum != nil && rnum != nil {
				res, err := arith(vm, op, lnum, rnum)
				if err != nil {
					return nil, err
				}
				return []any{res}, err
			}
			return nil, fmt.Errorf("cannot %v %v with %v", op, typeName(lval), typeName(rval))
		},
	}
}

func stdStringByte(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "string.byte", "string", "~number", "~number"); err != nil {
		return nil, err
	}

	start, end := int64(1), int64(1)
	if len(args) > 1 {
		start = toInt(args[1])
		end = start
	}
	if len(args) > 2 {
		end = toInt(args[2])
	}

	str := args[0].(string)
	substr := lstring.Substring(str, start, end)
	if len(substr) == 0 {
		return []any{nil}, nil
	}

	out := []any{}
	for _, b := range substr {
		out = append(out, int64(b))
	}
	return out, nil
}

func stdStringChar(_ *VM, args []any) ([]any, error) {
	var str strings.Builder
	for i, point := range args {
		if point != nil && !isNumber(point) {
			return nil, argumentErr(i+1, "string.char", fmt.Errorf("number expected, got %v", typeName(point)))
		} else if point == nil {
			continue
		}
		if _, err := str.WriteRune(rune(toInt(point))); err != nil {
			return nil, err
		}
	}
	return []any{str.String()}, nil
}

func stdStringDump(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "string.dump", "function", "~boolean"); err != nil {
		return nil, err
	}
	var fn *parse.FnProto
	switch cls := args[0].(type) {
	case *Closure:
		fn = cls.val
	default:
		return nil, argumentErr(1, "string.dump", fmt.Errorf("unable to dump %T", args[0]))
	}
	strip := false
	if len(args) > 1 {
		strip = toBool(args[0])
	}

	data, err := fn.Dump(strip)
	if err != nil {
		return nil, fmt.Errorf("could not dump fn: %w", err)
	}
	return []any{string(data)}, nil
}

func stdStringFind(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "string.find", "string", "string", "~number", "~boolean"); err != nil {
		return nil, err
	}
	src := args[0].(string)
	pat := args[1].(string)
	init := 1
	plain := false
	if len(args) > 2 {
		init = clamp(int(toInt(args[2])), 1, max(len(src), 1))
	}
	if len(args) > 3 {
		plain = toBool(args[3])
	}

	src = lstring.Substring(src, int64(init), int64(len(src)))

	if plain {
		if index := strings.Index(src, pat); index >= 0 {
			return []any{
				int64(index) + int64(init),
				int64(index+len(pat)) + int64(init) - 1,
			}, nil
		}
		return []any{nil}, nil
	}

	parsedPattern, err := pattern.Parse(pat)
	if err != nil {
		return nil, fmt.Errorf("bad pattern: %w", err)
	}
	matches, err := parsedPattern.Find(src, 1)
	if err != nil {
		return nil, fmt.Errorf("bad pattern: %w", err)
	}
	out := []any{}
	if len(matches) > 0 {
		m := matches[0]
		out = append(out, int64(m.Start)+int64(init), int64(m.End)+int64(init))
	}
	if len(matches) > 1 {
		for i := 1; i < len(matches); i++ {
			out = append(out, matches[i].Subs)
		}
		return out, nil
	}
	return []any{nil}, nil
}

func stdStringMatch(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "string.match", "string", "string", "~number"); err != nil {
		return nil, err
	}
	src := args[0].(string)
	pat := args[1].(string)
	init := 0
	if len(args) > 2 {
		init = clamp(int(toInt(args[2])), 1, len(src)) - 1
	}
	parsedPattern, err := pattern.Parse(pat)
	if err != nil {
		return nil, fmt.Errorf("bad pattern: %w", err)
	}
	matches, err := parsedPattern.Find(src[init:], 1)
	if err != nil {
		return nil, fmt.Errorf("bad pattern: %w", err)
	}
	if len(matches) == 0 {
		return []any{nil}, nil
	}
	out := make([]any, len(matches))
	for i := range matches {
		out[i] = matches[i].Subs
	}
	return out, nil
}

// for k, v in string.gmatch("from=world, to=Lua", "%w+=(%w+)") do print(k, v) end.
func stdStringGMatchNext(iter pattern.Iterator) func(*VM, []any) ([]any, error) {
	return func(_ *VM, args []any) ([]any, error) {
		if err := assertArguments(args, "string.match.next"); err != nil {
			return nil, err
		}
		matches, err := iter.Next()
		if err != nil {
			return nil, err
		} else if len(matches) == 0 {
			return []any{nil}, nil
		}
		result := []any{}
		if len(matches) > 1 { // we have submatches so lets surface those
			matches = matches[1:]
		}
		for _, match := range matches {
			result = append(result, match.Subs)
		}
		return result, nil
	}
}

func stdStringGMatch(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "string.gmatch", "string", "string"); err != nil {
		return nil, err
	}
	src := args[0].(string)
	parsedPattern, err := pattern.Parse(args[1].(string))
	if err != nil {
		return nil, fmt.Errorf("bad pattern: %w", err)
	}
	return []any{Fn("string.gmatch.next", stdStringGMatchNext(parsedPattern.Iter(src))), args[0], nil}, nil
}

/*
Repl:
  - string: then its value is used for replacement. The character % works
    as an escape character: any sequence in repl of the form %d, with d between 1
    and 9, stands for the value of the d-th captured substring. The sequence %0
    stands for the whole match. The sequence %% stands for a single %.
  - table: then the table is queried for every match, using the first capture as
    the key.
  - function: this function is called every time a match occurs, with all captured
    substrings passed as arguments, in order.
*/
func stdStringGSub(vm *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "string.gsub", "string", "string", "string|table|function"); err != nil {
		return nil, err
	}
	src := args[0].(string)
	parsedPattern, err := pattern.Parse(args[1].(string))
	if err != nil {
		return nil, fmt.Errorf("bad pattern: %w", err)
	}
	iter := parsedPattern.Iter(src)

	var outputStr strings.Builder
	start := 0
	for {
		matches, err := iter.Next()
		if err != nil {
			return nil, err
		} else if len(matches) == 0 {
			break
		}

		var toSub string
		switch tval := args[2].(type) {
		case string:
			repSubs := strings.Clone(tval)
			for i, m := range matches {
				repSubs = strings.ReplaceAll(repSubs, fmt.Sprintf("%%%v", i), m.Subs)
			}
			toSub = repSubs
		case *Table:
			key := matches[0].Subs
			if len(matches) > 1 {
				key = matches[1].Subs
			}
			val, ok := tval.hashtable[key]
			if !ok {
				val = ""
			}
			resStr, err := vm.toString(val)
			if err != nil {
				return nil, err
			}
			toSub = resStr
		case *GoFunc, *Closure:
			params := []any{}
			for _, match := range matches {
				params = append(params, match.Subs)
			}
			res, err := vm.call(tval, params)
			if err != nil {
				return nil, err
			}
			if len(res) > 0 {
				resStr, err := vm.toString(res[0])
				if err != nil {
					return nil, err
				}
				toSub = resStr
			}
		}
		outputStr.WriteString(src[start:matches[0].Start])
		outputStr.WriteString(toSub)
		start = matches[0].End
	}
	outputStr.WriteString(src[start:])
	return []any{outputStr.String()}, nil
}

func stdStringFormat(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "string.format", "string"); err != nil {
		return nil, err
	}
	fmtStr, err := lstring.Format(args[0].(string), args[1:]...)
	return []any{fmtStr}, err
}

func stdStringLen(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "string.len", "string"); err != nil {
		return nil, err
	}
	return []any{int64(len(args[0].(string)))}, nil
}

func stdStringLower(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "string.lower", "string"); err != nil {
		return nil, err
	}
	return []any{strings.ToLower(args[0].(string))}, nil
}

func stdStringUpper(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "string.upper", "string"); err != nil {
		return nil, err
	}
	return []any{strings.ToUpper(args[0].(string))}, nil
}

func stdStringRep(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "string.rep", "string", "number", "~string"); err != nil {
		return nil, err
	}
	sep := ""
	if len(args) > 2 {
		sep = args[2].(string)
	}
	return []any{lstring.Repeat(args[0].(string), sep, toInt(args[1]))}, nil
}

func stdStringReverse(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "string.reverse", "string"); err != nil {
		return nil, err
	}
	return []any{lstring.Reverse(args[0].(string))}, nil
}

func stdStringSub(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "string.sub", "string", "number", "~number"); err != nil {
		return nil, err
	}
	end := int64(len(args[0].(string)))
	if len(args) > 2 {
		end = toInt(args[2])
	}
	return []any{lstring.Substring(args[0].(string), toInt(args[1]), end)}, nil
}

func stdStringPack(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "string.pack", "string"); err != nil {
		return nil, err
	}
	str, err := pack.Pack(args[0].(string), args[1:]...)
	return []any{str}, err
}

func stdStringPacksize(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "string.packsize", "string"); err != nil {
		return nil, err
	}
	size, err := pack.Packsize(args[0].(string))
	return []any{int64(size)}, err
}

func stdStringUnpack(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "string.unpack", "string", "string"); err != nil {
		return nil, err
	}
	return pack.Unpack(args[0].(string), args[1].(string))
}

func clamp(f, low, high int) int {
	return min(max(f, low), high)
}
