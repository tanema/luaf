package luaf

import (
	"fmt"
	"strings"

	"github.com/tanema/luaf/string/pack"
	"github.com/tanema/luaf/string/pattern"
)

type String struct{ val string }

func createStringLib() *Table {
	return &Table{
		hashtable: map[any]Value{
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
}

func (s *String) Type() string   { return string(typeString) }
func (s *String) Val() any       { return string(s.val) }
func (s *String) String() string { return string(s.val) }
func (s *String) Meta() *Table   { return stringMetaTable }

// for some reason lua implements arithmetic operations on strings that will work
// if the strings are convertable into numbers
var stringMetaTable = &Table{
	hashtable: map[any]Value{
		"__name":  &String{val: "STRING"},
		"__add":   strArith(metaAdd),
		"__sub":   strArith(metaSub),
		"__mul":   strArith(metaMul),
		"__mod":   strArith(metaMod),
		"__pow":   strArith(metaPow),
		"__div":   strArith(metaDiv),
		"__idiv":  strArith(metaIDiv),
		"__unm":   strArith(metaUNM),
		"__index": createStringLib(),
	},
}

func strArith(op metaMethod) *GoFunc {
	return &GoFunc{
		name: fmt.Sprintf("string:%s", op),
		val: func(vm *VM, args []Value) ([]Value, error) {
			var lval, rval Value
			if len(args) < 1 {
				return nil, fmt.Errorf("bad argument #1 to 'string:%v' (value expected)", op)
			}
			lval = args[0]
			if op == metaUNM || op == metaBNot {
				rval = &Integer{} // mock second value for unary
			} else if len(args) < 2 {
				return nil, fmt.Errorf("bad argument #2 to 'string:%v' (value expected)", op)
			} else {
				rval = args[1]
			}
			lnum := toNumber(lval, 10)
			rnum := toNumber(rval, 10)
			nilval := &Nil{}
			if lnum != nilval && rnum != nilval {
				res, err := arith(vm, op, lnum, rnum)
				if err != nil {
					return nil, err
				}
				return []Value{res}, err
			}
			return nil, fmt.Errorf("cannot %v %v with %v", op, lval.Type(), rval.Type())
		}}
}

func stdStringByte(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "string.byte", "string", "~number", "~number"); err != nil {
		return nil, err
	}
	str := []byte(args[0].(*String).val)
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
		return []Value{}, nil
	}
	if j >= len(str) {
		j = len(str)
	}
	out := []Value{}
	for _, b := range str[i:j] {
		out = append(out, &Integer{val: int64(b)})
	}
	return out, nil
}

func stdStringChar(vm *VM, args []Value) ([]Value, error) {
	points := []byte{}
	for i, point := range args {
		if !isNumber(point) {
			return nil, argumentErr(i+1, "string.char", fmt.Errorf("number expected, got %v", point.Type()))
		}
		points = append(points, byte(toInt(point)))
	}
	return []Value{&String{val: string(points)}}, nil
}

func stdStringDump(vm *VM, args []Value) ([]Value, error) {
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
		strip = toBool(args[0]).val
	}

	data, err := fn.Dump(strip)
	if err != nil {
		return nil, fmt.Errorf("could not dump fn: %v", err)
	}
	return []Value{&String{val: string(data)}}, nil
}

func stdStringFind(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "string.find", "string", "string", "~number", "~boolean"); err != nil {
		return nil, err
	}
	src := args[0].(*String).val
	pat := args[1].(*String).val
	init := 1
	plain := false
	if len(args) > 2 {
		init = clamp(int(toInt(args[2])), 1, max(len(src), 1))
	}
	if len(args) > 3 {
		plain = toBool(args[3]).val
	}

	if plain {
		if index := strings.Index(src[init-1:], pat); index >= 0 {
			return []Value{
				&Integer{val: int64(index) + int64(init)},
				&Integer{val: int64(index+len(pat)) + int64(init) - 1},
			}, nil
		}
		return []Value{&Nil{}}, nil
	}

	parsedPattern, err := pattern.Parse(pat)
	if err != nil {
		return nil, fmt.Errorf("bad pattern: %v", err)
	}
	matches, err := parsedPattern.Find(src[init-1:], 1)
	if err != nil {
		return nil, fmt.Errorf("bad pattern: %v", err)
	}
	out := []Value{}
	if len(matches) > 0 {
		m := matches[0]
		out = append(out, &Integer{val: int64(m.Start) + int64(init)}, &Integer{val: int64(m.End) + int64(init)})
	}
	if len(matches) > 1 {
		for i := 1; i < len(matches); i++ {
			out = append(out, &String{val: matches[i].Subs})
		}
		return out, nil
	}
	return []Value{&Nil{}}, nil
}

func stdStringMatch(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "string.match", "string", "string", "~number"); err != nil {
		return nil, err
	}
	src := args[0].(*String).val
	pat := args[1].(*String).val
	init := 0
	if len(args) > 2 {
		init = clamp(int(toInt(args[2])), 1, len(src))
	}
	parsedPattern, err := pattern.Parse(pat)
	if err != nil {
		return nil, fmt.Errorf("bad pattern: %v", err)
	}
	matches, err := parsedPattern.Find(src[init-1:], 1)
	if err != nil {
		return nil, fmt.Errorf("bad pattern: %v", err)
	}
	if len(matches) == 0 {
		return []Value{&Nil{}}, nil
	}
	out := make([]Value, len(matches))
	for i := 0; i < len(matches); i++ {
		out[i] = &String{val: matches[i].Subs}
	}
	return out, nil
}

// for k, v in string.gmatch("from=world, to=Lua", "%w+=(%w+)") do print(k, v) end
func stdStringGMatchNext(iter pattern.PatternIterator) func(*VM, []Value) ([]Value, error) {
	return func(vm *VM, args []Value) ([]Value, error) {
		if err := assertArguments(args, "string.match.next"); err != nil {
			return nil, err
		}
		matches, err := iter.Next()
		if err != nil {
			return nil, err
		} else if len(matches) == 0 {
			return []Value{&Nil{}}, nil
		}
		result := []Value{}
		if len(matches) > 1 { // we have submatches so lets surface those
			matches = matches[1:]
		}
		for _, match := range matches {
			result = append(result, &String{val: match.Subs})
		}
		return result, nil
	}
}

func stdStringGMatch(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "string.gmatch", "string", "string"); err != nil {
		return nil, err
	}
	src := args[0].(*String).val
	parsedPattern, err := pattern.Parse(args[1].(*String).val)
	if err != nil {
		return nil, fmt.Errorf("bad pattern: %v", err)
	}
	return []Value{Fn("string.gmatch.next", stdStringGMatchNext(parsedPattern.Iter(src))), args[0], &Nil{}}, nil
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
func stdStringGSub(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "string.gsub", "string", "string", "string|table|function"); err != nil {
		return nil, err
	}
	src := args[0].(*String).val
	parsedPattern, err := pattern.Parse(args[1].(*String).val)
	if err != nil {
		return nil, fmt.Errorf("bad pattern: %v", err)
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
		case *String:
			repSubs := strings.Clone(tval.val)
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
				val = &String{val: ""}
			}
			resStr, err := toString(vm, val)
			if err != nil {
				return nil, err
			}
			toSub = resStr.val
		case *GoFunc, *Closure:
			params := []Value{}
			for _, match := range matches {
				params = append(params, &String{val: match.Subs})
			}
			res, err := vm.call(tval, params)
			if err != nil {
				return nil, err
			}
			if len(res) > 0 {
				resStr, err := toString(vm, res[0])
				if err != nil {
					return nil, err
				}
				toSub = resStr.val
			}
		}
		outputStr.WriteString(src[start:matches[0].Start])
		outputStr.WriteString(toSub)
		start = matches[0].End
	}
	outputStr.WriteString(src[start:])
	return []Value{&String{val: outputStr.String()}}, nil
}

func stdStringFormat(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "string.format", "string"); err != nil {
		return nil, err
	}
	pattern := args[0].(*String).val
	data := []any{}
	for _, value := range args[1:] {
		data = append(data, value.Val())
	}
	return []Value{&String{val: fmt.Sprintf(pattern, data...)}}, nil
}

func stdStringLen(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "string.len", "string"); err != nil {
		return nil, err
	}
	return []Value{&Integer{val: int64(len(args[0].(*String).val))}}, nil
}

func stdStringLower(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "string.lower", "string"); err != nil {
		return nil, err
	}
	lowerStr := strings.ToLower(args[0].(*String).val)
	return []Value{&String{val: lowerStr}}, nil
}

func stdStringUpper(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "string.upper", "string"); err != nil {
		return nil, err
	}
	upperStr := strings.ToUpper(args[0].(*String).val)
	return []Value{&String{val: upperStr}}, nil
}

func stdStringRep(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "string.rep", "string", "number", "~string"); err != nil {
		return nil, err
	}
	str := args[0].(*String).val
	count := toInt(args[1])
	parts := repeat(str, int(count))
	sep := ""
	if len(args) > 2 {
		sep = args[2].(*String).val
	}
	return []Value{&String{val: strings.Join(parts, sep)}}, nil
}

func stdStringReverse(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "string.reverse", "string"); err != nil {
		return nil, err
	}
	revStr := reverse([]rune(args[0].(*String).val))
	return []Value{&String{val: string(revStr)}}, nil
}

func stdStringSub(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "string.sub", "string", "number", "~number"); err != nil {
		return nil, err
	}
	str := args[0].(*String).val
	i := substringIndex(args[1], len(str))
	if len(args) == 2 {
		if i == 0 {
			return []Value{args[0]}, nil
		} else if int(i) > len(str) {
			return []Value{&String{}}, nil
		}
		return []Value{&String{val: str[i-1:]}}, nil
	}

	if int(i) > len(str) {
		return []Value{&String{}}, nil
	}

	j := substringIndex(args[2], len(str))
	if j < i {
		return []Value{&String{}}, nil
	}
	return []Value{&String{val: str[i-1 : clamp(int(j), int(i), len(str))]}}, nil
}

func stdStringPack(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "string.pack", "string"); err != nil {
		return nil, err
	}
	params := make([]any, len(args)-1)
	for i, a := range args[1:] {
		params[i] = a.Val()
	}
	str, err := pack.Pack(args[0].(*String).val, params...)
	if err != nil {
		return nil, err
	}
	return []Value{&String{val: str}}, nil
}

func stdStringPacksize(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "string.packsize", "string"); err != nil {
		return nil, err
	}
	size, err := pack.Packsize(args[0].(*String).val)
	return []Value{&Integer{val: int64(size)}}, err
}

func stdStringUnpack(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "string.unpack", "string", "string"); err != nil {
		return nil, err
	}
	data, err := pack.Unpack(args[0].(*String).val, args[1].(*String).val)
	if err != nil {
		return nil, err
	}
	values := make([]Value, len(data))
	for i, v := range data {
		values[i] = ToValue(v)
	}
	return values, nil
}
