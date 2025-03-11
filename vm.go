package luaf

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"sync"
)

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
type (
	LoadMode uint
	frame    struct {
		prev         *frame // parent frames
		fn           *FnProto
		framePointer int64 // stack pointer to 0 of the running frame
		pc           int64
		xargs        []Value
		upvals       []*upvalueBroker // upvals passed to the scope
		openBrokers  []*upvalueBroker // upvals created by the scope
		tbcValues    []int64          // values that require closing
	}
	callInfo struct {
		LineInfo
		filename string
		name     string
	}
	VM struct {
		ctx         context.Context
		callStack   stack[callInfo]
		yieldable   bool
		env         *Table
		gcOff       bool
		stackLock   sync.Mutex
		top         int64 // end of stack the rest may be garbage
		Stack       []Value
		garbageSize int
		garbageHeap []Value
	}
	RuntimeErr struct {
		msg   string
		trace string
	}
	InterruptKind int
	Interrupt     struct {
		kind InterruptKind
		code int
		flag bool
	}
)

const (
	ModeText   LoadMode = 0b01
	ModeBinary LoadMode = 0b10

	InterruptExit InterruptKind = iota
	InterruptYield
	InterruptDebug
)

var forNumNames = []string{"initial", "limit", "step"}

func (err *RuntimeErr) Error() string {
	return fmt.Sprintf(`%v
stack traceback:
%v`, err.msg, err.trace)
}

func (interrupt *Interrupt) Error() string {
	return fmt.Sprintf("VM interrupt %v", interrupt.kind)
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
	_, _ = newEnv.push(argValues[splitidx:]...)
	return newEnv
}

func NewEnvVM(ctx context.Context, env *Table) *VM {
	return &VM{
		ctx:         ctx,
		callStack:   newStack[callInfo](100),
		Stack:       make([]Value, INITIALSTACKSIZE),
		top:         0,
		env:         env,
		garbageSize: 0,
		garbageHeap: make([]Value, INITIALSTACKSIZE),
	}
}

func (vm *VM) runtimeErr(li LineInfo, err error) error {
	if rerr, isRuntimeErr := err.(*RuntimeErr); isRuntimeErr {
		return rerr
	}
	ci := &callInfo{LineInfo: li}
	if uerr, isUserErr := err.(*UserError); isUserErr {
		if csl := vm.callStack.Len(); csl > 0 && uerr.level > 0 && uerr.level < csl {
			ci = vm.callStack.data[uerr.level]
		}
	} else if vm.callStack.Len() > 0 {
		ci.filename = vm.callStack.Top().filename
	}
	return &RuntimeErr{
		msg:   fmt.Sprintf("lua:%v:%v:%v %v ", ci.filename, ci.Line, ci.Column, err),
		trace: printStackTrace(vm.callStack),
	}
}

func (vm *VM) Eval(fn *FnProto, params, xargs []Value) ([]Value, error) {
	// push the fn because the vm always expects that the fn value is at framePointer-1
	ifn, err := vm.push(append([]Value{&Closure{val: fn}}, params...)...)
	if err != nil {
		return nil, err
	}
	vm.callStack.Push(&callInfo{LineInfo: fn.LineInfo, name: fn.Name, filename: fn.Filename})
	return vm.eval(vm.newEnvFrame(fn, ifn+1, 0, xargs))
}

func (vm *VM) newEnvFrame(fn *FnProto, fp, pc int64, xargs []Value) *frame {
	return vm.newFrame(fn, fp, pc, []*upvalueBroker{{name: "_ENV", val: vm.env}}, xargs...)
}

func (vm *VM) newFrame(fn *FnProto, fp, pc int64, upvals []*upvalueBroker, xargs ...Value) *frame {
	return &frame{
		fn:           fn,
		framePointer: fp,
		pc:           pc,
		xargs:        xargs,
		upvals:       upvals,
	}
}

func (vm *VM) eval(f *frame) ([]Value, error) {
	extraArg := func(index int64) int64 {
		if index == 0 {
			f.pc++
			return int64(f.fn.ByteCodes[f.pc])
		} else {
			return index - 1
		}
	}

	for {
		if err := vm.ctx.Err(); err != nil {
			// cancelled context
			return nil, fmt.Errorf("vm interrupted")
		}
		var err error
		if int64(len(f.fn.ByteCodes)) <= f.pc {
			return nil, nil
		}

		instruction := f.fn.ByteCodes[f.pc]
		var lineInfo LineInfo
		// guard here really only so that line traces are not required for tests
		if f.pc < int64(len(f.fn.LineTrace)) {
			lineInfo = f.fn.LineTrace[f.pc]
		}
		switch instruction.op() {
		case MOVE:
			err = vm.setStack(f.framePointer+instruction.getA(), vm.getStack(f.framePointer+instruction.getB()))
		case LOADK:
			err = vm.setStack(f.framePointer+instruction.getA(), f.fn.getConst(instruction.getBx()))
		case LOADBOOL:
			err = vm.setStack(f.framePointer+instruction.getA(), &Boolean{val: instruction.getB() == 1})
			if instruction.getC() != 0 {
				f.pc++
			}
		case LOADI:
			err = vm.setStack(f.framePointer+instruction.getA(), &Integer{val: instruction.getsBx()})
		case LOADF:
			err = vm.setStack(f.framePointer+instruction.getA(), &Float{val: float64(instruction.getsBx())})
		case LOADNIL:
			a := instruction.getA()
			b := instruction.getBx()
			for i := a; i <= a+b; i++ {
				if err := vm.setStack(f.framePointer+i, &Nil{}); err != nil {
					return nil, vm.runtimeErr(lineInfo, err)
				}
			}
		case ADD:
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			if val, err := arith(vm, metaAdd, vm.get(f.fn, f.framePointer, b, bK), vm.get(f.fn, f.framePointer, c, cK)); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			} else if err = vm.setStack(f.framePointer+instruction.getA(), val); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			}
		case SUB:
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			if val, err := arith(vm, metaSub, vm.get(f.fn, f.framePointer, b, bK), vm.get(f.fn, f.framePointer, c, cK)); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			} else if err = vm.setStack(f.framePointer+instruction.getA(), val); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			}
		case MUL:
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			if val, err := arith(vm, metaMul, vm.get(f.fn, f.framePointer, b, bK), vm.get(f.fn, f.framePointer, c, cK)); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			} else if err = vm.setStack(f.framePointer+instruction.getA(), val); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			}
		case DIV:
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			if val, err := arith(vm, metaDiv, vm.get(f.fn, f.framePointer, b, bK), vm.get(f.fn, f.framePointer, c, cK)); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			} else if err = vm.setStack(f.framePointer+instruction.getA(), val); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			}
		case MOD:
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			if val, err := arith(vm, metaMod, vm.get(f.fn, f.framePointer, b, bK), vm.get(f.fn, f.framePointer, c, cK)); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			} else if err = vm.setStack(f.framePointer+instruction.getA(), val); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			}
		case POW:
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			if val, err := arith(vm, metaPow, vm.get(f.fn, f.framePointer, b, bK), vm.get(f.fn, f.framePointer, c, cK)); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			} else if err = vm.setStack(f.framePointer+instruction.getA(), val); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			}
		case IDIV:
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			if val, err := arith(vm, metaIDiv, vm.get(f.fn, f.framePointer, b, bK), vm.get(f.fn, f.framePointer, c, cK)); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			} else if err = vm.setStack(f.framePointer+instruction.getA(), val); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			}
		case BAND:
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			if val, err := arith(vm, metaBAnd, vm.get(f.fn, f.framePointer, b, bK), vm.get(f.fn, f.framePointer, c, cK)); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			} else if err = vm.setStack(f.framePointer+instruction.getA(), val); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			}
		case BOR:
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			if val, err := arith(vm, metaBOr, vm.get(f.fn, f.framePointer, b, bK), vm.get(f.fn, f.framePointer, c, cK)); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			} else if err = vm.setStack(f.framePointer+instruction.getA(), val); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			}
		case BXOR:
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			if val, err := arith(vm, metaBXOr, vm.get(f.fn, f.framePointer, b, bK), vm.get(f.fn, f.framePointer, c, cK)); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			} else if err = vm.setStack(f.framePointer+instruction.getA(), val); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			}
		case SHL:
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			if val, err := arith(vm, metaShl, vm.get(f.fn, f.framePointer, b, bK), vm.get(f.fn, f.framePointer, c, cK)); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			} else if err = vm.setStack(f.framePointer+instruction.getA(), val); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			}
		case SHR:
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			if val, err := arith(vm, metaShr, vm.get(f.fn, f.framePointer, b, bK), vm.get(f.fn, f.framePointer, c, cK)); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			} else if err = vm.setStack(f.framePointer+instruction.getA(), val); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			}
		case UNM:
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			if val, err := arith(vm, metaUNM, vm.get(f.fn, f.framePointer, b, bK), vm.get(f.fn, f.framePointer, c, cK)); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			} else if err = vm.setStack(f.framePointer+instruction.getA(), val); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			}
		case BNOT:
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			if val, err := arith(vm, metaBNot, vm.get(f.fn, f.framePointer, b, bK), vm.get(f.fn, f.framePointer, c, cK)); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			} else if err = vm.setStack(f.framePointer+instruction.getA(), val); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			}
		case NOT:
			b, bK := instruction.getBK()
			val := toBool(vm.get(f.fn, f.framePointer, b, bK)).Not()
			err = vm.setStack(f.framePointer+instruction.getA(), val)
		case CONCAT:
			b := instruction.getB()
			c := instruction.getC()
			if c < b {
				c = b + 1
			}
			result := vm.getStack(f.framePointer + b)
			for i := b + 1; i <= c; i++ {
				next := vm.getStack(f.framePointer + i)
				aCoercable := isString(result) || isNumber(result)
				bCoercable := isString(next) || isNumber(next)
				if aCoercable && bCoercable {
					result = &String{val: result.String() + next.String()}
				} else if didDelegate, res, err := vm.delegateMetamethodBinop(metaConcat, result, next); err != nil {
					return nil, vm.runtimeErr(lineInfo, err)
				} else if didDelegate && len(res) > 0 {
					result = res[0]
				} else {
					return nil, vm.runtimeErr(lineInfo, fmt.Errorf("attempted to concatenate a %v value", next.Type()))
				}
			}
			err = vm.setStack(f.framePointer+instruction.getA(), result)
		case TBC:
			f.tbcValues = append(f.tbcValues, f.framePointer+instruction.getA())
		case JMP:
			if from := int64(instruction.getA() - 1); from >= 0 {
				for i := from; i < vm.top; i++ {
					if j, ok := search(f.openBrokers, uint64(f.framePointer+i), findBroker); ok {
						f.openBrokers[j].Close()
						f.openBrokers = append(f.openBrokers[:j], f.openBrokers[j+1:]...) // remove broker
					}
				}
				vm.setTop(f.framePointer + from)
			}
			f.pc += instruction.getsBx()
		case CLOSE:
			from := instruction.getA()
			for i := from; i < vm.top; i++ {
				if j, ok := search(f.openBrokers, uint64(f.framePointer+i), findBroker); ok {
					f.openBrokers[j].Close()
					f.openBrokers = append(f.openBrokers[:j], f.openBrokers[j+1:]...) // remove broker
				}
			}
			vm.setTop(f.framePointer + from)
		case EQ:
			expected := instruction.getA() != 0
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			lVal := vm.get(f.fn, f.framePointer, b, bK)
			rVal := vm.get(f.fn, f.framePointer, c, cK)
			if isEq, err := eq(vm, lVal, rVal); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			} else if isEq != expected {
				f.pc++
			}
		case LT:
			expected := instruction.getA() != 0
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			if res, err := compareVal(vm, metaLt, vm.get(f.fn, f.framePointer, b, bK), vm.get(f.fn, f.framePointer, c, cK)); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			} else if isMatch := res < 0; isMatch != expected {
				f.pc++
			}
		case LE:
			expected := instruction.getA() != 0
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			if res, err := compareVal(vm, metaLe, vm.get(f.fn, f.framePointer, b, bK), vm.get(f.fn, f.framePointer, c, cK)); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			} else if isMatch := res <= 0; isMatch != expected {
				f.pc++
			}
		case TEST:
			expected := instruction.getB() != 0
			actual := toBool(vm.getStack(f.framePointer + instruction.getA())).val
			if expected != actual {
				f.pc++
			}
		case LEN:
			b, bK := instruction.getBK()
			val := vm.get(f.fn, f.framePointer, b, bK)
			dst := f.framePointer + instruction.getA()
			if isString(val) {
				err = vm.setStack(dst, &Integer{val: int64(len(val.(*String).val))})
			} else if tbl, isTbl := val.(*Table); isTbl {
				if method := findMetavalue(metaLen, tbl); method != nil {
					res, err := vm.call(method, []Value{tbl})
					if err != nil {
						return nil, vm.runtimeErr(lineInfo, err)
					} else if len(res) > 0 {
						if err = vm.setStack(dst, res[0]); err != nil {
							return nil, vm.runtimeErr(lineInfo, err)
						}
					} else if err = vm.setStack(dst, &Nil{}); err != nil {
						return nil, vm.runtimeErr(lineInfo, err)
					}
				} else {
					if err = vm.setStack(dst, &Integer{val: int64(len(tbl.val))}); err != nil {
						return nil, vm.runtimeErr(lineInfo, err)
					}
				}
			} else {
				err = vm.runtimeErr(lineInfo, fmt.Errorf("attempt to get length of a %v value", val.Type()))
			}
		case NEWTABLE:
			err = vm.setStack(f.framePointer+instruction.getA(), NewSizedTable(int(instruction.getB()), int(instruction.getC())))
		case GETTABLE:
			keyIdx, keyK := instruction.getCK()
			tbl := vm.getStack(f.framePointer + instruction.getB())
			if val, err := vm.index(tbl, nil, vm.get(f.fn, f.framePointer, keyIdx, keyK)); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			} else if err = vm.setStack(f.framePointer+instruction.getA(), val); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			}
		case SETTABLE:
			keyIdx, keyK := instruction.getBK()
			valueIdx, valueK := instruction.getCK()
			err = vm.newIndex(
				vm.getStack(f.framePointer+instruction.getA()),
				vm.get(f.fn, f.framePointer, keyIdx, keyK),
				vm.get(f.fn, f.framePointer, valueIdx, valueK),
			)
		case SETLIST:
			itbl := instruction.getA()
			tbl, ok := vm.getStack(f.framePointer + itbl).(*Table)
			if !ok {
				return nil, vm.runtimeErr(lineInfo, fmt.Errorf("attempt to index a %v value", vm.getStack(f.framePointer+instruction.getA()).Type()))
			}
			start := itbl + 1
			nvals := (instruction.getB() - 1)
			if nvals < 0 {
				nvals = vm.top - start - 1
			}
			index := extraArg(instruction.getC())
			ensureSize(&tbl.val, int(index+nvals)-1)
			for i := int64(0); i < nvals; i++ {
				tbl.val[i+index] = vm.getStack(f.framePointer + start + i)
			}
			vm.setTop(f.framePointer + itbl + 1)
		case GETUPVAL:
			err = vm.setStack(f.framePointer+instruction.getA(), f.upvals[instruction.getB()].Get())
		case SETUPVAL:
			f.upvals[instruction.getB()].Set(vm.getStack(f.framePointer + instruction.getA()))
		case GETTABUP:
			keyIdx, keyK := instruction.getCK()
			tbl := f.upvals[instruction.getB()].Get()
			if val, err := vm.index(tbl, nil, vm.get(f.fn, f.framePointer, keyIdx, keyK)); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			} else if err = vm.setStack(f.framePointer+instruction.getA(), val); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			}
		case SETTABUP:
			keyIdx, keyK := instruction.getBK()
			valueIdx, valueK := instruction.getCK()
			err = vm.newIndex(
				f.upvals[instruction.getA()].Get(),
				vm.get(f.fn, f.framePointer, keyIdx, keyK),
				vm.get(f.fn, f.framePointer, valueIdx, valueK),
			)
		case SELF:
			tbl := vm.getStack(f.framePointer + instruction.getB())
			keyIdx, keyK := instruction.getCK()
			fn, err := vm.index(tbl, nil, vm.get(f.fn, f.framePointer, keyIdx, keyK))
			if err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			}
			ra := instruction.getA()
			if err = vm.setStack(f.framePointer+ra, fn); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			} else if err = vm.setStack(f.framePointer+ra+1, tbl); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			}
		case CALL, TAILCALL:
			var ifn int64
			if instruction.op() == CALL {
				ifn = f.framePointer + instruction.getA()
			} else if instruction.op() == TAILCALL {
				vm.callStack.Pop()
				vm.cleanup(f)
				ifn := f.framePointer - 1
				frameEnd := f.framePointer + instruction.getA()
				cutout(&vm.Stack, int(ifn), int(frameEnd))
				vm.setTop(vm.top - (frameEnd - ifn))
				f = f.prev
			}
			nargs := instruction.getB() - 1
			nret := instruction.getC() - 1
			fnVal := vm.getStack(ifn)
		CALLLOOP:
			for {
				switch tfn := fnVal.(type) {
				case *Closure:
					vm.callStack.Push(tfn.callinfo())
					var xargs []Value
					if ifn+1+tfn.val.Arity < vm.top {
						xargs = make([]Value, max(vm.top-(ifn+tfn.val.Arity)-1, 0))
						copy(xargs, vm.Stack[ifn+1+tfn.val.Arity:vm.top])
					}
					f = &frame{
						prev:         f,
						fn:           tfn.val,
						framePointer: ifn + 1,
						pc:           -1, // because at the end of this instruction it will be incremented
						xargs:        xargs,
						upvals:       tfn.upvalues,
						openBrokers:  []*upvalueBroker{},
						tbcValues:    []int64{},
					}
					if err := vm.ensureArgsInStack(f, nargs); err != nil {
						return nil, vm.runtimeErr(lineInfo, err)
					}
					break CALLLOOP
				case *GoFunc:
					vm.callStack.Push(tfn.callinfo())
					retVals, err := tfn.val(vm, vm.argsFromStack(ifn+1, nargs))
					if err != nil {
						if inrp, isInter := err.(*Interrupt); isInter {
							switch inrp.kind {
							case InterruptExit:
								if inrp.flag {
									vm.cleanInterruptShutdown(f)
								}
								os.Exit(inrp.code)
							case InterruptYield:
								if !vm.yieldable {
									return nil, vm.runtimeErr(lineInfo, fmt.Errorf("cannot yield on the main thread"))
								}
								// TODO return from eval but with ability to resume
							case InterruptDebug:
								// TODO allow repl in fn context
								if err := vm.REPL(); err != nil {
									return nil, vm.runtimeErr(lineInfo, err)
								}
							}
						} else {
							return nil, vm.runtimeErr(lineInfo, err)
						}
					}
					vm.callStack.Pop()
					vm.setTop(ifn)
					if nret > 0 && len(retVals) > int(nret) {
						retVals = retVals[:nret]
					} else if len(retVals) < int(nret) {
						retVals = ensureLenNil(retVals, int(nret))
					}
					if _, err = vm.push(retVals...); err != nil {
						return nil, vm.runtimeErr(lineInfo, err)
					}
					break CALLLOOP
				case *Table:
					fnVal = findMetavalue(metaCall, fnVal)
				default:
					return nil, vm.runtimeErr(lineInfo, fmt.Errorf("expected callable but found %s", vm.getStack(f.framePointer+ifn).Type()))
				}
			}
		case RETURN:
			vm.callStack.Pop()
			vm.cleanup(f)
			ifn := f.framePointer - 1
			start := f.framePointer + instruction.getA()
			nret := (instruction.getB() - 1)
			if nret == -1 {
				nret = vm.top - start
			}
			cutout(&vm.Stack, int(ifn), int(start))
			vm.setTop(ifn + nret)
			if f.prev == nil {
				return vm.argsFromStack(ifn, nret), nil
			}
			f = f.prev
		case VARARG:
			vm.setTop(f.framePointer + instruction.getA())
			_, err = vm.push(ensureLenNil(f.xargs, int(instruction.getB()-1))...)
		case CLOSURE:
			cls := f.fn.FnTable[instruction.getB()]
			closureUpvals := make([]*upvalueBroker, len(cls.UpIndexes))
			for i, idx := range cls.UpIndexes {
				if idx.FromStack {
					if j, ok := search(f.openBrokers, uint64(f.framePointer)+uint64(idx.Index), findBroker); ok {
						closureUpvals[i] = f.openBrokers[j]
					} else {
						newBroker := vm.newUpValueBroker(idx.Name, vm.getStack(f.framePointer+int64(idx.Index)), uint64(f.framePointer)+uint64(idx.Index))
						f.openBrokers = append(f.openBrokers, newBroker)
						closureUpvals[i] = newBroker
					}
				} else {
					closureUpvals[i] = f.upvals[idx.Index]
				}
			}
			err = vm.setStack(f.framePointer+instruction.getA(), &Closure{val: cls, upvalues: closureUpvals})
		case FORPREP:
			ivar := instruction.getA()
			hasFloat := false
			for i := ivar; i < ivar+3; i++ {
				switch vm.getStack(f.framePointer + int64(i)).(type) {
				case *Integer:
				case *Float:
					hasFloat = true
				default:
					return nil, vm.runtimeErr(lineInfo, fmt.Errorf("non-numeric %v value", forNumNames[i]))
				}
			}
			if hasFloat {
				for i := ivar; i < ivar+3; i++ {
					if _, ok := vm.getStack(f.framePointer + int64(i)).(*Integer); !ok {
						if err := vm.setStack(f.framePointer+int64(i), &Float{val: toFloat(vm.getStack(f.framePointer + int64(i)))}); err != nil {
							return nil, vm.runtimeErr(lineInfo, err)
						}
					}
				}
			}
			if toFloat(vm.getStack(f.framePointer+ivar+2)) == 0 {
				return nil, vm.runtimeErr(lineInfo, fmt.Errorf("0 Step in numerical for"))
			}

			i := vm.getStack(f.framePointer + ivar)
			step := vm.getStack(f.framePointer + ivar + 2)
			if iVal, isInt := i.(*Integer); isInt {
				stepVal := step.(*Integer)
				err = vm.setStack(f.framePointer+ivar, &Integer{val: iVal.val - stepVal.val})
			} else {
				iVal := i.(*Float)
				stepVal := step.(*Float)
				err = vm.setStack(f.framePointer+ivar, &Float{val: iVal.val - stepVal.val})
			}
			f.pc += instruction.getsBx()
		case FORLOOP:
			ivar := instruction.getA()
			i := vm.getStack(f.framePointer + ivar)
			limit := vm.getStack(f.framePointer + ivar + 1)
			step := vm.getStack(f.framePointer + ivar + 2)
			if iVal, isInt := i.(*Integer); isInt {
				stepVal := step.(*Integer)
				err = vm.setStack(f.framePointer+ivar, &Integer{val: iVal.val + stepVal.val})
			} else {
				iVal := i.(*Float)
				stepVal := step.(*Float)
				err = vm.setStack(f.framePointer+ivar, &Float{val: iVal.val + stepVal.val})
			}
			i = vm.getStack(f.framePointer + ivar)
			check := (toFloat(step) > 0 && toFloat(i) <= toFloat(limit)) ||
				(toFloat(step) < 0 && toFloat(i) >= toFloat(limit))
			if check {
				f.pc += instruction.getsBx()
			}
		case TFORCALL:
			idx := instruction.getA()
			fn := vm.getStack(f.framePointer + idx)
			values, err := vm.call(fn, vm.argsFromStack(f.framePointer+idx+1, 2))
			if err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			}
			var ctrl Value = &Nil{}
			if len(values) > 0 {
				ctrl = values[0]
			}
			if err := vm.setStack(f.framePointer+idx+2, ctrl); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			}
			// TODO set range instead of iteration
			for i := 0; i < int(instruction.getB()); i++ {
				var val Value = &Nil{}
				if i < len(values) {
					val = values[i]
				}
				if err := vm.setStack(f.framePointer+idx+int64(i)+3, val); err != nil {
					return nil, vm.runtimeErr(lineInfo, err)
				}
			}
		case TFORLOOP:
			idx := instruction.getA()
			control := vm.getStack(f.framePointer + idx + 1)
			if _, isNil := control.(*Nil); !isNil {
				f.pc += instruction.getsBx()
			}
		default:
			panic("unknown opcode this should never happen")
		}
		if err != nil {
			return nil, vm.runtimeErr(lineInfo, err)
		}
		f.pc++
	}
}

func (vm *VM) ensureArgsInStack(f *frame, nargs int64) error {
	if diff := f.fn.Arity - nargs; nargs > 0 && diff > 0 {
		for i := nargs; i <= f.fn.Arity; i++ {
			if err := vm.setStack(f.framePointer+i, &Nil{}); err != nil {
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

func (vm *VM) get(fn *FnProto, fp, id int64, isConst bool) Value {
	if isConst {
		return fn.getConst(id)
	}
	return vm.getStack(fp + id)
}

func (vm *VM) getStack(id int64) Value {
	if id >= vm.top || id < 0 || vm.Stack[id] == nil {
		return &Nil{}
	}
	return vm.Stack[id]
}

func (vm *VM) setStack(dst int64, val Value) error {
	if dst < 0 {
		return errors.New("cannot address negatively in the stack")
	} else if err := vm.ensureStackSize(dst); err != nil {
		return err
	}
	vm.Stack[dst] = val
	if dst+1 > vm.top {
		vm.setTop(dst + 1)
	}
	vm.collectGarbage()
	return nil
}

func (vm *VM) push(vals ...Value) (int64, error) {
	if len(vals) == 0 {
		return vm.top, nil
	}
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
		return fmt.Errorf("stack overflow %v", index)
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
		case *GoFunc, *Closure:
			if res, err := vm.call(metaVal, []Value{source, key}); err != nil {
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
		case *GoFunc, *Closure:
			_, err := vm.call(metaVal, []Value{table, key})
			return err
		default:
			return vm._newIndex(source, metaVal, key, value)
		}
	}
	if isTbl {
		return tbl.SetIndex(key, value)
	}
	return fmt.Errorf("attempt to index a %v value", table.Type())
}

func (vm *VM) delegateMetamethodBinop(op metaMethod, lval, rval Value) (bool, []Value, error) {
	if method := findMetavalue(op, lval); method != nil {
		ret, err := vm.call(method, []Value{lval, rval})
		return true, ret, err
	} else if method := findMetavalue(op, rval); method != nil {
		ret, err := vm.call(method, []Value{rval, lval})
		return true, ret, err
	}
	return false, nil, nil
}

func (vm *VM) call(fn Value, params []Value) ([]Value, error) {
	switch tfn := fn.(type) {
	case *GoFunc:
		vm.callStack.Push(tfn.callinfo())
		defer vm.callStack.Pop()
		return tfn.val(vm, params)
	case *Closure:
		vm.callStack.Push(tfn.callinfo())
		return tfn.eval(vm, params)
	case nil:
		return nil, fmt.Errorf("expected callable but found nil")
	default:
		return nil, fmt.Errorf("expected callable but found %s", fn.Type())
	}
}

func (vm *VM) cleanInterruptShutdown(f *frame) {
	for f != nil {
		vm.cleanup(f)
		f = f.prev
	}
	vm.Close()
}

func (vm *VM) Close() error {
	_, err := stdIOClose(vm, nil)
	return err
}

func (vm *VM) cleanup(f *frame) {
	for _, broker := range f.openBrokers {
		broker.Close()
	}
	for _, idx := range f.tbcValues {
		val := vm.getStack(idx)
		if method := findMetavalue(metaClose, val); method != nil {
			if _, err := vm.call(method, []Value{val}); err != nil {
				Warn(err.Error())
			}
		} else {
			Warn("__close not defined on closable table")
		}
	}
}

func (vm *VM) setTop(newTop int64) {
	if newTop < 0 {
		newTop = 0
	}
	if newTop < vm.top {
		shrinkSize := int(vm.top - newTop)
		ensureSize(&vm.garbageHeap, vm.garbageSize+shrinkSize)
		copy(vm.garbageHeap[vm.garbageSize:], vm.Stack[newTop:])
		for i := newTop; i < vm.top && i < int64(len(vm.Stack)); i++ {
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
			if _, err := vm.call(method, []Value{val}); err != nil {
				Warn(err.Error())
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
