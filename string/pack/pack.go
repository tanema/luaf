package pack

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

type operation struct {
	opt   byte
	opt2  byte
	param int
}

var typeDesc = map[byte]string{
	'i': "number",
	'I': "number",
	's': "string",
	'c': "string",
	'b': "number",
	'B': "number",
	'h': "number",
	'H': "number",
	'l': "number",
	'L': "number",
	'j': "number",
	'J': "number",
	'T': "value",
	'f': "number",
	'd': "number",
	'n': "number",
	'z': "string",
}

func isPowOf2(x int) bool {
	return (x != 0) && ((x & (x - 1)) == 0)
}

func consumeOperation(format string, i int) (int, *operation, error) {
	switch format[i] {
	case '!', 'i', 'I', 's', 'c': //![n]: sets maximum alignment to n (default is native alignment)
		j := i + 1
		numBuff := []byte{}
		for ; j < len(format); j++ {
			if unicode.IsDigit(rune(format[j])) {
				numBuff = append(numBuff, format[j])
			} else {
				break
			}
		}
		var param int
		var err error
		if len(numBuff) > 0 {
			param, err = strconv.Atoi(string(numBuff))
			if err != nil {
				return 0, nil, fmt.Errorf("invalid number for operation %v", format[i])
			} else if !isPowOf2(param) {
				return 0, nil, fmt.Errorf("invalid number for operation %v", format[i])
			}
		} else if format[i] == 'c' {
			return 0, nil, fmt.Errorf("string size required for c operation")
		}
		return j, &operation{opt: format[i], param: param}, nil
	case 'b':
		return i + 1, &operation{opt: 'i', param: 1}, nil
	case 'B':
		return i + 1, &operation{opt: 'I', param: 1}, nil
	case 'h':
		return i + 1, &operation{opt: 'i', param: 2}, nil
	case 'H':
		return i + 1, &operation{opt: 'I', param: 2}, nil
	case 'l':
		return i + 1, &operation{opt: 'i', param: 4}, nil
	case 'L':
		return i + 1, &operation{opt: 'I', param: 4}, nil
	case 'j':
		return i + 1, &operation{opt: 'i'}, nil
	case 'J':
		return i + 1, &operation{opt: 'I'}, nil
	case 'f', 'd', 'n', 'z', 'T', 'x':
		return i + 1, &operation{opt: format[i]}, nil
	case 'X': //Xop: an empty item that aligns according to option op (which is otherwise ignored)
		j, nextOp, err := consumeOperation(format, i+1)
		if err != nil {
			return 0, nil, err
		}
		return j + 1, &operation{opt: format[i], opt2: nextOp.opt, param: nextOp.param}, nil
	case ' ': // (empty space) ignored
		return i + 1, nil, nil
	default:
		return 0, nil, fmt.Errorf("unknown pack operation %s", string(format[i]))
	}
}

func parseFmt(format string) (binary.ByteOrder, []operation, error) {
	var end binary.ByteOrder = binary.NativeEndian
	if strings.HasPrefix(format, "<") {
		format = strings.TrimPrefix(format, "<")
		end = binary.LittleEndian
	} else if strings.HasPrefix(format, ">") {
		format = strings.TrimPrefix(format, ">")
		end = binary.BigEndian
	} else if strings.HasPrefix(format, "=") {
		format = strings.TrimPrefix(format, "=")
	}

	var err error
	var op *operation
	operations := []operation{}
	for i := 0; i < len(format); {
		if i, op, err = consumeOperation(format, i); err != nil {
			return nil, nil, err
		} else if op != nil {
			operations = append(operations, *op)
		}
	}
	return end, operations, nil
}

func Pack(format string, data ...any) (string, error) {
	end, ops, err := parseFmt(format)
	if err != nil {
		return "", err
	}

	var ival int64
	var fval float64
	dataIndex := 0
	buf := []byte{}
	for i, op := range ops {
		if dataIndex >= len(data) {
			return "", fmt.Errorf("bad argument #%v to 'pack', expected %v but got nil", i, typeDesc[op.opt])
		}
		switch op.opt {
		case '!': //![n]: sets maximum alignment to n (default is native alignment)
			panic("unsupported")
		case 'i': // i[n]: int with n bytes (default is 64)
			ival, err = toInt(data[dataIndex])
			if err != nil {
				return "", fmt.Errorf("bad argument #%v to 'pack', %v", i, err.Error())
			}
			switch op.param {
			case 1:
				buf, err = binary.Append(buf, end, int8(ival))
			case 2:
				buf, err = binary.Append(buf, end, int16(ival))
			case 4:
				buf, err = binary.Append(buf, end, int32(ival))
			default:
				buf, err = binary.Append(buf, end, ival)
			}
		case 'I': // I[n]: uint with n bytes (default is 64)
			ival, err = toInt(data[dataIndex])
			if err != nil {
				return "", fmt.Errorf("bad argument #%v to 'pack', %v", i, err.Error())
			}
			switch op.param {
			case 1:
				buf, err = binary.Append(buf, end, uint8(ival))
			case 2:
				buf, err = binary.Append(buf, end, uint16(ival))
			case 4:
				buf, err = binary.Append(buf, end, uint32(ival))
			default:
				buf, err = binary.Append(buf, end, uint64(ival))
			}
		case 's': // s[n]: a string preceded by its length coded as an unsigned integer with n bytes
			str := fmt.Sprint(data[dataIndex])
			strLen := len(str)
			if op.param > 0 {
				if op.param > strLen {
					str = strings.Repeat(" ", op.param-strLen) + str
				} else if op.param < strLen {
					str = str[:op.param]
				}
				strLen = op.param
			}
			buf, err = binary.Append(buf, end, strLen)
			if err != nil {
				return "", err
			}
			buf, err = binary.Append(buf, end, []byte(str))
			dataIndex++
		case 'c': // cn: fixed-sized string with n bytes
			str := fmt.Sprint(data[dataIndex])
			if op.param > len(str) {
				str = strings.Repeat(" ", op.param-len(str)) + str
			} else if op.param < len(str) {
				str = str[:op.param]
			}
			buf, err = binary.Append(buf, end, []byte(str))
		case 'z': // z: zero-terminated string
			buf, err = binary.Append(buf, end, []byte(fmt.Sprintf("%v\000", data[dataIndex])))
		case 'f': // f: float32
			fval, err = toFloat(data[dataIndex])
			if err != nil {
				return "", fmt.Errorf("bad argument #%v to 'pack', %v", i, err.Error())
			}
			buf, err = binary.Append(buf, end, float32(fval))
		case 'd', 'n': // d: float64
			fval, err = toFloat(data[dataIndex])
			if err != nil {
				return "", fmt.Errorf("bad argument #%v to 'pack', %v", i, err.Error())
			}
			buf, err = binary.Append(buf, end, float64(fval))
		case 'T': // T: a size_t (native size)
			panic("unsupported")
		case 'x': // x: one byte of padding
			if buf, err = binary.Append(buf, end, byte(' ')); err != nil {
				return "", err
			}
			continue
		case 'X': // Xop: an empty item that aligns according to option op (which is otherwise ignored)
			size, err := opSize(operation{opt: op.opt2, param: op.param})
			if err != nil {
				return "", err
			}
			if buf, err = binary.Append(buf, end, make([]byte, size)); err != nil {
				return "", err
			}
			continue
		}
		if err != nil {
			return "", err
		}
		dataIndex++
	}
	return string(buf), nil
}

func Packsize(format string) (int, error) {
	_, ops, err := parseFmt(format)
	if err != nil {
		return 0, err
	}
	byteCount := 0
	for _, op := range ops {
		size, err := opSize(op)
		if err != nil {
			return 0, err
		}
		byteCount += size
	}
	return byteCount, nil
}

func opSize(op operation) (int, error) {
	switch op.opt {
	case '!', 'T':
		panic("unsupported")
	case 'i', 'I', 'c':
		if op.param <= 0 {
			return 64, nil
		}
		return op.param, nil
	case 's':
		if op.param <= 0 {
			return 0, fmt.Errorf("cannot count variable sized format op 's'")
		}
		return op.param, nil
	case 'z':
		return 0, fmt.Errorf("cannot count variable sized format op 'z'")
	case 'x':
		return 1, nil
	case 'd', 'n':
		return 8, nil
	case 'X': // Xop: an empty item that aligns according to option op (which is otherwise ignored)
		return opSize(operation{opt: op.opt2, param: op.param})
	default:
		return 0, fmt.Errorf("unkown op")
	}
}

func Unpack(format, str string) ([]any, error) {
	end, ops, err := parseFmt(format)
	if err != nil {
		return nil, err
	}

	buf := bytes.NewBufferString(str)
	data := []any{}
	for _, op := range ops {
		if val, err := unpack(buf, end, op); err != nil {
			return nil, err
		} else {
			data = append(data, val)
		}
	}

	return data, nil
}

func unpack(buf *bytes.Buffer, end binary.ByteOrder, op operation) (any, error) {
	switch op.opt {
	case '!': //![n]: sets maximum alignment to n (default is native alignment)
		panic("unsupported")
	case 'i': // i[n]: int with n bytes (default is 64)
		switch op.param {
		case 1:
			var val int8
			err := binary.Read(buf, end, &val)
			return int64(val), err
		case 2:
			var val int16
			err := binary.Read(buf, end, &val)
			return int64(val), err
		case 4:
			var val int32
			err := binary.Read(buf, end, &val)
			return int64(val), err
		default:
			var val int64
			err := binary.Read(buf, end, &val)
			return int64(val), err
		}
	case 'I': // I[n]: uint with n bytes (default is 64)
		switch op.param {
		case 1:
			var val uint8
			err := binary.Read(buf, end, &val)
			return int64(val), err
		case 2:
			var val uint16
			err := binary.Read(buf, end, &val)
			return int64(val), err
		case 4:
			var val uint32
			err := binary.Read(buf, end, &val)
			return int64(val), err
		default:
			var val uint64
			err := binary.Read(buf, end, &val)
			return int64(val), err
		}
	case 's': // s[n]: a string preceded by its length coded as an unsigned integer with n bytes
		lenVal, err := unpack(buf, end, operation{opt: 'I', param: op.param})
		if err != nil {
			return nil, err
		}
		valBuf := make([]byte, lenVal.(int64))
		return string(valBuf), binary.Read(buf, end, &valBuf)
	case 'c': // cn: fixed-sized string with n bytes
		valBuf := make([]byte, op.param)
		return string(valBuf), binary.Read(buf, end, &valBuf)
	case 'z': // z: zero-terminated string
		strBuf := []byte{}
		for {
			var b byte
			if err := binary.Read(buf, end, &b); err != nil {
				return nil, err
			} else if b == '\000' {
				break
			}
			strBuf = append(strBuf, b)
		}
		return string(strBuf), nil
	case 'f': // f: float32
		var fval float32
		err := binary.Read(buf, end, &fval)
		return float64(fval), err
	case 'd', 'n': // d: float64
		var fval float64
		return fval, binary.Read(buf, end, &fval)
	case 'T': // T: a size_t (native size)
		panic("unsupported")
	case 'x': // x: one byte of padding
		var b byte
		return nil, binary.Read(buf, end, &b)
	case 'X': // Xop: an empty item that aligns according to option op (which is otherwise ignored)
		size, err := opSize(operation{opt: op.opt2, param: op.param})
		if err != nil {
			return "", err
		}
		b := make([]byte, size)
		return nil, binary.Read(buf, end, &b)
	}
	return nil, fmt.Errorf("unknown op") // shouldnt happen because already validated in parse
}

func toInt(data any) (int64, error) {
	ival, isInt := data.(int64)
	if !isInt {
		fval, isFloat := data.(float64)
		if !isFloat {
			return 0, fmt.Errorf("expected number but found string")
		}
		ival = int64(fval)
	}
	return ival, nil
}

func toFloat(data any) (float64, error) {
	fval, isFloat := data.(float64)
	if !isFloat {
		ival, isInt := data.(int64)
		if !isInt {
			return 0, fmt.Errorf("expected number but found string")
		}
		fval = float64(ival)
	}
	return fval, nil
}
