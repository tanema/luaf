package shine

import (
	"fmt"
	"math"
)

type (
	VM struct {
		pc    int64
		base  int64
		stack [256]Value
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
		stack: [256]Value{},
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
		if int64(len(fn.ByteCodes)) <= vm.pc {
			return nil
		}
		instruction := fn.ByteCodes[vm.pc]
		switch instruction.Op() {
		case MOVE:
			a, b, _ := instruction.ABC()
			vm.stack[vm.base+a] = vm.stack[vm.base+b]
		case LOADK:
			a, b := instruction.ABx()
			vm.stack[vm.base+a] = fn.Constants[b]
		case LOADBOOL:
			a, b, _ := instruction.ABC()
			vm.stack[vm.base+a] = &Boolean{val: b == 1}
		case LOADINT:
			a, b := instruction.AsBx()
			vm.stack[vm.base+a] = &Integer{val: b}
			vm.pc++
		case LOADNIL:
			a, _, _ := instruction.ABC()
			vm.stack[vm.base+a] = &Nil{}
		case GETUPVAL:
		case GETTABUP:
		case GETTABLE:
		case SETTABUP:
		case SETUPVAL:
		case SETTABLE:
		case NEWTABLE:
		case SELF:
		case ADD:
			if err := vm.setBinOp(instruction,
				func(x, y int64) Value { return &Integer{val: x + y} },
				func(x, y float64) Value { return &Float{val: x + y} }); err != nil {
				return err
			}
		case SUB:
			if err := vm.setBinOp(instruction,
				func(x, y int64) Value { return &Integer{val: x - y} },
				func(x, y float64) Value { return &Float{val: x - y} }); err != nil {
				return err
			}
		case MUL:
			if err := vm.setBinOp(instruction,
				func(x, y int64) Value { return &Integer{val: x * y} },
				func(x, y float64) Value { return &Float{val: x * y} }); err != nil {
				return err
			}
		case DIV:
			if err := vm.setBinOp(instruction,
				func(x, y int64) Value { return &Integer{val: x / y} },
				func(x, y float64) Value { return &Float{val: x / y} }); err != nil {
				return err
			}
		case MOD:
			if err := vm.setBinOp(instruction,
				func(x, y int64) Value { return &Integer{val: x % y} },
				func(x, y float64) Value { return &Float{val: math.Mod(x, y)} }); err != nil {
				return err
			}
		case POW:
			if err := vm.setBinOp(instruction,
				func(x, y int64) Value { return &Integer{val: x ^ y} },
				func(x, y float64) Value { return &Float{val: math.Pow(x, y)} }); err != nil {
				return err
			}
		case IDIV:
			if err := vm.setBinOp(instruction,
				func(x, y int64) Value { return &Integer{val: x / y} },
				func(x, y float64) Value { return &Float{val: math.Floor(x / y)} }); err != nil {
				return err
			}
		case BAND:
			if err := vm.setiBinOp(instruction, func(x, y int64) Value { return &Integer{val: x & y} }); err != nil {
				return err
			}
		case BOR:
			if err := vm.setiBinOp(instruction, func(x, y int64) Value { return &Integer{val: x | y} }); err != nil {
				return err
			}
		case BXOR:
			if err := vm.setiBinOp(instruction, func(x, y int64) Value { return &Integer{val: x ^ y} }); err != nil {
				return err
			}
		case SHL:
			if err := vm.setiBinOp(instruction, func(x, y int64) Value { return &Integer{val: x << y} }); err != nil {
				return err
			}
		case SHR:
			if err := vm.setiBinOp(instruction, func(x, y int64) Value { return &Integer{val: x >> y} }); err != nil {
				return err
			}
		case UNM:
		case BNOT:
		case NOT:
		case LEN:
		case CONCAT:
		case JMP:
		case EQ:
		case LT:
		case LE:
		case TEST:
		case TESTSET:
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
		vm.pc++
	}
}

func (vm *VM) setBinOp(instruction Bytecode, ifn func(a, b int64) Value, ffn func(a, b float64) Value) error {
	a, b, c := instruction.ABC()
	lVal, rVal := vm.stack[vm.base+b], vm.stack[vm.base+c]
	val, err := vm.binOp(lVal, rVal, ifn, ffn)
	if err != nil {
		return err
	}
	vm.stack[vm.base+a] = val
	return nil
}

func (vm *VM) binOp(lVal, rVal Value, ifn func(a, b int64) Value, ffn func(a, b float64) Value) (Value, error) {
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

func (vm *VM) setiBinOp(instruction Bytecode, ifn func(a, b int64) Value) error {
	a, b, c := instruction.ABC()
	lVal, rVal := vm.stack[vm.base+b], vm.stack[vm.base+c]
	val, err := vm.ibinOp(lVal, rVal, ifn)
	if err != nil {
		return err
	}
	vm.stack[vm.base+a] = val
	return nil
}

func (vm *VM) ibinOp(lVal, rVal Value, ifn func(a, b int64) Value) (Value, error) {
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
