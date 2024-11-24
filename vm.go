package luaf

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"
)

type (
	UpvalueBroker struct {
		index int
		open  bool
		name  string
		stack *[]Value
		val   Value
	}
	VM struct {
		framePointer int64
		Stack        []Value
		env          *Table
	}
	RuntimeErr struct {
		msg string
	}
)

var forNumNames = []string{"initial", "limit", "step"}

func (err *RuntimeErr) Error() string {
	return err.msg
}

func NewVM() *VM {
	env := NewTable(nil, stdlib)
	env.hashtable["_G"] = env
	return &VM{
		Stack:        []Value{env},
		framePointer: 1,
		env:          env,
	}
}

func (vm *VM) err(tmpl string, args ...any) error {
	return &RuntimeErr{msg: fmt.Sprintf(tmpl, args...)}
}

func (vm *VM) Env() *Table {
	return vm.env
}

func (vm *VM) Eval(fn *FuncProto) ([]Value, error) {
	values, _, err := vm.eval(fn, []*UpvalueBroker{vm.newUpValueBroker("_ENV", vm.env, 0)})
	return values, err
}

func (vm *VM) eval(fn *FuncProto, upvals []*UpvalueBroker) ([]Value, int64, error) {
	var programCounter int64
	xargs := vm.truncate(int64(fn.Arity))
	openBrokers := []*UpvalueBroker{}
	tbcValues := []int64{}

	extraArg := func(index int) int {
		if index == 0 {
			programCounter++
			return int(fn.ByteCodes[programCounter])
		} else {
			return index - 1
		}
	}

	for {
		var err error
		if int64(len(fn.ByteCodes)) <= programCounter {
			return nil, programCounter, nil
		}

		instruction := fn.ByteCodes[programCounter]
		switch instruction.op() {
		case MOVE:
			err = vm.SetStack(instruction.getA(), vm.GetStack(instruction.getB()))
		case LOADK:
			err = vm.SetStack(instruction.getA(), fn.getConst(instruction.getBx()))
		case LOADBOOL:
			err = vm.SetStack(instruction.getA(), &Boolean{val: instruction.getB() == 1})
			if instruction.getC() != 0 {
				programCounter++
			}
		case LOADI:
			err = vm.SetStack(instruction.getA(), &Integer{val: instruction.getBx()})
		case LOADNIL:
			a := instruction.getA()
			b := instruction.getBx()
			for i := a; i <= a+b; i++ {
				if err = vm.SetStack(i, &Nil{}); err != nil {
					return nil, programCounter, err
				}
			}
		case ADD:
			err = vm.setABCArith(fn, instruction, metaAdd)
		case SUB:
			err = vm.setABCArith(fn, instruction, metaSub)
		case MUL:
			err = vm.setABCArith(fn, instruction, metaMul)
		case DIV:
			err = vm.setABCArith(fn, instruction, metaDiv)
		case MOD:
			err = vm.setABCArith(fn, instruction, metaMod)
		case POW:
			err = vm.setABCArith(fn, instruction, metaPow)
		case IDIV:
			err = vm.setABCArith(fn, instruction, metaIDiv)
		case BAND:
			err = vm.setABCArith(fn, instruction, metaBAnd)
		case BOR:
			err = vm.setABCArith(fn, instruction, metaBOr)
		case BXOR:
			err = vm.setABCArith(fn, instruction, metaBXOr)
		case SHL:
			err = vm.setABCArith(fn, instruction, metaShl)
		case SHR:
			err = vm.setABCArith(fn, instruction, metaShr)
		case UNM:
			err = vm.setABCArith(fn, instruction, metaUNM)
		case BNOT:
			err = vm.setABCArith(fn, instruction, metaBNot)
		case NOT:
			err = vm.setABCFn(fn, instruction, func(lVal, rVal Value) (Value, error) { return toBool(lVal).Not(), nil })
		case CONCAT:
			err = vm.concat(instruction)
		case TBC:
			tbcValues = append(tbcValues, instruction.getA())
		case JMP:
			upvalIdx := instruction.getA() - 1
			if idx := int(upvalIdx); idx >= 0 {
				for i := int(upvalIdx); i < len(openBrokers); i++ {
					openBrokers[i].Close()
				}
				truncate(&openBrokers, idx)
			}
			programCounter += instruction.getsBx()
		case CLOSE:
			idx := int(instruction.getA())
			for i := idx; i < len(openBrokers); i++ {
				openBrokers[i].Close()
			}
			truncate(&openBrokers, idx)
		case EQ:
			expected := instruction.getA() != 0
			isEq, err := vm.eq(fn, instruction)
			if err != nil {
				return nil, programCounter, err
			} else if isEq != expected {
				programCounter++
			}
		case LT:
			expected := instruction.getA() != 0
			res, err := vm.compare(metaLt, fn, instruction)
			if err != nil {
				return nil, programCounter, err
			} else if isMatch := res < 0; isMatch != expected {
				programCounter++
			}
		case LE:
			expected := instruction.getA() != 0
			res, err := vm.compare(metaLe, fn, instruction)
			if err != nil {
				return nil, programCounter, err
			} else if isMatch := res <= 0; isMatch != expected {
				programCounter++
			}
		case TEST:
			expected := instruction.getB() != 0
			actual := toBool(vm.GetStack(instruction.getA())).val
			if expected != actual {
				programCounter++
			}
		case TESTSET:
			expected := instruction.getC() != 0
			actual := toBool(vm.GetStack(instruction.getB())).val
			if expected != actual {
				programCounter++
			} else {
				err = vm.SetStack(instruction.getA(), &Boolean{val: actual})
			}
		case LEN:
			b, bK := instruction.getBK()
			val := vm.Get(fn, b, bK)
			if isString(val) {
				err = vm.SetStack(instruction.getA(), &Integer{val: int64(len(val.(*String).val))})
			} else if tbl, isTbl := val.(*Table); isTbl {
				if didDelegate, res, err := vm.delegateMetamethod(metaLen, tbl); err != nil {
					return nil, programCounter, err
				} else if didDelegate && len(res) > 0 {
					if err = vm.SetStack(instruction.getA(), res[0]); err != nil {
						return nil, programCounter, err
					}
				} else {
					if err = vm.SetStack(instruction.getA(), &Integer{val: int64(len(tbl.val))}); err != nil {
						return nil, programCounter, err
					}
				}
			} else {
				err = fmt.Errorf("attempt to get length of a %v value", val.Type())
			}
		case NEWTABLE:
			err = vm.SetStack(instruction.getA(), NewSizedTable(int(instruction.getB()), int(instruction.getC())))
		case GETTABLE:
			keyIdx, keyK := instruction.getCK()
			tbl := vm.GetStack(instruction.getB())
			if val, err := vm.index(tbl, nil, vm.Get(fn, keyIdx, keyK)); err != nil {
				return nil, programCounter, err
			} else if err = vm.SetStack(instruction.getA(), val); err != nil {
				return nil, programCounter, err
			}
		case SETTABLE:
			keyIdx, keyK := instruction.getBK()
			valueIdx, valueK := instruction.getCK()
			err = vm.newIndex(
				vm.GetStack(instruction.getA()),
				nil,
				vm.Get(fn, keyIdx, keyK),
				vm.Get(fn, valueIdx, valueK),
			)
		case SETLIST:
			tbl, ok := vm.GetStack(instruction.getA()).(*Table)
			if !ok {
				return nil, programCounter, fmt.Errorf("attempt to index a %v value", vm.GetStack(instruction.getA()).Type())
			}
			start := instruction.getA() + 1
			nvals := (instruction.getB() - 1)
			if (instruction.getB() - 1) < 0 {
				nvals = int64(len(vm.Stack))
			}
			values := make([]Value, 0, nvals)
			for i := start; i < start+nvals; i++ {
				values = append(values, vm.GetStack(i))
			}
			index := extraArg(int(instruction.getC()))
			ensureSize(&tbl.val, index-1)
			tbl.val = slices.Insert(tbl.val, index, values...)
		case GETUPVAL:
			err = vm.SetStack(instruction.getA(), upvals[instruction.getB()].Get())
		case SETUPVAL:
			upvals[instruction.getB()].Set(vm.GetStack(instruction.getA()))
		case GETTABUP:
			keyIdx, keyK := instruction.getCK()
			tbl := upvals[instruction.getB()].Get()
			if val, err := vm.index(tbl, nil, vm.Get(fn, keyIdx, keyK)); err != nil {
				return nil, programCounter, err
			} else if err = vm.SetStack(instruction.getA(), val); err != nil {
				return nil, programCounter, err
			}
		case SETTABUP:
			keyIdx, keyK := instruction.getBK()
			valueIdx, valueK := instruction.getCK()
			err = vm.newIndex(
				upvals[instruction.getA()].Get(),
				nil,
				vm.Get(fn, keyIdx, keyK),
				vm.Get(fn, valueIdx, valueK),
			)
		case RETURN:
			vm.closeBrokers(openBrokers)
			if err := vm.closeTBC(tbcValues); err != nil {
				return nil, programCounter, err
			}
			nret := (instruction.getB() - 1)
			retVals := vm.truncate(instruction.getA())
			if nret > 0 {
				retVals = ensureLenNil(retVals, int(nret))
			}
			return retVals, programCounter, nil
		case VARARG:
			vm.truncate(instruction.getA())
			if want := instruction.getB() - 1; want > 0 {
				xargs = ensureLenNil(xargs, int(want))
			}
			vm.Stack = append(vm.Stack, xargs...)
		case CALL:
			ifn := instruction.getA()
			nargs := instruction.getB() - 1
			nret := instruction.getC() - 1
			retVals, err := vm.callFn(ifn, nargs)
			if err != nil {
				return nil, programCounter, err
			}
			if nret > 0 && len(retVals) > int(nret) {
				retVals = retVals[:nret]
			} else if len(retVals) < int(nret) {
				retVals = append(retVals, repeat[Value](&Nil{}, int(nret)-len(retVals))...)
			}
			vm.truncate(ifn)
			vm.Stack = append(vm.Stack, retVals...)
		case TAILCALL:
			ifn := int(vm.framePointer)
			cutout(&vm.Stack, ifn, int(vm.framePointer+instruction.getA()+1))
			stackFn := vm.Stack[int(vm.framePointer)-1]
			fn, isCallable := stackFn.(callable)
			if !isCallable {
				return nil, programCounter, fmt.Errorf("expected callable but found %v", stackFn.Type())
			}
			nargs := instruction.getB() - 1
			if nargs < 0 {
				nargs = int64(len(vm.Stack)) - vm.framePointer
			}
			retVals, err := fn.Call(vm, nargs)
			if err != nil {
				return nil, programCounter, err
			}
			truncate(&vm.Stack, ifn)
			vm.Stack = append(vm.Stack, retVals...)
		case CLOSURE:
			cls := fn.FnTable[instruction.getB()]
			closureUpvals := make([]*UpvalueBroker, len(cls.UpIndexes))
			for i, idx := range cls.UpIndexes {
				if idx.fromStack {
					if j, ok := search(openBrokers, int(idx.index), findBroker); ok {
						closureUpvals[i] = openBrokers[j]
					} else {
						newBroker := vm.newUpValueBroker(idx.name, vm.GetStack(int64(idx.index)), int(vm.framePointer)+int(idx.index))
						openBrokers = append(openBrokers, newBroker)
						closureUpvals[i] = newBroker
					}
				} else {
					closureUpvals[i] = upvals[idx.index]
				}
			}
			err = vm.SetStack(instruction.getA(), &Closure{val: cls, upvalues: closureUpvals})
		case SELF:
			tbl := vm.GetStack(instruction.getB()).(*Table)
			keyIdx, keyK := instruction.getCK()
			fn, err := tbl.Index(vm.Get(fn, keyIdx, keyK))
			if err != nil {
				return nil, programCounter, err
			}
			ra := instruction.getA()
			if err = vm.SetStack(ra, fn); err != nil {
				return nil, programCounter, err
			} else if err = vm.SetStack(ra+1, tbl); err != nil {
				return nil, programCounter, err
			}
		case FORPREP:
			ivar := instruction.getA()
			hasFloat := false
			for i := ivar; i < ivar+3; i++ {
				switch vm.GetStack(int64(i)).(type) {
				case *Integer:
				case *Float:
					hasFloat = true
				default:
					return nil, programCounter, fmt.Errorf("non-numeric %v value", forNumNames[i])
				}
			}
			if hasFloat {
				for i := ivar; i < ivar+3; i++ {
					if _, ok := vm.GetStack(int64(i)).(*Integer); !ok {
						if err := vm.SetStack(int64(i), &Float{val: toFloat(vm.GetStack(int64(i)))}); err != nil {
							return nil, programCounter, err
						}
					}
				}
			}
			if toFloat(vm.GetStack(ivar+2)) == 0 {
				return nil, programCounter, fmt.Errorf("0 Step in numerical for")
			}

			i := vm.GetStack(ivar)
			step := vm.GetStack(ivar + 2)
			if iVal, isInt := i.(*Integer); isInt {
				stepVal := step.(*Integer)
				err = vm.SetStack(ivar, &Integer{val: iVal.val - stepVal.val})
			} else {
				iVal := i.(*Float)
				stepVal := step.(*Float)
				err = vm.SetStack(ivar, &Float{val: iVal.val - stepVal.val})
			}
			programCounter += instruction.getsBx()
		case FORLOOP:
			ivar := instruction.getA()
			i := vm.GetStack(ivar)
			limit := vm.GetStack(ivar + 1)
			step := vm.GetStack(ivar + 2)
			if iVal, isInt := i.(*Integer); isInt {
				stepVal := step.(*Integer)
				err = vm.SetStack(ivar, &Integer{val: iVal.val + stepVal.val})
			} else {
				iVal := i.(*Float)
				stepVal := step.(*Float)
				err = vm.SetStack(ivar, &Float{val: iVal.val + stepVal.val})
			}
			i = vm.GetStack(ivar)
			check := (toFloat(step) > 0 && toFloat(i) <= toFloat(limit)) ||
				(toFloat(step) < 0 && toFloat(i) >= toFloat(limit))
			if check {
				programCounter += instruction.getsBx()
			}
		case TFORCALL:
			idx := instruction.getA()
			values, err := vm.callFn(idx, 2)
			if err != nil {
				return nil, programCounter, err
			}
			var ctrl Value = &Nil{}
			if len(values) > 0 {
				ctrl = values[0]
			}
			if err := vm.SetStack(idx+2, ctrl); err != nil {
				return nil, programCounter, err
			}
			for i := 0; i < int(instruction.getB()); i++ {
				var val Value = &Nil{}
				if i < len(values) {
					val = values[i]
				}
				if err := vm.SetStack(idx+int64(i)+3, val); err != nil {
					return nil, programCounter, err
				}
			}
		case TFORLOOP:
			idx := instruction.getA()
			control := vm.GetStack(idx + 1)
			if _, isNil := control.(*Nil); !isNil {
				programCounter += instruction.getsBx()
			}
		default:
			panic("unknown opcode this should never happen")
		}
		if err != nil {
			return nil, programCounter, err
		}
		programCounter++
	}
}

func (vm *VM) callFn(fnR, nargs int64) ([]Value, error) {
	fnVal := vm.GetStack(fnR)
	if nargs < 0 {
		nargs = int64(len(vm.Stack)) - vm.framePointer
	}
	vm.framePointer += fnR + 1
	fn, isCallable := fnVal.(callable)
	if !isCallable {
		method, err := vm.findMetamethod(metaCall, fnVal)
		if err != nil {
			return nil, err
		} else if method != nil {
			fn = method
		} else {
			return nil, fmt.Errorf("expected callable but found %v", vm.GetStack(fnR).Type())
		}
	}
	values, err := fn.Call(vm, nargs)
	vm.framePointer -= fnR + 1
	return values, err
}

func (vm *VM) Get(fn *FuncProto, id int64, isConst bool) Value {
	if isConst {
		return fn.getConst(id)
	}
	return vm.GetStack(id)
}

func (vm *VM) GetStack(id int64) Value {
	if int(vm.framePointer+id) >= len(vm.Stack) || id < 0 || vm.Stack[vm.framePointer+id] == nil {
		return &Nil{}
	}
	return vm.Stack[vm.framePointer+id]
}

func (vm *VM) SetStack(id int64, val Value) error {
	dst := vm.framePointer + id
	if id < 0 {
		return errors.New("cannot address negatively in the stack")
	}
	ensureSizeGrow(&vm.Stack, int(dst))
	vm.Stack[dst] = val
	return nil
}

func (vm *VM) Push(val ...Value) int64 {
	addr := len(vm.Stack)
	vm.Stack = append(vm.Stack, val...)
	return int64(addr)
}

func (vm *VM) truncate(dst int64) []Value {
	return trimEndNil(truncate(&vm.Stack, int(vm.framePointer+dst)))
}

type opFn func(lVal, rVal Value) (Value, error)

func (vm *VM) setABCFn(fp *FuncProto, instruction Bytecode, fn opFn) error {
	b, bK := instruction.getBK()
	c, cK := instruction.getCK()
	val, err := fn(vm.Get(fp, b, bK), vm.Get(fp, c, cK))
	if err != nil {
		return err
	}
	return vm.SetStack(instruction.getA(), val)
}

func (vm *VM) setABCArith(fp *FuncProto, instruction Bytecode, op metaMethod) error {
	return vm.setABCFn(fp, instruction, func(lVal, rVal Value) (Value, error) {
		return vm.arith(op, lVal, rVal)
	})
}

func (vm *VM) arith(op metaMethod, lval, rval Value) (Value, error) {
	if isNumber(lval) && isNumber(rval) {
		switch op {
		case metaBAnd, metaBOr, metaBXOr, metaShl, metaShr, metaBNot:
			return &Integer{val: intArith(op, toInt(lval), toInt(rval))}, nil
		case metaDiv, metaPow:
			return &Float{val: floatArith(op, toFloat(lval), toFloat(rval))}, nil
		default:
			liva, lisInt := lval.(*Integer)
			riva, risInt := rval.(*Integer)
			if lisInt && risInt {
				return &Integer{val: intArith(op, liva.val, riva.val)}, nil
			} else {
				return &Float{val: floatArith(op, toFloat(lval), toFloat(rval))}, nil
			}
		}
	}
	if didDelegate, res, err := vm.delegateMetamethod(op, lval, rval); err != nil {
		return nil, err
	} else if !didDelegate {
		return nil, vm.err("cannot %v %v and %v", op, lval.Type(), rval.Type())
	} else if len(res) > 0 {
		return res[0], nil
	}
	return nil, fmt.Errorf("error object is a nil value")
}

func intArith(op metaMethod, lval, rval int64) int64 {
	switch op {
	case metaAdd:
		return lval + rval
	case metaSub:
		return lval - rval
	case metaMul:
		return lval * rval
	case metaIDiv:
		return lval / rval
	case metaUNM:
		return -lval
	case metaMod:
		return lval % rval
	case metaBAnd:
		return lval & rval
	case metaBOr:
		return lval | rval
	case metaBXOr:
		return lval | rval
	case metaShl:
		return lval << rval
	case metaShr:
		return lval >> rval
	case metaBNot:
		return ^lval
	default:
		panic(fmt.Sprintf("cannot perform float %v op", op))
	}
}

func floatArith(op metaMethod, lval, rval float64) float64 {
	switch op {
	case metaAdd:
		return lval + rval
	case metaSub:
		return lval - rval
	case metaMul:
		return lval * rval
	case metaDiv:
		return lval / rval
	case metaPow:
		return math.Pow(lval, rval)
	case metaIDiv:
		return math.Floor(lval / rval)
	case metaUNM:
		return -lval
	case metaMod:
		return math.Mod(lval, rval)
	default:
		panic(fmt.Sprintf("cannot perform float %v op", op))
	}
}

func (vm *VM) eq(fn *FuncProto, instruction Bytecode) (bool, error) {
	b, bK := instruction.getBK()
	c, cK := instruction.getCK()
	lVal := vm.Get(fn, b, bK)
	rVal := vm.Get(fn, c, cK)

	typeA, typeB := lVal.Type(), rVal.Type()
	if typeA != typeB {
		return false, nil
	}

	switch typeA {
	case "string":
		strA, strB := lVal.(*String), rVal.(*String)
		return strA.val == strB.val, nil
	case "number":
		vA, vB := toFloat(lVal), toFloat(rVal)
		return vA == vB, nil
	case "boolean":
		strA, strB := lVal.(*Boolean), rVal.(*Boolean)
		return strA.val == strB.val, nil
	case "nil":
		return true, nil
	case "table":
		if lVal == rVal {
			return true, nil
		}
		didDelegate, res, err := vm.delegateMetamethod(metaEq, lVal, rVal)
		if err != nil {
			return false, err
		} else if didDelegate && len(res) > 0 {
			return toBool(res[0]).val, nil
		}
		return false, nil
	case "function", "closure":
		return lVal == rVal, nil
	default:
		return false, nil
	}
}

func (vm *VM) compare(op metaMethod, fn *FuncProto, instruction Bytecode) (int, error) {
	b, bK := instruction.getBK()
	c, cK := instruction.getCK()
	lVal, rVal := vm.Get(fn, b, bK), vm.Get(fn, c, cK)

	if isNumber(lVal) && isNumber(rVal) {
		vA, vB := toFloat(lVal), toFloat(rVal)
		if vA < vB {
			return -1, nil
		} else if vA > vB {
			return 1, nil
		}
		return 0, nil
	} else if isString(lVal) && isString(rVal) {
		strA, strB := lVal.(*String), rVal.(*String)
		return strings.Compare(strA.val, strB.val), nil
	} else if didDelegate, res, err := vm.delegateMetamethod(op, lVal, rVal); err != nil {
		return 0, err
	} else if !didDelegate {
		return 0, vm.err("cannot %v %v and %v", op, lVal.Type(), rVal.Type())
	} else if len(res) > 0 {
		if toBool(res[0]).val {
			return -1, nil
		}
		return 1, nil
	}
	return 0, fmt.Errorf("attempted to compare two %v and %v values", lVal.Type(), rVal.Type())
}

func (vm *VM) concat(instruction Bytecode) error {
	b := instruction.getB()
	c := instruction.getC()
	if c < b {
		c = b + 1
	}

	concatBinOp := func(a, b Value) (Value, error) {
		aCoercable := isString(a) || isNumber(a)
		bCoercable := isString(b) || isNumber(b)
		if aCoercable && bCoercable {
			return &String{val: a.String() + b.String()}, nil
		} else if didDelegate, res, err := vm.delegateMetamethod(metaConcat, a, b); err != nil {
			return nil, err
		} else if didDelegate && len(res) > 0 {
			return res[0], nil
		} else {
			return nil, fmt.Errorf("attempted to concatenate a %v value", b.Type())
		}
	}

	result := vm.GetStack(b)
	for i := b + 1; i <= c; i++ {
		if str, err := concatBinOp(result, vm.GetStack(i)); err != nil {
			return err
		} else {
			result = str
		}
	}
	return vm.SetStack(instruction.getA(), &String{val: result.String()})
}

func (vm *VM) index(source, table, key Value) (Value, error) {
	if table == nil {
		table = source
	}
	tbl, isTable := table.(*Table)
	if isTable {
		res, err := tbl.Index(key)
		if err != nil {
			return nil, err
		} else if _, isNil := res.(*Nil); !isNil {
			return res, nil
		}
	}
	metatable := table.Meta()
	if metatable != nil && metatable.hashtable[string(metaIndex)] != nil {
		switch metaVal := metatable.hashtable[string(metaIndex)].(type) {
		case callable:
			res, err := vm.Call(metaVal, []Value{source, key})
			if err != nil {
				return nil, err
			} else if len(res) > 0 {
				return res[0], nil
			} else {
				return &Nil{}, nil
			}
		default:
			return vm.index(source, metaVal, key)
		}
	}
	if isTable {
		return &Nil{}, nil
	}
	return nil, fmt.Errorf("attempt to index a %v value", table.Type())
}

func (vm *VM) newIndex(source, table, key, value Value) error {
	if table == nil {
		table = source
	}
	tbl, isTbl := table.(*Table)
	if isTbl {
		res, err := tbl.Index(key)
		if err != nil {
			return err
		} else if _, isNil := res.(*Nil); !isNil {
			return tbl.SetIndex(key, value)
		}
	}
	metatable := table.Meta()
	if metatable != nil && metatable.hashtable[string(metaNewIndex)] != nil {
		switch metaVal := metatable.hashtable[string(metaNewIndex)].(type) {
		case callable:
			_, err := vm.Call(metaVal, []Value{table, key})
			if err != nil {
				return err
			}
			return nil
		default:
			return vm.newIndex(source, metaVal, key, value)
		}
	}
	if isTbl {
		return tbl.SetIndex(key, value)
	}
	return fmt.Errorf("attempt to index a %v value", table.Type())
}

func (vm *VM) findMetamethod(op metaMethod, params ...Value) (callable, error) {
	var method callable
	for _, val := range params {
		if val.Meta() != nil && val.Meta().hashtable[string(op)] != nil {
			metamethod := val.Meta().hashtable[string(op)]
			fn, isCallable := metamethod.(callable)
			if !isCallable {
				return nil, vm.err("expected %v metamethod to be callable but found %v", op, metamethod.Type())
			}
			method = fn
			break
		}
	}
	return method, nil
}

func (vm *VM) delegateMetamethod(op metaMethod, params ...Value) (bool, []Value, error) {
	method, err := vm.findMetamethod(op, params...)
	if err != nil {
		return false, nil, err
	} else if method != nil {
		ret, err := vm.Call(method, params)
		return true, ret, err
	}
	// unable to delegate
	return false, nil, nil
}

func (vm *VM) Call(fn callable, params []Value) ([]Value, error) {
	val, isValue := fn.(Value)
	if !isValue {
		return nil, fmt.Errorf("callable is not value")
	}
	ifn := vm.Push(val)
	vm.Push(params...)
	retVals, err := vm.callFn(ifn-vm.framePointer, int64(len(params)))
	if err != nil {
		return nil, err
	}
	vm.truncate(ifn)
	return retVals, nil
}

func (vm *VM) closeBrokers(brokers []*UpvalueBroker) {
	for _, broker := range brokers {
		broker.Close()
	}
}

func (vm *VM) closeTBC(tbcs []int64) error {
	for _, idx := range tbcs {
		if didDelegate, _, err := vm.delegateMetamethod(metaClose, vm.GetStack(idx)); err != nil {
			return err
		} else if !didDelegate {
			return fmt.Errorf("__close not defined on closable table")
		}
	}
	return nil
}

func (vm *VM) newUpValueBroker(name string, val Value, index int) *UpvalueBroker {
	return &UpvalueBroker{
		stack: &vm.Stack,
		name:  name,
		val:   val,
		index: index,
		open:  true,
	}
}

func (b *UpvalueBroker) Get() Value {
	if b.open {
		return (*b.stack)[b.index]
	}
	return b.val
}

func (b *UpvalueBroker) Set(val Value) {
	if b.open {
		(*b.stack)[b.index] = val
	}
	b.val = val
}

func (b *UpvalueBroker) Close() {
	b.val = (*b.stack)[b.index]
	b.open = false
}
