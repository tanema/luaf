package runtime

import (
	"encoding"
	"errors"
	"fmt"
	"os"
	"reflect"
)

var errCyclicTable = errors.New("cannot decode a cyclic table")

// GetGlobal decodes the global name from the vm into dst with FromLuaValue.
func (vm *VM) GetGlobal(name string, dst any) error {
	val, err := vm.env.Get(name)
	if err != nil {
		return err
	}
	return FromLuaValue(val, dst)
}

// FromLuaValue decodes a lua runtime value into dst, which must be a non-nil pointer.
// A nil lua value leaves dst unchanged.
func FromLuaValue(val, dst any) error {
	out := reflect.ValueOf(dst)
	if out.Kind() != reflect.Pointer || out.IsNil() {
		return fmt.Errorf("expected a non-nil pointer but got %T", dst)
	}
	return decodeValue(val, out.Elem(), map[*Table]bool{})
}

func decodeValue(val any, out reflect.Value, visiting map[*Table]bool) error {
	if val == nil {
		return nil
	}

	if out.Kind() == reflect.Interface && out.NumMethod() == 0 {
		goVal, err := toGoValue(val, visiting)
		if err != nil {
			return err
		}
		out.Set(reflect.ValueOf(goVal))
		return nil
	}

	lval := reflect.ValueOf(val)
	if lval.Type().AssignableTo(out.Type()) {
		out.Set(lval)
		return nil
	}

	if file, ok := val.(*File); ok && out.Type() == reflect.TypeFor[*os.File]() {
		handle, ok := file.handle.(*os.File)
		if !ok {
			return fmt.Errorf("cannot decode %v into *os.File", file.Path)
		}
		out.Set(reflect.ValueOf(handle))
		return nil
	}

	if out.Kind() == reflect.Pointer {
		ptr := reflect.New(out.Type().Elem())
		if !out.IsNil() {
			ptr.Elem().Set(out.Elem())
		}
		if err := decodeValue(val, ptr.Elem(), visiting); err != nil {
			return err
		}
		out.Set(ptr)
		return nil
	}

	if unmarshaler, ok := reflect.TypeAssert[encoding.TextUnmarshaler](out.Addr()); ok {
		str, ok := val.(string)
		if !ok {
			return decodeErr(val, out)
		}
		return unmarshaler.UnmarshalText([]byte(str))
	}

	switch tval := val.(type) {
	case string:
		return decodeString(tval, out)
	case bool:
		if out.Kind() != reflect.Bool {
			return decodeErr(val, out)
		}
		out.SetBool(tval)
		return nil
	case int64:
		return decodeInt(tval, out)
	case float64:
		return decodeFloat(tval, out)
	case *Table:
		if visiting[tval] {
			return errCyclicTable
		}
		visiting[tval] = true
		defer delete(visiting, tval)
		return decodeTable(tval, out, visiting)
	default:
		return decodeErr(val, out)
	}
}

func decodeErr(val any, out reflect.Value) error {
	return fmt.Errorf("cannot decode lua %s into %v", typeName(val), out.Type())
}

func decodeString(val string, out reflect.Value) error {
	switch {
	case out.Kind() == reflect.String:
		out.SetString(val)
	case out.Kind() == reflect.Slice && out.Type().Elem().Kind() == reflect.Uint8:
		out.SetBytes([]byte(val))
	case out.Kind() == reflect.Slice && out.Type().Elem().Kind() == reflect.Int32:
		out.Set(reflect.ValueOf([]rune(val)).Convert(out.Type()))
	default:
		return decodeErr(val, out)
	}
	return nil
}

func decodeInt(val int64, out reflect.Value) error {
	switch out.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if out.OverflowInt(val) {
			return fmt.Errorf("%v overflows %v", val, out.Type())
		}
		out.SetInt(val)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		if val < 0 || out.OverflowUint(uint64(val)) {
			return fmt.Errorf("%v overflows %v", val, out.Type())
		}
		out.SetUint(uint64(val))
	case reflect.Float32, reflect.Float64:
		out.SetFloat(float64(val))
	default:
		return decodeErr(val, out)
	}
	return nil
}

func decodeFloat(val float64, out reflect.Value) error {
	switch out.Kind() {
	case reflect.Float32, reflect.Float64:
		if out.OverflowFloat(val) {
			return fmt.Errorf("%v overflows %v", val, out.Type())
		}
		out.SetFloat(val)
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		ival, ok := toIntExact(val)
		if !ok {
			return fmt.Errorf("number %v has no integer representation", val)
		}
		return decodeInt(ival, out)
	default:
		return decodeErr(val, out)
	}
}

func decodeTable(tbl *Table, out reflect.Value, visiting map[*Table]bool) error {
	switch out.Kind() {
	case reflect.Struct:
		return decodeTableIntoStruct(tbl, out, visiting)
	case reflect.Slice:
		slice := reflect.MakeSlice(out.Type(), len(tbl.val), len(tbl.val))
		if err := decodeTableIntoArray(tbl, slice, visiting); err != nil {
			return err
		}
		out.Set(slice)
		return nil
	case reflect.Array:
		if len(tbl.val) > out.Len() {
			return fmt.Errorf("table of length %v does not fit in %v", len(tbl.val), out.Type())
		}
		out.SetZero()
		return decodeTableIntoArray(tbl, out, visiting)
	case reflect.Map:
		return decodeTableIntoMap(tbl, out, visiting)
	default:
		return decodeErr(tbl, out)
	}
}

func decodeTableIntoStruct(tbl *Table, out reflect.Value, visiting map[*Table]bool) error {
	t := out.Type()
	for i := range t.NumField() {
		fieldType := t.Field(i)
		if !fieldType.IsExported() {
			continue
		}

		luaName := luaFieldName(fieldType)
		lval, err := tbl.Get(luaName)
		if err != nil {
			return err
		}

		if err := decodeValue(lval, out.Field(i), visiting); err != nil {
			return fmt.Errorf("%s: %w", luaName, err)
		}
	}
	return nil
}

func decodeTableIntoArray(tbl *Table, out reflect.Value, visiting map[*Table]bool) error {
	for i, lval := range tbl.val {
		if err := decodeValue(lval, out.Index(i), visiting); err != nil {
			return fmt.Errorf("[%v]: %w", i+1, err)
		}
	}
	return nil
}

func decodeTableIntoMap(tbl *Table, out reflect.Value, visiting map[*Table]bool) error {
	if out.IsNil() {
		out.Set(reflect.MakeMap(out.Type()))
	}

	set := func(lkey, lval any) error {
		key := reflect.New(out.Type().Key()).Elem()
		if err := decodeValue(lkey, key, visiting); err != nil {
			return err
		}
		val := reflect.New(out.Type().Elem()).Elem()
		if err := decodeValue(lval, val, visiting); err != nil {
			return fmt.Errorf("[%v]: %w", lkey, err)
		}
		out.SetMapIndex(key, val)
		return nil
	}

	for i, lval := range tbl.val {
		if lval == nil {
			continue
		}
		if err := set(int64(i+1), lval); err != nil {
			return err
		}
	}
	for lkey, lval := range tbl.hashtable {
		if err := set(lkey, lval); err != nil {
			return err
		}
	}
	return nil
}

func toGoValue(val any, visiting map[*Table]bool) (any, error) {
	tbl, ok := val.(*Table)
	if !ok {
		return val, nil
	}

	if visiting[tbl] {
		return nil, errCyclicTable
	}
	visiting[tbl] = true
	defer delete(visiting, tbl)

	if len(tbl.hashtable) == 0 {
		out := make([]any, len(tbl.val))
		for i, lval := range tbl.val {
			goVal, err := toGoValue(lval, visiting)
			if err != nil {
				return nil, fmt.Errorf("[%v]: %w", i+1, err)
			}
			out[i] = goVal
		}
		return out, nil
	}

	out := make(map[any]any, len(tbl.val)+len(tbl.hashtable))
	for i, lval := range tbl.val {
		if lval == nil {
			continue
		}
		goVal, err := toGoValue(lval, visiting)
		if err != nil {
			return nil, fmt.Errorf("[%v]: %w", i+1, err)
		}
		out[int64(i+1)] = goVal
	}
	for lkey, lval := range tbl.hashtable {
		goVal, err := toGoValue(lval, visiting)
		if err != nil {
			return nil, fmt.Errorf("[%v]: %w", lkey, err)
		}
		out[lkey] = goVal
	}
	return out, nil
}
