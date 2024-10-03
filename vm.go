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
		pc    int64
		base  int64
		top   int64
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
	env.SetKey("print", &ExternFunc{func(args []Value) ([]Value, error) {
		var strBuilder strings.Builder
		for _, arg := range args {
			fmt.Fprint(&strBuilder, arg.String())
		}
		fmt.Println(strBuilder.String())
		return nil, nil
	}})
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
	xargs := vm.truncate(vm.base + int64(fn.Arity))
	vm.top = vm.base + int64(fn.Arity)
	openBrokers := []Broker{}
	for {
		var err error
		if int64(len(fn.ByteCodes)) <= vm.pc {
			return nil
		}
		instruction := fn.ByteCodes[vm.pc]
		switch instruction.op() {
		case MOVE:
			b, _ := instruction.getB()
			err = vm.SetStack(instruction.getA(), vm.GetStack(b))
		case LOADK:
			val, err := fn.getConst(instruction.getBx())
			if err != nil {
				return err
			}
			err = vm.SetStack(instruction.getA(), val)
		case LOADBOOL:
			b, _ := instruction.getB()
			c, _ := instruction.getC()
			err = vm.SetStack(instruction.getA(), &Boolean{val: b == 1})
			if c != 0 {
				vm.pc++
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
			b, _ := instruction.getB()
			c, _ := instruction.getC()
			var strBuilder strings.Builder
			if c < b {
				c = b
			}
			for i := b; b < c; i++ {
				fmt.Fprint(&strBuilder, vm.GetStack(i).String())
			}
			err = vm.SetStack(instruction.getA(), &String{val: strBuilder.String()})
		case JMP: // TODO if A is not 0 then upvalues need to be closed
			vm.pc += instruction.getsBx()
		case EQ:
			expected := instruction.getA() != 0
			isEq, err := vm.eq(fn, instruction)
			if err != nil {
				return err
			} else if isEq != expected {
				vm.pc++
			}
		case LT:
			expected := instruction.getA() != 0
			res, err := vm.compare(fn, instruction)
			if err != nil {
				return err
			} else if isMatch := res < 0; isMatch != expected {
				vm.pc++
			}
		case LE:
			expected := instruction.getA() != 0
			res, err := vm.compare(fn, instruction)
			if err != nil {
				return err
			} else if isMatch := res <= 0; isMatch != expected {
				vm.pc++
			}
		case TEST:
			b, _ := instruction.getB()
			expected := b != 0
			actual := vm.GetStack(instruction.getA()).Bool().val
			if expected != actual {
				vm.pc++
			}
		case TESTSET:
			b, _ := instruction.getB()
			c, _ := instruction.getC()
			expected := c != 0
			actual := vm.GetStack(b).Bool().val
			if expected != actual {
				vm.pc++
			} else {
				err = vm.SetStack(instruction.getA(), &Boolean{val: actual})
			}
		case LEN:
			b, bK := instruction.getB()
			val, err := vm.Get(fn, b, bK)
			if err != nil {
				return err
			}
			switch tval := val.(type) {
			case *String:
				err = vm.SetStack(instruction.getA(), &Integer{val: int64(len(tval.val))})
			case *Table:
				err = vm.SetStack(instruction.getA(), &Integer{val: int64(len(tval.val))})
			default:
				err = fmt.Errorf("attempt to get length of a %v value", val.Type())
			}
		case NEWTABLE:
			arraySize, _ := instruction.getB()
			hashSize, _ := instruction.getC()
			err = vm.SetStack(instruction.getA(), NewSizedTable(int(arraySize), int(hashSize)))
		case GETTABLE:
			tblIdx, _ := instruction.getB()
			keyIdx, keyK := instruction.getC()
			tbl, ok := vm.GetStack(tblIdx).(*Table)
			if !ok {
				return fmt.Errorf("attempt to index a %v value", vm.GetStack(instruction.getA()).Type())
			}
			key, err := vm.Get(fn, keyIdx, keyK)
			if err != nil {
				return err
			}
			val, err := tbl.Index(key)
			if err != nil {
				return err
			}
			err = vm.SetStack(instruction.getA(), val)
		case SETTABLE:
			tbl, ok := vm.GetStack(instruction.getA()).(*Table)
			if !ok {
				return fmt.Errorf("attempt to index a %v value", vm.GetStack(instruction.getA()).Type())
			}
			keyIdx, keyK := instruction.getB()
			key, err := vm.Get(fn, keyIdx, keyK)
			if err != nil {
				return err
			}
			valueIdx, valueK := instruction.getC()
			value, err := vm.Get(fn, valueIdx, valueK)
			if err != nil {
				return err
			}
			err = tbl.SetIndex(key, value)
		case SETLIST:
			// TODO Extended C usage is not supported yet
			// If C is 0, the next instruction is cast as an integer, and used as the C value.
			// This happens only when operand C is unable to encode the block number,
			// i.e. when C > 511, equivalent to an array index greater than 25550.
			tbl, ok := vm.GetStack(instruction.getA()).(*Table)
			if !ok {
				return fmt.Errorf("attempt to index a %v value", vm.GetStack(instruction.getA()).Type())
			}
			nElems, _ := instruction.getB()
			startIndex, _ := instruction.getC()
			start := instruction.getA() + 1
			end := nElems - 1
			if end < 0 {
				end = int64(len(vm.Stack)) - (vm.base + start)
			}
			values := make([]Value, 0, end+1)
			for i := start; i <= end; i++ {
				values = append(values, vm.GetStack(i))
			}
			tbl.val = slices.Insert(tbl.val, int(startIndex), values...)
		case VARARG:
			vm.truncate(instruction.getA())
			want, _ := instruction.getB()
			if diff := int(want) - len(xargs); diff > 0 {
				for i := 0; i <= diff; i++ {
					xargs = append(xargs, &Nil{})
				}
			} else if int(want) < len(xargs) && want != 0 {
				xargs = xargs[:want]
			}
			vm.Stack = append(vm.Stack, xargs...)
		case GETUPVAL:
			b, _ := instruction.getB()
			err = vm.SetStack(instruction.getA(), upvals[b].Get(vm.Stack))
		case SETUPVAL:
			b, _ := instruction.getB()
			upvals[b].Set(vm.Stack, vm.GetStack(instruction.getA()))
		case GETTABUP:
			b, _ := instruction.getB()
			upval := upvals[b].Get(vm.Stack)
			tbl, ok := upval.(*Table)
			if !ok {
				return fmt.Errorf("cannot index upvalue type %v", upval.Type())
			}
			c, cK := instruction.getC()
			key, err := vm.Get(fn, c, cK)
			if err != nil {
				return err
			}
			val, err := tbl.Index(key)
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
			b, bK := instruction.getB()
			key, err := vm.Get(fn, b, bK)
			if err != nil {
				return err
			}
			c, cK := instruction.getC()
			value, err := vm.Get(fn, c, cK)
			if err != nil {
				return err
			}
			err = tbl.SetIndex(key, value)
		case CALL:
			//b, _ := instruction.getB()
			//nargs := b - 1
			//fnR := instruction.getA()
			//callable := vm.GetStack(fnR)
			//vm.base += fnR + 1
			// switch fn := callable.(type) {
			// case *Function:
			// case *ExternFunc:
			// case *Closure:
			// }
			//vm.base -= fnR + 1
			// a register of loaded fn
			// b = 0 : B = ‘top’, the function parameters range from R(A+1) to the
			//         top of the stack. This form is used when the number of parameters
			//         to pass is set by the previous VM instruction, which has to be
			//         one of OP_CALL or OP_VARARG
			//  >= 1 : there are (B-1) parameters and upon entry to the called function,
			//         R(A+1) will become the base
			// c = 0 : ‘top’ is set to last_result+1, so that the next open instruction
			//         (OP_CALL, OP_RETURN, OP_SETLIST) can use ‘top’
			//  >= 1 : (C-1) return values
		case CLOSURE:
			b, _ := instruction.getB()
			upindexes := fn.FnTable[b].UpIndexes
			closureUpvals := make([]Broker, len(upindexes))
			for i, idx := range upindexes {
				if idx.fromStack {
					if i, ok := slices.BinarySearchFunc(openBrokers, int(idx.index), findBroker); ok {
						closureUpvals[i] = openBrokers[i]
					} else {
						newBroker := Broker{val: vm.GetStack(int64(idx.index)), open: true, index: int(idx.index), name: idx.name}
						openBrokers = append(openBrokers, newBroker)
						closureUpvals[i] = newBroker
					}
				} else {
					closureUpvals[i] = upvals[idx.index]
				}
			}
			err = vm.SetStack(instruction.getA(), &Closure{val: fn.FnTable[b], upvalues: closureUpvals})
		case SELF:
			// loads the table as the first param in the fn
			// SELF  A B C
			// R(A) := R(B)[RK(C)]
			// R(A+1) := R(B);
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
		vm.pc++
	}
}

func (vm *VM) Get(fn *FuncProto, id int64, isConst bool) (Value, error) {
	if isConst {
		return fn.getConst(id)
	}
	return vm.GetStack(id), nil
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
	b, bK := instruction.getB()
	lVal, err := vm.Get(fp, b, bK)
	if err != nil {
		return err
	}

	c, cK := instruction.getC()
	rVal, err := vm.Get(fp, c, cK)
	if err != nil {
		return err
	}

	val, err := fn(lVal, rVal)
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
	b, bK := instruction.getB()
	lVal, err := vm.Get(fn, b, bK)
	if err != nil {
		return false, err
	}

	c, cK := instruction.getC()
	rVal, err := vm.Get(fn, c, cK)
	if err != nil {
		return false, err
	}

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
	b, bK := instruction.getB()
	lVal, err := vm.Get(fn, b, bK)
	if err != nil {
		return 0, err
	}

	c, cK := instruction.getC()
	rVal, err := vm.Get(fn, c, cK)
	if err != nil {
		return 0, err
	}

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
