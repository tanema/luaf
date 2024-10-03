package shine

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
		val   Value
	}
	VM struct {
		base  int64
		Stack []Value
		env   *Table
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
			fmt.Fprint(&strBuilder, arg)
		}
		fmt.Println(strBuilder.String())
		return nil, nil
	}}
	return &VM{
		Stack: []Value{env},
		base:  1,
		env:   env,
	}
}

func (vm *VM) err(tmpl string, args ...any) error {
	return &RuntimeErr{msg: fmt.Sprintf(tmpl, args...)}
}

func (vm *VM) Env() *Table {
	return vm.env
}

func (vm *VM) Eval(fn *FuncProto) error {
	return vm.eval(fn, []Broker{{val: vm.env, open: true, name: "_ENV", index: 0}})
}

func (vm *VM) eval(fn *FuncProto, upvals []Broker) error {
	var programCounter int64
	xargs := vm.truncate(vm.base + int64(fn.Arity))
	openBrokers := []Broker{}
	for {
		var err error
		if int64(len(fn.ByteCodes)) <= programCounter {
			return nil
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
		case LOADINT:
			err = vm.SetStack(instruction.getA(), &Integer{val: instruction.getBx()})
		case LOADNIL:
			a := instruction.getA()
			b := instruction.getBx()
			for i := a; i <= a+b; i++ {
				if err = vm.SetStack(i, &Nil{}); err != nil {
					return err
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
			err = vm.setABCFn(fn, instruction, vm.binOp(func(x, y int64) Value { return &Integer{val: x / y} }, func(x, y float64) Value { return &Float{val: math.Floor(x / y)} }))
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
			for i := b; b < c; i++ {
				fmt.Fprint(&strBuilder, vm.GetStack(i).String())
			}
			err = vm.SetStack(instruction.getA(), &String{val: strBuilder.String()})
		case JMP: // TODO if A is not 0 then upvalues need to be closed
			programCounter += instruction.getsBx()
		case EQ:
			expected := instruction.getA() != 0
			isEq, err := vm.eq(fn, instruction)
			if err != nil {
				return err
			} else if isEq != expected {
				programCounter++
			}
		case LT:
			expected := instruction.getA() != 0
			res, err := vm.compare(fn, instruction)
			if err != nil {
				return err
			} else if isMatch := res < 0; isMatch != expected {
				programCounter++
			}
		case LE:
			expected := instruction.getA() != 0
			res, err := vm.compare(fn, instruction)
			if err != nil {
				return err
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
				return fmt.Errorf("attempt to index a %v value", vm.GetStack(instruction.getA()).Type())
			}
			val, err := tbl.Index(vm.Get(fn, keyIdx, keyK))
			if err != nil {
				return err
			}
			err = vm.SetStack(instruction.getA(), val)
		case SETTABLE:
			tbl, ok := vm.GetStack(instruction.getA()).(*Table)
			if !ok {
				return fmt.Errorf("attempt to index a %v value", vm.GetStack(instruction.getA()).Type())
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
				return fmt.Errorf("attempt to index a %v value", vm.GetStack(instruction.getA()).Type())
			}
			start := instruction.getA() + 1
			end := instruction.getB() - 1
			if end < 0 {
				end = int64(len(vm.Stack)) - (vm.base + start)
			}
			values := make([]Value, 0, end+1)
			for i := start; i <= end; i++ {
				values = append(values, vm.GetStack(i))
			}
			tbl.val = slices.Insert(tbl.val, int(instruction.getC()), values...)
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
		case GETUPVAL:
			err = vm.SetStack(instruction.getA(), upvals[instruction.getB()].Get(vm.Stack))
		case SETUPVAL:
			upvals[instruction.getB()].Set(vm.Stack, vm.GetStack(instruction.getA()))
		case GETTABUP:
			upval := upvals[instruction.getB()].Get(vm.Stack)
			tbl, ok := upval.(*Table)
			if !ok {
				return fmt.Errorf("cannot index upvalue type %v", upval.Type())
			}
			c, cK := instruction.getCK()
			val, err := tbl.Index(vm.Get(fn, c, cK))
			if err != nil {
				return err
			}
			err = vm.SetStack(instruction.getA(), val)
		case SETTABUP:
			upval := upvals[instruction.getA()].Get(vm.Stack)
			tbl, ok := upval.(*Table)
			if !ok {
				return fmt.Errorf("cannot index upvalue type %v", upval.Type())
			}
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			err = tbl.SetIndex(vm.Get(fn, b, bK), vm.Get(fn, c, cK))
		case CALL:
			fnR := instruction.getA()
			callable := vm.GetStack(fnR)
			vm.base += fnR + 1

			nargs := instruction.getB() - 1
			if nargs < 0 {
				nargs = int64(len(vm.Stack)) - vm.base
			}
			args := []Value{}
			for _, val := range vm.Stack[vm.base : vm.base+nargs] {
				if val != nil {
					args = append(args, val)
				}
			}

			switch closure := callable.(type) {
			case *Closure:
				if err := vm.eval(closure.val, closure.upvalues); err != nil {
					return err
				}
			case *ExternFunc:
				_, err := closure.val(args)
				if err != nil {
					return err
				}
			default:
				return fmt.Errorf("expected callable but found %v", callable.Type())
			}
			vm.base -= fnR + 1
		case CLOSURE:
			cls := fn.FnTable[instruction.getB()]
			closureUpvals := make([]Broker, len(cls.UpIndexes))
			for i, idx := range cls.UpIndexes {
				if idx.fromStack {
					if j, ok := search(openBrokers, int(idx.index), findBroker); ok {
						closureUpvals[i] = openBrokers[j]
					} else {
						newBroker := Broker{val: vm.GetStack(int64(idx.index)), open: true, index: int(vm.base) + int(idx.index), name: idx.name}
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
				return err
			}
			ra := instruction.getA()
			vm.SetStack(ra, fn)
			vm.SetStack(ra+1, tbl)
		case RETURN:
			// RETURN  A B return R(A), ... ,R(A+B-2)
			// First OP_RETURN closes any open upvalues
			// b = 0 : the set of values range from R(A) to the top of the stack and
			//         the previous instruction (which must be either OP_CALL or OP_VARARG )
			//         would have set L->top to indicate how many values to return.
			//         The number of values to be returned in this case is R(A) to L->top.
			//  >= 1 : there are (B-1) return values, located in consecutive registers from R(A) onwards.
		case TAILCALL:
		case FORLOOP:
		case FORPREP:
		case TFORLOOP:
		case TFORPREP:
		default:
		}
		if err != nil {
			return err
		}
		programCounter++
	}
}

func (vm *VM) Get(fn *FuncProto, id int64, isConst bool) Value {
	if isConst {
		return fn.getConst(id)
	}
	return vm.GetStack(id)
}

func (vm *VM) GetStack(id int64) Value {
	if int(vm.base+id) >= len(vm.Stack) || id < 0 || vm.Stack[vm.base+id] == nil {
		return &Nil{}
	}
	return vm.Stack[vm.base+id]
}

func (vm *VM) SetStack(id int64, val Value) error {
	dst := vm.base + id
	if int(dst) >= len(vm.Stack) {
		newStack := make([]Value, 2*len(vm.Stack)+1)
		copy(newStack, vm.Stack)
		vm.Stack = newStack
	} else if id < 0 {
		return errors.New("cannot address negatively in the stack")
	}
	vm.Stack[dst] = val
	return nil
}

func (vm *VM) truncate(dst int64) []Value {
	vm.fillStackNil(int(dst))
	out := vm.Stack[dst:]
	vm.Stack = vm.Stack[:dst]
	return out
}

func (vm *VM) fillStackNil(dst int) {
	idx := vm.base + int64(dst)
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
		//TODO
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

func (b *Broker) Get(stack []Value) Value {
	if b.open {
		return stack[b.index]
	}
	return b.val
}

func (b *Broker) Set(stack []Value, val Value) {
	if b.open {
		stack[b.index] = val
	}
	b.val = val
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
