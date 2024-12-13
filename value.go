package luaf

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

type (
	callable interface {
		Call(vm *VM, nargs int64) ([]Value, error)
	}
	GoFunc func(*VM, []Value) ([]Value, error)
	Value  interface {
		fmt.Stringer
		Type() string
		Val() any
		Meta() *Table
	}
	Nil        struct{}
	Boolean    struct{ val bool }
	Integer    struct{ val int64 }
	Float      struct{ val float64 }
	ExternFunc struct{ val GoFunc }
	Closure    struct {
		val      *FnProto
		upvalues []*UpvalueBroker
	}
	Error struct {
		val   Value
		addr  string
		trace string
	}
)

func ToValue(in any) Value {
	switch val := unifyType(in).(type) {
	case int64:
		return &Integer{val: val}
	case float64:
		return &Float{val: val}
	case bool:
		return &Boolean{val: val}
	case string:
		return &String{val: val}
	case nil:
		return &Nil{}
	case Value:
		return val
	default:
		return nil
	}
}

func toBool(in Value) *Boolean {
	switch tin := in.(type) {
	case *Error, *String, *Closure, *ExternFunc, *Table:
		return &Boolean{val: true}
	case *Boolean:
		return tin
	case *Integer:
		return &Boolean{val: tin.val != 0}
	case *Float:
		return &Boolean{val: tin.val != 0}
	default:
		return &Boolean{val: false}
	}
}

func toKey(in Value) any {
	switch tin := in.(type) {
	case *Nil:
		panic("dont use nil as a key!")
	case *String:
		return tin.val
	case *Boolean:
		return tin.val
	case *Integer:
		return tin.val
	case *Float:
		return tin.val
	default:
		return in
	}
}

func isNumber(in Value) bool {
	switch in.(type) {
	case *Integer, *Float:
		return true
	default:
		return false
	}
}

func isString(in Value) bool {
	switch in.(type) {
	case *String:
		return true
	default:
		return false
	}
}

func toInt(val Value) int64 {
	switch tval := val.(type) {
	case *Integer:
		return tval.val
	case *Float:
		return int64(tval.val)
	default:
		return int64(math.NaN())
	}
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

func toNumber(in Value, base int) Value {
	switch tin := in.(type) {
	case *Integer, *Float:
		return in
	case *String:
		if strings.Contains(tin.val, ".") {
			fval, err := strconv.ParseFloat(tin.val, 64)
			if err != nil {
				return &Nil{}
			}
			return &Float{val: fval}
		}
		ival, err := strconv.ParseInt(tin.val, base, 64)
		if err != nil {
			return &Nil{}
		}
		return &Integer{val: ival}
	default:
		return &Nil{}
	}
}

func toString(vm *VM, val Value) (*String, error) {
	didDelegate, res, err := vm.delegateMetamethod(metaToString, val)
	if err != nil {
		return nil, err
	} else if !didDelegate || len(res) == 0 {
		didDelegate, res, err := vm.delegateMetamethod(metaName, val)
		if err != nil {
			return nil, err
		} else if !didDelegate || len(res) == 0 {
			return &String{val: val.String()}, nil
		}
		return &String{val: res[0].String()}, nil
	}
	return &String{val: res[0].String()}, nil
}

func toError(vm *VM, val Value, level int) (*Error, error) {
	didDelegate, res, err := vm.delegateMetamethod(metaToString, val)
	if err != nil {
		return nil, err
	} else if didDelegate && len(res) > 0 {
		val = &String{val: res[0].String()}
	}
	newError := &Error{val: val}
	if len(vm.callStack) > 0 && level > 0 {
		ci := vm.callStack[len(vm.callStack)-level]
		newError.addr = fmt.Sprintf(" %v:%v: ", ci.filename, ci.Line)
		newError.trace = printStackTrace(vm.callStack)
	}
	return newError, nil
}

func (err *Error) Type() string { return "error" }
func (err *Error) Val() any     { return err.val }
func (err *Error) String() string {
	msg := err.addr
	if str, isStr := err.val.(*String); isStr {
		msg += ": " + str.val
	} else {
		msg += fmt.Sprintf(" (error object is a %v value)", err.val.Type())
	}
	if err.trace != "" {
		msg += "\n" + err.trace
	}
	return msg
}
func (err *Error) Error() string { return err.String() }
func (err *Error) Meta() *Table  { return nil }

func (n *Nil) Type() string   { return "nil" }
func (n *Nil) Val() any       { return nil }
func (n *Nil) String() string { return "nil" }
func (n *Nil) Meta() *Table   { return nil }

func (b *Boolean) Type() string   { return "boolean" }
func (b *Boolean) Val() any       { return bool(b.val) }
func (b *Boolean) String() string { return fmt.Sprintf("%v", b.val) }
func (b *Boolean) Not() *Boolean  { return &Boolean{val: !b.val} }
func (b *Boolean) Meta() *Table   { return nil }

func (i *Integer) Type() string   { return "number" }
func (i *Integer) Val() any       { return int64(i.val) }
func (i *Integer) String() string { return fmt.Sprintf("%v", i.val) }
func (i *Integer) Meta() *Table   { return nil }

func (f *Float) Type() string   { return "number" }
func (f *Float) Val() any       { return float64(f.val) }
func (f *Float) String() string { return fmt.Sprintf("%v", f.val) }
func (f *Float) Meta() *Table   { return nil }

func (c *Closure) Type() string   { return "function" }
func (c *Closure) Val() any       { return c.val }
func (c *Closure) String() string { return "function" }
func (c *Closure) Meta() *Table   { return nil }
func (c *Closure) Call(vm *VM, nargs int64) ([]Value, error) {
	values, _, err := vm.eval(c.val, c.upvalues)
	return values, err
}

func (f *ExternFunc) Type() string   { return "function" }
func (f *ExternFunc) Val() any       { return f.val }
func (f *ExternFunc) String() string { return "function" }
func (f *ExternFunc) Meta() *Table   { return nil }
func (f *ExternFunc) Call(vm *VM, nargs int64) ([]Value, error) {
	args := []Value{}
	if nargs >= 0 {
		ensureSize(&vm.Stack, int(vm.framePointer+nargs))
	} else {
		nargs = vm.top - vm.framePointer
	}
	for _, val := range vm.Stack[vm.framePointer : vm.framePointer+nargs] {
		if val != nil {
			args = append(args, val)
		} else {
			args = append(args, &Nil{})
		}
	}
	return f.val(vm, args)
}
