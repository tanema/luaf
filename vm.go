package luaf

import (
	"context"
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"
	"sync"
)

type (
	LoadMode      uint
	UpvalueBroker struct {
		index     uint64
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
	// VMState struct {
	//	prev           *VMState
	//	next           *VMState
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

func (vm *VM) Env() *Table {
	return vm.env
}

func (vm *VM) Eval(fn *FnProto, args ...string) ([]Value, error) {
	return vm.EvalEnv(fn, vm.env)
}

func (vm *VM) EvalEnv(fn *FnProto, env *Table) ([]Value, error) {
	envUpval := &UpvalueBroker{name: "_ENV", val: env}
	values, _, err := vm.eval(fn, []*UpvalueBroker{envUpval})
	return values, err
}

func (vm *VM) eval(fn *FnProto, upvals []*UpvalueBroker) ([]Value, int64, error) {
	var programCounter int64
	xargs := vm.truncateGet(vm.framePointer + fn.Arity)
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
			val, err := vm.arith(metaAdd, vm.Get(fn, vm.framePointer, b, bK), vm.Get(fn, vm.framePointer, c, cK))
			if err != nil {
				return nil, programCounter, err
			} else if err = vm.SetStack(vm.framePointer+instruction.getA(), val); err != nil {
				return nil, programCounter, err
			}
		case SUB:
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			val, err := vm.arith(metaSub, vm.Get(fn, vm.framePointer, b, bK), vm.Get(fn, vm.framePointer, c, cK))
			if err != nil {
				return nil, programCounter, err
			} else if err = vm.SetStack(vm.framePointer+instruction.getA(), val); err != nil {
				return nil, programCounter, err
			}
		case MUL:
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			val, err := vm.arith(metaMul, vm.Get(fn, vm.framePointer, b, bK), vm.Get(fn, vm.framePointer, c, cK))
			if err != nil {
				return nil, programCounter, err
			} else if err = vm.SetStack(vm.framePointer+instruction.getA(), val); err != nil {
				return nil, programCounter, err
			}
		case DIV:
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			val, err := vm.arith(metaDiv, vm.Get(fn, vm.framePointer, b, bK), vm.Get(fn, vm.framePointer, c, cK))
			if err != nil {
				return nil, programCounter, err
			} else if err = vm.SetStack(vm.framePointer+instruction.getA(), val); err != nil {
				return nil, programCounter, err
			}
		case MOD:
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			val, err := vm.arith(metaMod, vm.Get(fn, vm.framePointer, b, bK), vm.Get(fn, vm.framePointer, c, cK))
			if err != nil {
				return nil, programCounter, err
			} else if err = vm.SetStack(vm.framePointer+instruction.getA(), val); err != nil {
				return nil, programCounter, err
			}
		case POW:
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			val, err := vm.arith(metaPow, vm.Get(fn, vm.framePointer, b, bK), vm.Get(fn, vm.framePointer, c, cK))
			if err != nil {
				return nil, programCounter, err
			} else if err = vm.SetStack(vm.framePointer+instruction.getA(), val); err != nil {
				return nil, programCounter, err
			}
		case IDIV:
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			val, err := vm.arith(metaIDiv, vm.Get(fn, vm.framePointer, b, bK), vm.Get(fn, vm.framePointer, c, cK))
			if err != nil {
				return nil, programCounter, err
			} else if err = vm.SetStack(vm.framePointer+instruction.getA(), val); err != nil {
				return nil, programCounter, err
			}
		case BAND:
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			val, err := vm.arith(metaBAnd, vm.Get(fn, vm.framePointer, b, bK), vm.Get(fn, vm.framePointer, c, cK))
			if err != nil {
				return nil, programCounter, err
			} else if err = vm.SetStack(vm.framePointer+instruction.getA(), val); err != nil {
				return nil, programCounter, err
			}
		case BOR:
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			val, err := vm.arith(metaBOr, vm.Get(fn, vm.framePointer, b, bK), vm.Get(fn, vm.framePointer, c, cK))
			if err != nil {
				return nil, programCounter, err
			} else if err = vm.SetStack(vm.framePointer+instruction.getA(), val); err != nil {
				return nil, programCounter, err
			}
		case BXOR:
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			val, err := vm.arith(metaBXOr, vm.Get(fn, vm.framePointer, b, bK), vm.Get(fn, vm.framePointer, c, cK))
			if err != nil {
				return nil, programCounter, err
			} else if err = vm.SetStack(vm.framePointer+instruction.getA(), val); err != nil {
				return nil, programCounter, err
			}
		case SHL:
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			val, err := vm.arith(metaShl, vm.Get(fn, vm.framePointer, b, bK), vm.Get(fn, vm.framePointer, c, cK))
			if err != nil {
				return nil, programCounter, err
			} else if err = vm.SetStack(vm.framePointer+instruction.getA(), val); err != nil {
				return nil, programCounter, err
			}
		case SHR:
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			val, err := vm.arith(metaShr, vm.Get(fn, vm.framePointer, b, bK), vm.Get(fn, vm.framePointer, c, cK))
			if err != nil {
				return nil, programCounter, err
			} else if err = vm.SetStack(vm.framePointer+instruction.getA(), val); err != nil {
				return nil, programCounter, err
			}
		case UNM:
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			val, err := vm.arith(metaUNM, vm.Get(fn, vm.framePointer, b, bK), vm.Get(fn, vm.framePointer, c, cK))
			if err != nil {
				return nil, programCounter, err
			} else if err = vm.SetStack(vm.framePointer+instruction.getA(), val); err != nil {
				return nil, programCounter, err
			}
		case BNOT:
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			val, err := vm.arith(metaBNot, vm.Get(fn, vm.framePointer, b, bK), vm.Get(fn, vm.framePointer, c, cK))
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
				} else if didDelegate, res, err := vm.delegateMetamethod(metaConcat, result, next); err != nil {
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
			if isEq, err := vm.eq(lVal, rVal); err != nil {
				return nil, programCounter, err
			} else if isEq != expected {
				programCounter++
			}
		case LT:
			expected := instruction.getA() != 0
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			res, err := vm.compareVal(metaLt, vm.Get(fn, vm.framePointer, b, bK), vm.Get(fn, vm.framePointer, c, cK))
			if err != nil {
				return nil, programCounter, err
			} else if isMatch := res < 0; isMatch != expected {
				programCounter++
			}
		case LE:
			expected := instruction.getA() != 0
			b, bK := instruction.getBK()
			c, cK := instruction.getCK()
			res, err := vm.compareVal(metaLe, vm.Get(fn, vm.framePointer, b, bK), vm.Get(fn, vm.framePointer, c, cK))
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
			if isString(val) {
				err = vm.SetStack(vm.framePointer+instruction.getA(), &Integer{val: int64(len(val.(*String).val))})
			} else if tbl, isTbl := val.(*Table); isTbl {
				if didDelegate, res, err := vm.delegateMetamethod(metaLen, tbl); err != nil {
					return nil, programCounter, err
				} else if didDelegate && len(res) > 0 {
					if err = vm.SetStack(vm.framePointer+instruction.getA(), res[0]); err != nil {
						return nil, programCounter, err
					}
				} else {
					if err = vm.SetStack(vm.framePointer+instruction.getA(), &Integer{val: int64(len(tbl.val))}); err != nil {
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
			fnVal := vm.GetStack(vm.framePointer + ifn)
			args := vm.argsFromStack(vm.framePointer+ifn+1, nargs)
			vm.framePointer += ifn + 1
			vm.callStack.Push(&callInfo{name: fn.Name, filename: fn.Filename, LineInfo: linfo})
			var retVals []Value
			switch tfn := fnVal.(type) {
			case *Closure:
				if retVals, err = tfn.Call(vm, nargs); err != nil {
					return nil, programCounter, err
				}
			case *ExternFunc:
				if retVals, err = tfn.Call(vm, nargs); err != nil {
					return nil, programCounter, err
				}
			default:
				if method, isCallable := findMetamethod(metaCall, fnVal); !isCallable {
					return nil, programCounter, vm.err("expected callable but found %v", fnVal.Type())
				} else if isNil(method) {
					return nil, programCounter, vm.err("could not find metavalue __call")
				} else if retVals, err = vm.Call("__call", method.(callable), append([]Value{fnVal}, args...)); err != nil {
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

			newFn, isCallable := vm.Stack[vm.framePointer-1].(callable)
			if !isCallable {
				return nil, programCounter, vm.err("expected callable but found %v", vm.Stack[vm.framePointer-1].Type())
			}
			retVals, err := newFn.Call(vm, instruction.getB()-1)
			if err != nil {
				return nil, programCounter, err
			}
			vm.setTop(vm.framePointer)
			vm.collectGarbage()
			if _, err = vm.Push(retVals...); err != nil {
				return nil, programCounter, err
			}
		case RETURN:
			nret := (instruction.getB() - 1)
			if err := vm.cleanup(vm.framePointer, openBrokers, tbcValues); err != nil {
				return nil, programCounter, err
			}
			retVals := vm.truncateGet(vm.framePointer + instruction.getA())
			if nret > 0 && len(retVals) > int(nret) {
				retVals = retVals[:nret]
			} else if len(retVals) < int(nret) {
				retVals = ensureLenNil(retVals, int(nret))
			} else if nret == 0 {
				retVals = nil
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
			fn, isCallable := vm.GetStack(vm.framePointer + idx).(callable)
			if !isCallable {
				return nil, programCounter, vm.err("iterator not callable, found: %v", vm.GetStack(vm.framePointer+idx))
			}
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

func (vm *VM) truncateGet(dst int64) []Value {
	if dst >= vm.top || dst < 0 {
		return nil
	}
	out := make([]Value, vm.top-dst)
	copy(out, vm.Stack[dst:vm.top])
	vm.setTop(dst)
	return out
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
		if rval == 0 {
			return int64(math.Inf(1))
		}
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
		if rval > 0 {
			return lval << rval
		} else {
			return lval >> int64(math.Abs(float64(rval)))
		}
	case metaShr:
		if rval > 0 {
			return lval >> rval
		} else {
			return lval << int64(math.Abs(float64(rval)))
		}
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

func (vm *VM) eq(lVal, rVal Value) (bool, error) {
	typeA, typeB := lVal.Type(), rVal.Type()
	if typeA != typeB {
		return false, nil
	}
	switch lVal.(type) {
	case *String:
		strA, strB := lVal.(*String), rVal.(*String)
		return strA.val == strB.val, nil
	case *Integer, *Float:
		vA, vB := toFloat(lVal), toFloat(rVal)
		return vA == vB, nil
	case *Boolean:
		strA, strB := lVal.(*Boolean), rVal.(*Boolean)
		return strA.val == strB.val, nil
	case *Nil:
		return true, nil
	case *Table:
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
	case *Closure:
		return lVal.Val() == rVal.Val(), nil
	case *ExternFunc:
		return lVal == rVal, nil
	default:
		return false, nil
	}
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
		case callable:
			_, err := vm.Call(mNewIndex, metaVal, []Value{table, key})
			if err != nil {
				return err
			}
			return nil
		default:
			return vm._newIndex(source, metaVal, key, value)
		}
	}
	if isTbl {
		return tbl.SetIndex(key, value)
	}
	return vm.err("attempt to index a %v value", table.Type())
}

func (vm *VM) delegateMetamethod(op metaMethod, params ...Value) (bool, []Value, error) {
	if method, isCallable := findMetamethod(op, params...); isCallable && !isNil(method) {
		ret, err := vm.Call(string(op), method.(callable), params)
		return true, ret, err
	} else if !isNil(method) {
		return true, []Value{method}, nil
	}
	return false, nil, nil // unable to delegate
}

func (vm *VM) Call(label string, fn callable, params []Value) ([]Value, error) {
	ifn, err := vm.Push(append([]Value{fn.(Value)}, params...)...)
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
	return fn.Call(vm, int64(len(params)))
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
		if didDelegate, _, err := vm.delegateMetamethod(metaClose, vm.GetStack(fp+idx)); err != nil {
			return err
		} else if !didDelegate {
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
		if _, _, err := vm.delegateMetamethod(metaGC, vm.garbageHeap[i]); err != nil {
			errVal := err.(*Error)
			str, _ := toString(vm, errVal.val)
			Warn(str.val)
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

func (vm *VM) newUpValueBroker(name string, val Value, index uint64) *UpvalueBroker {
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
