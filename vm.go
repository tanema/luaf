package shine

import (
	"fmt"
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
			vm.pc++
		case LOADK:
			a, b := instruction.ABx()
			vm.stack[vm.base+a] = fn.Constants[b]
			vm.pc++
		case LOADBOOL:
			a, b, _ := instruction.ABC()
			vm.stack[vm.base+a] = &Boolean{val: b == 1}
			vm.pc++
		case LOADINT:
			a, b := instruction.AsBx()
			vm.stack[vm.base+a] = &Integer{val: b}
			vm.pc++
		case LOADNIL:
			a, _, _ := instruction.ABC()
			vm.stack[vm.base+a] = &Nil{}
			vm.pc++
		case GETUPVAL:
		case GETTABUP:
		case GETTABLE:
		case SETTABUP:
		case SETUPVAL:
		case SETTABLE:
		case NEWTABLE:
		case SELF:
		case ADD:
			a, b, c := instruction.ABC()
			lVal, rVal := vm.stack[vm.base+b], vm.stack[vm.base+c]
			switch lVal.(type) {
			case *Integer:
				switch rVal.(type) {
				case *Integer:
					vm.stack[vm.base+a] = &Integer{val: lVal.Val().(int64) + rVal.Val().(int64)}
					vm.pc++
					continue
				case *Float:
					vm.stack[vm.base+a] = &Float{val: float64(lVal.Val().(int64)) + rVal.Val().(float64)}
				default:
					return vm.err("cannot add %v and %v", lVal.Type(), rVal.Type())
				}
			case *Float:
				switch rVal.(type) {
				case *Integer:
					vm.stack[vm.base+a] = &Float{val: lVal.Val().(float64) + float64(rVal.Val().(int64))}
				case *Float:
					vm.stack[vm.base+a] = &Float{val: lVal.Val().(float64) + rVal.Val().(float64)}
				default:
					return vm.err("cannot add %v and %v", lVal.Type(), rVal.Type())
				}
			default:
				return vm.err("cannot add %v and %v", lVal.Type(), rVal.Type())
			}
			vm.pc++
		case SUB:
			a, b, c := instruction.ABC()
			lVal, rVal := vm.stack[vm.base+b], vm.stack[vm.base+c]
			switch lVal.(type) {
			case *Integer:
				switch rVal.(type) {
				case *Integer:
					vm.stack[vm.base+a] = &Integer{val: lVal.Val().(int64) - rVal.Val().(int64)}
					vm.pc++
					continue
				case *Float:
					vm.stack[vm.base+a] = &Float{val: float64(lVal.Val().(int64)) - rVal.Val().(float64)}
				default:
					return vm.err("cannot subtract %v and %v", lVal.Type(), rVal.Type())
				}
			case *Float:
				switch rVal.(type) {
				case *Integer:
					vm.stack[vm.base+a] = &Float{val: lVal.Val().(float64) - float64(rVal.Val().(int64))}
				case *Float:
					vm.stack[vm.base+a] = &Float{val: lVal.Val().(float64) - rVal.Val().(float64)}
				default:
					return vm.err("cannot subtract %v and %v", lVal.Type(), rVal.Type())
				}
			default:
				return vm.err("cannot subtract %v and %v", lVal.Type(), rVal.Type())
			}
			vm.pc++
		case MUL:
			a, b, c := instruction.ABC()
			lVal, rVal := vm.stack[vm.base+b], vm.stack[vm.base+c]
			switch lVal.(type) {
			case *Integer:
				switch rVal.(type) {
				case *Integer:
					vm.stack[vm.base+a] = &Integer{val: lVal.Val().(int64) * rVal.Val().(int64)}
					vm.pc++
					continue
				case *Float:
					vm.stack[vm.base+a] = &Float{val: float64(lVal.Val().(int64)) * rVal.Val().(float64)}
				default:
					return vm.err("cannot multiply %v and %v", lVal.Type(), rVal.Type())
				}
			case *Float:
				switch rVal.(type) {
				case *Integer:
					vm.stack[vm.base+a] = &Float{val: lVal.Val().(float64) * float64(rVal.Val().(int64))}
				case *Float:
					vm.stack[vm.base+a] = &Float{val: lVal.Val().(float64) * rVal.Val().(float64)}
				default:
					return vm.err("cannot multiply %v and %v", lVal.Type(), rVal.Type())
				}
			default:
				return vm.err("cannot multiply %v and %v", lVal.Type(), rVal.Type())
			}
			vm.pc++
		case DIV:
			// only floats
		case MOD:
			// only int
		case POW:
			// float only
		case IDIV:
			// float
		case BAND:
		case BOR:
		case BXOR:
		case SHL:
		case SHR:
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
	}
}
