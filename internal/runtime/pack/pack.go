// Package pack allows for serialization and deserialization of data into a string.
// All of this is supported by format strings. A format string is a sequence of
// conversion options. The conversion options are as follows:
//   - <: sets little endian
//   - >: sets big endian
//   - =: sets native endian
//   - ![n]: sets maximum alignment to n (default is native alignment)
//   - b: a signed byte (char)
//   - B: an unsigned byte (char)
//   - h: a signed short (native size)
//   - H: an unsigned short (native size)
//   - l: a signed long (native size)
//   - L: an unsigned long (native size)
//   - j: a lua_Integer
//   - J: a lua_Unsigned
//   - T: a size_t (native size)
//   - i[n]: a signed int with n bytes (default is native size)
//   - I[n]: an unsigned int with n bytes (default is native size)
//   - f: a float (native size)
//   - d: a double (native size)
//   - n: a lua_Number
//   - cn: a fixed-sized string with n bytes
//   - z: a zero-terminated string
//   - s[n]: a string preceded by its length coded as an unsigned integer with n bytes (default is a size_t)
//   - x: one byte of padding
//   - Xop: an empty item that aligns according to option op (which is otherwise ignored)
//   - ' ': (empty space) ignored
package pack

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"

	"github.com/tanema/luaf/internal/types"
)

const (
	defaultSize     = 8  // default width, in bytes, for i/I/s/j/J/T/d/n when no [n] is given
	maxIntSize      = 16 // largest width, in bytes, allowed for i[n]/I[n]/s[n]/X's target
	defaultMaxAlign = 8  // maxalign used by a bare '!' with no digits
)

// nativeLittleEndian reports this platform's native byte order without relying
// on comparing binary.NativeEndian (an interface value) against a sentinel.
var nativeLittleEndian = func() bool {
	var buf [2]byte
	binary.NativeEndian.PutUint16(buf[:], 1)
	return buf[0] == 1
}()

type operation struct {
	opt          byte // resolved base option: i, I, s, c, z, f, d, n, x, X, T
	opt2         byte // for X: the target option's resolved opt
	param        int  // size in bytes for i/I/s/c/X's target; unused otherwise
	align        int  // alignment requirement for this item (1 means "no alignment")
	littleEndian bool
}

var typeDesc = map[byte]string{
	'i': types.NameNumber,
	'I': types.NameNumber,
	's': types.NameString,
	'c': types.NameString,
	'f': types.NameNumber,
	'd': types.NameNumber,
	'n': types.NameNumber,
	'z': types.NameString,
}

func isPowOf2(x int) bool {
	return (x != 0) && ((x & (x - 1)) == 0)
}

// alignFor returns the alignment requirement of an item of the given size
// under the current maximum alignment: never more than either.
func alignFor(size, maxAlign int) int {
	if size < maxAlign {
		return size
	}
	return maxAlign
}

func scanDigits(format string, start int) (int, []byte) {
	j := start
	numBuff := []byte{}
	for ; j < len(format); j++ {
		if !unicode.IsDigit(rune(format[j])) {
			break
		}
		numBuff = append(numBuff, format[j])
	}
	return j, numBuff
}

// parseSize turns a scanned digit run into a size, without risking an overflow
// panic/huge-allocation from strconv on pathological input like a 40-digit run.
func parseSize(numBuff []byte) (int, error) {
	if len(numBuff) == 0 {
		return 0, nil
	}
	if len(numBuff) > 8 {
		return 0, errors.New("invalid format (size overflow)")
	}
	n, err := strconv.Atoi(string(numBuff))
	if err != nil {
		return 0, errors.New("invalid format (size overflow)")
	}
	return n, nil
}

func isDisallowedXTarget(ch byte) bool {
	switch ch {
	case 'c', 's', 'z', 'x', 'X', '!', '<', '>', '=', ' ':
		return true
	}
	return false
}

// consumeOperation parses a single format item starting at i, threading the
// current byte order and max alignment (mutated by <, >, =, and !, and applied
// to every item parsed afterward, per Lua's format string semantics).
func consumeOperation(format string, i int, little *bool, maxAlign *int) (int, *operation, error) {
	switch format[i] {
	case '<':
		*little = true
		return i + 1, nil, nil
	case '>':
		*little = false
		return i + 1, nil, nil
	case '=':
		*little = nativeLittleEndian
		return i + 1, nil, nil
	case ' ':
		return i + 1, nil, nil
	case '!':
		j, numBuff := scanDigits(format, i+1)
		n := defaultMaxAlign
		if len(numBuff) > 0 {
			var err error
			if n, err = parseSize(numBuff); err != nil {
				return 0, nil, err
			}
		}
		if n <= 0 || n > maxIntSize {
			return 0, nil, fmt.Errorf("integer size (%d) out of limits [1,16]", n)
		} else if !isPowOf2(n) {
			return 0, nil, errors.New("format asks for alignment not power of 2")
		}
		*maxAlign = n
		return j, nil, nil
	case 'i', 'I':
		j, numBuff := scanDigits(format, i+1)
		size, err := parseSize(numBuff)
		if err != nil {
			return 0, nil, err
		} else if len(numBuff) == 0 {
			size = defaultSize
		}
		if size <= 0 || size > maxIntSize {
			return 0, nil, fmt.Errorf("integer size (%d) out of limits [1,16]", size)
		}
		align := alignFor(size, *maxAlign)
		if !isPowOf2(align) {
			return 0, nil, errors.New("format asks for alignment not power of 2")
		}
		return j, &operation{opt: format[i], param: size, align: align, littleEndian: *little}, nil
	case 's':
		j, numBuff := scanDigits(format, i+1)
		size, err := parseSize(numBuff)
		if err != nil {
			return 0, nil, err
		} else if len(numBuff) == 0 {
			size = defaultSize
		}
		if size <= 0 || size > maxIntSize {
			return 0, nil, fmt.Errorf("integer size (%d) out of limits [1,16]", size)
		}
		return j, &operation{opt: 's', param: size, align: 1, littleEndian: *little}, nil
	case 'c':
		j, numBuff := scanDigits(format, i+1)
		if len(numBuff) == 0 {
			return 0, nil, errors.New("missing size for format option 'c'")
		}
		size, err := parseSize(numBuff)
		if err != nil {
			return 0, nil, err
		}
		return j, &operation{opt: 'c', param: size, align: 1, littleEndian: *little}, nil
	case 'b':
		return i + 1, &operation{opt: 'i', param: 1, align: alignFor(1, *maxAlign), littleEndian: *little}, nil
	case 'B':
		return i + 1, &operation{opt: 'I', param: 1, align: alignFor(1, *maxAlign), littleEndian: *little}, nil
	case 'h':
		return i + 1, &operation{opt: 'i', param: 2, align: alignFor(2, *maxAlign), littleEndian: *little}, nil
	case 'H':
		return i + 1, &operation{opt: 'I', param: 2, align: alignFor(2, *maxAlign), littleEndian: *little}, nil
	case 'l':
		return i + 1, &operation{opt: 'i', param: 4, align: alignFor(4, *maxAlign), littleEndian: *little}, nil
	case 'L':
		return i + 1, &operation{opt: 'I', param: 4, align: alignFor(4, *maxAlign), littleEndian: *little}, nil
	case 'j':
		return i + 1, &operation{opt: 'i', param: 8, align: alignFor(8, *maxAlign), littleEndian: *little}, nil
	case 'J':
		return i + 1, &operation{opt: 'I', param: 8, align: alignFor(8, *maxAlign), littleEndian: *little}, nil
	case 'T':
		return i + 1, &operation{opt: 'T', param: 8, align: alignFor(8, *maxAlign), littleEndian: *little}, nil
	case 'f':
		return i + 1, &operation{opt: 'f', param: 4, align: alignFor(4, *maxAlign), littleEndian: *little}, nil
	case 'd', 'n':
		return i + 1, &operation{opt: format[i], param: 8, align: alignFor(8, *maxAlign), littleEndian: *little}, nil
	case 'z':
		return i + 1, &operation{opt: 'z', align: 1, littleEndian: *little}, nil
	case 'x':
		return i + 1, &operation{opt: 'x', align: 1}, nil
	case 'X':
		if i+1 >= len(format) || isDisallowedXTarget(format[i+1]) {
			return 0, nil, errors.New("invalid next option for option 'X'")
		}
		j, nextOp, err := consumeOperation(format, i+1, little, maxAlign)
		if err != nil {
			return 0, nil, err
		}
		return j, &operation{opt: 'X', opt2: nextOp.opt, param: nextOp.param, align: nextOp.align}, nil
	default:
		return 0, nil, fmt.Errorf("invalid format option '%c'", format[i])
	}
}

func parseFmt(format string) ([]operation, error) {
	little := nativeLittleEndian
	maxAlign := 1
	operations := []operation{}
	for i := 0; i < len(format); {
		next, op, err := consumeOperation(format, i, &little, &maxAlign)
		if err != nil {
			return nil, err
		} else if op != nil {
			operations = append(operations, *op)
		}
		i = next
	}
	return operations, nil
}

func padding(total, align int) int {
	if align <= 1 {
		return 0
	}
	if rem := total % align; rem != 0 {
		return align - rem
	}
	return 0
}

// widen packs v (as its low 8 bytes, little-endian) into a buffer of size
// bytes, sign-extending (or zero-extending) the remaining bytes when size > 8,
// then lays it out in the requested byte order.
func widen(v uint64, size int, little bool, signExtend bool) []byte {
	buf := make([]byte, size)
	fill := byte(0)
	if signExtend && int64(v) < 0 {
		fill = 0xFF
	}
	for i := range buf {
		buf[i] = fill
	}
	raw := make([]byte, 8)
	binary.LittleEndian.PutUint64(raw, v)
	n := min(size, 8)
	copy(buf, raw[:n])
	if !little {
		reverse(buf)
	}
	return buf
}

// narrow is the inverse of widen: given the raw bytes of a packed integer (in
// its original byte order), returns the 64-bit value, erroring if a >8-byte
// integer doesn't actually fit in 64 bits (i.e. the extra bytes aren't pure
// sign/zero extension).
func narrow(raw []byte, little, signed bool) (int64, error) {
	size := len(raw)
	buf := append([]byte(nil), raw...)
	if !little {
		reverse(buf)
	}
	if size <= 8 {
		full := make([]byte, 8)
		copy(full, buf)
		if signed && size < 8 && buf[size-1]&0x80 != 0 {
			for i := size; i < 8; i++ {
				full[i] = 0xFF
			}
		}
		return int64(binary.LittleEndian.Uint64(full)), nil
	}
	fill := byte(0)
	if signed && buf[7]&0x80 != 0 {
		fill = 0xFF
	}
	for i := 8; i < size; i++ {
		if buf[i] != fill {
			return 0, fmt.Errorf("%d-byte integer does not fit into Lua Integer", size)
		}
	}
	return int64(binary.LittleEndian.Uint64(buf[:8])), nil
}

func reverse(b []byte) {
	for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
}

// Pack will pack many datatypes into a serialized string formatted using the passed
// in format.
func Pack(format string, data ...any) (string, error) {
	ops, err := parseFmt(format)
	if err != nil {
		return "", err
	}

	buf := []byte{}
	total := 0
	dataIndex := 0
	for _, op := range ops {
		if pad := padding(total, op.align); pad > 0 {
			buf = append(buf, make([]byte, pad)...)
			total += pad
		}

		switch op.opt {
		case 'x':
			buf = append(buf, 0)
			total++
			continue
		case 'X':
			continue
		}

		if dataIndex >= len(data) {
			return "", fmt.Errorf("bad argument #%v to 'pack', expected %v but got nil", dataIndex+2, typeDesc[op.opt])
		}
		val := data[dataIndex]
		dataIndex++

		switch op.opt {
		case 'i':
			ival, ierr := toInt(val)
			if ierr != nil {
				return "", fmt.Errorf("bad argument #%v to 'pack', %v", dataIndex+1, ierr.Error())
			}
			buf = append(buf, widen(uint64(ival), op.param, op.littleEndian, true)...)
			total += op.param
		case 'I':
			ival, ierr := toInt(val)
			if ierr != nil {
				return "", fmt.Errorf("bad argument #%v to 'pack', %v", dataIndex+1, ierr.Error())
			}
			buf = append(buf, widen(uint64(ival), op.param, op.littleEndian, false)...)
			total += op.param
		case 's':
			str := fmt.Sprint(val)
			if op.param < 8 && uint64(len(str)) >= uint64(1)<<(uint(op.param)*8) {
				return "", errors.New("string length does not fit in given size")
			}
			buf = append(buf, widen(uint64(len(str)), op.param, op.littleEndian, false)...)
			buf = append(buf, []byte(str)...)
			total += op.param + len(str)
		case 'c':
			str := fmt.Sprint(val)
			if len(str) > op.param {
				return "", errors.New("string longer than given size")
			}
			buf = append(buf, []byte(str)...)
			buf = append(buf, make([]byte, op.param-len(str))...)
			total += op.param
		case 'z':
			str := fmt.Sprint(val)
			if strings.IndexByte(str, 0) >= 0 {
				return "", errors.New("string contains zeros")
			}
			buf = append(buf, []byte(str)...)
			buf = append(buf, 0)
			total += len(str) + 1
		case 'f':
			fval, ferr := toFloat(val)
			if ferr != nil {
				return "", fmt.Errorf("bad argument #%v to 'pack', %v", dataIndex+1, ferr.Error())
			}
			buf = append(buf, widen(uint64(math.Float32bits(float32(fval))), op.param, op.littleEndian, false)...)
			total += op.param
		case 'd', 'n':
			fval, ferr := toFloat(val)
			if ferr != nil {
				return "", fmt.Errorf("bad argument #%v to 'pack', %v", dataIndex+1, ferr.Error())
			}
			buf = append(buf, widen(math.Float64bits(fval), op.param, op.littleEndian, false)...)
			total += op.param
		}
	}
	return string(buf), nil
}

// Packsize will return the size of string for a given format.
func Packsize(format string) (int, error) {
	ops, err := parseFmt(format)
	if err != nil {
		return 0, err
	}
	total := 0
	for _, op := range ops {
		if op.opt == 's' || op.opt == 'z' {
			return 0, errors.New("variable-length format in packsize")
		}
		total += padding(total, op.align)
		if op.opt == 'x' {
			total++
		} else if op.opt != 'X' {
			total += op.param
		}
	}
	return total, nil
}

// Unpack will deserialize a string using a format as definition of what data is
// in the string, starting at the given 0-indexed byte offset. It returns the
// unpacked values along with the 0-indexed offset just past the last byte read.
func Unpack(format, str string, pos int) ([]any, int, error) {
	ops, err := parseFmt(format)
	if err != nil {
		return nil, 0, err
	}

	src := []byte(str)
	total := pos
	data := []any{}
	for _, op := range ops {
		total += padding(total, op.align)

		switch op.opt {
		case 'x':
			if total >= len(src) {
				return nil, 0, errors.New("data string too short")
			}
			total++
			continue
		case 'X':
			continue
		}

		val, size, uerr := unpackOne(src, total, op)
		if uerr != nil {
			return nil, 0, uerr
		}
		data = append(data, val)
		total += size
	}

	return data, total, nil
}

func unpackOne(src []byte, offset int, op operation) (any, int, error) {
	switch op.opt {
	case 'i':
		if offset+op.param > len(src) {
			return nil, 0, errors.New("data string too short")
		}
		v, err := narrow(src[offset:offset+op.param], op.littleEndian, true)
		return v, op.param, err
	case 'I':
		if offset+op.param > len(src) {
			return nil, 0, errors.New("data string too short")
		}
		v, err := narrow(src[offset:offset+op.param], op.littleEndian, false)
		return v, op.param, err
	case 'f':
		if offset+op.param > len(src) {
			return nil, 0, errors.New("data string too short")
		}
		v, err := narrow(src[offset:offset+op.param], op.littleEndian, false)
		if err != nil {
			return nil, 0, err
		}
		return float64(math.Float32frombits(uint32(v))), op.param, nil
	case 'd', 'n':
		if offset+op.param > len(src) {
			return nil, 0, errors.New("data string too short")
		}
		v, err := narrow(src[offset:offset+op.param], op.littleEndian, false)
		if err != nil {
			return nil, 0, err
		}
		return math.Float64frombits(uint64(v)), op.param, nil
	case 'c':
		if offset+op.param > len(src) {
			return nil, 0, errors.New("data string too short")
		}
		return string(src[offset : offset+op.param]), op.param, nil
	case 's':
		if offset+op.param > len(src) {
			return nil, 0, errors.New("data string too short")
		}
		lenVal, err := narrow(src[offset:offset+op.param], op.littleEndian, false)
		if err != nil {
			return nil, 0, err
		}
		strStart := offset + op.param
		strEnd := strStart + int(lenVal)
		if lenVal < 0 || strEnd > len(src) {
			return nil, 0, errors.New("data string too short")
		}
		return string(src[strStart:strEnd]), op.param + int(lenVal), nil
	case 'z':
		end := offset
		for end < len(src) && src[end] != 0 {
			end++
		}
		if end >= len(src) {
			return nil, 0, errors.New("unfinished string for format 'z'")
		}
		return string(src[offset:end]), end - offset + 1, nil
	}
	return nil, 0, fmt.Errorf("unknown op %q", op.opt) // shouldnt happen because already validated in parse
}

func toInt(data any) (int64, error) {
	ival, isInt := data.(int64)
	if !isInt {
		fval, isFloat := data.(float64)
		if !isFloat {
			return 0, errors.New("expected number but found string")
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
			return 0, errors.New("expected number but found string")
		}
		fval = float64(ival)
	}
	return fval, nil
}
