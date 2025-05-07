package luaf

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

type (
	typeName string
	// GoFunc is a go func usable by the vm.
	GoFunc struct {
		val  func(*VM, []any) ([]any, error)
		name string
	}
	// Closure is a lua function encapsulated in the vm.
	Closure struct {
		val      *FnProto
		upvalues []*upvalueBroker
	}
	// UserError is a vm error that comes from user defined error behaviour.
	UserError struct {
		val   any
		level int
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

// TypeName will return the vm value's type name string.
func TypeName(in any) string {
	switch in.(type) {
	case int64, float64:
		return string(typeNumber)
	case bool:
		return string(typeBool)
	case *Closure, GoFunc:
		return string(typeFunc)
	case *Table:
		return string(typeTable)
	case *UserError:
		return string(typeError)
	case *File:
		return string(typeFile)
	default:
		return fmt.Sprintf("%T", in)
	}
}

func getMetatable(in any) *Table {
	switch tin := in.(type) {
	case *Table:
		return tin.metatable
	case string:
		return stringMetaTable
	case *File:
		return fileMetatable
	case *Thread:
		return threadMetatable
	default:
		return nil
	}
}

func toBool(in any) bool {
	switch tin := in.(type) {
	case string, *Closure, *GoFunc, *Table, int64, float64, error:
		return true
	case bool:
		return tin
	default:
		return false
	}
}

func toKey(in any) any {
	switch in.(type) {
	case nil:
		panic("dont use nil as a key!")
	default:
		return in
	}
}

func isNumber(in any) bool {
	switch in.(type) {
	case int64, float64:
		return true
	default:
		return false
	}
}

func isString(in any) bool {
	switch in.(type) {
	case string:
		return true
	default:
		return false
	}
}

func toInt(val any) int64 {
	switch tval := val.(type) {
	case int64:
		return tval
	case float64:
		return int64(tval)
	default:
		return int64(math.NaN())
	}
}

func toFloat(val any) float64 {
	switch tval := val.(type) {
	case int64:
		return float64(tval)
	case float64:
		return tval
	default:
		return math.NaN()
	}
}

func toNumber(in any, base int) any {
	switch tin := in.(type) {
	case int64, float64:
		return in
	case string:
		if strings.Contains(tin, ".") {
			fval, err := strconv.ParseFloat(tin, 64)
			if err != nil {
				return nil
			}
			return fval
		}
		ival, err := strconv.ParseInt(tin, base, 64)
		if err != nil {
			return nil
		}
		return ival
	default:
		return math.NaN()
	}
}

// ToString will format a vm value to a printable string.
func ToString(val any) string {
	switch tin := val.(type) {
	case nil:
		return "nil"
	case string:
		return tin
	case *Table:
		return fmt.Sprintf("table %p", tin.val)
	case *File:
		return fmt.Sprintf("file %s %p", tin.path, tin)
	case *UserError:
		if str, isStr := tin.val.(string); isStr {
			return str
		}
		return fmt.Sprintf(" (error object is a %v value)", TypeName(tin.val))
	case bool:
		return strconv.FormatBool(tin)
	case int64:
		return strconv.FormatInt(tin, 10)
	case float64:
		return fmt.Sprintf("%v", tin)
	case *Closure:
		if tin.val.Name != "" {
			return fmt.Sprintf("function[%s()]", tin.val.Name)
		}
		return fmt.Sprintf("function[%p]", tin)
	case *GoFunc:
		return fmt.Sprintf("function[%s()]", tin.name)
	case *Thread:
		return fmt.Sprintf("thread %p", tin)
	default:
		return fmt.Sprintf("?%v?", val)
	}
}

func findMetavalue(op metaMethod, val any) any {
	if val == nil {
		return nil
	}
	if mt := getMetatable(val); mt != nil && mt.hashtable[string(op)] != nil {
		return mt.hashtable[string(op)]
	}
	return nil
}

func (err *UserError) Error() string { return ToString(err) }

// Fn creates a value that is usable by the vm from a function. This enables exposing
// a go functionn to the VM.
func Fn(name string, fn func(*VM, []any) ([]any, error)) *GoFunc {
	return &GoFunc{
		name: name,
		val:  fn,
	}
}

func arith(vm *VM, op metaMethod, lval, rval any) (any, error) {
	if op == metaUNM {
		if liva, lisInt := lval.(int64); lisInt {
			return intArith(op, liva, 0), nil
		} else if isNumber(lval) {
			return floatArith(op, toFloat(lval), 0), nil
		}
	} else if op == metaBNot {
		if isNumber(lval) {
			return intArith(op, toInt(lval), 0), nil
		}
	} else if isNumber(lval) && isNumber(rval) {
		switch op {
		case metaBAnd, metaBOr, metaBXOr, metaShl, metaShr:
			return intArith(op, toInt(lval), toInt(rval)), nil
		case metaDiv, metaPow:
			return floatArith(op, toFloat(lval), toFloat(rval)), nil
		default:
			liva, lisInt := lval.(int64)
			riva, risInt := rval.(int64)
			if lisInt && risInt {
				return intArith(op, liva, riva), nil
			}
			return floatArith(op, toFloat(lval), toFloat(rval)), nil
		}
	}
	if didDelegate, res, err := vm.delegateMetamethodBinop(op, lval, rval); err != nil {
		return nil, err
	} else if !didDelegate {
		if op == metaUNM || op == metaBNot {
			return nil, fmt.Errorf("cannot %v %v", op, TypeName(lval))
		}
		return nil, fmt.Errorf("cannot %v %v and %v", op, TypeName(lval), TypeName(rval))
	} else if len(res) > 0 {
		return res[0], nil
	}
	return nil, errors.New("error object is a nil value")
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
		}
		return lval >> int64(math.Abs(float64(rval)))
	case metaShr:
		if rval > 0 {
			return lval >> rval
		}
		return lval << int64(math.Abs(float64(rval)))
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

func eq(vm *VM, lVal, rVal any) (bool, error) {
	typeA, typeB := TypeName(lVal), TypeName(rVal)
	if typeA != typeB {
		return false, nil
	}
	switch tlval := lVal.(type) {
	case string:
		return tlval == rVal.(string), nil
	case int64, float64:
		return toFloat(lVal) == toFloat(rVal), nil
	case bool:
		return tlval == rVal.(bool), nil
	case nil:
		return true, nil
	case *Table:
		if lVal == rVal {
			return true, nil
		}
		didDelegate, res, err := vm.delegateMetamethodBinop(metaEq, lVal, rVal)
		if err != nil {
			return false, err
		} else if didDelegate && len(res) > 0 {
			return toBool(res[0]), nil
		}
		return false, nil
	case *Closure:
		return tlval.val == rVal.(*Closure).val, nil
	case *GoFunc:
		return lVal == rVal, nil
	default:
		return false, nil
	}
}

func compareVal(vm *VM, op metaMethod, lVal, rVal any) (int, error) {
	if isNumber(lVal) && isNumber(rVal) {
		vA, vB := toFloat(lVal), toFloat(rVal)
		if vA < vB {
			return -1, nil
		} else if vA > vB {
			return 1, nil
		}
		return 0, nil
	} else if isString(lVal) && isString(rVal) {
		strA, strB := lVal.(string), rVal.(string)
		return strings.Compare(strA, strB), nil
	} else if didDelegate, res, err := vm.delegateMetamethodBinop(op, lVal, rVal); err != nil {
		return 0, err
	} else if !didDelegate {
		return 0, fmt.Errorf("cannot %v %v and %v", op, TypeName(lVal), TypeName(rVal))
	} else if len(res) > 0 {
		if toBool(res[0]) {
			return -1, nil
		}
		return 1, nil
	}
	return 0, fmt.Errorf("attempted to compare two %v and %v values", TypeName(lVal), TypeName(rVal))
}
