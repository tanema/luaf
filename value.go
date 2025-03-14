package luaf

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

type (
	typeName string
	Value    interface {
		fmt.Stringer
		Type() string
		Val() any
		Meta() *Table
	}
	Nil     struct{}
	Boolean struct{ val bool }
	Integer struct{ val int64 }
	Float   struct{ val float64 }
	GoFunc  struct {
		name string
		val  func(*VM, []Value) ([]Value, error)
	}
	Closure struct {
		val      *FnProto
		upvalues []*upvalueBroker
	}
	UserError struct {
		level int
		val   Value
	}
)

const (
	typeUnknown typeName = "unknown" // used for type hinting
	typeString  typeName = "string"
	typeTable   typeName = "table"
	typeFunc    typeName = "function"
	typeNumber  typeName = "number"
	typeBool    typeName = "boolean"
	typeNil     typeName = "nil"
	typeClosure typeName = "closure"
	typeError   typeName = "error"
	typeFile    typeName = "file"
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
	case *String, *Closure, *GoFunc, *Table, *Integer, *Float, error:
		return &Boolean{val: true}
	case *Boolean:
		return tin
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

func isNil(in Value) bool {
	switch in.(type) {
	case *Nil, nil:
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
	if method := findMetavalue(metaToString, val); method != nil {
		if res, err := vm.call(method, []Value{val}); err != nil {
			return nil, err
		} else if len(res) > 0 {
			return &String{val: res[0].String()}, nil
		}
	}
	return &String{val: val.String()}, nil
}

func findMetavalue(op metaMethod, val Value) Value {
	if val != nil && val.Meta() != nil && val.Meta().hashtable[string(op)] != nil {
		return val.Meta().hashtable[string(op)]
	}
	return nil
}

func (err *UserError) Type() string { return string(typeError) }
func (err *UserError) Val() any     { return err.val }
func (err *UserError) String() string {
	if str, isStr := err.val.(*String); isStr {
		return str.val
	}
	return fmt.Sprintf(" (error object is a %v value)", err.val.Type())
}
func (err *UserError) Error() string { return err.String() }
func (err *UserError) Meta() *Table  { return nil }

func (n *Nil) Type() string   { return string(typeNil) }
func (n *Nil) Val() any       { return nil }
func (n *Nil) String() string { return "nil" }
func (n *Nil) Meta() *Table   { return nil }

func (b *Boolean) Type() string   { return string(typeBool) }
func (b *Boolean) Val() any       { return bool(b.val) }
func (b *Boolean) String() string { return fmt.Sprintf("%v", b.val) }
func (b *Boolean) Not() *Boolean  { return &Boolean{val: !b.val} }
func (b *Boolean) Meta() *Table   { return nil }

func (i *Integer) Type() string   { return string(typeNumber) }
func (i *Integer) Val() any       { return int64(i.val) }
func (i *Integer) String() string { return fmt.Sprintf("%v", i.val) }
func (i *Integer) Meta() *Table   { return nil }

func (f *Float) Type() string   { return string(typeNumber) }
func (f *Float) Val() any       { return float64(f.val) }
func (f *Float) String() string { return fmt.Sprintf("%v", f.val) }
func (f *Float) Meta() *Table   { return nil }

func (c *Closure) Type() string { return string(typeFunc) }
func (c *Closure) Val() any     { return c.val }
func (c *Closure) Meta() *Table { return nil }
func (c *Closure) String() string {
	if c.val.Name != "" {
		return fmt.Sprintf("function[%s()]", c.val.Name)
	}
	// anon functions
	return fmt.Sprintf("function[%p]", c)
}

func (c *Closure) callinfo() *callInfo {
	return &callInfo{
		name:     c.val.Name,
		filename: c.val.Filename,
		LineInfo: c.val.LineInfo,
	}
}

func Fn(name string, fn func(*VM, []Value) ([]Value, error)) *GoFunc {
	return &GoFunc{
		name: name,
		val:  fn,
	}
}
func (f *GoFunc) Type() string        { return string(typeFunc) }
func (f *GoFunc) Val() any            { return f.val }
func (f *GoFunc) String() string      { return fmt.Sprintf("function[%s()]", f.name) }
func (f *GoFunc) Meta() *Table        { return nil }
func (f *GoFunc) callinfo() *callInfo { return &callInfo{name: f.name, filename: "<core>"} }

func arith(vm *VM, op metaMethod, lval, rval Value) (Value, error) {
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
	if didDelegate, res, err := vm.delegateMetamethodBinop(op, lval, rval); err != nil {
		return nil, err
	} else if !didDelegate {
		if op == metaUNM || op == metaBNot {
			return nil, fmt.Errorf("cannot %v %v", op, lval.Type())
		} else {
			return nil, fmt.Errorf("cannot %v %v and %v", op, lval.Type(), rval.Type())
		}
	} else if len(res) > 0 {
		return res[0], nil
	}
	return nil, fmt.Errorf("error object is a nil value")
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

func eq(vm *VM, lVal, rVal Value) (bool, error) {
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
		didDelegate, res, err := vm.delegateMetamethodBinop(metaEq, lVal, rVal)
		if err != nil {
			return false, err
		} else if didDelegate && len(res) > 0 {
			return toBool(res[0]).val, nil
		}
		return false, nil
	case *Closure:
		return lVal.Val() == rVal.Val(), nil
	case *GoFunc:
		return lVal == rVal, nil
	default:
		return false, nil
	}
}

func compareVal(vm *VM, op metaMethod, lVal, rVal Value) (int, error) {
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
	} else if didDelegate, res, err := vm.delegateMetamethodBinop(op, lVal, rVal); err != nil {
		return 0, err
	} else if !didDelegate {
		return 0, fmt.Errorf("cannot %v %v and %v", op, lVal.Type(), rVal.Type())
	} else if len(res) > 0 {
		if toBool(res[0]).val {
			return -1, nil
		}
		return 1, nil
	}
	return 0, fmt.Errorf("attempted to compare two %v and %v values", lVal.Type(), rVal.Type())
}
