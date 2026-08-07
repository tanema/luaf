package runtime

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/tanema/luaf/internal/parse"
)

type (
	// GoFunc is a go func usable by the vm.
	GoFunc struct {
		val  func(*VM, []any) ([]any, error)
		name string
	}
	// Closure is a lua function encapsulated in the vm.
	Closure struct {
		val      *parse.FnProto
		upvalues []*upvalueBroker
	}
)

const (
	typeNameNumber   = "number"
	typeNameBoolean  = "boolean"
	typeNameFunction = "function"
	typeNameTable    = "table"
	typeNameError    = "error"
	typeNameFile     = "file"
	typeNameThread   = "thread"
	typeNameNil      = "nil"
)

func (fn *GoFunc) String() string {
	return fmt.Sprintf("function:[%s()]", fn.name)
}

func (fn *Closure) String() string {
	if fn.val.Name != "" {
		return fmt.Sprintf("function:[%s()]", fn.val.Name)
	}
	return fmt.Sprintf("function:[%p]", fn)
}

func typeName(in any) string {
	switch in.(type) {
	case int64, float64:
		return typeNameNumber
	case bool:
		return typeNameBoolean
	case *Closure, *GoFunc:
		return typeNameFunction
	case *Table:
		return typeNameTable
	case error:
		return typeNameError
	case *File:
		return typeNameFile
	case *VM:
		return typeNameThread
	case nil:
		return typeNameNil
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
	case *VM:
		return threadMetatable
	default:
		return nil
	}
}

func toBool(in any) bool {
	switch tin := in.(type) {
	case string, int64, float64, error, *Closure, *GoFunc, *Table, *File, *VM:
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

var (
	hexNumberPattern = regexp.MustCompile(`(?i)^[+-]?0x([0-9a-f]+\.?[0-9a-f]*|\.[0-9a-f]+)(p[+-]?[0-9]+)?$`)
	decNumberPattern = regexp.MustCompile(`(?i)^[+-]?([0-9]+\.?[0-9]*|\.[0-9]+)(e[+-]?[0-9]+)?$`)
)

func toNumber(in any, base int) any {
	switch tin := in.(type) {
	case int64, float64:
		return in
	case string:
		str := strings.TrimSpace(tin)
		if hexNumberPattern.MatchString(str) {
			if !strings.ContainsAny(str, ".pP") {
				ival, err := strconv.ParseInt(str, 0, 64)
				if err != nil {
					return nil
				}
				return ival
			}
			if !strings.ContainsAny(str, "pP") {
				str += "p0" // Go requires an exponent on hex floats, Lua does not
			}
			fval, err := strconv.ParseFloat(str, 64)
			if err != nil {
				return nil
			}
			return fval
		} else if decNumberPattern.MatchString(str) {
			if strings.ContainsAny(str, ".eE") {
				fval, err := strconv.ParseFloat(str, 64)
				if err != nil {
					return nil
				}
				return fval
			}
			ival, err := strconv.ParseInt(str, 10, 64)
			if err != nil {
				return nil
			}
			return ival
		}

		ival, err := strconv.ParseInt(str, base, 64)
		if err != nil {
			return nil
		}
		return ival
	default:
		return nil
	}
}

// ToString will format a vm value to a printable string.
func ToString(val any) string {
	switch tin := val.(type) {
	case nil:
		return typeName(tin)
	case string:
		return tin
	case *Table:
		return fmt.Sprintf("table: %p", tin.val)
	case error:
		return tin.Error()
	case bool:
		return strconv.FormatBool(tin)
	case int64:
		return strconv.FormatInt(tin, 10)
	case float64:
		return fmt.Sprintf("%v", tin)
	case fmt.Stringer:
		return tin.String()
	default:
		return fmt.Sprintf("Unknown value type: %v", val)
	}
}

func findMetavalue(op parse.MetaMethod, val any) any {
	if val == nil {
		return nil
	}
	if mt := getMetatable(val); mt != nil && mt.hashtable[string(op)] != nil {
		return mt.hashtable[string(op)]
	}
	return nil
}

// Fn creates a value that is usable by the vm from a function. This enables exposing
// a go functionn to the VM.
func Fn(name string, fn func(*VM, []any) ([]any, error)) *GoFunc {
	return &GoFunc{
		name: name,
		val:  fn,
	}
}

func arith(vm *VM, op parse.MetaMethod, lval, rval any) (any, error) {
	if op == parse.MetaUNM {
		if liva, lisInt := lval.(int64); lisInt {
			return intArith(op, liva, 0), nil
		} else if isNumber(lval) {
			return floatArith(op, toFloat(lval), 0), nil
		}
	} else if op == parse.MetaBNot {
		if isNumber(lval) {
			ival, ok := toIntExact(lval)
			if !ok {
				return nil, errors.New("number has no integer representation")
			}
			return intArith(op, ival, 0), nil
		}
	} else if isNumber(lval) && isNumber(rval) {
		switch op {
		case parse.MetaBAnd, parse.MetaBOr, parse.MetaBXOr, parse.MetaShl, parse.MetaShr, parse.MetaSar:
			liva, lok := toIntExact(lval)
			riva, rok := toIntExact(rval)
			if !lok || !rok {
				return nil, errors.New("number has no integer representation")
			}
			return intArith(op, liva, riva), nil
		case parse.MetaDiv, parse.MetaPow:
			return floatArith(op, toFloat(lval), toFloat(rval)), nil
		case parse.MetaIDiv, parse.MetaMod:
			liva, lisInt := lval.(int64)
			riva, risInt := rval.(int64)
			if lisInt && risInt {
				if riva == 0 {
					if op == parse.MetaIDiv {
						return nil, errors.New("attempt to divide by zero")
					}
					return nil, errors.New("attempt to perform 'n%0'")
				}
				return intArith(op, liva, riva), nil
			}
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
		bad := lval
		if op != parse.MetaUNM && op != parse.MetaBNot && isNumber(lval) {
			bad = rval
		}
		switch op {
		case parse.MetaBAnd, parse.MetaBOr, parse.MetaBXOr, parse.MetaShl, parse.MetaShr, parse.MetaSar, parse.MetaBNot:
			return nil, fmt.Errorf("attempt to perform bitwise operation on a %v value", nameOfType(bad))
		default:
			return nil, fmt.Errorf("attempt to perform arithmetic on a %v value", nameOfType(bad))
		}
	} else if len(res) > 0 {
		return res[0], nil
	}
	return nil, errors.New("error object is a nil value")
}

func arithMetamethodOnly(vm *VM, op parse.MetaMethod, lval, rval any) (any, error) {
	didDelegate, res, err := vm.delegateMetamethodBinop(op, lval, rval)
	if err != nil {
		return nil, err
	} else if !didDelegate {
		return nil, fmt.Errorf("attempt to perform arithmetic on a %v value", nameOfType(lval))
	} else if len(res) > 0 {
		return res[0], nil
	}
	return nil, nil
}

func toIntExact(val any) (int64, bool) {
	switch tval := val.(type) {
	case int64:
		return tval, true
	case float64:
		if math.IsNaN(tval) || math.IsInf(tval, 0) || tval != math.Trunc(tval) {
			return 0, false
		}
		if tval < -9223372036854775808.0 || tval >= 9223372036854775808.0 {
			return 0, false
		}
		return int64(tval), true
	default:
		return 0, false
	}
}

func nameOfType(val any) string {
	var mt *Table
	switch tval := val.(type) {
	case *Table:
		mt = tval.metatable
	case *File:
		mt = fileMetatable
	case *VM:
		mt = threadMetatable
	}
	if mt != nil {
		if name, ok := mt.hashtable[string(parse.MetaName)].(string); ok {
			return name
		}
	}
	return typeName(val)
}

func intArith(op parse.MetaMethod, lval, rval int64) int64 {
	switch op {
	case parse.MetaAdd:
		return lval + rval
	case parse.MetaSub:
		return lval - rval
	case parse.MetaMul:
		return lval * rval
	case parse.MetaIDiv:
		q := lval / rval
		if lval%rval != 0 && (lval < 0) != (rval < 0) {
			q--
		}
		return q
	case parse.MetaUNM:
		return -lval
	case parse.MetaMod:
		r := lval % rval
		if r != 0 && (r < 0) != (rval < 0) {
			r += rval
		}
		return r
	case parse.MetaBAnd:
		return lval & rval
	case parse.MetaBOr:
		return lval | rval
	case parse.MetaBXOr:
		return lval ^ rval
	case parse.MetaShl:
		if rval > 0 {
			return lval << rval
		}
		return lval >> int64(math.Abs(float64(rval)))
	case parse.MetaShr:
		// logical shift: fills vacant bits with zeros regardless of sign,
		// per the Lua 5.4 spec, unlike Go's native sign-extending int64 >>.
		if rval > 0 {
			return int64(uint64(lval) >> rval)
		}
		return lval << int64(math.Abs(float64(rval)))
	case parse.MetaSar:
		// arithmetic shift: sign-extends, matching Go's native int64 >>.
		if rval > 0 {
			return lval >> rval
		}
		return lval << int64(math.Abs(float64(rval)))
	case parse.MetaBNot:
		return ^lval
	default:
		panic(fmt.Sprintf("cannot perform float %v op", op))
	}
}

func floatArith(op parse.MetaMethod, lval, rval float64) float64 {
	switch op {
	case parse.MetaAdd:
		return lval + rval
	case parse.MetaSub:
		return lval - rval
	case parse.MetaMul:
		return lval * rval
	case parse.MetaDiv:
		if rval == 0 {
			return math.Inf(1)
		}
		return lval / rval
	case parse.MetaPow:
		return math.Pow(lval, rval)
	case parse.MetaIDiv:
		return math.Floor(lval / rval)
	case parse.MetaUNM:
		return -lval
	case parse.MetaMod:
		r := math.Mod(lval, rval)
		if r != 0 && (r < 0) != (rval < 0) {
			r += rval
		}
		return r
	default:
		panic(fmt.Sprintf("cannot perform float %v op", op))
	}
}

func eq(vm *VM, lVal, rVal any) (bool, error) {
	switch tlval := lVal.(type) {
	case string:
		rstr, ok := rVal.(string)
		if !ok {
			return false, nil
		}
		return tlval == rstr, nil
	case int64, float64:
		return toFloat(lVal) == toFloat(rVal), nil
	case bool:
		rbool, ok := rVal.(bool)
		if !ok {
			return false, nil
		}
		return tlval == rbool, nil
	case nil:
		return rVal == nil, nil
	case *Table:
		rtble, ok := rVal.(*Table)
		if !ok {
			return false, nil
		} else if lVal == rtble {
			return true, nil
		}
		didDelegate, res, err := vm.delegateMetamethodBinop(parse.MetaEq, lVal, rtble)
		if err != nil {
			return false, err
		} else if didDelegate && len(res) > 0 {
			return toBool(res[0]), nil
		}
		return false, nil
	case *Closure:
		rcls, ok := rVal.(*Closure)
		if !ok {
			return false, nil
		}
		return tlval.val == rcls.val, nil
	case *GoFunc:
		rfn, ok := rVal.(*GoFunc)
		if !ok {
			return false, nil
		}
		return tlval == rfn, nil
	default:
		return false, nil
	}
}

func compareVal(vm *VM, op parse.MetaMethod, lVal, rVal any) (int, error) {
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
		return 0, compareErr(lVal, rVal)
	} else if len(res) > 0 && toBool(res[0]) {
		return -1, nil
	}
	return 1, nil
}

func compareErr(lVal, rVal any) error {
	nameA, nameB := nameOfType(lVal), nameOfType(rVal)
	if nameA == nameB {
		return fmt.Errorf("attempt to compare two %v values", nameA)
	}
	return fmt.Errorf("attempt to compare %v with %v", nameA, nameB)
}
