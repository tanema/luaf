package luaf

import "fmt"

type String struct{ val string }

var libString = &Table{
	hashtable: map[any]Value{
		"byte":     &ExternFunc{stdStringByte},
		"char":     &ExternFunc{nil},
		"dump":     &ExternFunc{nil},
		"find":     &ExternFunc{nil},
		"format":   &ExternFunc{nil},
		"gmatch":   &ExternFunc{nil},
		"gsub":     &ExternFunc{nil},
		"len":      &ExternFunc{nil},
		"lower":    &ExternFunc{nil},
		"match":    &ExternFunc{nil},
		"pack":     &ExternFunc{nil},
		"packsize": &ExternFunc{nil},
		"rep":      &ExternFunc{nil},
		"reverse":  &ExternFunc{nil},
		"sub":      &ExternFunc{nil},
		"unpack":   &ExternFunc{nil},
		"upper":    &ExternFunc{nil},
	},
}

func (s *String) Type() string   { return "string" }
func (s *String) Val() any       { return string(s.val) }
func (s *String) String() string { return string(s.val) }
func (s *String) Meta() *Table   { return stringMetaTable }

// for some reason lua implements arithmetic operations on strings that will work
// if the strings are convertable into numbers
var stringMetaTable = &Table{
	hashtable: map[any]Value{
		string(metaAdd):   strArith(metaAdd),
		string(metaSub):   strArith(metaSub),
		string(metaMul):   strArith(metaMul),
		string(metaMod):   strArith(metaMod),
		string(metaPow):   strArith(metaPow),
		string(metaDiv):   strArith(metaDiv),
		string(metaIDiv):  strArith(metaIDiv),
		string(metaUNM):   strArith(metaUNM),
		string(metaIndex): libString,
	},
}

func strArith(op metaMethod) *ExternFunc {
	return &ExternFunc{func(vm *VM, args []Value) ([]Value, error) {
		var lval, rval Value
		if len(args) < 1 {
			return nil, fmt.Errorf("bad argument #1 to '%v' (value expected)", op)
		}
		lval = args[0]
		if op == metaUNM || op == metaBNot {
			rval = &Integer{} // mock second value for unary
		} else if len(args) < 2 {
			return nil, fmt.Errorf("bad argument #2 to '%v' (value expected)", op)
		} else {
			rval = args[1]
		}
		lnum := toNumber(lval, 10)
		rnum := toNumber(rval, 10)
		nilval := &Nil{}
		if lnum != nilval && rnum != nilval {
			res, err := vm.arith(op, lnum, rnum)
			if err != nil {
				return nil, err
			}
			return []Value{res}, err
		}
		return nil, fmt.Errorf("cannot %v %v with %v", op, lval.Type(), rval.Type())
	}}
}

func stdStringByte(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(vm, args, "string.byte", "string", "~number", "~number"); err != nil {
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
