package luaf

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"strings"
	"sync"
)

type (
	LoadMode      uint
	UpvalueBroker struct {
		index     int
		open      bool
		name      string
		stackLock *sync.Mutex
		stack     *[]Value
		val       Value
	}
	callInfo struct {
		LineInfo
		name     string
		filename string
	}
	VM struct {
		ctx          context.Context
		yieldable    bool
		framePointer int64
		top          int64
		usedLength   int64
		stackLock    sync.Mutex
		Stack        []Value
		env          *Table
		callStack    Stack[*callInfo]
	}
	RuntimeErr struct {
		msg   string
		trace string
	}
)

const (
	ModeText   LoadMode = 0b01
	ModeBinary LoadMode = 0b10
)

var forNumNames = []string{"initial", "limit", "step"}

func (err *RuntimeErr) Error() string {
	return fmt.Sprintf(`%v
stack traceback:
%v`, err.msg, err.trace)
}

func (i *callInfo) String() string {
	return fmt.Sprintf("%v:%v: in %v", i.filename, i.Line, i.name)
}

func NewVM(ctx context.Context) *VM {
	env := envTable
	env.hashtable["_G"] = env
	return NewEnvVM(ctx, env)
}

func NewEnvVM(ctx context.Context, env *Table) *VM {
	return &VM{
		ctx:          ctx,
		Stack:        make([]Value, INITIALSTACKSIZE),
		top:          0,
		framePointer: 0,
		env:          env,
	}
}

func (vm *VM) err(tmpl string, args ...any) error {
	var errAddrs string
	if len(vm.callStack) > 0 {
		ci := vm.callStack[len(vm.callStack)-1]
		errAddrs = fmt.Sprintf(" %v:%v: ", ci.filename, ci.Line)
	}
	return &RuntimeErr{
		msg:   fmt.Sprintf("lua:%v %v", errAddrs, fmt.Sprintf(tmpl, args...)),
		trace: printStackTrace(vm.callStack),
	}
}

func (vm *VM) Env() *Table {
	return vm.env
}

func (vm *VM) LoadFile(path string, mode LoadMode, env *Table) ([]Value, error) {
	src, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer src.Close()
	return vm.Load(path, src, mode, env)
}

func (vm *VM) LoadString(path, src string, mode LoadMode, env *Table) ([]Value, error) {
	return vm.Load(path, strings.NewReader(src), mode, env)
}

func (vm *VM) Load(name string, src io.ReadSeeker, mode LoadMode, env *Table) ([]Value, error) {
	if env == nil {
		env = vm.env
	}
	if mode&ModeBinary == ModeBinary {
		fn, err := UndumpFnProto(src)
		if err != nil && mode&ModeText != ModeText {
			return nil, err
		} else if err == nil {
			return vm.EvalEnv(fn, env)
		}
	}
	fn, err := Parse(name, src)
	if err != nil {
		return nil, err
	}
	return vm.EvalEnv(fn, env)
}

func (vm *VM) Eval(fn *FnProto) ([]Value, error) {
	return vm.EvalEnv(fn, vm.env)
}

func (vm *VM) EvalEnv(fn *FnProto, env *Table) ([]Value, error) {
	envUpval := &UpvalueBroker{name: "_ENV", val: env}
	values, _, err := vm.eval(fn, []*UpvalueBroker{envUpval})
	return values, err
}

func (vm *VM) eval(fn *FnProto, upvals []*UpvalueBroker) ([]Value, int64, error) {
	var programCounter int64
	xargs, err := vm.truncateGet(int64(fn.Arity))
	if err != nil {
		return nil, programCounter, err
	}
	openBrokers := []*UpvalueBroker{}
	tbcValues := []int64{}

	extraArg := func(index int64) int64 {
		if index == 0 {
			programCounter++
			return int64(fn.ByteCodes[programCounter])
		} else {
			return index - 1
		}
	}

	var linfo LineInfo
	for {
		if err := vm.ctx.Err(); err != nil {
			// cancelled context
			return nil, programCounter, err
		}
		var err error
		if int64(len(fn.ByteCodes)) <= programCounter {
			return nil, programCounter, nil
		}

		instruction := fn.ByteCodes[programCounter]
		if programCounter < int64(len(fn.LineTrace)) {
			// protection clause really only for tests, these should always be 1:1
			linfo = fn.LineTrace[programCounter]
		}
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
			err = vm.SetStack(instruction.getA(), &Integer{val: instruction.getsBx()})
		case LOADF:
			err = vm.SetStack(instruction.getA(), &Float{val: float64(instruction.getsBx())})
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
				err = vm.err("attempt to get length of a %v value", val.Type())
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
			itbl := instruction.getA()
			tbl, ok := vm.GetStack(itbl).(*Table)
			if !ok {
				return nil, programCounter, vm.err("attempt to index a %v value", vm.GetStack(instruction.getA()).Type())
			}
			start := itbl + 1
			nvals := (instruction.getB() - 1)
			if nvals < 0 {
				nvals = vm.top
			}
			index := extraArg(instruction.getC())
			ensureSize(&tbl.val, int(index+nvals)-1)
			for i := int64(0); i < nvals; i++ {
				tbl.val[i+index] = vm.GetStack(start + i)
			}
			if err := vm.setTop(itbl + 1); err != nil {
				return nil, programCounter, err
			}
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
			retVals, err := vm.truncateGet(instruction.getA())
			if err != nil {
				return nil, programCounter, err
			} else if nret > 0 && len(retVals) > int(nret) {
				retVals = retVals[:nret]
			} else if len(retVals) < int(nret) {
				retVals = ensureLenNil(retVals, int(nret))
			} else if nret == 0 {
				retVals = nil
			}
			return retVals, programCounter, nil
		case VARARG:
			if err := vm.truncate(instruction.getA()); err != nil {
				return nil, programCounter, err
			}
			if want := instruction.getB() - 1; want > 0 {
				xargs = ensureLenNil(xargs, int(want))
			}
			_, err = vm.Push(xargs...)
		case CALL:
			ifn := instruction.getA()
			nargs := instruction.getB() - 1
			nret := instruction.getC() - 1
			retVals, err := vm.callFn(ifn, nargs, fn.Name, fn.Filename, linfo)
			if err != nil {
				return nil, programCounter, err
			}
			if nret > 0 && len(retVals) > int(nret) {
				retVals = retVals[:nret]
			} else if len(retVals) < int(nret) {
				retVals = ensureLenNil(retVals, int(nret))
			}
			if err := vm.truncate(ifn); err != nil {
				return nil, programCounter, err
			} else if _, err = vm.Push(retVals...); err != nil {
				return nil, programCounter, err
			}
		case TAILCALL:
			ifn := int(vm.framePointer)
			cutout(&vm.Stack, ifn, int(vm.framePointer+instruction.getA()+1))
			stackFn := vm.Stack[int(vm.framePointer)-1]
			newFn, isCallable := stackFn.(callable)
			if !isCallable {
				return nil, programCounter, vm.err("expected callable but found %v", stackFn.Type())
			}
			retVals, err := newFn.Call(vm, instruction.getB()-1)
			if err != nil {
				return nil, programCounter, err
			}
			truncate(&vm.Stack, ifn)
			if _, err = vm.Push(retVals...); err != nil {
				return nil, programCounter, err
			}
		case CLOSURE:
			cls := fn.FnTable[instruction.getB()]
			closureUpvals := make([]*UpvalueBroker, len(cls.UpIndexes))
			for i, idx := range cls.UpIndexes {
				if idx.FromStack {
					if j, ok := search(openBrokers, int(idx.Index), findBroker); ok {
						closureUpvals[i] = openBrokers[j]
					} else {
						newBroker := vm.newUpValueBroker(idx.Name, vm.GetStack(int64(idx.Index)), int(vm.framePointer)+int(idx.Index))
						openBrokers = append(openBrokers, newBroker)
						closureUpvals[i] = newBroker
					}
				} else {
					closureUpvals[i] = upvals[idx.Index]
				}
			}
			err = vm.SetStack(instruction.getA(), &Closure{val: cls, upvalues: closureUpvals})
		case SELF:
			tbl := vm.GetStack(instruction.getB())
			keyIdx, keyK := instruction.getCK()
			fn, err := vm.index(tbl, nil, vm.Get(fn, keyIdx, keyK))
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
					return nil, programCounter, vm.err("non-numeric %v value", forNumNames[i])
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
				return nil, programCounter, vm.err("0 Step in numerical for")
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
			values, err := vm.callFn(idx, 2, fn.Name, fn.Filename, linfo)
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

func (vm *VM) callFn(fnR, nargs int64, fnName, filename string, linfo LineInfo) ([]Value, error) {
	fnVal := vm.GetStack(fnR)
	fn, isCallable := fnVal.(callable)
	if !isCallable {
		method, isCallable := vm.findMetamethod(metaCall, fnVal)
		if !isCallable {
			return nil, vm.err("expected callable but found %v", vm.GetStack(fnR).Type())
		} else if !isNil(method) {
			fn = method.(callable)
		} else {
			return nil, vm.err("could not find metavalue __call")
		}
	}
	vm.framePointer += fnR + 1
	vm.callStack.Push(&callInfo{name: fnName, filename: filename, LineInfo: linfo})
	defer func() {
		vm.callStack.Pop()
		vm.framePointer -= fnR + 1
	}()
	return fn.Call(vm, nargs)
}

func (vm *VM) Get(fn *FnProto, id int64, isConst bool) Value {
	if isConst {
		return fn.getConst(id)
	}
	return vm.GetStack(id)
}

func (vm *VM) GetStack(id int64) Value {
	if vm.framePointer+id >= vm.top || id < 0 || vm.Stack[vm.framePointer+id] == nil {
		return &Nil{}
	}
	return vm.Stack[vm.framePointer+id]
}

func (vm *VM) SetStack(id int64, val Value) error {
	dst := vm.framePointer + id
	if id < 0 {
		return errors.New("cannot address negatively in the stack")
	} else if err := vm.ensureStackSize(dst); err != nil {
		return err
	}
	vm.Stack[dst] = val
	if dst+1 > vm.top {
		if err := vm.setTop(dst + 1); err != nil {
			return err
		}
	}
	return nil
}

func (vm *VM) Push(val ...Value) (int64, error) {
	addr := vm.top
	if err := vm.ensureStackSize(vm.top + int64(len(val))); err != nil {
		return -1, err
	}
	for i := vm.top; i < int64(len(val))+vm.top; i++ {
		vm.Stack[i] = val[i-vm.top]
	}
	vm.top += int64(len(val))
	return int64(addr), nil
}

func (vm *VM) ensureStackSize(index int64) error {
	sliceLen := int64(len(vm.Stack))
	if index < sliceLen {
		return nil
	}
	growthAmount := (index - (sliceLen - 1)) * 2
	if growthAmount+sliceLen > MAXSTACKSIZE {
		growthAmount = MAXSTACKSIZE - sliceLen
	}
	if growthAmount <= 0 {
		return vm.err("stack overflow %v", index)
	}
	newSlice := make([]Value, sliceLen+growthAmount)
	copy(newSlice, vm.Stack)
	vm.Stack = newSlice
	return nil
}

func (vm *VM) truncate(id int64) error {
	dst := vm.framePointer + id
	if dst > vm.top || dst < 0 {
		return nil
	}
	return vm.setTop(dst)
}

func (vm *VM) truncateGet(id int64) ([]Value, error) {
	dst := vm.framePointer + id
	if dst >= vm.top || dst < 0 {
		return nil, nil
	}
	out := make([]Value, vm.top-dst)
	copy(out, vm.Stack[dst:vm.top])
	return out, vm.setTop(dst)
}

type opFn func(lVal, rVal Value) (Value, error)

func (vm *VM) setABCFn(fp *FnProto, instruction Bytecode, fn opFn) error {
	b, bK := instruction.getBK()
	c, cK := instruction.getCK()
	val, err := fn(vm.Get(fp, b, bK), vm.Get(fp, c, cK))
	if err != nil {
		return err
	}
	return vm.SetStack(instruction.getA(), val)
}

func (vm *VM) setABCArith(fp *FnProto, instruction Bytecode, op metaMethod) error {
	return vm.setABCFn(fp, instruction, func(lVal, rVal Value) (Value, error) {
		return vm.arith(op, lVal, rVal)
	})
}

func (vm *VM) arith(op metaMethod, lval, rval Value) (Value, error) {
	if op == metaUNM {
		if liva, lisInt := lval.(*Integer); lisInt {
			return &Integer{val: intArith(op, liva.val, 0)}, nil
		} else if isNumber(lval) {
			return &Float{val: floatArith(op, toFloat(lval), 0)}, nil
		}
	} else if op == metaBNot {
		if isNumber(lval) {
			return &Integer{val: intArith(op, toInt(lval), 0)}, nil
		}
	} else if isNumber(lval) && isNumber(rval) {
		switch op {
		case metaBAnd, metaBOr, metaBXOr, metaShl, metaShr:
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
		if op == metaUNM || op == metaBNot {
			return nil, vm.err("cannot %v %v", op, lval.Type())
		} else {
			return nil, vm.err("cannot %v %v and %v", op, lval.Type(), rval.Type())
		}
	} else if len(res) > 0 {
		return res[0], nil
	}
	return nil, vm.err("error object is a nil value")
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

func (vm *VM) eq(fn *FnProto, instruction Bytecode) (bool, error) {
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

func (vm *VM) compare(op metaMethod, fn *FnProto, instruction Bytecode) (int, error) {
	b, bK := instruction.getBK()
	c, cK := instruction.getCK()
	return vm.compareVal(op, vm.Get(fn, b, bK), vm.Get(fn, c, cK))
}

func (vm *VM) compareVal(op metaMethod, lVal, rVal Value) (int, error) {
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
	return 0, vm.err("attempted to compare two %v and %v values", lVal.Type(), rVal.Type())
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
			return nil, vm.err("attempted to concatenate a %v value", b.Type())
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
	mIndex := string(metaIndex)
	if metatable != nil && metatable.hashtable[mIndex] != nil {
		switch metaVal := metatable.hashtable[mIndex].(type) {
		case callable:
			res, err := vm.Call(mIndex, metaVal, []Value{source, key})
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
	return nil, vm.err("attempt to index a %v value", table.Type())
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
	mNewIndex := string(metaNewIndex)
	if metatable != nil && metatable.hashtable[mNewIndex] != nil {
		switch metaVal := metatable.hashtable[mNewIndex].(type) {
		case callable:
			_, err := vm.Call(mNewIndex, metaVal, []Value{table, key})
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
	return vm.err("attempt to index a %v value", table.Type())
}

func (vm *VM) findMetamethod(op metaMethod, params ...Value) (Value, bool) {
	for _, val := range params {
		if val != nil && val.Meta() != nil && val.Meta().hashtable[string(op)] != nil {
			metamethod := val.Meta().hashtable[string(op)]
			if isNil(metamethod) {
				continue
			}
			_, isCallable := metamethod.(callable)
			return metamethod, isCallable
		}
	}
	return nil, false
}

func (vm *VM) delegateMetamethod(op metaMethod, params ...Value) (bool, []Value, error) {
	if method, isCallable := vm.findMetamethod(op, params...); isCallable && !isNil(method) {
		ret, err := vm.Call(string(op), method.(callable), params)
		return true, ret, err
	} else if !isNil(method) {
		return true, []Value{method}, nil
	}
	return false, nil, nil // unable to delegate
}

func (vm *VM) Call(label string, fn callable, params []Value) ([]Value, error) {
	val, isValue := fn.(Value)
	if !isValue {
		return nil, vm.err("callable is not value")
	}
	ifn, err := vm.Push(val)
	if err != nil {
		return nil, err
	}
	if _, err := vm.Push(params...); err != nil {
		return nil, err
	}
	retVals, err := vm.callFn(ifn-vm.framePointer, int64(len(params)), label, "", LineInfo{})
	if err != nil {
		return nil, err
	}
	return retVals, vm.truncate(ifn)
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
			return vm.err("__close not defined on closable table")
		}
	}
	return nil
}

func (vm *VM) setTop(newTop int64) error {
	vm.top = newTop
	if vm.top > vm.usedLength {
		vm.usedLength = vm.top
	}
	return vm.collectGarbage(false)
}

func (vm *VM) collectGarbage(implicit bool) error {
	usedSize := vm.usedLength - vm.top
	if !implicit && usedSize < GCPAUSE {
		return nil
	}
	for i := vm.top; i < vm.usedLength; i++ {
		if _, _, err := vm.delegateMetamethod(metaGC, vm.Stack[i]); err != nil {
			return err
		}
		vm.Stack[i] = nil
	}
	vm.usedLength = vm.top
	return nil
}

func (vm *VM) newUpValueBroker(name string, val Value, index int) *UpvalueBroker {
	return &UpvalueBroker{
		stackLock: &vm.stackLock,
		stack:     &vm.Stack,
		name:      name,
		val:       val,
		index:     index,
		open:      true,
	}
}

func (b *UpvalueBroker) Get() Value {
	if b.open {
		b.stackLock.Lock()
		defer b.stackLock.Unlock()
		return (*b.stack)[b.index]
	}
	return b.val
}

func (b *UpvalueBroker) Set(val Value) {
	if b.open {
		b.stackLock.Lock()
		defer b.stackLock.Unlock()
		(*b.stack)[b.index] = val
	}
	b.val = val
}

func (b *UpvalueBroker) Close() {
	if !b.open {
		return
	}
	b.stackLock.Lock()
	defer b.stackLock.Unlock()
	b.val = (*b.stack)[b.index]
	b.open = false
	b.stack = nil
}
