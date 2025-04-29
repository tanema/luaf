package luaf

import (
	"fmt"
)

func createUtf8Lib() *Table {
	return &Table{
		hashtable: map[any]any{
			"char":        Fn("utf8.char", stdUtf8Char),
			"charpattern": charPattern,
			"codepoint":   Fn("utf8.codepoint", stdUtf8Codepoint),
			"len":         Fn("utf8.len", stdUtf8Len),
			"codes":       Fn("utf8.codes", stdUtf8Codes),
		},
	}
}

func stdUtf8Char(_ *VM, args []any) ([]any, error) {
	points := []byte{}
	for i, point := range args {
		if !isNumber(point) {
			return nil, argumentErr(i+1, "utf8.char", fmt.Errorf("number expected, got %v", TypeName(point)))
		}
		points = append(points, byte(toInt(point)))
	}
	return []any{string(points)}, nil
}

func stdUtf8Codepoint(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "utf8.codepoint", "string", "~number", "~number"); err != nil {
		return nil, err
	}
	str := []byte(args[0].(string))
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
		return []any{}, nil
	}
	if j >= len(str) {
		j = len(str)
	}
	out := []any{}
	for _, b := range str[i:j] {
		out = append(out, int64(b))
	}
	return out, nil
}

func stdUtf8Len(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "string.len", "string", "~number", "~number"); err != nil {
		return nil, err
	}
	str := args[0].(string)
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
		return []any{""}, nil
	}
	if j < 0 || j > strLen {
		j = strLen
	}
	return []any{int64(len(str[i:j]))}, nil
}

func stdCodesNext(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "utf8.codes", "string", "~value"); err != nil {
		return nil, err
	}
	str := args[0].(string)
	index := int64(0)
	if len(args) > 1 {
		if args[1] != nil {
			index = toInt(args[1])
		}
	}
	if index >= int64(len(str)) {
		return []any{nil}, nil
	}
	return []any{index + 1, int64(str[index])}, nil
}

func stdUtf8Codes(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "utf8.codes", "string"); err != nil {
		return nil, err
	}
	return []any{Fn("utf8.codes.next", stdCodesNext), args[0], nil}, nil
}
