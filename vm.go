package shine

type VM struct {
	stack []Value
}

func (vm *VM) Eval(res *ParseResult) error {
	for {
		instruction := res.Blocks[0].ByteCodes[0]
		switch instruction.Op {
		case MOVE:
		case LOADK:
		case LOADBOOL:
		case LOADNIL:
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
	}
}
