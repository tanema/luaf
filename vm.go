package luaf

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
)

type (
	LoadMode uint
	callInfo struct {
		LineInfo
		name     string
		filename string
	}
	// ctx
	// yieldable
	// stack
	// top
	// frame
	//   env
	//   fn
	//   framePointer
	//   program counter
	//   xargs
	//   upvalIndexes
	//   upvalBrokers
	//   tbc
	//   callInfo
	//
	// VMState struct {
	//	fn             *FnProto
	//	env            *Table
	//	framePointer   int64 // stack pointer to 0 of the running frame
	//	programCounter int64
	//	xargs          []Value
	//	upvals         []*UpvalueBroker // upvals passed to the scope
	//	openBrokers    []*UpvalueBroker // upvals created by the scope
	//	tbcValues      []int64          // values that require closing
	//	callInfo       callInfo
	// }
	/*
		VM Design
		One VM for each stack that you want because the VM contains the stack. This
		means a separate vm for each coroutine/thread.
		This also means the state that is kept of its current progress can not be shared
		between VMs as the position of variables in the stack for upvalues and calls
		are needed to resume that state.

		On yield the state is saved in the VM and the VM can be resumed instead of like
		other lua implementations
	*/
	VM struct {
		ctx          context.Context
		yieldable    bool
		framePointer int64 // stack pointer to 0 of the running frame
		top          int64 // end of frame in stack
		env          *Table
		callStack    Stack[callInfo]

		// stack management
		gcOff       bool
		stackLock   sync.Mutex
		Stack       []Value
		garbageSize int
		garbageHeap []Value
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

func NewVM(ctx context.Context, clargs ...string) *VM {
	env := envTable
	env.hashtable["_G"] = env

	splitidx := slices.Index(clargs, "--")
	if splitidx == -1 {
		splitidx = len(clargs)
	} else {
		splitidx++
	}

	argValues := make([]Value, len(clargs))
	for i, a := range clargs {
		argValues[i] = &String{val: a}
	}

	argVal := NewTable(argValues[splitidx:], nil)
	for i := 0; i < splitidx; i++ {
		argVal.hashtable[int64(-(splitidx-i)+1)] = argValues[i]
	}

	env.hashtable["arg"] = argVal
	newEnv := NewEnvVM(ctx, env)
	_, _ = newEnv.Push(argValues[splitidx:]...)
	return newEnv
}

func NewEnvVM(ctx context.Context, env *Table) *VM {
	return &VM{
		ctx:          ctx,
		Stack:        make([]Value, INITIALSTACKSIZE),
		top:          0,
		framePointer: 0,
		env:          env,
		callStack:    NewStack[callInfo](100),
		garbageSize:  0,
		garbageHeap:  make([]Value, INITIALSTACKSIZE),
	}
}

func (vm *VM) err(tmpl string, args ...any) error {
	var errAddrs string
	if ci := vm.callStack.Top(); ci != nil {
		errAddrs = fmt.Sprintf(" %v:%v: ", ci.filename, ci.Line)
	}
	return &RuntimeErr{
		msg:   fmt.Sprintf("lua:%v %v", errAddrs, fmt.Sprintf(tmpl, args...)),
		trace: printStackTrace(vm.callStack),
	}
}

func (vm *VM) Eval(fn *FnProto, args ...string) ([]Value, error) {
	envUpval := &UpvalueBroker{name: "_ENV", val: vm.env}
	res, _, err := vm.eval(fn, []*UpvalueBroker{envUpval})
	return res, err
}

func (vm *VM) eval(fn *FnProto, upvals []*UpvalueBroker) ([]Value, int64, error) {
	var programCounter int64

	xargs := make([]Value, vm.top-vm.framePointer+fn.Arity)
	copy(xargs, vm.Stack[vm.framePointer+fn.Arity:vm.top])
	vm.setTop(vm.framePointer + fn.Arity)

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
			err = vm.SetStack(vm.framePointer+instruction.getA(), vm.GetStack(vm.framePointer+instruction.getB()))
		case LOADK:
			err = vm.SetStack(vm.framePointer+instruction.getA(), fn.getConst(instruction.getBx()))
		case LOADBOOL:
			err = vm.SetStack(vm.framePointer+instruction.getA(), &Boolean{val: instruction.getB() == 1})
			if instruction.getC() != 0 {
				programCounter++
			}
		case LOADI:
			err = vm.SetStack(vm.framePointer+instruction.getA(), &Integer{val: instruction.getsBx()})
		case LOADF:
			err = vm.SetStack(vm.framePointer+instruction.getA(), &Float{val: float64(instruction.getsBx())})
		case LOADNIL:
			a := instruction.getA()
			b := instruction.getBx()
			for i := a; i <= a+b; i++ {
				if err = vm.SetStack(vm.framePointer+i, &Nil{}); err != nil {
					return nil, programCounter, err
				}
			}
		case ADD:
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			val, err := arith(vm, metaAdd, vm.Get(fn, vm.framePointer, b, bK), vm.Get(fn, vm.framePointer, c, cK))
			if err != nil {
				return nil, programCounter, err
			} else if err = vm.SetStack(vm.framePointer+instruction.getA(), val); err != nil {
				return nil, programCounter, err
			}
		case SUB:
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			val, err := arith(vm, metaSub, vm.Get(fn, vm.framePointer, b, bK), vm.Get(fn, vm.framePointer, c, cK))
			if err != nil {
				return nil, programCounter, err
			} else if err = vm.SetStack(vm.framePointer+instruction.getA(), val); err != nil {
				return nil, programCounter, err
			}
		case MUL:
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			val, err := arith(vm, metaMul, vm.Get(fn, vm.framePointer, b, bK), vm.Get(fn, vm.framePointer, c, cK))
			if err != nil {
				return nil, programCounter, err
			} else if err = vm.SetStack(vm.framePointer+instruction.getA(), val); err != nil {
				return nil, programCounter, err
			}
		case DIV:
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			val, err := arith(vm, metaDiv, vm.Get(fn, vm.framePointer, b, bK), vm.Get(fn, vm.framePointer, c, cK))
			if err != nil {
				return nil, programCounter, err
			} else if err = vm.SetStack(vm.framePointer+instruction.getA(), val); err != nil {
				return nil, programCounter, err
			}
		case MOD:
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			val, err := arith(vm, metaMod, vm.Get(fn, vm.framePointer, b, bK), vm.Get(fn, vm.framePointer, c, cK))
			if err != nil {
				return nil, programCounter, err
			} else if err = vm.SetStack(vm.framePointer+instruction.getA(), val); err != nil {
				return nil, programCounter, err
			}
		case POW:
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			val, err := arith(vm, metaPow, vm.Get(fn, vm.framePointer, b, bK), vm.Get(fn, vm.framePointer, c, cK))
			if err != nil {
				return nil, programCounter, err
			} else if err = vm.SetStack(vm.framePointer+instruction.getA(), val); err != nil {
				return nil, programCounter, err
			}
		case IDIV:
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			val, err := arith(vm, metaIDiv, vm.Get(fn, vm.framePointer, b, bK), vm.Get(fn, vm.framePointer, c, cK))
			if err != nil {
				return nil, programCounter, err
			} else if err = vm.SetStack(vm.framePointer+instruction.getA(), val); err != nil {
				return nil, programCounter, err
			}
		case BAND:
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			val, err := arith(vm, metaBAnd, vm.Get(fn, vm.framePointer, b, bK), vm.Get(fn, vm.framePointer, c, cK))
			if err != nil {
				return nil, programCounter, err
			} else if err = vm.SetStack(vm.framePointer+instruction.getA(), val); err != nil {
				return nil, programCounter, err
			}
		case BOR:
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			val, err := arith(vm, metaBOr, vm.Get(fn, vm.framePointer, b, bK), vm.Get(fn, vm.framePointer, c, cK))
			if err != nil {
				return nil, programCounter, err
			} else if err = vm.SetStack(vm.framePointer+instruction.getA(), val); err != nil {
				return nil, programCounter, err
			}
		case BXOR:
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			val, err := arith(vm, metaBXOr, vm.Get(fn, vm.framePointer, b, bK), vm.Get(fn, vm.framePointer, c, cK))
			if err != nil {
				return nil, programCounter, err
			} else if err = vm.SetStack(vm.framePointer+instruction.getA(), val); err != nil {
				return nil, programCounter, err
			}
		case SHL:
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			val, err := arith(vm, metaShl, vm.Get(fn, vm.framePointer, b, bK), vm.Get(fn, vm.framePointer, c, cK))
			if err != nil {
				return nil, programCounter, err
			} else if err = vm.SetStack(vm.framePointer+instruction.getA(), val); err != nil {
				return nil, programCounter, err
			}
		case SHR:
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			val, err := arith(vm, metaShr, vm.Get(fn, vm.framePointer, b, bK), vm.Get(fn, vm.framePointer, c, cK))
			if err != nil {
				return nil, programCounter, err
			} else if err = vm.SetStack(vm.framePointer+instruction.getA(), val); err != nil {
				return nil, programCounter, err
			}
		case UNM:
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			val, err := arith(vm, metaUNM, vm.Get(fn, vm.framePointer, b, bK), vm.Get(fn, vm.framePointer, c, cK))
			if err != nil {
				return nil, programCounter, err
			} else if err = vm.SetStack(vm.framePointer+instruction.getA(), val); err != nil {
				return nil, programCounter, err
			}
		case BNOT:
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			val, err := arith(vm, metaBNot, vm.Get(fn, vm.framePointer, b, bK), vm.Get(fn, vm.framePointer, c, cK))
			if err != nil {
				return nil, programCounter, err
			} else if err = vm.SetStack(vm.framePointer+instruction.getA(), val); err != nil {
				return nil, programCounter, err
			}
		case NOT:
			b, bK := instruction.getBK()
			val := toBool(vm.Get(fn, vm.framePointer, b, bK)).Not()
			err = vm.SetStack(vm.framePointer+instruction.getA(), val)
		case CONCAT:
			b := instruction.getB()
			c := instruction.getC()
			if c < b {
				c = b + 1
			}
			result := vm.GetStack(vm.framePointer + b)
			for i := b + 1; i <= c; i++ {
				next := vm.GetStack(vm.framePointer + i)
				aCoercable := isString(result) || isNumber(result)
				bCoercable := isString(next) || isNumber(next)
				if aCoercable && bCoercable {
					result = &String{val: result.String() + next.String()}
				} else if didDelegate, res, err := vm.delegateMetamethodBinop(metaConcat, result, next); err != nil {
					return nil, programCounter, err
				} else if didDelegate && len(res) > 0 {
					result = res[0]
				} else {
					return nil, programCounter, vm.err("attempted to concatenate a %v value", next.Type())
				}
			}
			err = vm.SetStack(vm.framePointer+instruction.getA(), result)
		case TBC:
			tbcValues = append(tbcValues, instruction.getA())
		case JMP:
			from := int64(instruction.getA() - 1)
			if from >= 0 {
				for i := from; i < vm.top; i++ {
					if j, ok := search(openBrokers, uint64(vm.framePointer+i), findBroker); ok {
						openBrokers[j].Close()
						openBrokers = append(openBrokers[:j], openBrokers[j+1:]...) // remove broker
					}
				}
				vm.setTop(vm.framePointer + from)
				vm.collectGarbage()
			}
			programCounter += instruction.getsBx()
		case CLOSE:
			from := instruction.getA()
			for i := from; i < vm.top; i++ {
				if j, ok := search(openBrokers, uint64(vm.framePointer+i), findBroker); ok {
					openBrokers[j].Close()
					openBrokers = append(openBrokers[:j], openBrokers[j+1:]...) // remove broker
				}
			}
			vm.setTop(vm.framePointer + from)
			vm.collectGarbage()
		case EQ:
			expected := instruction.getA() != 0
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			lVal := vm.Get(fn, vm.framePointer, b, bK)
			rVal := vm.Get(fn, vm.framePointer, c, cK)
			if isEq, err := eq(vm, lVal, rVal); err != nil {
				return nil, programCounter, err
			} else if isEq != expected {
				programCounter++
			}
		case LT:
			expected := instruction.getA() != 0
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			res, err := compareVal(vm, metaLt, vm.Get(fn, vm.framePointer, b, bK), vm.Get(fn, vm.framePointer, c, cK))
			if err != nil {
				return nil, programCounter, err
			} else if isMatch := res < 0; isMatch != expected {
				programCounter++
			}
		case LE:
			expected := instruction.getA() != 0
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			res, err := compareVal(vm, metaLe, vm.Get(fn, vm.framePointer, b, bK), vm.Get(fn, vm.framePointer, c, cK))
			if err != nil {
				return nil, programCounter, err
			} else if isMatch := res <= 0; isMatch != expected {
				programCounter++
			}
		case TEST:
			expected := instruction.getB() != 0
			actual := toBool(vm.GetStack(vm.framePointer + instruction.getA())).val
			if expected != actual {
				programCounter++
			}
		case LEN:
			b, bK := instruction.getBK()
			val := vm.Get(fn, vm.framePointer, b, bK)
			dst := vm.framePointer + instruction.getA()
			if isString(val) {
				err = vm.SetStack(dst, &Integer{val: int64(len(val.(*String).val))})
			} else if tbl, isTbl := val.(*Table); isTbl {
				if method := findMetavalue(metaLen, tbl); method != nil {
					res, err := vm.Call(string(metaLen), method, []Value{tbl})
					if err != nil {
						return nil, programCounter, err
					} else if len(res) > 0 {
						if err = vm.SetStack(dst, res[0]); err != nil {
							return nil, programCounter, err
						}
					} else if err = vm.SetStack(dst, &Nil{}); err != nil {
						return nil, programCounter, err
					}
				} else {
					if err = vm.SetStack(dst, &Integer{val: int64(len(tbl.val))}); err != nil {
						return nil, programCounter, err
					}
				}
			} else {
				err = vm.err("attempt to get length of a %v value", val.Type())
			}
		case NEWTABLE:
			err = vm.SetStack(vm.framePointer+instruction.getA(), NewSizedTable(int(instruction.getB()), int(instruction.getC())))
		case GETTABLE:
			keyIdx, keyK := instruction.getCK()
			tbl := vm.GetStack(vm.framePointer + instruction.getB())
			if val, err := vm.index(tbl, nil, vm.Get(fn, vm.framePointer, keyIdx, keyK)); err != nil {
				return nil, programCounter, err
			} else if err = vm.SetStack(vm.framePointer+instruction.getA(), val); err != nil {
				return nil, programCounter, err
			}
		case SETTABLE:
			keyIdx, keyK := instruction.getBK()
			valueIdx, valueK := instruction.getCK()
			err = vm.newIndex(
				vm.GetStack(vm.framePointer+instruction.getA()),
				vm.Get(fn, vm.framePointer, keyIdx, keyK),
				vm.Get(fn, vm.framePointer, valueIdx, valueK),
			)
		case SETLIST:
			itbl := instruction.getA()
			tbl, ok := vm.GetStack(vm.framePointer + itbl).(*Table)
			if !ok {
				return nil, programCounter, vm.err("attempt to index a %v value", vm.GetStack(vm.framePointer+instruction.getA()).Type())
			}
			start := itbl + 1
			nvals := (instruction.getB() - 1)
			if nvals < 0 {
				nvals = vm.top - start - 1
			}
			index := extraArg(instruction.getC())
			ensureSize(&tbl.val, int(index+nvals)-1)
			for i := int64(0); i < nvals; i++ {
				tbl.val[i+index] = vm.GetStack(vm.framePointer + start + i)
			}
			vm.setTop(vm.framePointer + itbl + 1)
		case GETUPVAL:
			err = vm.SetStack(vm.framePointer+instruction.getA(), upvals[instruction.getB()].Get())
		case SETUPVAL:
			upvals[instruction.getB()].Set(vm.GetStack(vm.framePointer + instruction.getA()))
		case GETTABUP:
			keyIdx, keyK := instruction.getCK()
			tbl := upvals[instruction.getB()].Get()
			if val, err := vm.index(tbl, nil, vm.Get(fn, vm.framePointer, keyIdx, keyK)); err != nil {
				return nil, programCounter, err
			} else if err = vm.SetStack(vm.framePointer+instruction.getA(), val); err != nil {
				return nil, programCounter, err
			}
		case SETTABUP:
			keyIdx, keyK := instruction.getBK()
			valueIdx, valueK := instruction.getCK()
			err = vm.newIndex(
				upvals[instruction.getA()].Get(),
				vm.Get(fn, vm.framePointer, keyIdx, keyK),
				vm.Get(fn, vm.framePointer, valueIdx, valueK),
			)
		case CALL:
			ifn := instruction.getA()
			nargs := instruction.getB() - 1
			nret := instruction.getC() - 1
			args := vm.argsFromStack(vm.framePointer+ifn+1, nargs)
			fnVal := vm.GetStack(vm.framePointer + ifn)
			vm.framePointer += ifn + 1
			vm.callStack.Push(&callInfo{name: fn.Name, filename: fn.Filename, LineInfo: linfo})
			var retVals []Value
			switch tfn := fnVal.(type) {
			case *Closure:
				if err := vm.ensureArgsInStack(nargs, tfn.val.Arity); err != nil {
					return nil, programCounter, err
				} else if retVals, _, err = vm.eval(tfn.val, tfn.upvalues); err != nil {
					return nil, programCounter, err
				}
			case *ExternFunc:
				if retVals, err = tfn.val(vm, vm.argsFromStack(vm.framePointer, nargs)); err != nil {
					return nil, programCounter, err
				}
			default:
				if retVals, err = vm.Call("__call", findMetavalue(metaCall, fnVal), append([]Value{fnVal}, args...)); err != nil {
					return nil, programCounter, err
				}
			}
			vm.callStack.Pop()
			vm.framePointer -= ifn + 1
			vm.setTop(vm.framePointer + ifn)
			vm.collectGarbage()
			if nret > 0 && len(retVals) > int(nret) {
				retVals = retVals[:nret]
			} else if len(retVals) < int(nret) {
				retVals = ensureLenNil(retVals, int(nret))
			}
			if _, err = vm.Push(retVals...); err != nil {
				return nil, programCounter, err
			}
		case TAILCALL:
			if err := vm.cleanup(vm.framePointer, openBrokers, tbcValues); err != nil {
				return nil, programCounter, err
			}
			openBrokers = []*UpvalueBroker{}
			tbcValues = []int64{}

			frameStart := vm.framePointer - 1
			frameEnd := vm.framePointer + instruction.getA()
			cutout(&vm.Stack, int(frameStart), int(frameEnd))
			vm.setTop(vm.top - (frameEnd - frameStart))

			ifn := vm.framePointer - 1
			fnVal := vm.Stack[vm.framePointer-1]
			nargs := instruction.getB() - 1
			args := vm.argsFromStack(vm.framePointer+ifn+1, nargs)
			var retVals []Value
			switch tfn := fnVal.(type) {
			case *Closure:
				if err := vm.ensureArgsInStack(nargs, tfn.val.Arity); err != nil {
					return nil, programCounter, err
				} else if retVals, _, err = vm.eval(tfn.val, tfn.upvalues); err != nil {
					return nil, programCounter, err
				}
			case *ExternFunc:
				if retVals, err = tfn.val(vm, vm.argsFromStack(vm.framePointer, nargs)); err != nil {
					return nil, programCounter, err
				}
			default:
				if retVals, err = vm.Call("__call", findMetavalue(metaCall, fnVal), append([]Value{fnVal}, args...)); err != nil {
					return nil, programCounter, err
				}
			}

			vm.setTop(vm.framePointer)
			vm.collectGarbage()
			if _, err = vm.Push(retVals...); err != nil {
				return nil, programCounter, err
			}
		case RETURN:
			if err := vm.cleanup(vm.framePointer, openBrokers, tbcValues); err != nil {
				return nil, programCounter, err
			}
			var retVals []Value
			start := vm.framePointer + instruction.getA()
			nret := (instruction.getB() - 1)
			if nret == -1 {
				nret = vm.top - start
			}
			end := min(start+nret, vm.top)
			if nret > 0 {
				retVals = make([]Value, nret)
				copy(retVals, vm.Stack[start:end])
				if diff := start + nret - end; diff > 0 {
					for i := end - 1; i < end+diff-1; i++ {
						retVals[i] = &Nil{}
					}
				}
			}
			vm.setTop(vm.framePointer)
			vm.collectGarbage()
			return retVals, programCounter, nil
		case VARARG:
			vm.setTop(vm.framePointer + instruction.getA())
			vm.collectGarbage()
			if want := instruction.getB() - 1; want > 0 {
				xargs = ensureLenNil(xargs, int(want))
			}
			_, err = vm.Push(xargs...)
		case CLOSURE:
			cls := fn.FnTable[instruction.getB()]
			closureUpvals := make([]*UpvalueBroker, len(cls.UpIndexes))
			for i, idx := range cls.UpIndexes {
				if idx.FromStack {
					if j, ok := search(openBrokers, uint64(vm.framePointer)+uint64(idx.Index), findBroker); ok {
						closureUpvals[i] = openBrokers[j]
					} else {
						newBroker := vm.newUpValueBroker(idx.Name, vm.GetStack(vm.framePointer+int64(idx.Index)), uint64(vm.framePointer)+uint64(idx.Index))
						openBrokers = append(openBrokers, newBroker)
						closureUpvals[i] = newBroker
					}
				} else {
					closureUpvals[i] = upvals[idx.Index]
				}
			}
			err = vm.SetStack(vm.framePointer+instruction.getA(), &Closure{val: cls, upvalues: closureUpvals})
		case SELF:
			tbl := vm.GetStack(vm.framePointer + instruction.getB())
			keyIdx, keyK := instruction.getCK()
			fn, err := vm.index(tbl, nil, vm.Get(fn, vm.framePointer, keyIdx, keyK))
			if err != nil {
				return nil, programCounter, err
			}
			ra := instruction.getA()
			if err = vm.SetStack(vm.framePointer+ra, fn); err != nil {
				return nil, programCounter, err
			} else if err = vm.SetStack(vm.framePointer+ra+1, tbl); err != nil {
				return nil, programCounter, err
			}
		case FORPREP:
			ivar := instruction.getA()
			hasFloat := false
			for i := ivar; i < ivar+3; i++ {
				switch vm.GetStack(vm.framePointer + int64(i)).(type) {
				case *Integer:
				case *Float:
					hasFloat = true
				default:
					return nil, programCounter, vm.err("non-numeric %v value", forNumNames[i])
				}
			}
			if hasFloat {
				for i := ivar; i < ivar+3; i++ {
					if _, ok := vm.GetStack(vm.framePointer + int64(i)).(*Integer); !ok {
						if err := vm.SetStack(vm.framePointer+int64(i), &Float{val: toFloat(vm.GetStack(vm.framePointer + int64(i)))}); err != nil {
							return nil, programCounter, err
						}
					}
				}
			}
			if toFloat(vm.GetStack(vm.framePointer+ivar+2)) == 0 {
				return nil, programCounter, vm.err("0 Step in numerical for")
			}

			i := vm.GetStack(vm.framePointer + ivar)
			step := vm.GetStack(vm.framePointer + ivar + 2)
			if iVal, isInt := i.(*Integer); isInt {
				stepVal := step.(*Integer)
				err = vm.SetStack(vm.framePointer+ivar, &Integer{val: iVal.val - stepVal.val})
			} else {
				iVal := i.(*Float)
				stepVal := step.(*Float)
				err = vm.SetStack(vm.framePointer+ivar, &Float{val: iVal.val - stepVal.val})
			}
			programCounter += instruction.getsBx()
		case FORLOOP:
			ivar := instruction.getA()
			i := vm.GetStack(vm.framePointer + ivar)
			limit := vm.GetStack(vm.framePointer + ivar + 1)
			step := vm.GetStack(vm.framePointer + ivar + 2)
			if iVal, isInt := i.(*Integer); isInt {
				stepVal := step.(*Integer)
				err = vm.SetStack(vm.framePointer+ivar, &Integer{val: iVal.val + stepVal.val})
			} else {
				iVal := i.(*Float)
				stepVal := step.(*Float)
				err = vm.SetStack(vm.framePointer+ivar, &Float{val: iVal.val + stepVal.val})
			}
			i = vm.GetStack(vm.framePointer + ivar)
			check := (toFloat(step) > 0 && toFloat(i) <= toFloat(limit)) ||
				(toFloat(step) < 0 && toFloat(i) >= toFloat(limit))
			if check {
				programCounter += instruction.getsBx()
			}
		case TFORCALL:
			idx := instruction.getA()
			fn := vm.GetStack(vm.framePointer + idx)
			values, err := vm.Call("tfor", fn, vm.argsFromStack(vm.framePointer+idx+1, 2))
			if err != nil {
				return nil, programCounter, err
			}
			var ctrl Value = &Nil{}
			if len(values) > 0 {
				ctrl = values[0]
			}
			if err := vm.SetStack(vm.framePointer+idx+2, ctrl); err != nil {
				return nil, programCounter, err
			}
			// TODO set range instead of iteration
			for i := 0; i < int(instruction.getB()); i++ {
				var val Value = &Nil{}
				if i < len(values) {
					val = values[i]
				}
				if err := vm.SetStack(vm.framePointer+idx+int64(i)+3, val); err != nil {
					return nil, programCounter, err
				}
			}
		case TFORLOOP:
			idx := instruction.getA()
			control := vm.GetStack(vm.framePointer + idx + 1)
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

func (vm *VM) ensureArgsInStack(nargs, arity int64) error {
	if diff := arity - nargs; nargs > 0 && diff > 0 {
		for i := nargs; i <= arity; i++ {
			if err := vm.SetStack(vm.framePointer+i, &Nil{}); err != nil {
				return err
			}
		}
	}
	return nil
}

func (vm *VM) argsFromStack(offset, nargs int64) []Value {
	args := []Value{}
	if nargs <= 0 {
		nargs = vm.top - offset
	}
	for _, val := range vm.Stack[offset : offset+nargs] {
		if val != nil {
			args = append(args, val)
		} else {
			args = append(args, &Nil{})
		}
	}
	if diff := int(nargs) - len(args); diff > 0 {
		for i := 0; i < diff; i++ {
			args = append(args, &Nil{})
		}
	}
	return args
}

func (vm *VM) Get(fn *FnProto, fp, id int64, isConst bool) Value {
	if isConst {
		return fn.getConst(id)
	}
	return vm.GetStack(fp + id)
}

func (vm *VM) GetStack(id int64) Value {
	if id >= vm.top || id < 0 || vm.Stack[id] == nil {
		return &Nil{}
	}
	return vm.Stack[id]
}

func (vm *VM) SetStack(dst int64, val Value) error {
	if dst < 0 {
		return errors.New("cannot address negatively in the stack")
	} else if err := vm.ensureStackSize(dst); err != nil {
		return err
	}
	vm.Stack[dst] = val
	if dst+1 > vm.top {
		vm.setTop(dst + 1)
	}
	return nil
}

func (vm *VM) Push(vals ...Value) (int64, error) {
	addr := vm.top
	if err := vm.ensureStackSize(vm.top + int64(len(vals))); err != nil {
		return -1, err
	}
	for _, val := range vals {
		vm.Stack[vm.top] = val
		vm.top++
	}
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
		case *ExternFunc, *Closure:
			if res, err := vm.Call(mIndex, metaVal, []Value{source, key}); err != nil {
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

func (vm *VM) newIndex(table, key, value Value) error {
	return vm._newIndex(table, table, key, value)
}

func (vm *VM) _newIndex(source, table, key, value Value) error {
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
		case *ExternFunc, *Closure:
			_, err := vm.Call(mNewIndex, metaVal, []Value{table, key})
			return err
		default:
			return vm._newIndex(source, metaVal, key, value)
		}
	}
	if isTbl {
		return tbl.SetIndex(key, value)
	}
	return vm.err("attempt to index a %v value", table.Type())
}

func (vm *VM) delegateMetamethodBinop(op metaMethod, lval, rval Value) (bool, []Value, error) {
	if method := findMetavalue(op, lval); method != nil {
		ret, err := vm.Call(string(op), method, []Value{lval, rval})
		return true, ret, err
	} else if method := findMetavalue(op, rval); method != nil {
		ret, err := vm.Call(string(op), method, []Value{rval, lval})
		return true, ret, err
	}
	return false, nil, nil
}

func (vm *VM) Call(label string, fn Value, params []Value) ([]Value, error) {
	ifn, err := vm.Push(append([]Value{fn}, params...)...)
	if err != nil {
		return nil, err
	}
	lastPointer := vm.framePointer
	vm.framePointer = ifn + 1
	vm.callStack.Push(&callInfo{name: label, filename: label, LineInfo: LineInfo{}})
	defer func() {
		vm.callStack.Pop()
		vm.framePointer = lastPointer
		vm.setTop(ifn)
		vm.collectGarbage()
	}()
	nargs := int64(len(params))
	switch tfn := fn.(type) {
	case *ExternFunc:
		return tfn.val(vm, vm.argsFromStack(vm.framePointer, nargs))
	case *Closure:
		if err := vm.ensureArgsInStack(nargs, tfn.val.Arity); err != nil {
			return nil, err
		}
		values, _, err := vm.eval(tfn.val, tfn.upvalues)
		return values, err
	default:
		return nil, vm.err("expected callable but found %s", fn.Type())
	}
}

func (vm *VM) Close() error {
	_, err := stdIOClose(vm, nil)
	return err
}

func (vm *VM) cleanup(fp int64, brokers []*UpvalueBroker, tbcs []int64) error {
	for _, broker := range brokers {
		broker.Close()
	}
	for _, idx := range tbcs {
		val := vm.GetStack(fp + idx)
		if method := findMetavalue(metaClose, val); method != nil {
			if _, err := vm.Call(string(metaClose), method, []Value{val}); err != nil {
				return err
			}
		} else {
			return vm.err("__close not defined on closable table")
		}
	}
	return nil
}

func (vm *VM) setTop(newTop int64) {
	if newTop < vm.top {
		shrinkSize := int(vm.top - newTop)
		ensureSize(&vm.garbageHeap, vm.garbageSize+shrinkSize)
		copy(vm.garbageHeap[vm.garbageSize:], vm.Stack[newTop:])
		for i := newTop; i < vm.top; i++ {
			vm.Stack[i] = nil
		}
		vm.garbageSize += shrinkSize
	}
	vm.top = newTop
}

func (vm *VM) collectGarbage() {
	if vm.gcOff || vm.garbageSize < GCPAUSE {
		return
	}
	// pause gc while calling __gc metamethods so that the gc is not triggered again
	// while it is running. All garbage will be added to the end of the heap while
	// __gc is called so this is still safe
	vm.gcOff = true
	for i := 0; i <= GCPAUSE; i++ {
		val := vm.garbageHeap[i]
		if method := findMetavalue(metaGC, val); method != nil {
			if _, err := vm.Call(string(metaGC), method, []Value{val}); err != nil {
				errVal := err.(*Error)
				str, _ := toString(vm, errVal.val)
				Warn(str.val)
			}
		}
	}
	copy(vm.garbageHeap, vm.garbageHeap[GCPAUSE:])
	if vm.garbageSize-GCPAUSE > 0 {
		for i := GCPAUSE; i < vm.garbageSize; i++ {
			vm.garbageHeap[i] = nil
		}
	}
	vm.garbageSize -= GCPAUSE
	vm.gcOff = false
}
