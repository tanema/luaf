package runtime

func createUtf8Lib() *Table {
	return &Table{
		hashtable: map[any]any{
			"char":        Fn("utf8.char", stdStringChar),
			"charpattern": charPattern,
			"codepoint":   Fn("utf8.codepoint", stdStringByte),
			"len":         Fn("utf8.len", stdStringLen),
			"codes":       Fn("utf8.codes", stdUtf8Codes),
		},
	}
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
