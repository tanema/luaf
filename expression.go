package luaf

import (
	"errors"
	"fmt"
	"math"
	"strconv"
)

const defaultRetN = 2

type (
	expression interface {
		discharge(fn *FnProto, dst uint8) error
	}
	exString struct {
		val string
		lineInfo
	}
	exInteger struct {
		lineInfo
		val int64
	}
	exFloat struct {
		lineInfo
		val float64
	}
	exNil struct {
		lineInfo
		num uint16
	}
	exBool struct {
		lineInfo
		val, skip bool
	}
	exClosure struct {
		lineInfo
		fn uint16
	}
	exVariable struct { // upvalue or local
		lvar *local
		name string
		lineInfo
		local, attrConst, attrClose bool
		address                     uint8
	}
	exIndex struct {
		table, key expression
		lineInfo
	}
	exConcat struct { // upvalue or local
		exprs []expression
		lineInfo
	}
	exCall struct {
		fn   expression
		args []expression
		lineInfo
		self        bool
		tail        bool
		nargs, nret uint8
	}
	exVarArgs struct {
		lineInfo
		want uint8
	}
	tableField struct {
		key expression
		val expression
	}
	exTable struct {
		array  []expression
		fields []tableField
		lineInfo
	}
	exInfixOp struct {
		left, right expression
		operand     tokenType
		lineInfo
	}
	exUnaryOp struct {
		val expression
		lineInfo
		op BytecodeOp
	}
)

func newCallExpr(fn expression, args []expression, self bool, li lineInfo) *exCall {
	nargs := uint8(len(args) + 1)
	if self {
		nargs++
	}
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
	return &exCall{
		fn:       fn,
		self:     self,
		nargs:    nargs,
		nret:     defaultRetN,
		args:     args,
		lineInfo: li,
	}
}

func newInfixExpr(op *token, left, right expression) expression {
	return constFold(&exInfixOp{
		operand:  op.Kind,
		left:     left,
		right:    right,
		lineInfo: op.lineInfo,
	})
}

func (ex *exString) discharge(fn *FnProto, dst uint8) error {
	kaddr, err := fn.addConst(ex.val)
	fn.code(iABx(LOADK, dst, kaddr), ex.lineInfo)
	return err
}

func (ex *exInteger) discharge(fn *FnProto, dst uint8) error {
	if ex.val > math.MinInt16 && ex.val < math.MaxInt16-1 {
		fn.code(iAsBx(LOADI, dst, int16(ex.val)), ex.lineInfo)
		return nil
	}
	kaddr, err := fn.addConst(ex.val)
	fn.code(iABx(LOADK, dst, kaddr), ex.lineInfo)
	return err
}

func (ex *exFloat) discharge(fn *FnProto, dst uint8) error {
	if ex.val == math.Trunc(ex.val) && (ex.val > math.MinInt16 && ex.val < math.MaxInt16-1) {
		fn.code(iAsBx(LOADF, dst, int16(ex.val)), ex.lineInfo)
	}
	kaddr, err := fn.addConst(ex.val)
	fn.code(iABx(LOADK, dst, kaddr), ex.lineInfo)
	return err
}

func (ex *exNil) discharge(fn *FnProto, dst uint8) error {
	fn.code(iABx(LOADNIL, dst, ex.num), ex.lineInfo)
	return nil
}

func (ex *exClosure) discharge(fn *FnProto, dst uint8) error {
	fn.code(iABx(CLOSURE, dst, ex.fn), ex.lineInfo)
	return nil
}

func (ex *exCall) discharge(fn *FnProto, dst uint8) error {
	offset := uint8(1)
	if ex.self {
		index, isIndex := ex.fn.(*exIndex)
		if !isIndex {
			panic("found non index on self fn call expression")
		}
		if err := index.table.discharge(fn, dst); err != nil {
			return err
		}
		kaddr, err := fn.addConst(index.key.(*exString).val)
		if err != nil {
			return err
		}
		fn.code(iABCK(SELF, dst, dst, false, uint8(kaddr), true), index.lineInfo)
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
		fn.code(iAB(TAILCALL, dst, ex.nargs), ex.lineInfo)
	} else {
		fn.code(iABC(CALL, dst, ex.nargs, ex.nret), ex.lineInfo)
	}
	return nil
}

func (ex *exVarArgs) discharge(fn *FnProto, dst uint8) error {
	fn.code(iAB(VARARG, dst, ex.want), ex.lineInfo)
	return nil
}

func (ex *exUnaryOp) discharge(fn *FnProto, dst uint8) error {
	if err := ex.val.discharge(fn, dst); err != nil {
		return err
	}
	fn.code(iAB(ex.op, dst, dst), ex.lineInfo)
	return nil
}

func (ex *exBool) discharge(fn *FnProto, dst uint8) error {
	fn.code(iABC(LOADBOOL, dst, b2U8(ex.val), b2U8(ex.skip)), ex.lineInfo)
	return nil
}

func (ex *exConcat) discharge(fn *FnProto, dst uint8) error {
	for i, expr := range ex.exprs {
		if err := expr.discharge(fn, dst+uint8(i)); err != nil {
			return err
		}
	}
	fn.code(iABC(CONCAT, dst, dst, dst+uint8(len(ex.exprs)-1)), ex.lineInfo)
	return nil
}

func (ex *exVariable) discharge(fn *FnProto, dst uint8) error {
	if !ex.local {
		fn.code(iAB(GETUPVAL, dst, ex.address), ex.lineInfo)
	} else if dst != ex.address { // already there
		fn.code(iAB(MOVE, dst, ex.address), ex.lineInfo)
	}
	return nil
}

func (ex *exTable) discharge(fn *FnProto, dst uint8) error {
	fn.code(iABC(NEWTABLE, dst, uint8(len(ex.array)), uint8(len(ex.fields))), ex.lineInfo)

	numOut := 0
	tableIndex := uint64(1)

	dischargeValues := func() error {
		if tableIndex > math.MaxUint8 && tableIndex <= math.MaxUint32 {
			fn.code(iABC(SETLIST, dst, uint8(numOut+1), 0), ex.lineInfo)
			fn.code(Bytecode(tableIndex), ex.lineInfo)
		} else if tableIndex > math.MaxUint32 {
			return errors.New("table index overflow")
		} else {
			fn.code(iABC(SETLIST, dst, uint8(numOut+1), uint8(tableIndex)), ex.lineInfo)
		}
		tableIndex += uint64(numOut)
		numOut = 0
		return nil
	}

	if len(ex.array) > 0 {
		for i := range len(ex.array) - 1 {
			if err := ex.array[i].discharge(fn, dst+1+uint8(numOut)); err != nil {
				return err
			}
			numOut++
			if numOut+1 == math.MaxUint8 {
				if err := dischargeValues(); err != nil {
					return err
				}
			}
		}

		lastExpr := ex.array[len(ex.array)-1]
		if err := lastExpr.discharge(fn, dst+1+uint8(numOut)); err != nil {
			return err
		}
		numOut++
		switch lastExpr.(type) {
		case *exCall, *exVarArgs:
			fn.code(iABC(SETLIST, dst, 0, uint8(tableIndex)), ex.lineInfo)
		default:
			if err := dischargeValues(); err != nil {
				return err
			}
		}
	}

	for _, field := range ex.fields {
		ikey, keyIsConst, err := dischargeMaybeConst(fn, field.key, dst+1)
		if err != nil {
			return err
		}
		valAddr := dst + 1
		if !keyIsConst {
			valAddr++
		}
		ival, valIsConst, err := dischargeMaybeConst(fn, field.val, valAddr)
		if err != nil {
			return err
		}
		fn.code(iABCK(SETTABLE, dst, ikey, keyIsConst, ival, valIsConst), ex.lineInfo)
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
			fn.code(iABCK(GETTABLE, dst, val.address, false, ikey, keyIsConst), ex.lineInfo)
		} else {
			fn.code(iABCK(GETTABUP, dst, val.address, false, ikey, keyIsConst), ex.lineInfo)
		}
		return nil
	}
	// if the table is not a value, it is a value that will be colocated in the stack
	// after discharging.
	err = ex.table.discharge(fn, dst)
	fn.code(iABCK(GETTABLE, dst, dst, false, ikey, keyIsConst), ex.lineInfo)
	return err
}

func (ex *exInfixOp) discharge(fn *FnProto, dst uint8) error {
	switch ex.operand {
	case tokenBitwiseOr, tokenBitwiseNotOrXOr, tokenBitwiseAnd, tokenShiftLeft, tokenShiftRight,
		tokenModulo, tokenDivide, tokenFloorDivide, tokenExponent, tokenMinus, tokenAdd, tokenMultiply:
		if err := ex.dischargeBoth(fn, dst); err != nil {
			return err
		}
		fn.code(iABC(tokenToBytecodeOp[ex.operand], dst, dst, dst+1), ex.lineInfo)
	case tokenLt, tokenLe, tokenEq:
		if err := ex.dischargeBoth(fn, dst); err != nil {
			return err
		}
		fn.code(iABC(tokenToBytecodeOp[ex.operand], 0, dst, dst+1), ex.lineInfo) // if false skip next
		fn.code(iABC(LOADBOOL, dst, 0, 1), ex.lineInfo)                          // set false don't skip next
		fn.code(iABC(LOADBOOL, dst, 1, 0), ex.lineInfo)                          // set true then skip next
	case tokenNe:
		if err := ex.dischargeBoth(fn, dst); err != nil {
			return err
		}
		fn.code(iABC(EQ, 1, dst, dst+1), ex.lineInfo)   // if not eq skip next
		fn.code(iABC(LOADBOOL, dst, 0, 1), ex.lineInfo) // set false don't skip next
		fn.code(iABC(LOADBOOL, dst, 1, 0), ex.lineInfo) // set true then skip next
	case tokenAnd:
		if err := ex.left.discharge(fn, dst); err != nil {
			return err
		}
		fn.code(iAB(TEST, dst, 0), ex.lineInfo)
		ijmp := fn.code(iABx(JMP, 0, 0), ex.lineInfo)
		if err := ex.right.discharge(fn, dst); err != nil {
			return err
		}
		fn.ByteCodes[ijmp] = iAsBx(JMP, 0, int16(len(fn.ByteCodes)-ijmp-1))
	case tokenOr:
		if err := ex.left.discharge(fn, dst); err != nil {
			return err
		}
		fn.code(iAB(TEST, dst, 1), ex.lineInfo)
		ijmp := fn.code(iABx(JMP, 0, 0), ex.lineInfo)
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
	switch ex.operand {
	case tokenGt:
		ex.operand = tokenLt
		ex.left, ex.right = ex.right, ex.left
	case tokenGe:
		ex.operand = tokenLe
		ex.left, ex.right = ex.right, ex.left
	}

	if ex.operand == tokenConcat {
		return ex.foldConcat()
	} else if exIsNum(ex.left) && exIsNum(ex.right) {
		return ex.foldConstArith()
	} else if ex.operand == tokenEq && exIsString(ex.left) && exIsString(ex.right) {
		return &exBool{val: exToString(ex.left) == exToString(ex.right), lineInfo: ex.lineInfo}
	} else if ex.operand == tokenNe && exIsString(ex.left) && exIsString(ex.right) {
		return &exBool{val: exToString(ex.left) != exToString(ex.right), lineInfo: ex.lineInfo}
	}
	return ex
}

func (ex *exInfixOp) foldConcat() expression {
	if exIsStringOrNumber(ex.left) && exIsStringOrNumber(ex.right) {
		return &exString{val: exToString(ex.left) + exToString(ex.right), lineInfo: ex.lineInfo}
	} else if concat, isConcat := ex.right.(*exConcat); isConcat {
		concat.exprs = append([]expression{ex.left}, concat.exprs...)
		return concat
	}
	return &exConcat{
		exprs:    []expression{ex.left, ex.right},
		lineInfo: ex.lineInfo,
	}
}

func (ex *exInfixOp) foldConstArith() expression {
	op := tokenToMetaMethod[ex.operand]
	switch ex.operand {
	case tokenBitwiseAnd, tokenBitwiseOr, tokenBitwiseNotOrXOr, tokenShiftLeft, tokenShiftRight:
		return &exInteger{val: intArith(op, exToInt(ex.left), exToInt(ex.right)), lineInfo: ex.lineInfo}
	case tokenDivide, tokenExponent:
		return &exFloat{val: floatArith(op, exToFloat(ex.left), exToFloat(ex.right)), lineInfo: ex.lineInfo}
	case tokenEq:
		return &exBool{val: exToFloat(ex.left) == exToFloat(ex.right), lineInfo: ex.lineInfo}
	case tokenNe:
		return &exBool{val: exToFloat(ex.left) != exToFloat(ex.right), lineInfo: ex.lineInfo}
	case tokenLt:
		return &exBool{val: exToFloat(ex.left) < exToFloat(ex.right), lineInfo: ex.lineInfo}
	case tokenLe:
		return &exBool{val: exToFloat(ex.left) <= exToFloat(ex.right), lineInfo: ex.lineInfo}
	case tokenAnd:
		return ex.right
	case tokenOr:
		return ex.left
	default:
		liva, lisInt := ex.left.(*exInteger)
		riva, risInt := ex.right.(*exInteger)
		if lisInt && risInt {
			return &exInteger{val: intArith(op, liva.val, riva.val), lineInfo: ex.lineInfo}
		}
		return &exFloat{val: floatArith(op, exToFloat(ex.left), exToFloat(ex.right)), lineInfo: ex.lineInfo}
	}
}

// unaryExpression will process a unary token with a value. If the value can be
// folded then a simple expression is returned. However if it cannot be folded,
// the last expression is discharged and the unary expression is returned for future
// folding as well.
func unaryExpression(tk *token, valDesc expression) expression {
	switch tk.Kind {
	case tokenNot:
		switch tval := valDesc.(type) {
		case *exString:
			return &exBool{val: true, lineInfo: tk.lineInfo}
		case *exInteger:
			return &exBool{val: tval.val != 0, lineInfo: tk.lineInfo}
		case *exFloat:
			return &exBool{val: tval.val != 0, lineInfo: tk.lineInfo}
		case *exBool:
			return &exBool{val: !tval.val, lineInfo: tk.lineInfo}
		case *exNil:
			return &exBool{val: true, lineInfo: tk.lineInfo}
		}
		return &exUnaryOp{op: NOT, val: valDesc, lineInfo: tk.lineInfo}
	case tokenMinus:
		switch tval := valDesc.(type) {
		case *exInteger:
			return &exInteger{val: -tval.val, lineInfo: tk.lineInfo}
		case *exFloat:
			return &exFloat{val: -tval.val, lineInfo: tk.lineInfo}
		}
		return &exUnaryOp{op: UNM, val: valDesc, lineInfo: tk.lineInfo}
	case tokenLength:
		// if this is simply a string constant, we can just loan an integer instead of calling length
		if str, isStr := valDesc.(*exString); isStr {
			return &exInteger{val: int64(len(str.val)), lineInfo: tk.lineInfo}
		}
		return &exUnaryOp{op: LEN, val: valDesc, lineInfo: tk.lineInfo}
	case tokenBitwiseNotOrXOr:
		switch tval := valDesc.(type) {
		case *exInteger:
			return &exInteger{val: ^tval.val, lineInfo: tk.lineInfo}
		case *exFloat:
			return &exFloat{val: float64(^int64(tval.val)), lineInfo: tk.lineInfo}
		}
		return &exUnaryOp{op: BNOT, val: valDesc, lineInfo: tk.lineInfo}
	default:
		panic("unknown unary")
	}
}

func exIsStringOrNumber(ex expression) bool {
	return exIsString(ex) || exIsNum(ex)
}

func exIsString(ex expression) bool {
	_, isString := ex.(*exString)
	return isString
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
		return strconv.FormatInt(expr.val, 10)
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

func b2U8(val bool) uint8 {
	if val {
		return 1
	}
	return 0
}
