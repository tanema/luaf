package luaf

import (
	"fmt"
	"strings"

	"github.com/tanema/luaf/string/pack"
	"github.com/tanema/luaf/string/pattern"
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
			"__add":   strArith(metaAdd),
			"__sub":   strArith(metaSub),
			"__mul":   strArith(metaMul),
			"__mod":   strArith(metaMod),
			"__pow":   strArith(metaPow),
			"__div":   strArith(metaDiv),
			"__idiv":  strArith(metaIDiv),
			"__unm":   strArith(metaUNM),
			"__index": strLib,
		},
	}
	return strLib
}

func strArith(op metaMethod) *GoFunc {
	return &GoFunc{
		name: fmt.Sprintf("string:%s", op),
		val: func(vm *VM, args []any) ([]any, error) {
			var lval, rval any
			if len(args) < 1 {
				return nil, fmt.Errorf("bad argument #1 to 'string:%v' (value expected)", op)
			}
			lval = args[0]
			if op == metaUNM || op == metaBNot {
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
			return nil, fmt.Errorf("cannot %v %v with %v", op, TypeName(lval), TypeName(rval))
		},
	}
}

func stdStringByte(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "string.byte", "string", "~number", "~number"); err != nil {
		return nil, err
	}
	str := []byte(args[0].(string))
	i, j := 0, 1
	if len(args) > 1 {
		i = int(toInt(args[1])) - 1
		j = i + 1
	}
	if len(args) > 2 {
		j = int(toInt(args[2]))
	}
	if i < 0 {
		i = len(str) + i
	}
	if j < 0 {
		j = len(str) + j
	}
	if j < i || i >= len(str) {
		return []any{}, nil
	}
	if j >= len(str) {
		j = len(str)
	}
	out := []any{}
	for _, b := range str[i:j] {
		out = append(out, int64(b))
	}
	return out, nil
}

func stdStringChar(_ *VM, args []any) ([]any, error) {
	points := []byte{}
	for i, point := range args {
		if !isNumber(point) {
			return nil, argumentErr(i+1, "string.char", fmt.Errorf("number expected, got %v", TypeName(point)))
		}
		points = append(points, byte(toInt(point)))
	}
	return []any{string(points)}, nil
}

func stdStringDump(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "string.dump", "function", "~boolean"); err != nil {
		return nil, err
	}
	var fn *FnProto
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

	if plain {
		if index := strings.Index(src[init-1:], pat); index >= 0 {
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
	matches, err := parsedPattern.Find(src[init-1:], 1)
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
		init = clamp(int(toInt(args[2])), 1, len(src))
	}
	parsedPattern, err := pattern.Parse(pat)
	if err != nil {
		return nil, fmt.Errorf("bad pattern: %w", err)
	}
	matches, err := parsedPattern.Find(src[init-1:], 1)
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
	pattern := args[0].(string)
	return []any{fmt.Sprintf(pattern, args[1:]...)}, nil
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
	lowerStr := strings.ToLower(args[0].(string))
	return []any{lowerStr}, nil
}

func stdStringUpper(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "string.upper", "string"); err != nil {
		return nil, err
	}
	upperStr := strings.ToUpper(args[0].(string))
	return []any{upperStr}, nil
}

func stdStringRep(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "string.rep", "string", "number", "~string"); err != nil {
		return nil, err
	}
	str := args[0].(string)
	count := toInt(args[1])
	parts := make([]string, count)
	for i := range count {
		parts[i] = str
	}
	sep := ""
	if len(args) > 2 {
		sep = args[2].(string)
	}
	return []any{strings.Join(parts, sep)}, nil
}

func stdStringReverse(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "string.reverse", "string"); err != nil {
		return nil, err
	}
	str := []rune(args[0].(string))
	for i, j := 0, len(str)-1; i < j; i, j = i+1, j-1 {
		str[i], str[j] = str[j], str[i]
	}
	return []any{string(str)}, nil
}

func stdStringSub(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "string.sub", "string", "number", "~number"); err != nil {
		return nil, err
	}
	str := args[0].(string)
	i := substringIndex(args[1], len(str))
	if len(args) == 2 {
		if i == 0 {
			return []any{args[0]}, nil
		} else if int(i) > len(str) {
			return []any{""}, nil
		}
		return []any{str[i-1:]}, nil
	}

	if int(i) > len(str) {
		return []any{""}, nil
	}

	j := substringIndex(args[2], len(str))
	if j < i {
		return []any{""}, nil
	}
	return []any{str[i-1 : clamp(int(j), int(i), len(str))]}, nil
}

func stdStringPack(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "string.pack", "string"); err != nil {
		return nil, err
	}
	str, err := pack.Pack(args[0].(string), args[1:]...)
	if err != nil {
		return nil, err
	}
	return []any{str}, nil
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
