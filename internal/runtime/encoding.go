package runtime

import (
	"encoding"
	"errors"
	"fmt"
	"math"
	"os"
	"reflect"
)

// SetGlobal converts a go value with ToLuaValue and sets it as a global in the vm.
func (vm *VM) SetGlobal(name string, val any) error {
	lval, err := ToLuaValue(val)
	if err != nil {
		return err
	}
	return vm.env.Set(name, lval)
}

// ToLuaValue converts most go values into a value that the lua runtime will be able
// to use safely.
func ToLuaValue(s any) (any, error) {
	return encodeValue(reflect.ValueOf(s))
}

func encodeValue(v reflect.Value) (any, error) {
	if !v.IsValid() {
		return nil, nil
	}

	switch v.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice, reflect.Func:
		if v.IsNil() {
			return nil, nil
		}
	}

	switch tin := v.Interface().(type) {
	case *File, *GoFunc, *Closure, *Table:
		return tin, nil
	case *os.File:
		return ToLuaFile(tin), nil
	case []byte:
		return string(tin), nil
	case []rune:
		return string(tin), nil
	case encoding.TextMarshaler:
		text, err := tin.MarshalText()
		if err != nil {
			return nil, err
		}
		return string(text), nil
	}

	switch v.Kind() {
	case reflect.Bool:
		return v.Bool(), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int(), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		if v.Uint() > math.MaxInt64 {
			return nil, fmt.Errorf("%v overflows a lua integer", v.Uint())
		}
		return int64(v.Uint()), nil
	case reflect.Float32, reflect.Float64:
		return v.Float(), nil
	case reflect.String:
		return v.String(), nil
	case reflect.Pointer, reflect.Interface:
		return encodeValue(v.Elem())
	case reflect.Func:
		apiFuncType := reflect.TypeFor[APIFunc]()
		if !v.Type().ConvertibleTo(apiFuncType) {
			return nil, fmt.Errorf("function of type %v does not match APIFunc", v.Type())
		}
		return Fn("anon", v.Convert(apiFuncType).Interface().(APIFunc)), nil
	case reflect.Slice, reflect.Array:
		return encodeArray(v)
	case reflect.Map:
		return encodeMap(v)
	case reflect.Struct:
		return encodeStruct(v)
	case reflect.Complex64, reflect.Complex128:
		return nil, errors.New("complex numbers unsupported currently")
	default:
		return nil, fmt.Errorf("unsupported type %v", v.Type())
	}
}

func encodeArray(s reflect.Value) (*Table, error) {
	out := make([]any, s.Len())
	for i := range s.Len() {
		val, err := encodeValue(s.Index(i))
		if err != nil {
			return nil, err
		}
		out[i] = val
	}
	return NewTable(out, nil), nil
}

func encodeMap(s reflect.Value) (*Table, error) {
	out := NewTable(nil, nil)
	for k, v := range s.Seq2() {
		key, err := encodeValue(k)
		if err != nil {
			return nil, err
		}

		val, err := encodeValue(v)
		if err != nil {
			return nil, err
		}

		if err := out.Set(key, val); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func encodeStruct(s reflect.Value) (*Table, error) {
	t := s.Type()
	out := NewTable(nil, nil)
	for i := range s.NumField() {
		fieldType := t.Field(i)
		if !fieldType.IsExported() {
			continue
		}

		luaVal, err := encodeValue(s.Field(i))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", fieldType.Name, err)
		}

		if err := out.Set(luaFieldName(fieldType), luaVal); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func luaFieldName(field reflect.StructField) string {
	if name := field.Tag.Get("lua"); name != "" {
		return name
	}
	return field.Name
}
