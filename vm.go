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
		stack []Value
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

func (err *RuntimeErr) Error() string {
	return err.msg
}

func NewVM() *VM {
	env := NewTable()
	env.hashtable["print"] = &ExternFunc{func(args []Value) ([]Value, error) {
		var strBuilder strings.Builder
		for _, arg := range args {
			if _, err := fmt.Fprint(&strBuilder, arg); err != nil {
				return nil, err
			}
		}
		fmt.Println(strBuilder.String())
		return nil, nil
	}}
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
	values, _, err := vm.eval(fn, []Broker{vm.newValueBroker("_ENV", vm.env, 0)})
	return values, err
}

func (vm *VM) eval(fn *FuncProto, upvals []Broker) ([]Value, int64, error) {
	var programCounter int64
	xargs := vm.truncate(int64(fn.Arity))
	openBrokers := []Broker{}
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
			err = vm.setABCFn(fn, instruction, vm.binOp(func(x, y int64) Value { return &Integer{val: x + y} }, func(x, y float64) Value { return &Float{val: x + y} }))
		case SUB:
			err = vm.setABCFn(fn, instruction, vm.binOp(func(x, y int64) Value { return &Integer{val: x - y} }, func(x, y float64) Value { return &Float{val: x - y} }))
		case MUL:
			err = vm.setABCFn(fn, instruction, vm.binOp(func(x, y int64) Value { return &Integer{val: x * y} }, func(x, y float64) Value { return &Float{val: x * y} }))
		case DIV:
			err = vm.setABCFn(fn, instruction, vm.binOp(func(x, y int64) Value { return &Integer{val: x / y} }, func(x, y float64) Value { return &Float{val: x / y} }))
		case MOD:
			err = vm.setABCFn(fn, instruction, vm.binOp(func(x, y int64) Value { return &Integer{val: x % y} }, func(x, y float64) Value { return &Float{val: math.Mod(x, y)} }))
		case POW:
			err = vm.setABCFn(fn, instruction, vm.binOp(func(x, y int64) Value { return &Integer{val: x ^ y} }, func(x, y float64) Value { return &Float{val: math.Pow(x, y)} }))
		case IDIV:
			err = vm.setABCFn(fn, instruction, vm.binOp(func(x, y int64) Value { return &Integer{val: x / y} }, func(x, y float64) Value { return &Integer{val: int64(math.Floor(x / y))} }))
		case BAND:
			err = vm.setABCFn(fn, instruction, vm.ibinOp(func(x, y int64) Value { return &Integer{val: x & y} }))
		case BOR:
			err = vm.setABCFn(fn, instruction, vm.ibinOp(func(x, y int64) Value { return &Integer{val: x | y} }))
		case BXOR:
			err = vm.setABCFn(fn, instruction, vm.ibinOp(func(x, y int64) Value { return &Integer{val: x ^ y} }))
		case SHL:
			err = vm.setABCFn(fn, instruction, vm.ibinOp(func(x, y int64) Value { return &Integer{val: x << y} }))
		case SHR:
			err = vm.setABCFn(fn, instruction, vm.ibinOp(func(x, y int64) Value { return &Integer{val: x >> y} }))
		case UNM:
			err = vm.setABCFn(fn, instruction, vm.binOp(func(x, y int64) Value { return &Integer{val: -x} }, func(x, y float64) Value { return &Float{val: -x} }))
		case BNOT:
			err = vm.setABCFn(fn, instruction, vm.ibinOp(func(x, y int64) Value { return &Integer{val: ^x} }))
		case NOT:
			err = vm.setABCFn(fn, instruction, func(lVal, rVal Value) (Value, error) { return lVal.Bool().Not(), nil })
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
			offset := instruction.getsBx()
			if instruction.getA() != 0 {
				vm.closeBrokers(openBrokers)
			}
			programCounter += offset
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
			actual := vm.GetStack(instruction.getA()).Bool().val
			if expected != actual {
				programCounter++
			}
		case TESTSET:
			expected := instruction.getC() != 0
			actual := vm.GetStack(instruction.getB()).Bool().val
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
			keyIdx, keyK := instruction.getCK()
			tbl, ok := vm.GetStack(instruction.getB()).(*Table)
			if !ok {
				return nil, programCounter, fmt.Errorf("attempt to index a %v value", vm.GetStack(instruction.getA()).Type())
			}
			val, err := tbl.Index(vm.Get(fn, keyIdx, keyK))
			if err != nil {
				return nil, programCounter, err
			}
			err = vm.SetStack(instruction.getA(), val)
		case SETTABLE:
			tbl, ok := vm.GetStack(instruction.getA()).(*Table)
			if !ok {
				return nil, programCounter, fmt.Errorf("attempt to index a %v value", vm.GetStack(instruction.getA()).Type())
			}
			keyIdx, keyK := instruction.getBK()
			valueIdx, valueK := instruction.getCK()
			err = tbl.SetIndex(vm.Get(fn, keyIdx, keyK), vm.Get(fn, valueIdx, valueK))
		case SETLIST:
			// TODO Extended C usage is not supported yet
			// If C is 0, the next instruction is cast as an integer, and used as the C value.
			// This happens only when operand C is unable to encode the block number,
			// i.e. when C > 511, equivalent to an array index greater than 25550.
			tbl, ok := vm.GetStack(instruction.getA()).(*Table)
			if !ok {
				return nil, programCounter, fmt.Errorf("attempt to index a %v value", vm.GetStack(instruction.getA()).Type())
			}
			start := instruction.getA() + 1
			end := instruction.getB() - 1
			if end < 0 {
				end = int64(len(vm.Stack)) - (vm.framePointer + start)
			}
			values := make([]Value, 0, end+1)
			for i := start; i <= end; i++ {
				values = append(values, vm.GetStack(i))
			}
			index := int(instruction.getC())
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
			}
			err = vm.SetStack(instruction.getA(), val)
		case SETTABUP:
			upval := upvals[instruction.getA()].Get()
			tbl, ok := upval.(*Table)
			if !ok {
				return nil, programCounter, fmt.Errorf("cannot index upvalue type %v", upval.Type())
			}
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			err = tbl.SetIndex(vm.Get(fn, b, bK), vm.Get(fn, c, cK))
		case VARARG:
			vm.truncate(instruction.getA())
			want := instruction.getB()
			if diff := int(want) - len(xargs); diff > 0 {
				for i := 0; i <= diff; i++ {
					xargs = append(xargs, &Nil{})
				}
			} else if int(want) < len(xargs) && want != 0 {
				xargs = xargs[:want]
			}
			vm.Stack = append(vm.Stack, xargs...)
		case CALL:
			ifn := instruction.getA()
			nargs := instruction.getB() - 1
			nret := instruction.getC()
			retVals, err := vm.callFn(ifn, nargs)
			if err != nil {
				return nil, programCounter, err
			}
			vm.truncate(ifn)
			if len(retVals) > 0 {
				if nret > 0 && len(retVals) > int(nret) {
					retVals = retVals[:nret-1]
				}
				vm.Stack = append(vm.Stack, retVals...)
			}
		case CLOSURE:
			cls := fn.FnTable[instruction.getB()]
			closureUpvals := make([]Broker, len(cls.UpIndexes))
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
			vm.SetStack(ra, fn)
			vm.SetStack(ra+1, tbl)
		case TAILCALL:
			// TODO how to do this without messing up framePointer
			// err = vm.callFn(instruction.getA(), instruction.getB()-1)
		case RETURN:
			vm.closeBrokers(openBrokers)
			nret := (instruction.getB() - 1)
			retVals := vm.truncate(instruction.getA())
			if len(retVals) > int(nret) {
				retVals = retVals[:nret]
			} else if len(retVals) < int(nret) {
				for i := 0; i < int(nret)-len(retVals); i++ {
					retVals = append(retVals, &Nil{})
				}
			}
			return retVals, programCounter, nil
		case FORLOOP:
			// ivar := instruction.getA()
			// loopVar := vm.Get(ivar)
			// limit := vm.Get(ivar + 1)
			// step := vm.Get(ivar + 2)
			// jmp := instruction.getsBx()
			// programCounter += jmp
		case FORPREP:
		case TFORLOOP:
		case TFORPREP:
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

func (vm *VM) truncate(dst int64) []Value {
	vm.fillStackNil(int(dst))
	return truncate(&vm.Stack, int(vm.framePointer+dst))
}

func (vm *VM) fillStackNil(dst int) {
	idx := vm.framePointer + int64(dst)
	if diff := idx - int64(len(vm.Stack)-1); diff > 0 {
		for i := 0; i < int(diff); i++ {
			vm.Stack = append(vm.Stack, &Nil{})
		}
	}
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

func (vm *VM) binOp(ifn func(a, b int64) Value, ffn func(a, b float64) Value) opFn {
	return func(lVal, rVal Value) (Value, error) {
		switch lVal.(type) {
		case *Integer:
			switch rVal.(type) {
			case *Integer:
				val := ifn(lVal.Val().(int64), rVal.Val().(int64))
				return val, nil
			case *Float:
				val := ffn(float64(lVal.Val().(int64)), rVal.Val().(float64))
				return val, nil
			}
		case *Float:
			switch rVal.(type) {
			case *Integer:
				val := ffn(lVal.Val().(float64), float64(rVal.Val().(int64)))
				return val, nil
			case *Float:
				val := ffn(lVal.Val().(float64), rVal.Val().(float64))
				return val, nil
			}
		}
		return nil, vm.err("cannot <> %v and %v", lVal.Type(), rVal.Type())
	}
}

func (vm *VM) ibinOp(ifn func(a, b int64) Value) opFn {
	return func(lVal, rVal Value) (Value, error) {
		switch lVal.(type) {
		case *Integer:
			switch rVal.(type) {
			case *Integer:
				val := ifn(lVal.Val().(int64), rVal.Val().(int64))
				return val, nil
			case *Float:
				val := ifn(lVal.Val().(int64), int64(rVal.Val().(float64)))
				return val, nil
			}
		case *Float:
			switch rVal.(type) {
			case *Integer:
				val := ifn(int64(lVal.Val().(float64)), rVal.Val().(int64))
				return val, nil
			case *Float:
				val := ifn(int64(lVal.Val().(float64)), int64(rVal.Val().(float64)))
				return val, nil
			}
		}
		return nil, vm.err("cannot <> %v and %v", lVal.Type(), rVal.Type())
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
	case "table", "function", "closure":
		fallthrough
	default:
		return false, fmt.Errorf("cannot eq %v right now", typeA)
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

func (vm *VM) closeBrokers(brokers []Broker) {
	for _, broker := range brokers {
		broker.Close(vm.Stack)
	}
}

func (vm *VM) newValueBroker(name string, val Value, index int) Broker {
	return Broker{
		stack: vm.Stack,
		name:  name,
		val:   val,
		index: index,
		open:  true,
	}
}

func (b *Broker) Get() Value {
	if b.open {
		return b.stack[b.index]
	}
	return b.val
}

func (b *Broker) Set(val Value) {
	if b.open {
		b.stack[b.index] = val
	}
	b.val = val
}

func (b *Broker) Close(stack []Value) {
	b.val = stack[b.index]
	b.open = false
}

func toFloat(val Value) float64 {
	switch tval := val.(type) {
	case *Integer:
		return float64(tval.val)
	case *Float:
		return tval.val
	default:
		return math.NaN()
	}
}
