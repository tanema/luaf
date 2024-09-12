package shine

type VM struct {
	pc int64
	stack [256]Value
}

func (vm *VM) Eval(res *ParseResult) error {
	fn := res.Blocks[0]
	for {
		if len(fn.Bytecodes) <= vm.pc {
			return nil
		}
		instruction := fn.ByteCodes[vm.pc]
		switch instruction.Op() {
		case MOVE:
   a, b, _ := instruction.ABC()
			stack[a] = stack[b]	
		case LOADK:
			a, b := instruction.ABx()
			stack[a] = res.Constants[b]
		case LOADBOOL:
			a, b, _ := instruction.ABC()
			stack[a] = Bool{val: b==1}
		case LOADNIL:
			a, _, _ := instruction.ABC()
			stack[a] = Nil{}
		case GETUPVAL:
		case GETTABUP:
		case GETTABLE:
		case SETTABUP:
		case SETUPVAL:
		case SETTABLE:
		case NEWTABLE:
		case SELF:
		case ADD:
		case SUB:
		case MUL:
		case MOD:
		case POW:
		case DIV:
		case IDIV:
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
		vm.pc++
	}
}
