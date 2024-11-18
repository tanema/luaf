package luaf

import (
	"bytes"
	"fmt"
)

type (
	callable interface {
		Call(vm *VM, nargs int64) ([]Value, error)
	}
	GoFunc func([]Value) ([]Value, error)
	Value  interface {
		fmt.Stringer
		Type() string
		Val() any
		Bool() *Boolean
		ToKey() any
	}
	Nil        struct{}
	String     struct{ val string }
	Boolean    struct{ val bool }
	Integer    struct{ val int64 }
	Float      struct{ val float64 }
	ExternFunc struct{ val GoFunc }
	Closure    struct {
		val      *FuncProto
		upvalues []*Broker
	}
	Table struct {
		val       []Value
		hashtable map[any]Value
		metatable *Table
		keyCache  []any
	}
	Error struct {
		val Value
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

func (err *Error) Type() string { return "error" }
func (err *Error) Val() any     { return err.val }
func (err *Error) String() string {
	switch msgVal := err.val.(type) {
	case *String, *Float, *Integer:
		return err.val.String()
	default:
		return fmt.Sprintf("(error object is a %v value", msgVal.Type())
	}
}
func (err *Error) Bool() *Boolean { return &Boolean{val: true} }
func (err *Error) ToKey() any     { return err }
func (err *Error) Error() string  { return err.String() }

func (n *Nil) Type() string   { return "nil" }
func (n *Nil) Val() any       { return nil }
func (n *Nil) String() string { return "nil" }
func (n *Nil) Bool() *Boolean { return &Boolean{val: false} }
func (n *Nil) ToKey() any     { panic("dont use nil as a key!") }

func (s *String) Type() string   { return "string" }
func (s *String) Val() any       { return string(s.val) }
func (s *String) String() string { return string(s.val) }
func (s *String) Bool() *Boolean { return &Boolean{val: true} }
func (s *String) ToKey() any     { return s.val }

func (b *Boolean) Type() string   { return "boolean" }
func (b *Boolean) Val() any       { return bool(b.val) }
func (b *Boolean) String() string { return fmt.Sprintf("%v", b.val) }
func (b *Boolean) Bool() *Boolean { return b }
func (b *Boolean) Not() *Boolean  { return &Boolean{val: !b.val} }
func (b *Boolean) ToKey() any     { return b.val }

func (i *Integer) Type() string   { return "number" }
func (i *Integer) Val() any       { return int64(i.val) }
func (i *Integer) String() string { return fmt.Sprintf("%v", i.val) }
func (i *Integer) Bool() *Boolean { return &Boolean{val: i.val != 0} }
func (i *Integer) ToKey() any     { return i.val }

func (f *Float) Type() string   { return "number" }
func (f *Float) Val() any       { return float64(f.val) }
func (f *Float) String() string { return fmt.Sprintf("%v", f.val) }
func (f *Float) Bool() *Boolean { return &Boolean{val: f.val != 0} }
func (f *Float) ToKey() any     { return f.val }

func (c *Closure) Type() string   { return "function" }
func (c *Closure) Val() any       { return c.val }
func (c *Closure) String() string { return "function" }
func (c *Closure) Bool() *Boolean { return &Boolean{val: true} }
func (c *Closure) ToKey() any     { return c }
func (c *Closure) Call(vm *VM, nargs int64) ([]Value, error) {
	values, _, err := vm.eval(c.val, c.upvalues)
	return values, err
}

func (f *ExternFunc) Type() string   { return "function" }
func (f *ExternFunc) Val() any       { return f.val }
func (f *ExternFunc) String() string { return "function" }
func (f *ExternFunc) Bool() *Boolean { return &Boolean{val: true} }
func (f *ExternFunc) ToKey() any     { return f }
func (f *ExternFunc) Call(vm *VM, nargs int64) ([]Value, error) {
	args := []Value{}
	ensureSize(&vm.Stack, int(vm.framePointer+nargs))
	for _, val := range vm.Stack[vm.framePointer : vm.framePointer+nargs] {
		if val != nil {
			args = append(args, val)
		} else {
			args = append(args, &Nil{})
		}
	}
	return f.val(args)
}

func NewTable(arr []Value, hash map[any]Value) *Table {
	if hash == nil {
		hash = map[any]Value{}
	}
	return &Table{
		val:       arr,
		hashtable: hash,
		keyCache:  []any{},
	}
}

func NewSizedTable(arraySize, tableSize int) *Table {
	return &Table{
		val:       make([]Value, 0, arraySize),
		hashtable: make(map[any]Value, tableSize),
	}
}
func (t *Table) Type() string   { return "table" }
func (t *Table) Val() any       { return nil }
func (t *Table) Bool() *Boolean { return &Boolean{val: true} }
func (t *Table) ToKey() any     { return t }
func (t *Table) Keys() []any    { return t.keyCache }
func (t *Table) String() string {
	var buf bytes.Buffer
	fmt.Fprint(&buf, "{")
	for _, v := range t.val {
		if v != nil {
			fmt.Fprintf(&buf, " %s", v)
		}
	}
	for k, v := range t.hashtable {
		fmt.Fprintf(&buf, " %s = %s", k, v)
	}
	fmt.Fprint(&buf, " }")
	return buf.String()
}

func (t *Table) Index(key Value) (Value, error) {
	switch keyval := key.(type) {
	case *Integer:
		if i := keyval.val; i > 0 && int(i) <= len(t.val) {
			return t.val[i-1], nil
		} else if int(i) > len(t.val) {
			return &Nil{}, nil
		}
	case *Nil:
		return nil, fmt.Errorf("table index is nil")
	}
	val, ok := t.hashtable[key.ToKey()]
	if !ok {
		return &Nil{}, nil
	}
	return val, nil
}

func (t *Table) SetIndex(key, val Value) error {
	switch keyval := key.(type) {
	case *Integer:
		if i := keyval.val; i >= 0 {
			if int(i) > len(t.val) {
				t.val = t.val[:cap(t.val)]
			}
			t.val[i] = val
			return nil
		}
	case *Nil:
		return fmt.Errorf("table index is nil")
	}
	fmtKey := key.ToKey()
	_, exists := t.hashtable[fmtKey]
	if !exists {
		t.keyCache = append(t.keyCache, fmtKey)
	}
	if _, isNil := val.(*Nil); isNil {
		for i, kc := range t.keyCache {
			if fmtKey == kc {
				t.keyCache = t.keyCache[:i+copy(t.keyCache[i:], t.keyCache[i+1:])]
				break
			}
		}
		delete(t.hashtable, fmtKey)
	} else {
		t.hashtable[fmtKey] = val
	}
	return nil
}
