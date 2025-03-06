package luaf

import (
	"fmt"
)

var libUtf8 = &Table{
	hashtable: map[any]Value{
		"char":        Fn("utf8.char", stdUtf8Char),
		"charpattern": &String{val: charPattern},
		"codepoint":   Fn("utf8.codepoint", stdUtf8Codepoint),
		"len":         Fn("utf8.len", stdUtf8Len),
		"codes":       Fn("utf8.codes", stdUtf8Codes),
	},
}

func stdUtf8Char(vm *VM, args []Value) ([]Value, error) {
	points := []byte{}
	for i, point := range args {
		if !isNumber(point) {
			return nil, argumentErr(vm, i+1, "utf8.char", fmt.Errorf("number expected, got %v", point.Type()))
		}
		points = append(points, byte(toInt(point)))
	}
	return []Value{&String{val: string(points)}}, nil
}

func stdUtf8Codepoint(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(vm, args, "utf8.codepoint", "string", "~number", "~number"); err != nil {
		return nil, err
	}
	str := []byte(args[0].(*String).val)
	i, j := 0, 1
	if len(args) > 1 {
		i = int(toInt(args[1])) - 1
		j = i + 1
	}
	if len(args) > 2 {
		j = int(toInt(args[2]))
	}
	if i < 0 {
		i = len(str) + i
	}
	if j < 0 {
		j = len(str) + j
	}
	if j < i || i >= len(str) {
		return []Value{}, nil
	}
	if j >= len(str) {
		j = len(str)
	}
	out := []Value{}
	for _, b := range str[i:j] {
		out = append(out, &Integer{val: int64(b)})
	}
	return out, nil
}

func stdUtf8Len(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(vm, args, "string.len", "string", "~number", "~number"); err != nil {
		return nil, err
	}
	str := args[0].(*String).val
	strLen := int64(len(str))
	i := int64(0)
	j := strLen
	if len(args) > 1 {
		i = toInt(args[1])
	}
	if len(args) > 2 {
		j = toInt(args[2])
	}
	if i < 0 {
		i = strLen + i
	}
	if j < 0 {
		j = strLen + j
	}
	if i < 0 || i > strLen {
		return []Value{&String{}}, nil
	}
	if j < 0 || j > strLen {
		j = strLen
	}
	return []Value{&Integer{val: int64(len(str[i:j]))}}, nil
}

func stdCodesNext(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(vm, args, "utf8.codes", "string", "~value"); err != nil {
		return nil, err
	}
	str := args[0].(*String).val
	index := int64(0)
	if len(args) > 1 {
		if _, isNil := args[1].(*Nil); !isNil {
			index = toInt(args[1])
		}
	}
	if index >= int64(len(str)) {
		return []Value{&Nil{}}, nil
	}
	return []Value{&Integer{val: index + 1}, &Integer{val: int64(str[index])}}, nil
}

func stdUtf8Codes(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(vm, args, "utf8.codes", "string"); err != nil {
		return nil, err
	}
	return []Value{Fn("utf8.codes.next", stdCodesNext), args[0].(*String), &Nil{}}, nil
}
