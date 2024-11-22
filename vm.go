package luaf

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"
)

type (
	Broker struct {
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
	values, _, err := vm.eval(fn, []*Broker{vm.newValueBroker("_ENV", vm.env, 0)})
	return values, err
}

func (vm *VM) eval(fn *FuncProto, upvals []*Broker) ([]Value, int64, error) {
	var programCounter int64
	xargs := vm.truncate(int64(fn.Arity))
	openBrokers := []*Broker{}

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
			err = vm.setABCFn(fn, instruction, vm.binOp("__add", func(x, y int64) Value { return &Integer{val: x + y} }, func(x, y float64) Value { return &Float{val: x + y} }))
		case SUB:
			err = vm.setABCFn(fn, instruction, vm.binOp("__sub", func(x, y int64) Value { return &Integer{val: x - y} }, func(x, y float64) Value { return &Float{val: x - y} }))
		case MUL:
			err = vm.setABCFn(fn, instruction, vm.binOp("__mul", func(x, y int64) Value { return &Integer{val: x * y} }, func(x, y float64) Value { return &Float{val: x * y} }))
		case DIV:
			err = vm.setABCFn(fn, instruction, vm.binOp("__div", func(x, y int64) Value { return &Integer{val: x / y} }, func(x, y float64) Value { return &Float{val: x / y} }))
		case MOD:
			err = vm.setABCFn(fn, instruction, vm.binOp("__mod", func(x, y int64) Value { return &Integer{val: x % y} }, func(x, y float64) Value { return &Float{val: math.Mod(x, y)} }))
		case POW:
			err = vm.setABCFn(fn, instruction, vm.binOp("__pow", func(x, y int64) Value { return &Integer{val: x ^ y} }, func(x, y float64) Value { return &Float{val: math.Pow(x, y)} }))
		case IDIV:
			err = vm.setABCFn(fn, instruction, vm.binOp("__idiv", func(x, y int64) Value { return &Integer{val: x / y} }, func(x, y float64) Value { return &Integer{val: int64(math.Floor(x / y))} }))
		case BAND:
			err = vm.setABCFn(fn, instruction, vm.ibinOp("__band", func(x, y int64) Value { return &Integer{val: x & y} }))
		case BOR:
			err = vm.setABCFn(fn, instruction, vm.ibinOp("__bor", func(x, y int64) Value { return &Integer{val: x | y} }))
		case BXOR:
			err = vm.setABCFn(fn, instruction, vm.ibinOp("__bxor", func(x, y int64) Value { return &Integer{val: x ^ y} }))
		case SHL:
			err = vm.setABCFn(fn, instruction, vm.ibinOp("__shl", func(x, y int64) Value { return &Integer{val: x << y} }))
		case SHR:
			err = vm.setABCFn(fn, instruction, vm.ibinOp("__shr", func(x, y int64) Value { return &Integer{val: x >> y} }))
		case UNM:
			err = vm.setABCFn(fn, instruction, vm.binOp("__unm", func(x, y int64) Value { return &Integer{val: -x} }, func(x, y float64) Value { return &Float{val: -x} }))
		case BNOT:
			err = vm.setABCFn(fn, instruction, vm.ibinOp("__bnot", func(x, y int64) Value { return &Integer{val: ^x} }))
		case NOT:
			err = vm.setABCFn(fn, instruction, func(lVal, rVal Value) (Value, error) { return toBool(lVal).Not(), nil })
		case CONCAT:
			b := instruction.getB()
			c := instruction.getC()
			var strBuilder strings.Builder
			if c < b {
				c = b
			}
			for i := b; i <= c; i++ {
				if _, err := fmt.Fprint(&strBuilder, vm.GetStack(i).String()); err != nil {
					return nil, programCounter, err
				}
			}
			err = vm.SetStack(instruction.getA(), &String{val: strBuilder.String()})
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
			res, err := vm.compare(fn, instruction)
			if err != nil {
				return nil, programCounter, err
			} else if isMatch := res < 0; isMatch != expected {
				programCounter++
			}
		case LE:
			expected := instruction.getA() != 0
			res, err := vm.compare(fn, instruction)
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
			switch tval := val.(type) {
			case *String:
				err = vm.SetStack(instruction.getA(), &Integer{val: int64(len(tval.val))})
			case *Table:
				err = vm.SetStack(instruction.getA(), &Integer{val: int64(len(tval.val))})
			default:
				err = fmt.Errorf("attempt to get length of a %v value", val.Type())
			}
		case NEWTABLE:
			err = vm.SetStack(instruction.getA(), NewSizedTable(int(instruction.getB()), int(instruction.getC())))
		case GETTABLE:
			tbl, ok := vm.GetStack(instruction.getB()).(*Table)
			if !ok {
				return nil, programCounter, fmt.Errorf("attempt to index a %v value", vm.GetStack(instruction.getA()).Type())
			}
			keyIdx, keyK := instruction.getCK()
			val, err := tbl.Index(vm.Get(fn, keyIdx, keyK))
			if err != nil {
				return nil, programCounter, err
			} else if err = vm.SetStack(instruction.getA(), val); err != nil {
				return nil, programCounter, err
			}
		case SETTABLE:
			tbl, ok := vm.GetStack(instruction.getA()).(*Table)
			if !ok {
				return nil, programCounter, fmt.Errorf("attempt to index a %v value", vm.GetStack(instruction.getA()).Type())
			}
			keyIdx, keyK := instruction.getBK()
			valueIdx, valueK := instruction.getCK()
			err = tbl.SetIndex(vm.Get(fn, keyIdx, keyK), vm.Get(fn, valueIdx, valueK))
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
			upval := upvals[instruction.getB()].Get()
			tbl, ok := upval.(*Table)
			if !ok {
				return nil, programCounter, fmt.Errorf("cannot index upvalue type %v", upval.Type())
			}
			c, cK := instruction.getCK()
			val, err := tbl.Index(vm.Get(fn, c, cK))
			if err != nil {
				return nil, programCounter, err
			} else if err = vm.SetStack(instruction.getA(), val); err != nil {
				return nil, programCounter, err
			}
		case SETTABUP:
			upval := upvals[instruction.getA()].Get()
			tbl, ok := upval.(*Table)
			if !ok {
				return nil, programCounter, fmt.Errorf("cannot index upvalue type %v", upval.Type())
			}
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			err = tbl.SetIndex(vm.Get(fn, b, bK), vm.Get(fn, c, cK))
		case RETURN:
			vm.closeBrokers(openBrokers)
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
			closureUpvals := make([]*Broker, len(cls.UpIndexes))
			for i, idx := range cls.UpIndexes {
				if idx.fromStack {
					if j, ok := search(openBrokers, int(idx.index), findBroker); ok {
						closureUpvals[i] = openBrokers[j]
					} else {
						newBroker := vm.newValueBroker(idx.name, vm.GetStack(int64(idx.index)), int(vm.framePointer)+int(idx.index))
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
	fn, isCallable := vm.GetStack(fnR).(callable)
	if !isCallable {
		return nil, fmt.Errorf("expected callable but found %v", vm.GetStack(fnR).Type())
	}
	vm.framePointer += fnR + 1
	if nargs < 0 {
		nargs = int64(len(vm.Stack)) - vm.framePointer
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

func (vm *VM) binOp(op string, ifn func(a, b int64) Value, ffn func(a, b float64) Value) opFn {
	return func(lVal, rVal Value) (Value, error) {
		switch lVal.(type) {
		case *Integer:
			switch rVal.(type) {
			case *Integer:
				return ifn(toInt(lVal), toInt(rVal)), nil
			case *Float:
				return ffn(toFloat(lVal), toFloat(rVal)), nil
			}
		case *Float:
			switch rVal.(type) {
			case *Integer, *Float:
				return ffn(toFloat(lVal), toFloat(rVal)), nil
			}
		}
		// if none of the operations were valid then we should try to delegate
		didDelegate, res, err := vm.delegateMetamethod(op, lVal, rVal)
		if err != nil {
			return nil, err
		}
		if !didDelegate {
			return nil, vm.err("cannot %v %v and %v", op, lVal.Type(), rVal.Type())
		}
		if len(res) > 0 {
			return res[0], nil
		}
		return &Integer{val: 0}, nil
	}
}

func (vm *VM) ibinOp(op string, ifn func(a, b int64) Value) opFn {
	return func(lVal, rVal Value) (Value, error) {
		switch lVal.(type) {
		case *Integer, *Float:
			return ifn(toInt(lVal), toInt(rVal)), nil
		default:
			return nil, vm.err("cannot %v %v and %v", op, lVal.Type(), rVal.Type())
		}
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
		didDelegate, res, err := vm.delegateMetamethod("__eq", lVal, rVal)
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

func (vm *VM) compare(fn *FuncProto, instruction Bytecode) (int, error) {
	b, bK := instruction.getBK()
	c, cK := instruction.getCK()
	lVal := vm.Get(fn, b, bK)
	rVal := vm.Get(fn, c, cK)

	typeA, typeB := lVal.Type(), rVal.Type()
	if typeA != typeB {
		return 0, fmt.Errorf("attempt to compare %v with %v", typeA, typeB)
	}

	switch typeA {
	case "string":
		strA, strB := lVal.(*String), rVal.(*String)
		return strings.Compare(strA.val, strB.val), nil
	case "number":
		vA, vB := toFloat(lVal), toFloat(rVal)
		if vA < vB {
			return -1, nil
		} else if vA > vB {
			return 1, nil
		}
		return 0, nil
	default:
		return 0, fmt.Errorf("attempted to compare two %v values", typeA)
	}
}

func (vm *VM) delegateMetamethod(op string, lVal, rVal Value) (bool, []Value, error) {
	lmeta, rmeta := lVal.Meta(), rVal.Meta()
	var method callable
	if lmeta != nil && lmeta.hashtable[op] != nil {
		metamethod := lmeta.hashtable[op]
		fn, isCallable := metamethod.(callable)
		if !isCallable {
			return false, nil, vm.err("expected %v metamethod to be callable but found %v", op, metamethod.Type())
		}
		method = fn
	} else if rmeta != nil && rmeta.hashtable[op] != nil {
		metamethod := rmeta.hashtable[op]
		fn, isCallable := metamethod.(callable)
		if !isCallable {
			return false, nil, vm.err("expected %v metamethod to be callable but found %v", op, metamethod.Type())
		}
		method = fn
	}
	if method != nil {
		ret, err := vm.Call(method, []Value{lVal, rVal})
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

func (vm *VM) closeBrokers(brokers []*Broker) {
	for _, broker := range brokers {
		broker.Close()
	}
}

func (vm *VM) newValueBroker(name string, val Value, index int) *Broker {
	return &Broker{
		stack: &vm.Stack,
		name:  name,
		val:   val,
		index: index,
		open:  true,
	}
}

func (b *Broker) Get() Value {
	if b.open {
		return (*b.stack)[b.index]
	}
	return b.val
}

func (b *Broker) Set(val Value) {
	if b.open {
		(*b.stack)[b.index] = val
	}
	b.val = val
}

func (b *Broker) Close() {
	b.val = (*b.stack)[b.index]
	b.open = false
}
