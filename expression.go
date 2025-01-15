package luaf

import "math"

type (
	expression interface{ discharge(*FnProto, uint8) error }
	exString   struct {
		LineInfo
		val string
	}
	exInteger struct {
		LineInfo
		val int64
	}
	exFloat struct {
		LineInfo
		val float64
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
	exInfixOp struct {
		LineInfo
		operand     TokenType
		left, right expression
	}
	exUnaryOp struct {
		LineInfo
		op  BytecodeOp
		val uint8
	}
)

func (ex *exString) discharge(fn *FnProto, dst uint8) error {
	kaddr, err := fn.addConst(ex.val)
	if err != nil {
		return err
	}
	fn.code(iABx(LOADK, dst, kaddr), ex.LineInfo)
	return nil
}

func (ex *exInteger) discharge(fn *FnProto, dst uint8) error {
	if ex.val > math.MinInt16 && ex.val < math.MaxInt16-1 {
		fn.code(iAsBx(LOADI, dst, int16(ex.val)), ex.LineInfo)
		return nil
	}
	kaddr, err := fn.addConst(ex.val)
	if err != nil {
		return err
	}
	fn.code(iABx(LOADK, dst, kaddr), ex.LineInfo)
	return nil
}

func (ex *exFloat) discharge(fn *FnProto, dst uint8) error {
	if ex.val == math.Trunc(ex.val) && (ex.val > math.MinInt16 && ex.val < math.MaxInt16-1) {
		fn.code(iAsBx(LOADF, dst, int16(ex.val)), ex.LineInfo)
	}
	kaddr, err := fn.addConst(ex.val)
	if err != nil {
		return err
	}
	fn.code(iABx(LOADK, dst, kaddr), ex.LineInfo)
	return nil
}

func (ex *exNil) discharge(fn *FnProto, dst uint8) error {
	fn.code(iABx(LOADNIL, dst, ex.num), ex.LineInfo)
	return nil
}

func (ex *exClosure) discharge(fn *FnProto, dst uint8) error {
	fn.code(iABx(CLOSURE, dst, ex.fn), ex.LineInfo)
	return nil
}

func (ex *exCall) discharge(fn *FnProto, dst uint8) error {
	fn.code(iABC(CALL, ex.fn, ex.nargs, ex.nret), ex.LineInfo)
	return nil
}

func (ex *exVarArgs) discharge(fn *FnProto, dst uint8) error {
	fn.code(iAB(VARARG, dst, ex.want), ex.LineInfo)
	return nil
}

func (ex *exUnaryOp) discharge(fn *FnProto, dst uint8) error {
	fn.code(iAB(ex.op, dst, ex.val), ex.LineInfo)
	return nil
}

func (ex *exBool) discharge(fn *FnProto, dst uint8) error {
	fn.code(iABC(LOADBOOL, dst, b2U8(ex.val), b2U8(ex.skip)), ex.LineInfo)
	return nil
}

func (ex *exValue) discharge(fn *FnProto, dst uint8) error {
	if !ex.local {
		fn.code(iAB(GETUPVAL, dst, ex.address), ex.LineInfo)
	} else if uint8(dst) != ex.address { // already there
		fn.code(iAB(MOVE, dst, ex.address), ex.LineInfo)
	}
	return nil
}

func (ex *exIndex) discharge(fn *FnProto, dst uint8) error {
	if ex.local {
		fn.code(iABCK(GETTABLE, dst, ex.table, false, ex.key, ex.keyIsConst), ex.LineInfo)
	} else {
		fn.code(iABCK(GETTABUP, dst, ex.table, false, ex.key, ex.keyIsConst), ex.LineInfo)
	}
	return nil
}

func (ex *exInfixOp) discharge(fn *FnProto, dst uint8) error {
	if ex.operand == TokenGt {
		ex.operand = TokenLt
		ex.left, ex.right = ex.right, ex.left
	} else if ex.operand == TokenGe {
		ex.operand = TokenLe
		ex.left, ex.right = ex.right, ex.left
	}

	lval, rval := dst, dst+1
	if err := ex.left.discharge(fn, lval); err != nil {
		return err
	} else if err := ex.right.discharge(fn, rval); err != nil {
		return err
	}

	switch ex.operand {
	case TokenBitwiseOr, TokenBitwiseNotOrXOr, TokenBitwiseAnd, TokenShiftLeft, TokenShiftRight,
		TokenModulo, TokenDivide, TokenFloorDivide, TokenExponent:
		fn.code(iABC(tokenToBytecodeOp[ex.operand], dst, lval, rval), ex.LineInfo)
	case TokenAdd, TokenMultiply: // communicative operations
		fn.code(iABC(tokenToBytecodeOp[ex.operand], dst, lval, rval), ex.LineInfo)
	case TokenConcat: // merge concat operations
		fn.code(iABC(tokenToBytecodeOp[ex.operand], dst, lval, rval), ex.LineInfo)
	case TokenLt, TokenLe, TokenEq:
		fn.code(iABC(tokenToBytecodeOp[ex.operand], 0, lval, rval), ex.LineInfo) // if false skip next
		fn.code(iABx(JMP, 0, 1), ex.LineInfo)                                    // jump to set false
		fn.code(iABC(LOADBOOL, dst, 1, 1), ex.LineInfo)                          // set true then skip next
		fn.code(iABC(LOADBOOL, dst, 0, 0), ex.LineInfo)                          // set false don't skip next
	case TokenNe:
		fn.code(iABC(EQ, 1, lval, rval), ex.LineInfo)   // if false skip next
		fn.code(iABx(JMP, 0, 1), ex.LineInfo)           // jump to set false
		fn.code(iABC(LOADBOOL, dst, 1, 1), ex.LineInfo) // set true then skip next
		fn.code(iABC(LOADBOOL, dst, 0, 0), ex.LineInfo) // set false don't skip next
	case TokenAnd:
		fn.code(iAB(TEST, lval, 0), ex.LineInfo)          // if lval true skip next
		fn.code(iABx(JMP, 0, 1), ex.LineInfo)             // lval was false, short circuit jump to end
		fn.code(iABC(TESTSET, dst, rval, 0), ex.LineInfo) // if rval true set true
		fn.code(iABC(LOADBOOL, dst, 0, 0), ex.LineInfo)   // any were false set false
	case TokenOr:
		fn.code(iAB(TEST, lval, 1), ex.LineInfo)          // if lval true short circuit jump to end
		fn.code(iABx(JMP, 0, 1), ex.LineInfo)             // lval was true, short circuit jump to end
		fn.code(iABC(TESTSET, dst, rval, 1), ex.LineInfo) // if rval false return false
		fn.code(iABC(LOADBOOL, dst, 1, 0), ex.LineInfo)   // any were true set true
	default:
		panic("unknown binop")
	}
	return nil
}

func constFold(ex *exInfixOp) expression {
	if ex.operand == TokenConcat {
		return ex
	} else if exIsNum(ex.left) && exIsNum(ex.right) {
		return ex.foldConstArith()
	}
	return ex
}

func (ex *exInfixOp) foldConstArith() expression {
	op := tokenToMetaMethod[ex.operand]
	switch op {
	case metaBAnd, metaBOr, metaBXOr, metaShl, metaShr:
		return &exInteger{val: intArith(op, exToInt(ex.left), exToInt(ex.right)), LineInfo: ex.LineInfo}
	case metaDiv, metaPow:
		return &exFloat{val: floatArith(op, exToFloat(ex.left), exToFloat(ex.right)), LineInfo: ex.LineInfo}
	default:
		liva, lisInt := ex.left.(*exInteger)
		riva, risInt := ex.right.(*exInteger)
		if lisInt && risInt {
			return &exInteger{val: intArith(op, liva.val, riva.val), LineInfo: ex.LineInfo}
		}
		return &exFloat{val: floatArith(op, exToFloat(ex.left), exToFloat(ex.right)), LineInfo: ex.LineInfo}
	}
}

func exIsNum(ex expression) bool {
	switch ex.(type) {
	case *exInteger, *exFloat:
		return true
	}
	return false
}

func exToInt(ex expression) int64 {
	switch tex := ex.(type) {
	case *exInteger:
		return tex.val
	case *exFloat:
		return int64(tex.val)
	default:
		panic("tried to cast non number expression to int")
	}
}

func exToFloat(ex expression) float64 {
	switch tex := ex.(type) {
	case *exInteger:
		return float64(tex.val)
	case *exFloat:
		return tex.val
	default:
		panic("tried to cast non number expression to float")
	}
}

func exIsConst(expr expression) (any, bool) {
	switch ex := expr.(type) {
	case *exString:
		return ex.val, true
	case *exFloat:
		return ex.val, true
	case *exInteger:
		return ex.val, true
	default:
		return nil, false
	}
}
