package luaf

import (
	"fmt"
	"math"
)

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
	exClosure struct {
		LineInfo
		fn uint16
	}
	exVariable struct { // upvalue or local
		LineInfo
		name                        string
		local, attrConst, attrClose bool
		address                     uint8
		lvar                        *local
	}
	exIndex struct {
		LineInfo
		table, key expression
	}
	exConcat struct { // upvalue or local
		LineInfo
		exprs []expression
	}
	exCall struct {
		LineInfo
		self        bool
		tail        bool
		fn          expression
		args        []expression
		nargs, nret uint8
	}
	exVarArgs struct {
		LineInfo
		want uint8
	}
	tableField struct {
		key expression
		val expression
	}
	exTable struct {
		LineInfo
		array  []expression
		fields []tableField
	}
	exInfixOp struct {
		LineInfo
		operand     TokenType
		left, right expression
	}
	exUnaryOp struct {
		LineInfo
		op  BytecodeOp
		val expression
	}
)

func newCallExpr(fn expression, args []expression, self bool, li LineInfo) *exCall {
	nargs := uint8(len(args) + 1)
	if len(args) > 0 {
		switch arg := args[len(args)-1].(type) {
		case *exCall:
			nargs = 0
			arg.nret = 0 // all out
		case *exVarArgs:
			nargs = 0
			arg.want = 0 // var args all out
		}
	}
	if self {
		nargs++
	}
	return &exCall{
		fn:       fn,
		self:     self,
		nargs:    nargs,
		nret:     2,
		args:     args,
		LineInfo: li,
	}
}

func newInfixExpr(op *Token, left, right expression) expression {
	return constFold(&exInfixOp{
		operand:  op.Kind,
		left:     left,
		right:    right,
		LineInfo: op.LineInfo,
	})
}

func (ex *exString) discharge(fn *FnProto, dst uint8) error {
	kaddr, err := fn.addConst(ex.val)
	fn.code(iABx(LOADK, dst, kaddr), ex.LineInfo)
	return err
}

func (ex *exInteger) discharge(fn *FnProto, dst uint8) error {
	if ex.val > math.MinInt16 && ex.val < math.MaxInt16-1 {
		fn.code(iAsBx(LOADI, dst, int16(ex.val)), ex.LineInfo)
		return nil
	}
	kaddr, err := fn.addConst(ex.val)
	fn.code(iABx(LOADK, dst, kaddr), ex.LineInfo)
	return err
}

func (ex *exFloat) discharge(fn *FnProto, dst uint8) error {
	if ex.val == math.Trunc(ex.val) && (ex.val > math.MinInt16 && ex.val < math.MaxInt16-1) {
		fn.code(iAsBx(LOADF, dst, int16(ex.val)), ex.LineInfo)
	}
	kaddr, err := fn.addConst(ex.val)
	fn.code(iABx(LOADK, dst, kaddr), ex.LineInfo)
	return err
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
	offset := uint8(1)
	if ex.self {
		index := ex.fn.(*exIndex)
		if err := index.table.discharge(fn, dst); err != nil {
			return err
		}
		kaddr, err := fn.addConst(index.key.(*exString).val)
		if err != nil {
			return err
		}
		fn.code(iABCK(SELF, dst, dst, false, uint8(kaddr), true), index.LineInfo)
		offset++
	} else if err := ex.fn.discharge(fn, dst); err != nil {
		return err
	}
	for i, arg := range ex.args {
		if err := arg.discharge(fn, dst+offset+uint8(i)); err != nil {
			return err
		}
	}
	if ex.tail {
		fn.code(iAB(TAILCALL, dst, ex.nargs), ex.LineInfo)
	} else {
		fn.code(iABC(CALL, dst, ex.nargs, ex.nret), ex.LineInfo)
	}
	return nil
}

func (ex *exVarArgs) discharge(fn *FnProto, dst uint8) error {
	fn.code(iAB(VARARG, dst, ex.want), ex.LineInfo)
	return nil
}

func (ex *exUnaryOp) discharge(fn *FnProto, dst uint8) error {
	if err := ex.val.discharge(fn, dst); err != nil {
		return err
	}
	fn.code(iAB(ex.op, dst, dst), ex.LineInfo)
	return nil
}

func (ex *exBool) discharge(fn *FnProto, dst uint8) error {
	fn.code(iABC(LOADBOOL, dst, b2U8(ex.val), b2U8(ex.skip)), ex.LineInfo)
	return nil
}

func (ex *exConcat) discharge(fn *FnProto, dst uint8) error {
	for i, expr := range ex.exprs {
		if err := expr.discharge(fn, dst+uint8(i)); err != nil {
			return err
		}
	}
	fn.code(iABC(CONCAT, dst, dst, dst+uint8(len(ex.exprs)-1)), ex.LineInfo)
	return nil
}

func (ex *exVariable) discharge(fn *FnProto, dst uint8) error {
	if !ex.local {
		fn.code(iAB(GETUPVAL, dst, ex.address), ex.LineInfo)
	} else if uint8(dst) != ex.address { // already there
		fn.code(iAB(MOVE, dst, ex.address), ex.LineInfo)
	}
	return nil
}

func (ex *exTable) discharge(fn *FnProto, dst uint8) error {
	fn.code(iABC(NEWTABLE, dst, uint8(len(ex.array)), uint8(len(ex.fields))), ex.LineInfo)

	numOut := 0
	tableIndex := uint64(1)

	dischargeValues := func() error {
		if tableIndex > math.MaxUint8 && tableIndex <= math.MaxUint32 {
			fn.code(iABC(SETLIST, dst, uint8(numOut+1), 0), ex.LineInfo)
			fn.code(Bytecode(tableIndex), ex.LineInfo)
		} else if tableIndex > math.MaxUint32 {
			return fmt.Errorf("table index overflow")
		} else {
			fn.code(iABC(SETLIST, dst, uint8(numOut+1), uint8(tableIndex)), ex.LineInfo)
		}
		tableIndex += uint64(numOut)
		numOut = 0
		return nil
	}

	for _, val := range ex.array {
		if err := val.discharge(fn, dst+1+uint8(numOut)); err != nil {
			return err
		}
		numOut++
		if numOut+1 == math.MaxUint8 {
			if err := dischargeValues(); err != nil {
				return err
			}
		}
	}

	if numOut > 0 {
		if err := dischargeValues(); err != nil {
			return err
		}
	}

	for _, field := range ex.fields {
		ikey, keyIsConst, err := dischargeMaybeConst(fn, field.key, dst+1)
		if err != nil {
			return err
		}
		ival, valIsConst, err := dischargeMaybeConst(fn, field.val, dst+2)
		if err != nil {
			return err
		}
		fn.code(iABCK(SETTABLE, dst, ikey, keyIsConst, ival, valIsConst), ex.LineInfo)
	}

	return nil
}

func (ex *exIndex) discharge(fn *FnProto, dst uint8) error {
	ikey, keyIsConst, err := dischargeMaybeConst(fn, ex.key, dst+1)
	if err != nil {
		return err
	}
	if val, isVal := ex.table.(*exVariable); isVal {
		if val.local {
			fn.code(iABCK(GETTABLE, dst, val.address, false, ikey, keyIsConst), ex.LineInfo)
		} else {
			fn.code(iABCK(GETTABUP, dst, val.address, false, ikey, keyIsConst), ex.LineInfo)
		}
		return nil
	}
	// if the table is not a value, it is a value that will be colocated in the stack
	// after discharging.
	err = ex.table.discharge(fn, dst)
	fn.code(iABCK(GETTABLE, dst, dst, false, ikey, keyIsConst), ex.LineInfo)
	return err
}

/*
OR expression
Discharge left
TEST	0 1			// test left
JMP	2	to pc 12 // if true patch to true position (into if block)
discharge right // final result of or expression

AND expression
Discharge left
TEST	0 0			// test left
JMP	2	to pc 12 // if false patch to false position (after if block)
discharge right // final result of and expression
*/
func (ex *exInfixOp) discharge(fn *FnProto, dst uint8) error {
	switch ex.operand {
	case TokenBitwiseOr, TokenBitwiseNotOrXOr, TokenBitwiseAnd, TokenShiftLeft, TokenShiftRight,
		TokenModulo, TokenDivide, TokenFloorDivide, TokenExponent, TokenMinus, TokenAdd, TokenMultiply:
		if err := ex.dischargeBoth(fn, dst); err != nil {
			return err
		}
		fn.code(iABC(tokenToBytecodeOp[ex.operand], dst, dst, dst+1), ex.LineInfo)
	case TokenLt, TokenLe, TokenEq:
		if err := ex.dischargeBoth(fn, dst); err != nil {
			return err
		}
		fn.code(iABC(tokenToBytecodeOp[ex.operand], 0, dst, dst+1), ex.LineInfo) // if false skip next
		fn.code(iABC(LOADBOOL, dst, 0, 1), ex.LineInfo)                          // set false don't skip next
		fn.code(iABC(LOADBOOL, dst, 1, 0), ex.LineInfo)                          // set true then skip next
	case TokenNe:
		if err := ex.dischargeBoth(fn, dst); err != nil {
			return err
		}
		fn.code(iABC(EQ, 1, dst, dst+1), ex.LineInfo)   // if not eq skip next
		fn.code(iABC(LOADBOOL, dst, 0, 1), ex.LineInfo) // set false don't skip next
		fn.code(iABC(LOADBOOL, dst, 1, 0), ex.LineInfo) // set true then skip next
	case TokenAnd:
		if err := ex.left.discharge(fn, dst); err != nil {
			return err
		}
		fn.code(iAB(TEST, dst, 0), ex.LineInfo)
		ijmp := fn.code(iABx(JMP, 0, 0), ex.LineInfo)
		if err := ex.right.discharge(fn, dst); err != nil {
			return err
		}
		fn.ByteCodes[ijmp] = iAsBx(JMP, 0, int16(len(fn.ByteCodes)-ijmp-1))
	case TokenOr:
		if err := ex.left.discharge(fn, dst); err != nil {
			return err
		}
		fn.code(iAB(TEST, dst, 1), ex.LineInfo)
		ijmp := fn.code(iABx(JMP, 0, 0), ex.LineInfo)
		if err := ex.right.discharge(fn, dst); err != nil {
			return err
		}
		fn.ByteCodes[ijmp] = iAsBx(JMP, 0, int16(len(fn.ByteCodes)-ijmp-1))
	default:
		panic(fmt.Sprintf("unknown binop %s", ex.operand))
	}
	return nil
}

func (ex *exInfixOp) dischargeBoth(fn *FnProto, dst uint8) error {
	if err := ex.left.discharge(fn, dst); err != nil {
		return err
	}
	return ex.right.discharge(fn, dst+1)
}

func constFold(ex *exInfixOp) expression {
	if ex.operand == TokenGt {
		ex.operand = TokenLt
		ex.left, ex.right = ex.right, ex.left
	} else if ex.operand == TokenGe {
		ex.operand = TokenLe
		ex.left, ex.right = ex.right, ex.left
	}

	if ex.operand == TokenConcat {
		return ex.foldConcat()
	} else if exIsNum(ex.left) && exIsNum(ex.right) {
		return ex.foldConstArith()
	} else if ex.operand == TokenEq && exIsString(ex.left) && exIsString(ex.right) {
		return &exBool{val: exToString(ex.left) == exToString(ex.right), LineInfo: ex.LineInfo}
	} else if ex.operand == TokenNe && exIsString(ex.left) && exIsString(ex.right) {
		return &exBool{val: exToString(ex.left) != exToString(ex.right), LineInfo: ex.LineInfo}
	}
	return ex
}

func (ex *exInfixOp) foldConcat() expression {
	if exIsStringOrNumber(ex.left) && exIsStringOrNumber(ex.right) {
		return &exString{val: exToString(ex.left) + exToString(ex.right), LineInfo: ex.LineInfo}
	} else if concat, isConcat := ex.right.(*exConcat); isConcat {
		concat.exprs = append([]expression{ex.left}, concat.exprs...)
		return concat
	}
	return &exConcat{
		exprs:    []expression{ex.left, ex.right},
		LineInfo: ex.LineInfo,
	}
}

func (ex *exInfixOp) foldConstArith() expression {
	op := tokenToMetaMethod[ex.operand]
	switch ex.operand {
	case TokenBitwiseAnd, TokenBitwiseOr, TokenBitwiseNotOrXOr, TokenShiftLeft, TokenShiftRight:
		return &exInteger{val: intArith(op, exToInt(ex.left), exToInt(ex.right)), LineInfo: ex.LineInfo}
	case TokenDivide, TokenExponent:
		return &exFloat{val: floatArith(op, exToFloat(ex.left), exToFloat(ex.right)), LineInfo: ex.LineInfo}
	case TokenEq:
		return &exBool{val: exToFloat(ex.left) == exToFloat(ex.right), LineInfo: ex.LineInfo}
	case TokenNe:
		return &exBool{val: exToFloat(ex.left) != exToFloat(ex.right), LineInfo: ex.LineInfo}
	case TokenLt:
		return &exBool{val: exToFloat(ex.left) < exToFloat(ex.right), LineInfo: ex.LineInfo}
	case TokenLe:
		return &exBool{val: exToFloat(ex.left) <= exToFloat(ex.right), LineInfo: ex.LineInfo}
	default:
		liva, lisInt := ex.left.(*exInteger)
		riva, risInt := ex.right.(*exInteger)
		if lisInt && risInt {
			return &exInteger{val: intArith(op, liva.val, riva.val), LineInfo: ex.LineInfo}
		}
		return &exFloat{val: floatArith(op, exToFloat(ex.left), exToFloat(ex.right)), LineInfo: ex.LineInfo}
	}
}

// unaryExpression will process a unary token with a value. If the value can be
// folded then a simple expression is returned. However if it cannot be folded,
// the last expression is discharged and the unary expression is returned for future
// folding as well.
func unaryExpression(tk *Token, valDesc expression) expression {
	switch tk.Kind {
	case TokenNot:
		switch tval := valDesc.(type) {
		case *exString:
			return &exBool{val: true, LineInfo: tk.LineInfo}
		case *exInteger:
			return &exBool{val: tval.val != 0, LineInfo: tk.LineInfo}
		case *exFloat:
			return &exBool{val: tval.val != 0, LineInfo: tk.LineInfo}
		case *exBool:
			return &exBool{val: !tval.val, LineInfo: tk.LineInfo}
		case *exNil:
			return &exBool{val: true, LineInfo: tk.LineInfo}
		}
		return &exUnaryOp{op: NOT, val: valDesc, LineInfo: tk.LineInfo}
	case TokenMinus:
		switch tval := valDesc.(type) {
		case *exInteger:
			return &exInteger{val: -tval.val, LineInfo: tk.LineInfo}
		case *exFloat:
			return &exFloat{val: -tval.val, LineInfo: tk.LineInfo}
		}
		return &exUnaryOp{op: UNM, val: valDesc, LineInfo: tk.LineInfo}
	case TokenLength:
		// if this is simply a string constant, we can just loan an integer instead of calling length
		if str, isStr := valDesc.(*exString); isStr {
			return &exInteger{val: int64(len(str.val)), LineInfo: tk.LineInfo}
		}
		return &exUnaryOp{op: LEN, val: valDesc, LineInfo: tk.LineInfo}
	case TokenBitwiseNotOrXOr:
		switch tval := valDesc.(type) {
		case *exInteger:
			return &exInteger{val: ^tval.val, LineInfo: tk.LineInfo}
		case *exFloat:
			return &exFloat{val: float64(^int64(tval.val)), LineInfo: tk.LineInfo}
		}
		return &exUnaryOp{op: BNOT, val: valDesc, LineInfo: tk.LineInfo}
	default:
		panic("unknown unary")
	}
}

func exIsStringOrNumber(ex expression) bool {
	return exIsString(ex) || exIsNum(ex)
}

func exIsString(ex expression) bool {
	switch ex.(type) {
	case *exString:
		return true
	}
	return false
}

func exIsNum(ex expression) bool {
	switch ex.(type) {
	case *exInteger, *exFloat:
		return true
	}
	return false
}

func exToString(ex expression) string {
	switch expr := ex.(type) {
	case *exString:
		return expr.val
	case *exFloat:
		return fmt.Sprintf("%v", expr.val)
	case *exInteger:
		return fmt.Sprintf("%v", expr.val)
	default:
		panic("cannot convert to string")
	}
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

func dischargeMaybeConst(fn *FnProto, ex expression, dst uint8) (uint8, bool, error) {
	if k, isK := exIsConst(ex); isK {
		addr, err := fn.addConst(k)
		return uint8(addr), true, err
	}
	return dst, false, ex.discharge(fn, dst)
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
