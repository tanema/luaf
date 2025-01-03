package luaf

type (
	expression interface{ discharge(*FnProto, uint8) }
	exConstant struct {
		LineInfo
		index uint16
	}
	exInteger struct {
		LineInfo
		val int16
	}
	exFloat struct {
		LineInfo
		val int16
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
		lvar                        *local
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

func (ex *exInteger) discharge(fn *FnProto, dst uint8) {
	fn.code(iAsBx(LOADI, dst, ex.val), ex.LineInfo)
}

func (ex *exFloat) discharge(fn *FnProto, dst uint8) {
	fn.code(iAsBx(LOADF, dst, ex.val), ex.LineInfo)
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
