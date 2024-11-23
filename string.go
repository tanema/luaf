package luaf

import "fmt"

type String struct{ val string }

func (s *String) Type() string   { return "string" }
func (s *String) Val() any       { return string(s.val) }
func (s *String) String() string { return string(s.val) }
func (s *String) Meta() *Table   { return stringMetaTable }

// for some reason lua implements arithmetic operations on strings that will work
// if the strings are convertable into numbers
var stringMetaTable = NewTable(nil, map[any]Value{
	metaAdd:   strArith(metaAdd),
	metaSub:   strArith(metaSub),
	metaMul:   strArith(metaMul),
	metaMod:   strArith(metaMod),
	metaPow:   strArith(metaPow),
	metaDiv:   strArith(metaDiv),
	metaIDiv:  strArith(metaIDiv),
	metaUNM:   strArith(metaUNM),
	metaIndex: &ExternFunc{func(vm *VM, args []Value) ([]Value, error) { return []Value{&Nil{}}, nil }},
})

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
