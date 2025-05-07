package luaf

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"sync"
)

type (
	// LoadMode are flags to indicate how to load/parse a chunk of data.
	LoadMode uint
	frame    struct {
		prev         *frame // parent frames
		fn           *FnProto
		xargs        []any
		upvals       []*upvalueBroker // upvals passed to the scope
		openBrokers  []*upvalueBroker // upvals created by the scope
		tbcValues    []int64          // values that require closing
		framePointer int64            // stack pointer to 0 of the running frame
		pc           int64
	}
	callInfo struct {
		filename string
		name     string
		LineInfo
	}
	VM struct {
		ctx        context.Context
		env        *Table
		yieldFrame *frame
		vmargs     []any
		Stack      []any

		callDepth int64
		callStack []callInfo
		top       int64 // end of stack the rest may be garbage
		stackLock sync.Mutex
		gcOff     bool

		yieldable bool
		yielded   bool
	}
	RuntimeErr struct {
		addr  string
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
	// ModeText implies that the chunk of text being loaded is plain text.
	ModeText LoadMode = 0b01
	// ModeBinary implies that the chunk of data being loaded is pre parsed binary.
	ModeBinary LoadMode = 0b10

	// InterruptExit will interrupt the vm and exit the entire application.
	InterruptExit InterruptKind = iota
	// InterruptYield is only allowed in coroutines and will yield the coroutine to the parent.
	InterruptYield
	// InterruptDebug will interrupt the vm and start a repl in the context where debug was called.
	InterruptDebug
)

var forNumNames = []string{"initial", "limit", "step"}

func (err *RuntimeErr) Error() string {
	return fmt.Sprintf(`%v %v
stack traceback:
%v`, err.addr, err.msg, err.trace)
}

func (interrupt *Interrupt) Error() string {
	return fmt.Sprintf("VM interrupt %v", interrupt.kind)
}

func (i *callInfo) String() string {
	return fmt.Sprintf("%v:%v: in %v", i.filename, i.Line, i.name)
}

// NewVM will create a new vm for evaluating. It will establish the initial stack,
// setup the environment and globals, and make any extra arguments provided available
// as the arg value in luaf.
func NewVM(ctx context.Context, clargs ...string) *VM {
	env := createDefaultEnv(true)
	env.hashtable["_G"] = env

	splitidx := slices.Index(clargs, "--")
	if splitidx == -1 {
		splitidx = len(clargs)
	} else {
		splitidx++
	}

	argValues := make([]any, len(clargs))
	for i, a := range clargs {
		argValues[i] = a
	}

	argVal := NewTable(argValues[splitidx:], nil)
	for i := range splitidx {
		argVal.hashtable[int64(-(splitidx-i)+1)] = argValues[i]
	}

	env.hashtable["arg"] = argVal
	newEnv := newEnvVM(ctx, env)
	newEnv.vmargs = argValues[splitidx:]
	return newEnv
}

func newEnvVM(ctx context.Context, env *Table) *VM {
	return &VM{
		ctx:       ctx,
		callStack: make([]callInfo, 100),
		Stack:     make([]any, INITIALSTACKSIZE),
		top:       0,
		env:       env,
	}
}

func (vm *VM) runtimeErr(li LineInfo, err error) error {
	var rerr *RuntimeErr
	if errors.As(err, &rerr) {
		return rerr
	}
	ci := callInfo{LineInfo: li}
	var uerr *UserError
	if errors.As(err, &uerr) {
		if csl := len(vm.callStack); csl > 0 && uerr.level > 0 && uerr.level < csl {
			ci = vm.callStack[uerr.level]
		}
	} else if len(vm.callStack) > 0 {
		ci.filename = vm.callStack[len(vm.callStack)-1].filename
	}

	parts := []string{}
	for i := range vm.callDepth {
		parts = append(parts, fmt.Sprintf("\t%v", vm.callStack[i]))
	}
	return &RuntimeErr{
		addr:  fmt.Sprintf("lua:%v:%v:%v ", ci.filename, ci.Line, ci.Column),
		msg:   err.Error(),
		trace: strings.Join(parts, "\n"),
	}
}

// Eval will take in the parsed fnproto returned from parse and evaluate it.
func (vm *VM) Eval(fn *FnProto) ([]any, error) {
	// push the fn because the vm always expects that the fn value is at framePointer-1
	ifn, err := vm.push(&Closure{val: fn})
	if err != nil {
		return nil, err
	}
	vm.pushCallstack(fn)
	return vm.eval(vm.newEnvFrame(fn, ifn+1, vm.vmargs))
}

func (vm *VM) pushCallstack(fn *FnProto) {
	ensureSize(&vm.callStack, int(vm.callDepth+1))
	vm.callStack[vm.callDepth].LineInfo = fn.LineInfo
	vm.callStack[vm.callDepth].name = fn.Name
	vm.callStack[vm.callDepth].filename = fn.Filename
	vm.callDepth++
}

func (vm *VM) pushCoreCall(name string) {
	ensureSize(&vm.callStack, int(vm.callDepth+1))
	vm.callStack[vm.callDepth].LineInfo = LineInfo{}
	vm.callStack[vm.callDepth].name = name
	vm.callStack[vm.callDepth].filename = "<core>"
	vm.callDepth++
}

func (vm *VM) popCallstack() {
	vm.callDepth--
}

func (vm *VM) newEnvFrame(fn *FnProto, fp int64, xargs []any) *frame {
	return vm.newFrame(fn, fp, 0, []*upvalueBroker{{name: "_ENV", val: vm.env}}, xargs...)
}

func (vm *VM) newFrame(fn *FnProto, fp, pc int64, upvals []*upvalueBroker, xargs ...any) *frame {
	return &frame{
		fn:           fn,
		framePointer: fp,
		pc:           pc,
		xargs:        xargs,
		upvals:       upvals,
	}
}

func (vm *VM) resume() ([]any, error) {
	if !vm.yielded {
		return nil, errors.New("vm was not yielded and has no state to resume")
	}
	vm.yielded = false
	f := vm.yieldFrame
	vm.yieldFrame = nil
	return vm.eval(f)
}

func (vm *VM) eval(f *frame) ([]any, error) {
	extraArg := func(index int64) int64 {
		if index == 0 {
			f.pc++
			return int64(f.fn.ByteCodes[f.pc])
		}
		return index - 1
	}

	for {
		if err := vm.ctx.Err(); err != nil {
			// cancelled context
			return nil, errors.New("vm interrupted")
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
			err = vm.setStack(f.framePointer+instruction.getA(), instruction.getB() == 1)
			if instruction.getC() != 0 {
				f.pc++
			}
		case LOADI:
			err = vm.setStack(f.framePointer+instruction.getA(), instruction.getsBx())
		case LOADF:
			err = vm.setStack(f.framePointer+instruction.getA(), float64(instruction.getsBx()))
		case LOADNIL:
			a := instruction.getA()
			b := instruction.getBx()
			for i := a; i <= a+b; i++ {
				if err := vm.setStack(f.framePointer+i, nil); err != nil {
					return nil, vm.runtimeErr(lineInfo, err)
				}
			}
		case NEWTABLE:
			err = vm.setStack(f.framePointer+instruction.getA(), NewSizedTable(int(instruction.getB()), int(instruction.getC())))
		case ADD:
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			bVal := vm.get(f.fn, f.framePointer, b, bK)
			cVal := vm.get(f.fn, f.framePointer, c, cK)
			if val, err := arith(vm, metaAdd, bVal, cVal); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			} else if err = vm.setStack(f.framePointer+instruction.getA(), val); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			}
		case SUB:
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			bVal := vm.get(f.fn, f.framePointer, b, bK)
			cVal := vm.get(f.fn, f.framePointer, c, cK)
			if val, err := arith(vm, metaSub, bVal, cVal); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			} else if err = vm.setStack(f.framePointer+instruction.getA(), val); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			}
		case MUL:
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			bVal := vm.get(f.fn, f.framePointer, b, bK)
			cVal := vm.get(f.fn, f.framePointer, c, cK)
			if val, err := arith(vm, metaMul, bVal, cVal); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			} else if err = vm.setStack(f.framePointer+instruction.getA(), val); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			}
		case DIV:
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			bVal := vm.get(f.fn, f.framePointer, b, bK)
			cVal := vm.get(f.fn, f.framePointer, c, cK)
			if val, err := arith(vm, metaDiv, bVal, cVal); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			} else if err = vm.setStack(f.framePointer+instruction.getA(), val); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			}
		case MOD:
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			bVal := vm.get(f.fn, f.framePointer, b, bK)
			cVal := vm.get(f.fn, f.framePointer, c, cK)
			if val, err := arith(vm, metaMod, bVal, cVal); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			} else if err = vm.setStack(f.framePointer+instruction.getA(), val); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			}
		case POW:
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			bVal := vm.get(f.fn, f.framePointer, b, bK)
			cVal := vm.get(f.fn, f.framePointer, c, cK)
			if val, err := arith(vm, metaPow, bVal, cVal); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			} else if err = vm.setStack(f.framePointer+instruction.getA(), val); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			}
		case IDIV:
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			bVal := vm.get(f.fn, f.framePointer, b, bK)
			cVal := vm.get(f.fn, f.framePointer, c, cK)
			if val, err := arith(vm, metaIDiv, bVal, cVal); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			} else if err = vm.setStack(f.framePointer+instruction.getA(), val); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			}
		case BAND:
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			bVal := vm.get(f.fn, f.framePointer, b, bK)
			cVal := vm.get(f.fn, f.framePointer, c, cK)
			if val, err := arith(vm, metaBAnd, bVal, cVal); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			} else if err = vm.setStack(f.framePointer+instruction.getA(), val); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			}
		case BOR:
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			bVal := vm.get(f.fn, f.framePointer, b, bK)
			cVal := vm.get(f.fn, f.framePointer, c, cK)
			if val, err := arith(vm, metaBOr, bVal, cVal); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			} else if err = vm.setStack(f.framePointer+instruction.getA(), val); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			}
		case BXOR:
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			bVal := vm.get(f.fn, f.framePointer, b, bK)
			cVal := vm.get(f.fn, f.framePointer, c, cK)
			if val, err := arith(vm, metaBXOr, bVal, cVal); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			} else if err = vm.setStack(f.framePointer+instruction.getA(), val); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			}
		case SHL:
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			bVal := vm.get(f.fn, f.framePointer, b, bK)
			cVal := vm.get(f.fn, f.framePointer, c, cK)
			if val, err := arith(vm, metaShl, bVal, cVal); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			} else if err = vm.setStack(f.framePointer+instruction.getA(), val); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			}
		case SHR:
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			bVal := vm.get(f.fn, f.framePointer, b, bK)
			cVal := vm.get(f.fn, f.framePointer, c, cK)
			if val, err := arith(vm, metaShr, bVal, cVal); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			} else if err = vm.setStack(f.framePointer+instruction.getA(), val); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			}
		case UNM:
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			bVal := vm.get(f.fn, f.framePointer, b, bK)
			cVal := vm.get(f.fn, f.framePointer, c, cK)
			if val, err := arith(vm, metaUNM, bVal, cVal); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			} else if err = vm.setStack(f.framePointer+instruction.getA(), val); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			}
		case BNOT:
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			bVal := vm.get(f.fn, f.framePointer, b, bK)
			cVal := vm.get(f.fn, f.framePointer, c, cK)
			if val, err := arith(vm, metaBNot, bVal, cVal); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			} else if err = vm.setStack(f.framePointer+instruction.getA(), val); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			}
		case NOT:
			b, bK := instruction.getBK()
			val := !toBool(vm.get(f.fn, f.framePointer, b, bK))
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
					result = ToString(result) + ToString(next)
				} else if didDelegate, res, err := vm.delegateMetamethodBinop(metaConcat, result, next); err != nil {
					return nil, vm.runtimeErr(lineInfo, err)
				} else if didDelegate && len(res) > 0 {
					result = res[0]
				} else {
					return nil, vm.runtimeErr(lineInfo, fmt.Errorf("attempted to concatenate a %v value", TypeName(next)))
				}
			}
			err = vm.setStack(f.framePointer+instruction.getA(), result)
		case TBC:
			f.tbcValues = append(f.tbcValues, f.framePointer+instruction.getA())
		case JMP:
			if from := instruction.getA() - 1; from >= 0 {
				vm.closeRange(f, from)
			}
			f.pc += instruction.getsBx()
		case CLOSE:
			vm.closeRange(f, instruction.getA())
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
			bVal := vm.get(f.fn, f.framePointer, b, bK)
			cVal := vm.get(f.fn, f.framePointer, c, cK)
			if res, err := compareVal(vm, metaLt, bVal, cVal); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			} else if isMatch := res < 0; isMatch != expected {
				f.pc++
			}
		case LE:
			expected := instruction.getA() != 0
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			bVal := vm.get(f.fn, f.framePointer, b, bK)
			cVal := vm.get(f.fn, f.framePointer, c, cK)
			if res, err := compareVal(vm, metaLe, bVal, cVal); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			} else if isMatch := res <= 0; isMatch != expected {
				f.pc++
			}
		case TEST:
			expected := instruction.getB() != 0
			actual := toBool(vm.getStack(f.framePointer + instruction.getA()))
			if expected != actual {
				f.pc++
			}
		case LEN:
			b, bK := instruction.getBK()
			val := vm.get(f.fn, f.framePointer, b, bK)
			dst := f.framePointer + instruction.getA()
			if isString(val) {
				err = vm.setStack(dst, int64(len(val.(string))))
			} else if tbl, isTbl := val.(*Table); isTbl {
				if method := findMetavalue(metaLen, tbl); method != nil {
					res, err := vm.call(method, []any{tbl})
					if err != nil {
						return nil, vm.runtimeErr(lineInfo, err)
					} else if len(res) > 0 {
						if err = vm.setStack(dst, res[0]); err != nil {
							return nil, vm.runtimeErr(lineInfo, err)
						}
					} else if err = vm.setStack(dst, nil); err != nil {
						return nil, vm.runtimeErr(lineInfo, err)
					}
				} else {
					if err = vm.setStack(dst, int64(len(tbl.val))); err != nil {
						return nil, vm.runtimeErr(lineInfo, err)
					}
				}
			} else {
				err = vm.runtimeErr(lineInfo, fmt.Errorf("attempt to get length of a %v value", TypeName(val)))
			}
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
			if !keyK {
				vm.Stack[f.framePointer+keyIdx] = nil
			}
			if !valueK {
				vm.Stack[f.framePointer+valueIdx] = nil
			}
		case SETLIST:
			itbl := instruction.getA()
			tbl, ok := vm.getStack(f.framePointer + itbl).(*Table)
			if !ok {
				return nil, vm.runtimeErr(lineInfo,
					fmt.Errorf("attempt to index a %v value",
						TypeName(vm.getStack(f.framePointer+instruction.getA()))))
			}
			start := itbl + 1
			nvals := (instruction.getB() - 1)
			if nvals < 0 {
				nvals = vm.top - start - 1
			}
			index := extraArg(instruction.getC())
			ensureSize(&tbl.val, int(index+nvals)-1)
			for i := range nvals {
				tbl.val[i+index] = vm.getStack(f.framePointer + start + i)
			}
			vm.top = f.framePointer + itbl + 1
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
			ifn := f.framePointer + instruction.getA()
			nargs := instruction.getB() - 1
			nret := instruction.getC() - 1
			fnVal := vm.getStack(ifn)

			if instruction.op() == TAILCALL {
				vm.cleanup(f)
				copy(vm.Stack[f.framePointer-1:], vm.Stack[ifn:])
				vm.top -= (ifn - f.framePointer - 1)
				ifn = f.framePointer - 1
				f = f.prev
			}
		CALLLOOP:
			for {
				switch tfn := fnVal.(type) {
				case *Closure:
					vm.pushCallstack(tfn.val)
					var xargs []any
					if ifn+1+tfn.val.Arity < vm.top {
						xargs = make([]any, max(vm.top-(ifn+tfn.val.Arity)-1, 0))
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
					if diff := f.fn.Arity - nargs; nargs > 0 && diff > 0 {
						for i := nargs; i <= f.fn.Arity; i++ {
							if err := vm.setStack(f.framePointer+i, nil); err != nil {
								return nil, vm.runtimeErr(lineInfo, err)
							}
						}
					}
					break CALLLOOP
				case *GoFunc:
					vm.pushCoreCall(tfn.name)
					retVals, err := tfn.val(vm, vm.argsFromStack(ifn+1, nargs))
					if err != nil {
						var inrp *Interrupt
						if errors.As(err, &inrp) {
							switch inrp.kind {
							case InterruptExit:
								if inrp.flag {
									vm.cleanShutdown(f)
								}
								os.Exit(inrp.code)
							case InterruptYield:
								if !vm.yieldable {
									return nil, vm.runtimeErr(lineInfo, errors.New("cannot yield on the main thread"))
								}
								f.pc++
								vm.yieldFrame = f
								vm.yielded = true
								return retVals, inrp
							case InterruptDebug:
								replfn := newFnProtoFrom(f.fn)
								replframe := vm.newEnvFrame(replfn, f.framePointer, f.xargs)
								if err := vm.repl(replframe); err != nil {
									return nil, vm.runtimeErr(lineInfo, err)
								}
							}
						} else {
							return nil, vm.runtimeErr(lineInfo, err)
						}
					}
					vm.popCallstack()
					vm.top = ifn
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
					return nil, vm.runtimeErr(lineInfo,
						fmt.Errorf("expected callable but found %s",
							TypeName(vm.getStack(f.framePointer+ifn))))
				}
			}
		case RETURN:
			addr := f.framePointer + instruction.getA()
			nret := (instruction.getB() - 1)
			if nret == -1 {
				nret = vm.top - (f.framePointer + instruction.getA())
			}
			vm.cleanup(f)
			if f.prev == nil {
				retVals := make([]any, nret)
				copy(retVals, vm.Stack[addr:addr+nret])
				for i, val := range retVals {
					if val == nil {
						retVals[i] = nil
					}
				}
				vm.top = 0
				return retVals, nil
			}

			copy(vm.Stack[f.framePointer-1:], vm.Stack[addr:addr+nret])
			vm.top = f.framePointer - 1 + nret

			retVals := (vm.top - (f.framePointer - 1))
			if retVals < nret {
				for range nret - retVals {
					if _, err := vm.push(nil); err != nil {
						return nil, err
					}
				}
			} else if nret == 0 {
				if _, err := vm.push(nil); err != nil {
					return nil, err
				}
			}
			f = f.prev
		case VARARG:
			vm.top = f.framePointer + instruction.getA()
			_, err = vm.push(ensureLenNil(f.xargs, int(instruction.getB()-1))...)
		case CLOSURE:
			cls := f.fn.FnTable[instruction.getB()]
			closureUpvals := make([]*upvalueBroker, len(cls.UpIndexes))
			for i, idx := range cls.UpIndexes {
				if idx.FromStack {
					if j, ok := search(f.openBrokers, uint64(f.framePointer)+uint64(idx.Index), findBroker); ok {
						closureUpvals[i] = f.openBrokers[j]
					} else {
						newBroker := vm.newUpValueBroker(
							idx.Name,
							vm.getStack(f.framePointer+int64(idx.Index)),
							uint64(f.framePointer)+uint64(idx.Index),
						)
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
				switch vm.getStack(f.framePointer + i).(type) {
				case int64:
				case float64:
					hasFloat = true
				default:
					return nil, vm.runtimeErr(lineInfo, fmt.Errorf("non-numeric %v value", forNumNames[i]))
				}
			}
			if hasFloat {
				for i := ivar; i < ivar+3; i++ {
					if _, ok := vm.getStack(f.framePointer + i).(int64); !ok {
						fVal := toFloat(vm.getStack(f.framePointer + i))
						if err := vm.setStack(f.framePointer+i, fVal); err != nil {
							return nil, vm.runtimeErr(lineInfo, err)
						}
					}
				}
			}
			if toFloat(vm.getStack(f.framePointer+ivar+2)) == 0 {
				return nil, vm.runtimeErr(lineInfo, errors.New("0 Step in numerical for"))
			}

			i := vm.getStack(f.framePointer + ivar)
			step := vm.getStack(f.framePointer + ivar + 2)
			if iVal, isInt := i.(int64); isInt {
				stepVal := step.(int64)
				err = vm.setStack(f.framePointer+ivar, iVal-stepVal)
			} else {
				iVal := i.(float64)
				stepVal := step.(float64)
				err = vm.setStack(f.framePointer+ivar, iVal-stepVal)
			}
			f.pc += instruction.getsBx()
		case FORLOOP:
			ivar := instruction.getA()
			i := vm.getStack(f.framePointer + ivar)
			limit := vm.getStack(f.framePointer + ivar + 1)
			step := vm.getStack(f.framePointer + ivar + 2)
			if iVal, isInt := i.(int64); isInt {
				stepVal := step.(int64)
				err = vm.setStack(f.framePointer+ivar, iVal+stepVal)
			} else {
				iVal := i.(float64)
				stepVal := step.(float64)
				err = vm.setStack(f.framePointer+ivar, iVal+stepVal)
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
			var ctrl any
			if len(values) > 0 {
				ctrl = values[0]
			}
			if err := vm.setStack(f.framePointer+idx+2, ctrl); err != nil {
				return nil, vm.runtimeErr(lineInfo, err)
			}
			// TODO set range instead of iteration
			for i := range instruction.getB() {
				var val any
				if i < int64(len(values)) {
					val = values[i]
				}
				if err := vm.setStack(f.framePointer+idx+i+3, val); err != nil {
					return nil, vm.runtimeErr(lineInfo, err)
				}
			}
		case TFORLOOP:
			idx := instruction.getA()
			control := vm.getStack(f.framePointer + idx + 1)
			if control != nil {
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

func (vm *VM) argsFromStack(offset, nargs int64) []any {
	args := []any{}
	if nargs <= 0 {
		nargs = vm.top - offset
	}
	for _, val := range vm.Stack[offset : offset+nargs] {
		if val != nil {
			args = append(args, val)
		} else {
			args = append(args, nil)
		}
	}
	if diff := int(nargs) - len(args); diff > 0 {
		for range diff {
			args = append(args, nil)
		}
	}
	return args
}

func (vm *VM) get(fn *FnProto, fp, id int64, isConst bool) any {
	if isConst {
		return fn.getConst(id)
	}
	return vm.getStack(fp + id)
}

func (vm *VM) getStack(id int64) any {
	if id >= vm.top || id < 0 || vm.Stack[id] == nil {
		return nil
	}
	return vm.Stack[id]
}

func (vm *VM) setStack(dst int64, val any) error {
	if dst < 0 {
		return errors.New("cannot address negatively in the stack")
	} else if err := vm.ensureStackSize(dst); err != nil {
		return err
	}
	vm.Stack[dst] = val
	if dst+1 > vm.top {
		vm.top = dst + 1
	}
	return nil
}

func (vm *VM) push(vals ...any) (int64, error) {
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
	return addr, nil
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
	newSlice := make([]any, sliceLen+growthAmount)
	copy(newSlice, vm.Stack)
	vm.Stack = newSlice
	return nil
}

func (vm *VM) index(source, table, key any) (any, error) {
	if table == nil {
		table = source
	}
	tbl, isTable := table.(*Table)
	if isTable {
		res, err := tbl.Index(key)
		if err != nil {
			return nil, err
		} else if res != nil {
			return res, nil
		}
	}
	metatable := getMetatable(table)
	mIndex := string(metaIndex)
	if metatable != nil && metatable.hashtable[mIndex] != nil {
		switch metaVal := metatable.hashtable[mIndex].(type) {
		case *GoFunc, *Closure:
			if res, err := vm.call(metaVal, []any{source, key}); err != nil {
				return nil, err
			} else if len(res) > 0 {
				return res[0], nil
			}
			return nil, nil
		default:
			return vm.index(source, metaVal, key)
		}
	}
	if isTable {
		return nil, nil
	}
	return nil, fmt.Errorf("attempt to index a %v value", TypeName(table))
}

func (vm *VM) newIndex(table, key, value any) error {
	tbl, isTbl := table.(*Table)
	if isTbl {
		res, err := tbl.Index(key)
		if err != nil {
			return err
		} else if res != nil {
			return tbl.SetIndex(key, value)
		}
	}
	metatable := getMetatable(table)
	mNewIndex := string(metaNewIndex)
	if metatable != nil && metatable.hashtable[mNewIndex] != nil {
		switch metaVal := metatable.hashtable[mNewIndex].(type) {
		case *GoFunc, *Closure:
			_, err := vm.call(metaVal, []any{table, key})
			return err
		default:
			return vm.newIndex(metaVal, key, value)
		}
	}
	if isTbl {
		return tbl.SetIndex(key, value)
	}
	return fmt.Errorf("attempt to index a %v value", TypeName(table))
}

func (vm *VM) delegateMetamethodBinop(op metaMethod, lval, rval any) (bool, []any, error) {
	if method := findMetavalue(op, lval); method != nil {
		ret, err := vm.call(method, []any{lval, rval})
		return true, ret, err
	} else if method := findMetavalue(op, rval); method != nil {
		ret, err := vm.call(method, []any{rval, lval})
		return true, ret, err
	}
	return false, nil, nil
}

func (vm *VM) call(fn any, params []any) ([]any, error) {
	switch tfn := fn.(type) {
	case *GoFunc:
		vm.pushCoreCall(tfn.name)
		defer vm.popCallstack()
		return tfn.val(vm, params)
	case *Closure:
		vm.pushCallstack(tfn.val)
		ifn, err := vm.push(append([]any{tfn}, params...)...)
		if err != nil {
			return nil, err
		}
		return vm.eval(&frame{
			fn:           tfn.val,
			framePointer: ifn + 1,
			upvals:       tfn.upvalues,
		})
	case nil:
		return nil, errors.New("expected callable but found nil")
	default:
		return nil, fmt.Errorf("expected callable but found %s", TypeName(fn))
	}
}

func (vm *VM) toString(val any) (string, error) {
	switch tin := val.(type) {
	case *Table:
		if mt := getMetatable(val); mt != nil && mt.hashtable[string(metaToString)] != nil {
			res, err := vm.call(mt.hashtable[string(metaToString)], []any{val})
			if err != nil {
				return "", err
			} else if len(res) == 0 {
				return "", nil
			}
			return vm.toString(res[0])
		}
		return fmt.Sprintf("table %p", tin.val), nil
	default:
		return ToString(val), nil
	}
}

func (vm *VM) cleanShutdown(f *frame) {
	for f != nil {
		vm.cleanup(f)
		f = f.prev
	}
	for i := range vm.Stack[:vm.top] {
		vm.Stack[i] = nil
	}
	vm.top = 0
	_ = vm.Close()
}

// Close shuts down the vm cleanly and ensures all open files are closed.
func (vm *VM) Close() error {
	_, err := stdIOClose(vm, nil)
	return err
}

func (vm *VM) cleanup(f *frame) {
	vm.popCallstack()
	for _, broker := range f.openBrokers {
		broker.Close()
	}
	for _, idx := range f.tbcValues {
		val := vm.getStack(idx)
		if method := findMetavalue(metaClose, val); method != nil {
			if _, err := vm.call(method, []any{val}); err != nil {
				var rerr *RuntimeErr
				errors.As(err, &rerr)
				Warn(rerr.msg)
			}
		} else {
			Warn("__close not defined on closable table")
		}
	}
}

func (vm *VM) closeRange(f *frame, newTop int64) {
	for i := newTop; i < vm.top && i < int64(len(vm.Stack)); i++ {
		if j, ok := search(f.openBrokers, uint64(f.framePointer+i), findBroker); ok {
			f.openBrokers[j].Close()
			f.openBrokers = append(f.openBrokers[:j], f.openBrokers[j+1:]...) // remove broker
		}
	}
	vm.top = f.framePointer + newTop
}

func ensureLenNil(values []any, want int) []any {
	if want <= 0 {
		return values
	} else if len(values) > want {
		values = values[:want:want]
	} else if len(values) < want {
		for range want - len(values) {
			values = append(values, nil)
		}
	}
	return values
}

// ensures that we can safely use an index if required.
func ensureSize[T any](slice *[]T, index int) {
	sliceLen := len(*slice)
	if index < sliceLen {
		return
	}
	newSlice := make([]T, index+1)
	copy(newSlice, *slice)
	*slice = newSlice
}
