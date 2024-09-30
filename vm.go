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
		Stack []Value
		env   *Table
	}
	RuntimeErr struct {
		msg string
	}
)

func (err *RuntimeErr) Error() string {
	return err.msg
}

func NewVM() *VM {
	env := NewTable()
	return &VM{
		Stack: []Value{env},
		base:  1,
		env:   env,
	}
}

func (vm *VM) err(tmpl string, args ...any) error {
	return &RuntimeErr{msg: fmt.Sprintf(tmpl, args...)}
}

func (vm *VM) Env() *Table {
	return vm.env
}

func (vm *VM) Eval(fn *FuncProto) error {
	return vm.eval(fn, []Value{vm.env})
}

func (vm *VM) eval(fn *FuncProto, upvals []Value) error {
	xargs := vm.truncate(vm.base + int64(fn.Arity))
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
			val, err := fn.getConst(int(b))
			if err != nil {
				return err
			}
			err = vm.SetStack(a, val)
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
			a, b := instruction.ABx()
			for i := a; i < a+b; i++ {
				if err = vm.SetStack(i, &Nil{}); err != nil {
					return err
				}
			}
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
			err = vm.SetStack(a, &String{val: strBuilder.String()})
		case JMP: // TODO if A is not 0 then upvalues need to be closed
			_, b := instruction.AsBx()
			vm.pc += b
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
				err = vm.SetStack(a, &Boolean{val: actual})
			}
		case LEN:
			a, b, _ := instruction.ABC()
			val := vm.GetStack(b)
			switch tval := val.(type) {
			case *String:
				err = vm.SetStack(a, &Integer{val: int64(len(tval.val))})
			case *Table:
				err = vm.SetStack(a, &Integer{val: int64(len(tval.val))})
			default:
				err = fmt.Errorf("attempt to get length of a %v value", val.Type())
			}
		case NEWTABLE:
			dst, arraySize, hashSize := instruction.ABC()
			err = vm.SetStack(dst, NewSizedTable(int(arraySize), int(hashSize)))
		case GETTABLE: // todo allow using CONST
			dst, tblIdx, keyIdx := instruction.ABC()
			tblval := vm.GetStack(tblIdx)
			tbl, ok := tblval.(*Table)
			if !ok {
				return fmt.Errorf("attempt to index a %v value", tblval.Type())
			}
			val, err := tbl.Index(vm.GetStack(keyIdx))
			if err != nil {
				return err
			}
			err = vm.SetStack(dst, val)
		case SETTABLE: // todo allow using CONST
			tblIdx, keyIdx, valueIdx := instruction.ABC()
			tblval := vm.GetStack(tblIdx)
			tbl, ok := tblval.(*Table)
			if !ok {
				return fmt.Errorf("attempt to index a %v value", tblval.Type())
			}
			err = tbl.SetIndex(vm.GetStack(keyIdx), vm.GetStack(valueIdx))
		case VARARG:
			a, want := instruction.ABx()
			vm.truncate(a)
			if diff := int(want) - len(xargs); diff > 0 {
				for i := 0; i <= diff; i++ {
					xargs = append(xargs, &Nil{})
				}
			} else if int(want) < len(xargs) && want != 0 {
				xargs = xargs[:want]
			}
			vm.Stack = append(vm.Stack, xargs...)
		case GETUPVAL:
			a, b, _ := instruction.ABC()
			err = vm.SetStack(a, upvals[b])
		case SETUPVAL:
			a, b, _ := instruction.ABC()
			upvals[b] = vm.GetStack(a)
		case GETTABUP:
			a, b, c := instruction.ABC()
			upval := upvals[b]
			key := vm.GetStack(c)
			tbl, ok := upval.(*Table)
			if !ok {
				return fmt.Errorf("cannot index upvalue type %v", upval.Type())
			}
			val, err := tbl.Index(key)
			if err != nil {
				return err
			}
			err = vm.SetStack(a, val)
		case SETTABUP:
			a, b, c := instruction.ABC()
			upval := upvals[a]
			key, err := fn.getConst(int(b))
			if err != nil {
				return err
			}
			tbl, ok := upval.(*Table)
			if !ok {
				return fmt.Errorf("cannot index upvalue type %v", upval.Type())
			}
			err = tbl.SetIndex(key, vm.GetStack(c))
		case CALL:
			// a register of loaded fn
			// b = 0 : B = ‘top’, the function parameters range from R(A+1) to the top of the stack. This form is used when the number of parameters to pass is set by the previous VM instruction, which has to be one of OP_CALL or OP_VARARG
			//     1 : no parameters
			//  >= 2 : there are (B-1) parameters and upon entry to the called function, R(A+1) will become the base
			// c = 0 : ‘top’ is set to last_result+1, so that the next open instruction (OP_CALL, OP_RETURN, OP_SETLIST) can use ‘top’
			//   = 1 : no return results
			//  >= 2 : (C-1) return values
		case SELF:
		case CLOSURE:
		case TAILCALL:
		case RETURN:
		case FORLOOP:
		case FORPREP:
		case TFORLOOP:
		case TFORCALL:
		case SETLIST:
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
	if int(vm.base+id) >= len(vm.Stack) || id < 0 || vm.Stack[vm.base+id] == nil {
		return &Nil{}
	}
	return vm.Stack[vm.base+id]
}

func (vm *VM) SetStack(id int64, val Value) error {
	dst := vm.base + id
	if int(dst) >= len(vm.Stack) {
		newStack := make([]Value, 2*len(vm.Stack)+1)
		copy(newStack, vm.Stack)
		vm.Stack = newStack
	} else if id < 0 {
		return errors.New("cannot address negatively in the stack")
	}
	vm.Stack[dst] = val
	return nil
}

func (vm *VM) truncate(dst int64) []Value {
	vm.fillStackNil(int(dst))
	out := vm.Stack[dst:]
	vm.Stack = vm.Stack[:dst]
	return out
}

func (vm *VM) fillStackNil(dst int) {
	idx := vm.base + int64(dst)
	if diff := idx - int64(len(vm.Stack)-1); diff > 0 {
		for i := 0; i < int(diff); i++ {
			vm.Stack = append(vm.Stack, &Nil{})
		}
	}
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
