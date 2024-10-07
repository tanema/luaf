package shine

type (
	expression interface{ discharge(*FuncProto, uint8) }
	assignable interface{ assignTo(*FuncProto, uint8, bool) }
	exConstant struct{ index uint16 }
	exNil      struct{}
	exBool     struct{ value, skipnext bool }
	exValue    struct { // upvalue or local
		local   bool
		name    string
		address uint8
	}
	exIndex struct {
		local        bool
		table        uint8
		key          uint8
		keyIsConst   bool
		value        uint8
		valueIsConst bool
	}
	exClosure struct{ fn uint16 }
	exCall    struct{ fn, nargs, nret uint8 }
	exVarArgs struct{ limit, want uint8 }
	exBinOp   struct {
		op         BytecodeOp
		lval, rval uint8
	}
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
func (ex *exNil) discharge(fn *FuncProto, dst uint8)      { fn.code(iABx(LOADNIL, dst, 1)) }
func (ex *exBool) discharge(fn *FuncProto, dst uint8) {
	val := uint8(0)
	skip := uint8(0)
	if ex.value {
		val = 1
	}
	if ex.skipnext {
		skip = 1
	}
	fn.code(iABC(LOADBOOL, dst, val, skip))
}
func (ex *exValue) discharge(fn *FuncProto, dst uint8) {
	if !ex.local {
		fn.code(iAB(GETUPVAL, dst, ex.address))
	} else if uint8(dst) != ex.address { // already there
		fn.code(iAB(MOVE, dst, ex.address))
	}
}

func (ex *exValue) assignTo(fn *FuncProto, from uint8, fromIsConst bool) {
	if !ex.local {
		fn.code(iABCK(SETUPVAL, ex.address, from, fromIsConst, 0, false))
	} else if from != ex.address {
		fn.code(iABCK(MOVE, ex.address, from, fromIsConst, 0, false))
	}
}

func (ex *exIndex) discharge(fn *FuncProto, dst uint8) {
	if ex.local {
		fn.code(iABCK(GETTABLE, dst, ex.table, false, ex.key, ex.keyIsConst))
	} else {
		fn.code(iABCK(GETTABUP, dst, ex.table, false, ex.key, ex.keyIsConst))
	}
}

func (ex *exIndex) assignTo(fn *FuncProto, from uint8, fromIsConst bool) {
	if ex.local {
		fn.code(iABCK(SETTABLE, ex.table, ex.key, ex.keyIsConst, from, fromIsConst))
	} else {
		fn.code(iABCK(SETTABUP, ex.table, ex.key, ex.keyIsConst, from, fromIsConst))
	}
}

func (ex *exClosure) discharge(fn *FuncProto, dst uint8) { fn.code(iABx(CLOSURE, dst, ex.fn)) }
func (ex *exCall) discharge(fn *FuncProto, dst uint8)    { fn.code(iABC(CALL, ex.fn, ex.nargs, ex.nret)) }
func (ex *exVarArgs) discharge(fn *FuncProto, dst uint8) { fn.code(iAB(CALL, ex.limit, ex.want)) }
func (ex *exBinOp) discharge(fn *FuncProto, dst uint8)   { fn.code(iABC(ex.op, dst, ex.lval, ex.rval)) }
func (ex *exUnaryOp) discharge(fn *FuncProto, dst uint8) { fn.code(iAB(ex.op, dst, ex.val)) }
func (ex *exBoolBinOp) discharge(fn *FuncProto, dst uint8) {
	fn.code(iABC(ex.op, ex.expected, ex.lval, ex.rval)) // if false skip next
	fn.code(iABx(JMP, 0, 1))                            // jump to set false
	fn.code(iABC(LOADBOOL, dst, 1, 1))                  // set true skip next
	fn.code(iABC(LOADBOOL, dst, 0, 0))                  // set false don't skip next
}
