package shine

import (
	"errors"
	"fmt"
	"math"
	"strings"
)

type (
	VM struct {
		pc    int64
		base  int64
		stack []Value
	}
	RuntimeErr struct {
		msg string
	}
)

func (err *RuntimeErr) Error() string {
	return err.msg
}

func NewVM() *VM {
	return &VM{
		stack: []Value{},
		base:  0,
	}
}

func (vm *VM) Eval(res *ParseResult) error {
	return vm.eval(res, res.Blocks[0])
}

func (vm *VM) err(tmpl string, args ...any) error {
	return &RuntimeErr{msg: fmt.Sprintf(tmpl, args...)}
}

func (vm *VM) eval(res *ParseResult, fn *Scope) error {
	for {
		var err error
		if int64(len(fn.ByteCodes)) <= vm.pc {
			return nil
		}
		instruction := fn.ByteCodes[vm.pc]
		switch instruction.Op() {
		case MOVE:
			a, b, _ := instruction.ABC()
			err = vm.SetStack(a, vm.GetStack(b))
		case LOADK:
			a, b := instruction.ABx()
			if b < 0 || int(b) > len(fn.Constants) {
				return errors.New("Constant address out of bounds")
			}
			err = vm.SetStack(a, fn.Constants[b])
		case LOADBOOL:
			a, b, c := instruction.ABC()
			err = vm.SetStack(a, &Boolean{val: b == 1})
			if c != 0 {
				vm.pc++
			}
		case LOADINT:
			a, b := instruction.AsBx()
			err = vm.SetStack(a, &Integer{val: b})
		case LOADNIL:
			a, _, _ := instruction.ABC()
			err = vm.SetStack(a, &Nil{})
		case ADD:
			err = vm.setABCFn(instruction, vm.binOp(func(x, y int64) Value { return &Integer{val: x + y} }, func(x, y float64) Value { return &Float{val: x + y} }))
		case SUB:
			err = vm.setABCFn(instruction, vm.binOp(func(x, y int64) Value { return &Integer{val: x - y} }, func(x, y float64) Value { return &Float{val: x - y} }))
		case MUL:
			err = vm.setABCFn(instruction, vm.binOp(func(x, y int64) Value { return &Integer{val: x * y} }, func(x, y float64) Value { return &Float{val: x * y} }))
		case DIV:
			err = vm.setABCFn(instruction, vm.binOp(func(x, y int64) Value { return &Integer{val: x / y} }, func(x, y float64) Value { return &Float{val: x / y} }))
		case MOD:
			err = vm.setABCFn(instruction, vm.binOp(func(x, y int64) Value { return &Integer{val: x % y} }, func(x, y float64) Value { return &Float{val: math.Mod(x, y)} }))
		case POW:
			err = vm.setABCFn(instruction, vm.binOp(func(x, y int64) Value { return &Integer{val: x ^ y} }, func(x, y float64) Value { return &Float{val: math.Pow(x, y)} }))
		case IDIV:
			err = vm.setABCFn(instruction, vm.binOp(func(x, y int64) Value { return &Integer{val: x / y} }, func(x, y float64) Value { return &Float{val: math.Floor(x / y)} }))
		case BAND:
			err = vm.setABCFn(instruction, vm.ibinOp(func(x, y int64) Value { return &Integer{val: x & y} }))
		case BOR:
			err = vm.setABCFn(instruction, vm.ibinOp(func(x, y int64) Value { return &Integer{val: x | y} }))
		case BXOR:
			err = vm.setABCFn(instruction, vm.ibinOp(func(x, y int64) Value { return &Integer{val: x ^ y} }))
		case SHL:
			err = vm.setABCFn(instruction, vm.ibinOp(func(x, y int64) Value { return &Integer{val: x << y} }))
		case SHR:
			err = vm.setABCFn(instruction, vm.ibinOp(func(x, y int64) Value { return &Integer{val: x >> y} }))
		case UNM:
			err = vm.setABCFn(instruction, vm.binOp(func(x, y int64) Value { return &Integer{val: -x} }, func(x, y float64) Value { return &Float{val: -x} }))
		case BNOT:
			err = vm.setABCFn(instruction, vm.ibinOp(func(x, y int64) Value { return &Integer{val: ^x} }))
		case NOT:
			err = vm.setABCFn(instruction, func(lVal, rVal Value) (Value, error) { return lVal.Bool().Not(), nil })
		case CONCAT:
			a, b, c := instruction.ABC()
			var strBuilder strings.Builder
			if c < b {
				c = b
			}
			for i := b; b < c; i++ {
				fmt.Fprint(&strBuilder, vm.GetStack(i).String())
			}
			vm.SetStack(a, &String{val: strBuilder.String()})
		case JMP: // TODO if A is not 0 then upvalues need to be closed
			_, b := instruction.AsBx()
			if b != 0 {
				vm.pc += b
			}
		case EQ:
			a, b, c := instruction.ABC()
			expected := a != 0
			isEq, err := eq(vm.GetStack(b), vm.GetStack(c))
			if err != nil {
				return err
			} else if isEq != expected {
				vm.pc++
			}
		case LT:
			a, b, c := instruction.ABC()
			expected := a != 0
			res, err := compare(vm.GetStack(b), vm.GetStack(c))
			if err != nil {
				return err
			} else if isMatch := res < 0; isMatch != expected {
				vm.pc++
			}
		case LE:
			a, b, c := instruction.ABC()
			expected := a != 0
			res, err := compare(vm.GetStack(b), vm.GetStack(c))
			if err != nil {
				return err
			} else if isMatch := res <= 0; isMatch != expected {
				vm.pc++
			}
		case TEST:
			a, b, _ := instruction.ABC()
			expected := b != 0
			actual := vm.GetStack(a).Bool().val
			if expected != actual {
				vm.pc++
			}
		case TESTSET:
			a, b, c := instruction.ABC()
			expected := c != 0
			actual := vm.GetStack(b).Bool().val
			if expected != actual {
				vm.pc++
			} else {
				vm.SetStack(a, &Boolean{val: actual})
			}
		case LEN:
			a, b, _ := instruction.ABC()
			val := vm.GetStack(b)
			switch tval := val.(type) {
			case *String:
				vm.SetStack(a, &Integer{val: int64(len(tval.val))})
			case *Table:
				vm.SetStack(a, &Integer{val: int64(len(tval.val))})
			default:
				err = fmt.Errorf("attempt to get length of a %v value", val.Type())
			}
		case GETUPVAL:
		case GETTABUP:
		case GETTABLE:
		case SETTABUP:
		case SETUPVAL:
		case SETTABLE:
		case NEWTABLE:
		case SELF:
		case CALL:
		case TAILCALL:
		case RETURN:
		case FORLOOP:
		case FORPREP:
		case TFORLOOP:
		case TFORCALL:
		case SETLIST:
		case CLOSURE:
		case VARARG:
		default:
		}
		if err != nil {
			return err
		}
		vm.pc++
	}
}

type opFn func(lVal, rVal Value) (Value, error)

func (vm *VM) GetStack(id int64) Value {
	if int(vm.base+id) >= len(vm.stack)-1 || id < 0 {
		return &Nil{}
	}
	return vm.stack[vm.base+id]
}

func (vm *VM) SetStack(id int64, val Value) error {
	if int(vm.base+id) >= cap(vm.stack)-1 {
		newStack := make([]Value, len(vm.stack), 2*len(vm.stack)+1)
		copy(newStack, vm.stack)
		vm.stack = newStack
	} else if id < 0 {
		return errors.New("cannot address negatively in the stack")
	}
	vm.stack[vm.base+id] = val
	return nil
}

func (vm *VM) setABCFn(instruction Bytecode, fn opFn) error {
	a, b, c := instruction.ABC()
	val, err := fn(vm.GetStack(b), vm.GetStack(c))
	if err != nil {
		return err
	}
	return vm.SetStack(a, val)
}

func (vm *VM) binOp(ifn func(a, b int64) Value, ffn func(a, b float64) Value) opFn {
	return func(lVal, rVal Value) (Value, error) {
		switch lVal.(type) {
		case *Integer:
			switch rVal.(type) {
			case *Integer:
				val := ifn(lVal.Val().(int64), rVal.Val().(int64))
				return val, nil
			case *Float:
				val := ffn(float64(lVal.Val().(int64)), rVal.Val().(float64))
				return val, nil
			}
		case *Float:
			switch rVal.(type) {
			case *Integer:
				val := ffn(lVal.Val().(float64), float64(rVal.Val().(int64)))
				return val, nil
			case *Float:
				val := ffn(lVal.Val().(float64), rVal.Val().(float64))
				return val, nil
			}
		}
		return nil, vm.err("cannot <> %v and %v", lVal.Type(), rVal.Type())
	}
}

func (vm *VM) ibinOp(ifn func(a, b int64) Value) opFn {
	return func(lVal, rVal Value) (Value, error) {
		switch lVal.(type) {
		case *Integer:
			switch rVal.(type) {
			case *Integer:
				val := ifn(lVal.Val().(int64), rVal.Val().(int64))
				return val, nil
			case *Float:
				val := ifn(lVal.Val().(int64), int64(rVal.Val().(float64)))
				return val, nil
			}
		case *Float:
			switch rVal.(type) {
			case *Integer:
				val := ifn(int64(lVal.Val().(float64)), rVal.Val().(int64))
				return val, nil
			case *Float:
				val := ifn(int64(lVal.Val().(float64)), int64(rVal.Val().(float64)))
				return val, nil
			}
		}
		return nil, vm.err("cannot <> %v and %v", lVal.Type(), rVal.Type())
	}
}

func eq(lVal, rVal Value) (bool, error) {
	typeA, typeB := lVal.Type(), rVal.Type()
	if typeA != typeB {
		return false, nil
	}

	switch typeA {
	case "string":
		strA, strB := lVal.(*String), rVal.(*String)
		return strA.val == strB.val, nil
	case "number":
		vA, vB := toFloat(lVal), toFloat(rVal)
		return vA == vB, nil
	case "boolean":
		strA, strB := lVal.(*Boolean), rVal.(*Boolean)
		return strA.val == strB.val, nil
	case "table", "function", "closure":
		//TODO
		fallthrough
	default:
		return false, fmt.Errorf("cannot eq %v right now", typeA)
	}
}

func compare(lVal, rVal Value) (int, error) {
	typeA, typeB := lVal.Type(), rVal.Type()
	if typeA != typeB {
		return 0, fmt.Errorf("attempt to compare %v with %v", typeA, typeB)
	}

	switch typeA {
	case "string":
		strA, strB := lVal.(*String), rVal.(*String)
		return strings.Compare(strA.val, strB.val), nil
	case "number":
		vA, vB := toFloat(lVal), toFloat(rVal)
		if vA < vB {
			return -1, nil
		} else if vA > vB {
			return 1, nil
		}
		return 0, nil
	default:
		return 0, fmt.Errorf("attempted to compare two %v values", typeA)
	}
}

func toFloat(val Value) float64 {
	switch tval := val.(type) {
	case *Integer:
		return float64(tval.val)
	case *Float:
		return tval.val
	default:
		return math.NaN()
	}
}
