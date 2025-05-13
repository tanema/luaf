package luaf

import (
	"fmt"

	"github.com/tanema/luaf/lstring"
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
			return nil, argumentErr(i+1, "utf8.char", fmt.Errorf("number expected, got %v", typeName(point)))
		}
		points = append(points, byte(toInt(point)))
	}
	return []any{string(points)}, nil
}

func stdUtf8Codepoint(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "utf8.codepoint", "string", "~number", "~number"); err != nil {
		return nil, err
	}
	start, end := int64(0), int64(1)
	if len(args) > 1 {
		start = toInt(args[1]) - 1
		end = start + 1
	}
	if len(args) > 2 {
		end = toInt(args[2])
	}
	out := []any{}
	for _, b := range lstring.Substring(args[0].(string), start, end) {
		out = append(out, int64(b))
	}
	return out, nil
}

func stdUtf8Len(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "utf8.len", "string", "~number", "~number"); err != nil {
		return nil, err
	}
	src := args[0].(string)
	start := int64(0)
	end := int64(len(src))
	if len(args) > 1 {
		start = toInt(args[1])
	}
	if len(args) > 2 {
		end = toInt(args[2])
	}
	return []any{int64(len(lstring.Substring(src, start, end)))}, nil
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
