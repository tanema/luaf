package parse

import (
	"errors"
	"fmt"
	"math"
	"strconv"

	"github.com/tanema/luaf/src/bytecode"
	"github.com/tanema/luaf/src/types"
)

type (
	expression interface {
		discharge(fn *FnProto, dst uint8) error
		inferType() (types.Definition, error)
	}
	exString struct {
		val string
		LineInfo
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
		fnproto *FnProto
		fn      uint16
	}
	exTable struct {
		array []expression
		keys  []expression
		vals  []expression
		LineInfo
	}
	exVarArgs struct {
		LineInfo
		want uint8
	}
	exVariable struct { // upvalue or local
		lvar *local
		name string
		LineInfo
		typeDefn  types.Definition
		local     bool
		attrConst bool
		attrClose bool
		address   uint8
	}
	exIndex struct {
		table    expression
		key      expression
		typeDefn types.Definition
		LineInfo
	}
	exCall struct {
		fn   expression
		args []expression
		LineInfo
		self        bool
		tail        bool
		nargs, nret uint8
	}
	exInfixOp struct {
		exprs   []expression
		operand tokenType
		LineInfo
	}
	exUnaryOp struct {
		val expression
		LineInfo
		op bytecode.Op
	}
)

const defaultRetN = 2

func newCallExpr(fn expression, args []expression, self bool, li LineInfo) *exCall {
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
		LineInfo: li,
	}
}

func newInfixExpr(op *token, left, right expression) expression {
	return constFold(&exInfixOp{
		operand:  op.Kind,
		exprs:    []expression{left, right},
		LineInfo: op.LineInfo,
	})
}

func (ex *exString) discharge(fn *FnProto, dst uint8) error {
	kaddr, err := fn.addConst(ex.val)
	fn.code(bytecode.IABx(bytecode.LOADK, dst, kaddr), ex.LineInfo)
	return err
}

func (ex *exString) inferType() (types.Definition, error) { return types.String, nil }

func (ex *exInteger) discharge(fn *FnProto, dst uint8) error {
	if ex.val > math.MinInt16 && ex.val < math.MaxInt16-1 {
		fn.code(bytecode.IAsBx(bytecode.LOADI, dst, int16(ex.val)), ex.LineInfo)
		return nil
	}
	kaddr, err := fn.addConst(ex.val)
	fn.code(bytecode.IABx(bytecode.LOADK, dst, kaddr), ex.LineInfo)
	return err
}

func (ex *exInteger) inferType() (types.Definition, error) { return types.Int, nil }

func (ex *exFloat) discharge(fn *FnProto, dst uint8) error {
	if ex.val == math.Trunc(ex.val) && (ex.val > math.MinInt16 && ex.val < math.MaxInt16-1) {
		fn.code(bytecode.IAsBx(bytecode.LOADF, dst, int16(ex.val)), ex.LineInfo)
	}
	kaddr, err := fn.addConst(ex.val)
	fn.code(bytecode.IABx(bytecode.LOADK, dst, kaddr), ex.LineInfo)
	return err
}

func (ex *exFloat) inferType() (types.Definition, error) { return types.Float, nil }

func (ex *exNil) discharge(fn *FnProto, dst uint8) error {
	fn.code(bytecode.IABx(bytecode.LOADNIL, dst, ex.num), ex.LineInfo)
	return nil
}

func (ex *exNil) inferType() (types.Definition, error) { return types.Nil, nil }

func (ex *exClosure) discharge(fn *FnProto, dst uint8) error {
	fn.code(bytecode.IABx(bytecode.CLOSURE, dst, ex.fn), ex.LineInfo)
	return nil
}

func (ex *exClosure) inferType() (types.Definition, error) { return ex.fnproto.defn, nil }

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
		fn.code(bytecode.IABCK(bytecode.SELF, dst, dst, false, uint8(kaddr), true), index.LineInfo)
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
		fn.code(bytecode.IAB(bytecode.TAILCALL, dst, ex.nargs), ex.LineInfo)
	} else {
		fn.code(bytecode.IABC(bytecode.CALL, dst, ex.nargs, ex.nret), ex.LineInfo)
	}
	return nil
}

func (ex *exCall) inferType() (types.Definition, error) { return types.Any, nil }

func (ex *exVarArgs) discharge(fn *FnProto, dst uint8) error {
	fn.code(bytecode.IAB(bytecode.VARARG, dst, ex.want), ex.LineInfo)
	return nil
}

func (ex *exVarArgs) inferType() (types.Definition, error) { return types.Any, nil }

func (ex *exUnaryOp) discharge(fn *FnProto, dst uint8) error {
	if err := ex.val.discharge(fn, dst); err != nil {
		return err
	}
	fn.code(bytecode.IAB(ex.op, dst, dst), ex.LineInfo)
	return nil
}

func (ex *exUnaryOp) inferType() (types.Definition, error) {
	kind, err := ex.val.inferType()
	if err != nil {
		return types.Any, err
	}
	// TODO once all defined we should be able to get return types
	_, isTable := kind.(*types.Table)
	if isTable {
		isTable = true
	}
	switch ex.op {
	case bytecode.NOT:
		if isTable || kind == types.Any {
			return types.Any, nil
		}
		return types.Bool, nil
	case bytecode.UNM:
		if isTable {
			return types.Any, nil
		} else if kind != types.Number && kind != types.Int && kind != types.Float && kind != types.Any {
			return nil, fmt.Errorf("attempt to unm a %v", kind)
		}
		return kind, nil
	case bytecode.LEN:
		if kind != types.String && !isTable && kind != types.Any {
			return nil, fmt.Errorf("attempt to get length of a %v value", kind)
		} else if isTable {
			return types.Any, nil
		}
		return types.Int, nil
	case bytecode.BNOT:
		if kind != types.Number && kind != types.Int && kind != types.Float && !isTable && kind != types.Any {
			return nil, fmt.Errorf("attempt to bnot a %v", kind)
		} else if isTable {
			return types.Any, nil
		}
		return kind, nil
	default:
		return nil, fmt.Errorf("unexpected unary op %v", ex.op)
	}
}

func (ex *exBool) discharge(fn *FnProto, dst uint8) error {
	fn.code(bytecode.IABC(bytecode.LOADBOOL, dst, b2U8(ex.val), b2U8(ex.skip)), ex.LineInfo)
	return nil
}

func (ex *exBool) inferType() (types.Definition, error) {
	return types.Bool, nil
}

func (ex *exVariable) discharge(fn *FnProto, dst uint8) error {
	if !ex.local {
		fn.code(bytecode.IAB(bytecode.GETUPVAL, dst, ex.address), ex.LineInfo)
	} else if dst != ex.address { // already there
		fn.code(bytecode.IAB(bytecode.MOVE, dst, ex.address), ex.LineInfo)
	}
	return nil
}

func (ex *exVariable) inferType() (types.Definition, error) { return ex.typeDefn, nil }

func (ex *exTable) discharge(fn *FnProto, dst uint8) error {
	fn.code(bytecode.IABC(bytecode.NEWTABLE, dst, uint8(len(ex.array)), uint8(len(ex.vals))), ex.LineInfo)

	numOut := 0
	tableIndex := uint64(1)

	dischargeValues := func() error {
		if tableIndex > math.MaxUint8 && tableIndex <= math.MaxUint32 {
			fn.code(bytecode.IABC(bytecode.SETLIST, dst, uint8(numOut+1), 0), ex.LineInfo)
			fn.code(uint32(tableIndex), ex.LineInfo)
		} else if tableIndex > math.MaxUint32 {
			return errors.New("table index overflow")
		} else {
			fn.code(bytecode.IABC(bytecode.SETLIST, dst, uint8(numOut+1), uint8(tableIndex)), ex.LineInfo)
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
			fn.code(bytecode.IABC(bytecode.SETLIST, dst, 0, uint8(tableIndex)), ex.LineInfo)
		default:
			if err := dischargeValues(); err != nil {
				return err
			}
		}
	}

	for i, key := range ex.keys {
		ikey, keyIsConst, err := dischargeMaybeConst(fn, key, dst+1)
		if err != nil {
			return err
		}
		valAddr := dst + 1
		if !keyIsConst {
			valAddr++
		}
		ival, valIsConst, err := dischargeMaybeConst(fn, ex.vals[i], valAddr)
		if err != nil {
			return err
		}
		fn.code(bytecode.IABCK(bytecode.SETTABLE, dst, ikey, keyIsConst, ival, valIsConst), ex.LineInfo)
	}

	return nil
}

// TODO this is not right yet.
func (ex *exTable) inferType() (types.Definition, error) {
	defn := types.NewTable()
	if len(ex.array) > 0 && len(ex.keys) == 0 && len(ex.vals) == 0 {
		defn.Hint = types.TblArray
		valDefns, err := inferTypeArray(ex.vals)
		if err != nil {
			return nil, err
		}
		defn.ValDefn = types.Reduce(valDefns)
	} else if len(ex.array) == 0 && len(ex.keys) > 0 && len(ex.vals) > 0 {
		defn.Hint = types.TblMap
		keyDefns, err := inferTypeArray(ex.keys)
		if err != nil {
			return nil, err
		}
		defn.KeyDefn = types.Reduce(keyDefns)

		valDefns, err := inferTypeArray(ex.vals)
		if err != nil {
			return nil, err
		}
		defn.ValDefn = types.Reduce(valDefns)
	}
	return defn, nil
}

func (ex *exIndex) discharge(fn *FnProto, dst uint8) error {
	ikey, keyIsConst, err := dischargeMaybeConst(fn, ex.key, dst+1)
	if err != nil {
		return err
	}
	if val, isVal := ex.table.(*exVariable); isVal {
		if val.local {
			fn.code(bytecode.IABCK(bytecode.GETTABLE, dst, val.address, false, ikey, keyIsConst), ex.LineInfo)
		} else {
			fn.code(bytecode.IABCK(bytecode.GETTABUP, dst, val.address, false, ikey, keyIsConst), ex.LineInfo)
		}
		return nil
	}
	// if the table is not a value, it is a value that will be colocated in the stack
	// after discharging.
	err = ex.table.discharge(fn, dst)
	fn.code(bytecode.IABCK(bytecode.GETTABLE, dst, dst, false, ikey, keyIsConst), ex.LineInfo)
	return err
}

func (ex *exIndex) inferType() (types.Definition, error) { return ex.typeDefn, nil }

func (ex *exInfixOp) discharge(fn *FnProto, dst uint8) error {
	switch ex.operand {
	case tokenBitwiseOrUnion, tokenBitwiseNotOrXOr, tokenBitwiseAnd, tokenShiftLeft, tokenShiftRight,
		tokenModulo, tokenDivide, tokenFloorDivide, tokenExponent, tokenMinus, tokenAdd, tokenMultiply:
		if err := ex.dischargeBoth(fn, dst); err != nil {
			return err
		}
		fn.code(bytecode.IABC(tokenToBytecodeOp[ex.operand], dst, dst, dst+1), ex.LineInfo)
	case tokenLt, tokenLe, tokenEq:
		if err := ex.dischargeBoth(fn, dst); err != nil {
			return err
		}
		fn.code(bytecode.IABC(tokenToBytecodeOp[ex.operand], 0, dst, dst+1), ex.LineInfo) // if false skip next
		fn.code(bytecode.IABC(bytecode.LOADBOOL, dst, 0, 1), ex.LineInfo)                 // set false don't skip next
		fn.code(bytecode.IABC(bytecode.LOADBOOL, dst, 1, 0), ex.LineInfo)                 // set true then skip next
	case tokenNe:
		if err := ex.dischargeBoth(fn, dst); err != nil {
			return err
		}
		fn.code(bytecode.IABC(bytecode.EQ, 1, dst, dst+1), ex.LineInfo)   // if not eq skip next
		fn.code(bytecode.IABC(bytecode.LOADBOOL, dst, 0, 1), ex.LineInfo) // set false don't skip next
		fn.code(bytecode.IABC(bytecode.LOADBOOL, dst, 1, 0), ex.LineInfo) // set true then skip next
	case tokenAnd:
		if err := ex.exprs[0].discharge(fn, dst); err != nil {
			return err
		}
		fn.code(bytecode.IAB(bytecode.TEST, dst, 0), ex.LineInfo)
		ijmp := fn.code(bytecode.IABx(bytecode.JMP, 0, 0), ex.LineInfo)
		if err := ex.exprs[1].discharge(fn, dst); err != nil {
			return err
		}
		fn.ByteCodes[ijmp] = bytecode.IAsBx(bytecode.JMP, 0, int16(len(fn.ByteCodes)-ijmp-1))
	case tokenOr:
		if err := ex.exprs[0].discharge(fn, dst); err != nil {
			return err
		}
		fn.code(bytecode.IAB(bytecode.TEST, dst, 1), ex.LineInfo)
		ijmp := fn.code(bytecode.IABx(bytecode.JMP, 0, 0), ex.LineInfo)
		if err := ex.exprs[1].discharge(fn, dst); err != nil {
			return err
		}
		fn.ByteCodes[ijmp] = bytecode.IAsBx(bytecode.JMP, 0, int16(len(fn.ByteCodes)-ijmp-1))
	case tokenConcat:
		for i, expr := range ex.exprs {
			if err := expr.discharge(fn, dst+uint8(i)); err != nil {
				return err
			}
		}
		fn.code(bytecode.IABC(bytecode.CONCAT, dst, dst, dst+uint8(len(ex.exprs)-1)), ex.LineInfo)
	default:
		panic(fmt.Sprintf("unknown binop %s", ex.operand))
	}
	return nil
}

func (ex *exInfixOp) dischargeBoth(fn *FnProto, dst uint8) error {
	if err := ex.exprs[0].discharge(fn, dst); err != nil {
		return err
	}
	return ex.exprs[1].discharge(fn, dst+1)
}

func (ex *exInfixOp) inferType() (types.Definition, error) {
	switch ex.operand {
	case tokenConcat:
		// check all operands for string or coercable. If table, unknown, if others then error
		for _, x := range ex.exprs {
			kind, err := x.inferType()
			if err != nil {
				return nil, err
			}
			if kind != types.String && kind != types.Number && kind != types.Int && kind != types.Float {
				return types.Any, nil
			}
		}
		return types.String, nil
	case tokenFloorDivide:
		// should always be int except tables.
		for _, x := range ex.exprs {
			kind, err := x.inferType()
			if err != nil {
				return nil, err
			} else if kind != types.Number && kind != types.Int && kind != types.Float {
				return types.Any, nil
			}
		}
		return types.Int, nil
	case tokenBitwiseAnd, tokenBitwiseOrUnion, tokenBitwiseNotOrXOr, tokenShiftLeft,
		tokenShiftRight, tokenModulo, tokenMinus, tokenAdd, tokenMultiply:
		// could be number, int, float
		return types.Number, nil
	case tokenDivide, tokenExponent:
		// should always be float except tables
		for _, x := range ex.exprs {
			kind, err := x.inferType()
			if err != nil {
				return nil, err
			} else if kind != types.Number && kind != types.Int && kind != types.Float {
				return types.Any, nil
			}
		}
		return types.Float, nil
	case tokenEq, tokenNe:
		// should always be bool except tables
		for _, x := range ex.exprs {
			if kind, err := x.inferType(); err != nil {
				return nil, err
			} else if _, isTbl := kind.(*types.Table); isTbl {
				return types.Any, nil
			}
		}
		return types.Bool, nil
	case tokenLt, tokenLe, tokenGt, tokenGe:
		// should always be bool except tables
		for _, x := range ex.exprs {
			kind, err := x.inferType()
			if err != nil {
				return nil, err
			} else if kind != types.String && kind != types.Number && kind != types.Int && kind != types.Float {
				return types.Any, nil
			}
		}
		return types.Bool, nil
	case tokenAnd, tokenOr: // boolean operators any|or are often used to return the second or first value for assignment.
		return types.Any, nil
	default:
		return types.Any, nil
	}
}

func constFold(ex *exInfixOp) expression {
	switch ex.operand {
	case tokenGt:
		ex.operand = tokenLt
		ex.exprs[0], ex.exprs[1] = ex.exprs[1], ex.exprs[0]
	case tokenGe:
		ex.operand = tokenLe
		ex.exprs[0], ex.exprs[1] = ex.exprs[1], ex.exprs[0]
	}

	if ex.operand == tokenConcat {
		if exIsStringOrNumber(ex.exprs[0]) && exIsStringOrNumber(ex.exprs[1]) {
			return &exString{val: exToString(ex.exprs[0]) + exToString(ex.exprs[1]), LineInfo: ex.LineInfo}
		} else if infix, isInfix := ex.exprs[1].(*exInfixOp); isInfix && infix.operand == tokenConcat {
			infix.exprs = append([]expression{ex.exprs[0]}, infix.exprs...)
			return infix
		}
	} else if exIsNum(ex.exprs[0]) && exIsNum(ex.exprs[1]) {
		op := tokenToMetaMethod[ex.operand]
		switch ex.operand {
		case tokenBitwiseAnd, tokenBitwiseOrUnion, tokenBitwiseNotOrXOr, tokenShiftLeft, tokenShiftRight:
			return &exInteger{val: intArith(op, exToInt(ex.exprs[0]), exToInt(ex.exprs[1])), LineInfo: ex.LineInfo}
		case tokenDivide, tokenExponent:
			return &exFloat{val: floatArith(op, exToFloat(ex.exprs[0]), exToFloat(ex.exprs[1])), LineInfo: ex.LineInfo}
		case tokenEq:
			return &exBool{val: exToFloat(ex.exprs[0]) == exToFloat(ex.exprs[1]), LineInfo: ex.LineInfo}
		case tokenNe:
			return &exBool{val: exToFloat(ex.exprs[0]) != exToFloat(ex.exprs[1]), LineInfo: ex.LineInfo}
		case tokenLt:
			return &exBool{val: exToFloat(ex.exprs[0]) < exToFloat(ex.exprs[1]), LineInfo: ex.LineInfo}
		case tokenLe:
			return &exBool{val: exToFloat(ex.exprs[0]) <= exToFloat(ex.exprs[1]), LineInfo: ex.LineInfo}
		case tokenAnd:
			return ex.exprs[1]
		case tokenOr:
			return ex.exprs[0]
		default:
			liva, lisInt := ex.exprs[0].(*exInteger)
			riva, risInt := ex.exprs[1].(*exInteger)
			if lisInt && risInt {
				return &exInteger{val: intArith(op, liva.val, riva.val), LineInfo: ex.LineInfo}
			}
			return &exFloat{val: floatArith(op, exToFloat(ex.exprs[0]), exToFloat(ex.exprs[1])), LineInfo: ex.LineInfo}
		}
	} else if ex.operand == tokenEq && exIsString(ex.exprs[0]) && exIsString(ex.exprs[1]) {
		return &exBool{val: exToString(ex.exprs[0]) == exToString(ex.exprs[1]), LineInfo: ex.LineInfo}
	} else if ex.operand == tokenNe && exIsString(ex.exprs[0]) && exIsString(ex.exprs[1]) {
		return &exBool{val: exToString(ex.exprs[0]) != exToString(ex.exprs[1]), LineInfo: ex.LineInfo}
	}
	return ex
}

// unaryExpression will process a unary token with a value. If the value can be
// folded then a simple expression is returned. However if it cannot be folded,
// the last expression is discharged and the unary expression is returned for future
// folding as well.
func unaryExpression(tk *token, valDesc expression) expression {
	switch tk.Kind {
	case tokenNot:
		switch tval := valDesc.(type) {
		case *exString, *exInteger, *exFloat:
			return &exBool{val: false, LineInfo: tk.LineInfo}
		case *exBool:
			return &exBool{val: !tval.val, LineInfo: tk.LineInfo}
		case *exNil:
			return &exBool{val: true, LineInfo: tk.LineInfo}
		}
		return &exUnaryOp{op: bytecode.NOT, val: valDesc, LineInfo: tk.LineInfo}
	case tokenMinus:
		switch tval := valDesc.(type) {
		case *exInteger:
			return &exInteger{val: -tval.val, LineInfo: tk.LineInfo}
		case *exFloat:
			return &exFloat{val: -tval.val, LineInfo: tk.LineInfo}
		}
		return &exUnaryOp{op: bytecode.UNM, val: valDesc, LineInfo: tk.LineInfo}
	case tokenLength:
		// if this is simply a string constant, we can just loan an integer instead of calling length
		if str, isStr := valDesc.(*exString); isStr {
			return &exInteger{val: int64(len(str.val)), LineInfo: tk.LineInfo}
		}
		return &exUnaryOp{op: bytecode.LEN, val: valDesc, LineInfo: tk.LineInfo}
	case tokenBitwiseNotOrXOr:
		switch tval := valDesc.(type) {
		case *exInteger:
			return &exInteger{val: ^tval.val, LineInfo: tk.LineInfo}
		case *exFloat:
			return &exFloat{val: float64(^int64(tval.val)), LineInfo: tk.LineInfo}
		}
		return &exUnaryOp{op: bytecode.BNOT, val: valDesc, LineInfo: tk.LineInfo}
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

func intArith(op MetaMethod, lval, rval int64) int64 {
	switch op {
	case MetaAdd:
		return lval + rval
	case MetaSub:
		return lval - rval
	case MetaMul:
		return lval * rval
	case MetaIDiv:
		if rval == 0 {
			return int64(math.Inf(1))
		}
		return lval / rval
	case MetaUNM:
		return -lval
	case MetaMod:
		return lval % rval
	case MetaBAnd:
		return lval & rval
	case MetaBOr:
		return lval | rval
	case MetaBXOr:
		return lval | rval
	case MetaShl:
		if rval > 0 {
			return lval << rval
		}
		return lval >> int64(math.Abs(float64(rval)))
	case MetaShr:
		if rval > 0 {
			return lval >> rval
		}
		return lval << int64(math.Abs(float64(rval)))
	case MetaBNot:
		return ^lval
	default:
		panic(fmt.Sprintf("cannot perform float %v op", op))
	}
}

func floatArith(op MetaMethod, lval, rval float64) float64 {
	switch op {
	case MetaAdd:
		return lval + rval
	case MetaSub:
		return lval - rval
	case MetaMul:
		return lval * rval
	case MetaDiv:
		return lval / rval
	case MetaPow:
		return math.Pow(lval, rval)
	case MetaIDiv:
		return math.Floor(lval / rval)
	case MetaUNM:
		return -lval
	case MetaMod:
		return math.Mod(lval, rval)
	default:
		panic(fmt.Sprintf("cannot perform float %v op", op))
	}
}

func inferTypeArray(exprs []expression) ([]types.Definition, error) {
	var err error
	defns := make([]types.Definition, len(exprs))
	for i, ex := range exprs {
		defns[i], err = ex.inferType()
		if err != nil {
			return nil, err
		}
	}
	return defns, nil
}

func b2U8(val bool) uint8 {
	if val {
		return 1
	}
	return 0
}
