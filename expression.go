package shine

type (
	expression interface{ discharge(*FuncProto, uint8) }
	assignable interface {
		assignTo(*FuncProto, uint8, bool) error
	}
	exConstant struct{ index uint16 }
	exNil      struct{ num uint16 }
	exBool     struct{ val, skip bool }
	exValue    struct { // upvalue or local
		name                        string
		local, attrConst, attrClose bool
		address                     uint8
	}
	exIndex struct {
		local                    bool
		table, key, value        uint8
		keyIsConst, valueIsConst bool
	}
	exClosure struct{ fn uint16 }
	exCall    struct{ fn, nargs, nret uint8 }
	exVarArgs struct{ limit, want uint8 }
	exBinOp   struct {
		op         BytecodeOp
		lval, rval uint8
	}
	exAnd       struct{ lval, rval uint8 }
	exOr        struct{ lval, rval uint8 }
	exBoolBinOp struct {
		op                   BytecodeOp
		expected, lval, rval uint8
	}
	exUnaryOp struct {
		op  BytecodeOp
		val uint8
	}
)

func (ex *exConstant) discharge(fn *FuncProto, dst uint8) { fn.code(iABx(LOADK, dst, ex.index)) }
func (ex *exNil) discharge(fn *FuncProto, dst uint8)      { fn.code(iABx(LOADNIL, dst, ex.num)) }
func (ex *exClosure) discharge(fn *FuncProto, dst uint8)  { fn.code(iABx(CLOSURE, dst, ex.fn)) }
func (ex *exCall) discharge(fn *FuncProto, dst uint8)     { fn.code(iABC(CALL, ex.fn, ex.nargs, ex.nret)) }
func (ex *exVarArgs) discharge(fn *FuncProto, dst uint8)  { fn.code(iAB(CALL, ex.limit, ex.want)) }
func (ex *exBinOp) discharge(fn *FuncProto, dst uint8)    { fn.code(iABC(ex.op, dst, ex.lval, ex.rval)) }
func (ex *exUnaryOp) discharge(fn *FuncProto, dst uint8)  { fn.code(iAB(ex.op, dst, ex.val)) }
func (ex *exBool) discharge(fn *FuncProto, dst uint8) {
	fn.code(iABC(LOADBOOL, dst, b2U8(ex.val), b2U8(ex.skip)))
}

func (ex *exValue) discharge(fn *FuncProto, dst uint8) {
	if !ex.local {
		fn.code(iAB(GETUPVAL, dst, ex.address))
	} else if uint8(dst) != ex.address { // already there
		fn.code(iAB(MOVE, dst, ex.address))
	}
}

func (ex *exValue) assignTo(fn *FuncProto, from uint8, fromIsConst bool) error {
	if !ex.local {
		fn.code(iABCK(SETUPVAL, ex.address, from, fromIsConst, 0, false))
	} else if from != ex.address {
		fn.code(iABCK(MOVE, ex.address, from, fromIsConst, 0, false))
	}
	return nil
}

func (ex *exIndex) discharge(fn *FuncProto, dst uint8) {
	if ex.local {
		fn.code(iABCK(GETTABLE, dst, ex.table, false, ex.key, ex.keyIsConst))
	} else {
		fn.code(iABCK(GETTABUP, dst, ex.table, false, ex.key, ex.keyIsConst))
	}
}

func (ex *exIndex) assignTo(fn *FuncProto, from uint8, fromIsConst bool) error {
	if ex.local {
		fn.code(iABCK(SETTABLE, ex.table, ex.key, ex.keyIsConst, from, fromIsConst))
	} else {
		fn.code(iABCK(SETTABUP, ex.table, ex.key, ex.keyIsConst, from, fromIsConst))
	}
	return nil
}

func (ex *exBoolBinOp) discharge(fn *FuncProto, dst uint8) {
	fn.code(iABC(ex.op, ex.expected, ex.lval, ex.rval)) // if false skip next
	fn.code(iABx(JMP, 0, 1))                            // jump to set false
	fn.code(iABC(LOADBOOL, dst, 1, 1))                  // set true then skip next
	fn.code(iABC(LOADBOOL, dst, 0, 0))                  // set false don't skip next
}

func (ex *exAnd) discharge(fn *FuncProto, dst uint8) {
	fn.code(iAB(TEST, ex.lval, 0))          // if lval true skip next
	fn.code(iABx(JMP, 0, 1))                // lval was false, short circuit jump to end
	fn.code(iABC(TESTSET, dst, ex.rval, 0)) // if rval true set true
	fn.code(iABC(LOADBOOL, dst, 0, 0))      // any were false set false
}

func (ex *exOr) discharge(fn *FuncProto, dst uint8) {
	fn.code(iAB(TEST, ex.lval, 1))          // if lval true short circuit jump to end
	fn.code(iABx(JMP, 0, 1))                // lval was true, short circuit jump to end
	fn.code(iABC(TESTSET, dst, ex.rval, 1)) // if rval false return false
	fn.code(iABC(LOADBOOL, dst, 1, 0))      // any were true set true
}

func tokenToBinopExpression(kind TokenType, lval, rval uint8) expression {
	switch kind {
	case TokenBitwiseOr:
		return &exBinOp{op: BOR, lval: lval, rval: rval}
	case TokenBitwiseNotOrXOr:
		return &exBinOp{op: BXOR, lval: lval, rval: rval}
	case TokenBitwiseAnd:
		return &exBinOp{op: BAND, lval: lval, rval: rval}
	case TokenShiftLeft:
		return &exBinOp{op: SHL, lval: lval, rval: rval}
	case TokenShiftRight:
		return &exBinOp{op: SHR, lval: lval, rval: rval}
	case TokenConcat:
		return &exBinOp{op: CONCAT, lval: lval, rval: rval}
	case TokenAdd:
		return &exBinOp{op: ADD, lval: lval, rval: rval}
	case TokenMinus:
		return &exBinOp{op: SUB, lval: lval, rval: rval}
	case TokenMultiply:
		return &exBinOp{op: MUL, lval: lval, rval: rval}
	case TokenModulo:
		return &exBinOp{op: MOD, lval: lval, rval: rval}
	case TokenDivide:
		return &exBinOp{op: DIV, lval: lval, rval: rval}
	case TokenFloorDivide:
		return &exBinOp{op: IDIV, lval: lval, rval: rval}
	case TokenExponent:
		return &exBinOp{op: POW, lval: lval, rval: rval}
	case TokenLt:
		return &exBoolBinOp{op: LT, expected: 1, lval: lval, rval: rval}
	case TokenLe:
		return &exBoolBinOp{op: LE, expected: 1, lval: lval, rval: rval}
	case TokenGt:
		return &exBoolBinOp{op: LT, expected: 1, lval: rval, rval: lval}
	case TokenGe:
		return &exBoolBinOp{op: LE, expected: 1, lval: rval, rval: lval}
	case TokenEq:
		return &exBoolBinOp{op: EQ, expected: 1, lval: lval, rval: rval}
	case TokenNe:
		return &exBoolBinOp{op: EQ, expected: 0, lval: lval, rval: rval}
	case TokenAnd:
		return &exAnd{lval: lval, rval: rval}
	case TokenOr:
		return &exOr{lval: lval, rval: rval}
	default:
		panic("unknown binop")
	}
}

func tokenToUnary(kind TokenType, val uint8) expression {
	switch kind {
	case TokenNot:
		return &exUnaryOp{op: NOT, val: val}
	case TokenLength:
		return &exUnaryOp{op: LEN, val: val}
	case TokenMinus:
		return &exUnaryOp{op: UNM, val: val}
	case TokenBitwiseNotOrXOr:
		return &exUnaryOp{op: BNOT, val: val}
	default:
		panic("unknown unary")
	}
}

func b2U8(val bool) uint8 {
	if val {
		return 1
	}
	return 0
}
