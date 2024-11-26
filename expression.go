package luaf

type (
	expression interface{ discharge(*FnProto, uint8) }
	exConstant struct {
		LineInfo
		index uint16
	}
	exNil struct {
		LineInfo
		num uint16
	}
	exBool struct {
		LineInfo
		val, skip bool
	}
	exValue struct { // upvalue or local
		LineInfo
		name                        string
		local, attrConst, attrClose bool
		address                     uint8
		lvar                        *Local
	}
	exIndex struct {
		LineInfo
		local      bool
		table, key uint8
		keyIsConst bool
	}
	exClosure struct {
		LineInfo
		fn uint16
	}
	exCall struct {
		LineInfo
		fn, nargs, nret uint8
	}
	exVarArgs struct {
		LineInfo
		want uint8
	}
	exBinOp struct {
		LineInfo
		op         BytecodeOp
		lval, rval uint8
	}
	exAnd struct {
		LineInfo
		lval, rval uint8
	}
	exOr struct {
		LineInfo
		lval, rval uint8
	}
	exBoolBinOp struct {
		LineInfo
		op                   BytecodeOp
		expected, lval, rval uint8
	}
	exUnaryOp struct {
		LineInfo
		op  BytecodeOp
		val uint8
	}
)

func (ex *exConstant) discharge(fn *FnProto, dst uint8) {
	fn.code(iABx(LOADK, dst, ex.index), ex.LineInfo)
}
func (ex *exNil) discharge(fn *FnProto, dst uint8) {
	fn.code(iABx(LOADNIL, dst, ex.num), ex.LineInfo)
}
func (ex *exClosure) discharge(fn *FnProto, dst uint8) {
	fn.code(iABx(CLOSURE, dst, ex.fn), ex.LineInfo)
}
func (ex *exCall) discharge(fn *FnProto, dst uint8) {
	fn.code(iABC(CALL, ex.fn, ex.nargs, ex.nret), ex.LineInfo)
}
func (ex *exVarArgs) discharge(fn *FnProto, dst uint8) {
	fn.code(iAB(VARARG, dst, ex.want), ex.LineInfo)
}
func (ex *exBinOp) discharge(fn *FnProto, dst uint8) {
	fn.code(iABC(ex.op, dst, ex.lval, ex.rval), ex.LineInfo)
}
func (ex *exUnaryOp) discharge(fn *FnProto, dst uint8) {
	fn.code(iAB(ex.op, dst, ex.val), ex.LineInfo)
}
func (ex *exBool) discharge(fn *FnProto, dst uint8) {
	fn.code(iABC(LOADBOOL, dst, b2U8(ex.val), b2U8(ex.skip)), ex.LineInfo)
}

func (ex *exValue) discharge(fn *FnProto, dst uint8) {
	if !ex.local {
		fn.code(iAB(GETUPVAL, dst, ex.address), ex.LineInfo)
	} else if uint8(dst) != ex.address { // already there
		fn.code(iAB(MOVE, dst, ex.address), ex.LineInfo)
	}
}

func (ex *exIndex) discharge(fn *FnProto, dst uint8) {
	if ex.local {
		fn.code(iABCK(GETTABLE, dst, ex.table, false, ex.key, ex.keyIsConst), ex.LineInfo)
	} else {
		fn.code(iABCK(GETTABUP, dst, ex.table, false, ex.key, ex.keyIsConst), ex.LineInfo)
	}
}

func (ex *exBoolBinOp) discharge(fn *FnProto, dst uint8) {
	fn.code(iABC(ex.op, ex.expected, ex.lval, ex.rval), ex.LineInfo) // if false skip next
	fn.code(iABx(JMP, 0, 1), ex.LineInfo)                            // jump to set false
	fn.code(iABC(LOADBOOL, dst, 1, 1), ex.LineInfo)                  // set true then skip next
	fn.code(iABC(LOADBOOL, dst, 0, 0), ex.LineInfo)                  // set false don't skip next
}

func (ex *exAnd) discharge(fn *FnProto, dst uint8) {
	fn.code(iAB(TEST, ex.lval, 0), ex.LineInfo)          // if lval true skip next
	fn.code(iABx(JMP, 0, 1), ex.LineInfo)                // lval was false, short circuit jump to end
	fn.code(iABC(TESTSET, dst, ex.rval, 0), ex.LineInfo) // if rval true set true
	fn.code(iABC(LOADBOOL, dst, 0, 0), ex.LineInfo)      // any were false set false
}

func (ex *exOr) discharge(fn *FnProto, dst uint8) {
	fn.code(iAB(TEST, ex.lval, 1), ex.LineInfo)          // if lval true short circuit jump to end
	fn.code(iABx(JMP, 0, 1), ex.LineInfo)                // lval was true, short circuit jump to end
	fn.code(iABC(TESTSET, dst, ex.rval, 1), ex.LineInfo) // if rval false return false
	fn.code(iABC(LOADBOOL, dst, 1, 0), ex.LineInfo)      // any were true set true
}

func tokenToBinopExpression(tk *Token, lval, rval uint8) expression {
	switch tk.Kind {
	case TokenBitwiseOr:
		return &exBinOp{op: BOR, lval: lval, rval: rval, LineInfo: tk.LineInfo}
	case TokenBitwiseNotOrXOr:
		return &exBinOp{op: BXOR, lval: lval, rval: rval, LineInfo: tk.LineInfo}
	case TokenBitwiseAnd:
		return &exBinOp{op: BAND, lval: lval, rval: rval, LineInfo: tk.LineInfo}
	case TokenShiftLeft:
		return &exBinOp{op: SHL, lval: lval, rval: rval, LineInfo: tk.LineInfo}
	case TokenShiftRight:
		return &exBinOp{op: SHR, lval: lval, rval: rval, LineInfo: tk.LineInfo}
	case TokenConcat:
		return &exBinOp{op: CONCAT, lval: lval, rval: rval, LineInfo: tk.LineInfo}
	case TokenAdd:
		return &exBinOp{op: ADD, lval: lval, rval: rval, LineInfo: tk.LineInfo}
	case TokenMinus:
		return &exBinOp{op: SUB, lval: lval, rval: rval, LineInfo: tk.LineInfo}
	case TokenMultiply:
		return &exBinOp{op: MUL, lval: lval, rval: rval, LineInfo: tk.LineInfo}
	case TokenModulo:
		return &exBinOp{op: MOD, lval: lval, rval: rval, LineInfo: tk.LineInfo}
	case TokenDivide:
		return &exBinOp{op: DIV, lval: lval, rval: rval, LineInfo: tk.LineInfo}
	case TokenFloorDivide:
		return &exBinOp{op: IDIV, lval: lval, rval: rval, LineInfo: tk.LineInfo}
	case TokenExponent:
		return &exBinOp{op: POW, lval: lval, rval: rval, LineInfo: tk.LineInfo}
	case TokenLt:
		return &exBoolBinOp{op: LT, expected: 0, lval: lval, rval: rval, LineInfo: tk.LineInfo}
	case TokenLe:
		return &exBoolBinOp{op: LE, expected: 0, lval: lval, rval: rval, LineInfo: tk.LineInfo}
	case TokenGt:
		return &exBoolBinOp{op: LT, expected: 0, lval: rval, rval: lval, LineInfo: tk.LineInfo}
	case TokenGe:
		return &exBoolBinOp{op: LE, expected: 0, lval: rval, rval: lval, LineInfo: tk.LineInfo}
	case TokenEq:
		return &exBoolBinOp{op: EQ, expected: 0, lval: lval, rval: rval, LineInfo: tk.LineInfo}
	case TokenNe:
		return &exBoolBinOp{op: EQ, expected: 1, lval: lval, rval: rval, LineInfo: tk.LineInfo}
	case TokenAnd:
		return &exAnd{lval: lval, rval: rval, LineInfo: tk.LineInfo}
	case TokenOr:
		return &exOr{lval: lval, rval: rval, LineInfo: tk.LineInfo}
	default:
		panic("unknown binop")
	}
}

func tokenToUnary(tk *Token, val uint8) expression {
	switch tk.Kind {
	case TokenNot:
		return &exUnaryOp{op: NOT, val: val, LineInfo: tk.LineInfo}
	case TokenLength:
		return &exUnaryOp{op: LEN, val: val, LineInfo: tk.LineInfo}
	case TokenMinus:
		return &exUnaryOp{op: UNM, val: val, LineInfo: tk.LineInfo}
	case TokenBitwiseNotOrXOr:
		return &exUnaryOp{op: BNOT, val: val, LineInfo: tk.LineInfo}
	default:
		panic("unknown unary")
	}
}
