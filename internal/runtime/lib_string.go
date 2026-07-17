package runtime

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"
	"unsafe"

	"github.com/tanema/luaf/internal/parse"
	"github.com/tanema/luaf/internal/runtime/pack"
	"github.com/tanema/luaf/internal/runtime/pattern"
)

var stringMetaTable *Table

func createStringLib() *Table {
	strLib := &Table{
		hashtable: map[any]any{
			"byte":     Fn("string.byte", stdStringByte),
			"char":     Fn("string.char", stdStringChar),
			"dump":     Fn("string.dump", stdStringDump),
			"find":     Fn("string.find", stdStringFind),
			"match":    Fn("string.match", stdStringMatch),
			"gmatch":   Fn("string.gmatch", stdStringGMatch),
			"gsub":     Fn("string.gsub", stdStringGSub),
			"format":   Fn("string.format", stdStringFormat),
			"len":      Fn("string.len", stdStringLen),
			"lower":    Fn("string.lower", stdStringLower),
			"rep":      Fn("string.rep", stdStringRep),
			"reverse":  Fn("string.reverse", stdStringReverse),
			"upper":    Fn("string.upper", stdStringUpper),
			"sub":      Fn("string.sub", stdStringSub),
			"pack":     Fn("string.pack", stdStringPack),
			"packsize": Fn("string.packsize", stdStringPacksize),
			"unpack":   Fn("string.unpack", stdStringUnpack),
		},
	}

	// if the strings are convertable into numbers.
	stringMetaTable = &Table{
		hashtable: map[any]any{
			string(parse.MetaName):  "STRING",
			string(parse.MetaAdd):   strArith(parse.MetaAdd),
			string(parse.MetaSub):   strArith(parse.MetaSub),
			string(parse.MetaMul):   strArith(parse.MetaMul),
			string(parse.MetaMod):   strArith(parse.MetaMod),
			string(parse.MetaPow):   strArith(parse.MetaPow),
			string(parse.MetaDiv):   strArith(parse.MetaDiv),
			string(parse.MetaIDiv):  strArith(parse.MetaIDiv),
			string(parse.MetaUNM):   strArith(parse.MetaUNM),
			string(parse.MetaIndex): strLib,
		},
	}
	return strLib
}

func strArith(op parse.MetaMethod) *GoFunc {
	return &GoFunc{
		name: fmt.Sprintf("string:%s", op),
		val: func(vm *VM, args []any) ([]any, error) {
			var lval, rval any
			if len(args) < 1 {
				return nil, fmt.Errorf("bad argument #1 to 'string:%v' (value expected)", op)
			}
			lval = args[0]
			if op == parse.MetaUNM || op == parse.MetaBNot {
				rval = int64(0) // mock second value for unary
			} else if len(args) < 2 {
				return nil, fmt.Errorf("bad argument #2 to 'string:%v' (value expected)", op)
			} else {
				rval = args[1]
			}
			lnum := toNumber(lval, 10)
			rnum := toNumber(rval, 10)
			if lnum != nil && rnum != nil {
				res, err := arith(vm, op, lnum, rnum)
				if err != nil {
					return nil, err
				}
				return []any{res}, err
			}
			return nil, fmt.Errorf("cannot %v %v and %v", op, typeName(lval), typeName(rval))
		},
	}
}

func stdStringByte(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "string.byte", "string", "~number", "~number"); err != nil {
		return nil, err
	}

	start, end := int64(1), int64(1)
	if len(args) > 1 {
		start = toInt(args[1])
		end = start
	}
	if len(args) > 2 {
		end = toInt(args[2])
	}

	str := args[0].(string)
	substr := substring(str, start, end)
	if len(substr) == 0 {
		return []any{nil}, nil
	}

	out := []any{}
	for i := range len(substr) {
		out = append(out, int64(substr[i]))
	}
	return out, nil
}

func stdStringChar(_ *VM, args []any) ([]any, error) {
	var str strings.Builder
	for i, point := range args {
		if point != nil && !isNumber(point) {
			return nil, argumentErr(i+1, "string.char", fmt.Errorf("number expected, got %v", typeName(point)))
		} else if point == nil {
			continue
		}
		if err := str.WriteByte(byte(toInt(point))); err != nil {
			return nil, err
		}
	}
	return []any{str.String()}, nil
}

func stdStringDump(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "string.dump", "function", "~boolean"); err != nil {
		return nil, err
	}
	var fn *parse.FnProto
	switch cls := args[0].(type) {
	case *Closure:
		fn = cls.val
	default:
		return nil, argumentErr(1, "string.dump", fmt.Errorf("unable to dump %T", args[0]))
	}
	strip := false
	if len(args) > 1 {
		strip = toBool(args[0])
	}

	data, err := fn.Dump(strip)
	if err != nil {
		return nil, fmt.Errorf("could not dump fn: %w", err)
	}
	return []any{string(data)}, nil
}

func stdStringFind(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "string.find", "string", "string", "~number", "~boolean"); err != nil {
		return nil, err
	}
	src := args[0].(string)
	init := int64(1)
	if len(args) > 2 {
		init = toInt(args[2])
		srcLen := int64(len(src))
		if init < 0 {
			init += srcLen + 1
		}
		if init < 1 {
			init = 1
		} else if init > srcLen+1 {
			return []any{nil}, nil
		}
	}

	pat := args[1].(string)
	src = substring(src, init, int64(len(src)))
	if nospecials(pat) || (len(args) > 3 && toBool(args[3])) {
		return stdStringFindPlain(src, pat, init)
	}
	return stdStringFindPattern(src, pat, init)
}

// matchValue converts a pattern capture into the Lua value it should surface as:
// position captures ("()") are numbers, everything else is the captured text.
func matchValue(m *pattern.Match) any {
	if m.IsPosition {
		return int64(m.Start + 1)
	}
	return m.Subs
}

// matchText is matchValue rendered as text, for use inside a gsub string template
// where a position capture referenced via %1 etc. must be stringified.
func matchText(m *pattern.Match) string {
	if m.IsPosition {
		return strconv.FormatInt(int64(m.Start+1), 10)
	}
	return m.Subs
}

// gsubTemplate expands a gsub replacement string template in a single pass:
// %% is a literal %, %0 is the whole match, %1-%9 are captures. Doing this with
// sequential strings.ReplaceAll calls (the previous approach) is wrong in a
// subtle way: a capture's own text can happen to contain "%N", which a later
// ReplaceAll pass would then substitute again.
func gsubTemplate(tmpl string, matches []*pattern.Match) (string, error) {
	var out strings.Builder
	for i := 0; i < len(tmpl); i++ {
		if tmpl[i] != '%' {
			out.WriteByte(tmpl[i])
			continue
		}
		i++
		if i >= len(tmpl) {
			return "", errors.New("invalid use of '%' in replacement string")
		}
		switch {
		case tmpl[i] == '%':
			out.WriteByte('%')
		case tmpl[i] >= '0' && tmpl[i] <= '9':
			idx := int(tmpl[i] - '0')
			switch {
			case idx < len(matches):
				out.WriteString(matchText(matches[idx]))
			case idx == 1 && len(matches) == 1:
				// no explicit captures: %1 behaves like %0 (the whole match)
				out.WriteString(matchText(matches[0]))
			default:
				return "", fmt.Errorf("invalid capture index %%%d in replacement string", idx)
			}
		default:
			return "", errors.New("invalid use of '%' in replacement string")
		}
	}
	return out.String(), nil
}

func nospecials(pat string) bool {
	for _, ch := range pattern.SpecialChars {
		if strings.Contains(pat, string(ch)) {
			return false
		}
	}
	return true
}

func stdStringFindPlain(src, pat string, init int64) ([]any, error) {
	if index := strings.Index(src, pat); index >= 0 {
		return []any{
			init + int64(index),
			init + int64(index+len(pat)) - 1,
		}, nil
	}
	return []any{nil}, nil
}

func stdStringFindPattern(src, pat string, init int64) ([]any, error) {
	matches, err := pattern.Find(pat, src)
	if err != nil {
		return nil, err
	} else if len(matches) == 0 {
		return []any{nil}, nil
	}

	out := []any{}
	if len(matches) > 0 {
		out = append(
			out,
			init+int64(matches[0].Start),
			init+int64(matches[0].End)-1,
		)
	}

	for i := 1; i < len(matches); i++ {
		out = append(out, matchValue(matches[i]))
	}

	return out, nil
}

func stdStringMatch(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "string.match", "string", "string", "~number"); err != nil {
		return nil, err
	}
	src := args[0].(string)
	pat := args[1].(string)
	init := 0
	if len(args) > 2 {
		init = clamp(int(toInt(args[2])), 1, len(src)) - 1
	}
	parsedPattern, err := pattern.Parse(pat)
	if err != nil {
		return nil, err
	}
	matches, err := parsedPattern.Find(src[init:], 1)
	if err != nil {
		return nil, err
	} else if len(matches) == 0 {
		return []any{nil}, nil
	}

	if len(matches) > 1 { // explicit captures: surface only those, not the whole match
		matches = matches[1:]
	}
	out := make([]any, len(matches))
	for i := range matches {
		out[i] = matchValue(matches[i])
	}
	return out, nil
}

// for k, v in string.gmatch("from=world, to=Lua", "%w+=(%w+)") do print(k, v) end.
func stdStringGMatchNext(iter pattern.Iterator) func(*VM, []any) ([]any, error) {
	return func(_ *VM, args []any) ([]any, error) {
		if err := assertArguments(args, "string.match.next"); err != nil {
			return nil, err
		}
		matches, err := iter.Next()
		if err != nil {
			return nil, err
		} else if len(matches) == 0 {
			return []any{nil}, nil
		}
		result := []any{}
		if len(matches) > 1 { // we have submatches so lets surface those
			matches = matches[1:]
		}
		for _, match := range matches {
			result = append(result, matchValue(match))
		}
		return result, nil
	}
}

func stdStringGMatch(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "string.gmatch", "string", "string", "~number"); err != nil {
		return nil, err
	}
	src := args[0].(string)
	init := int64(1)
	if len(args) > 2 {
		init = toInt(args[2])
		srcLen := int64(len(src))
		if init < 0 {
			init += srcLen + 1
		}
		if init < 1 {
			init = 1
		}
		// No upper clamp: an init past srcLen+1 must produce zero matches, not the
		// same "empty match at the end" that init == srcLen+1 legitimately finds.
	}
	parsedPattern, err := pattern.Parse(args[1].(string))
	if err != nil {
		return nil, fmt.Errorf("bad pattern: %w", err)
	}
	iter := parsedPattern.Iter(src, int(init)-1)
	return []any{Fn("string.gmatch.next", stdStringGMatchNext(iter)), args[0], nil}, nil
}

/*
Repl:
  - string: then its value is used for replacement. The character % works
    as an escape character: any sequence in repl of the form %d, with d between 1
    and 9, stands for the value of the d-th captured substring. The sequence %0
    stands for the whole match. The sequence %% stands for a single %.
  - table: then the table is queried for every match, using the first capture as
    the key.
  - function: this function is called every time a match occurs, with all captured
    substrings passed as arguments, in order.
*/
// gsubReplacementString converts a table/function replacement value to the text
// to substitute in. Lua only allows a string or number here (nil/false, meaning
// "keep the original match", are handled by the caller before this is reached).
func gsubReplacementString(vm *VM, val any) (string, error) {
	if _, ok := val.(string); !ok && !isNumber(val) {
		return "", fmt.Errorf("invalid replacement value (a %s)", typeName(val))
	}
	return vm.toString(val)
}

func stdStringGSub(vm *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "string.gsub", "string", "string", "string|table|function", "~number"); err != nil {
		return nil, err
	}
	src := args[0].(string)
	parsedPattern, err := pattern.Parse(args[1].(string))
	if err != nil {
		return nil, fmt.Errorf("bad pattern: %w", err)
	}
	maxSubs := -1 // no limit
	if len(args) > 3 {
		maxSubs = int(toInt(args[3]))
	}
	iter := parsedPattern.Iter(src, 0)

	var outputStr strings.Builder
	start := 0
	count := 0
	changed := false // true once any match's replacement actually differs from the original text
	for maxSubs < 0 || count < maxSubs {
		matches, err := iter.Next()
		if err != nil {
			return nil, err
		} else if len(matches) == 0 {
			break
		}
		count++

		// The default (used whenever a table/function replacement yields nil or
		// false) is to keep the original matched text as-is.
		toSub := matches[0].Subs
		substituted := false
		switch tval := args[2].(type) {
		case string:
			repSubs, err := gsubTemplate(tval, matches)
			if err != nil {
				return nil, err
			}
			toSub = repSubs
			substituted = true // a string template always substitutes, even a no-op "%0"
		case *Table:
			key := matchValue(matches[0])
			if len(matches) > 1 {
				key = matchValue(matches[1])
			}
			val, err := vm.index(tval, nil, key)
			if err != nil {
				return nil, err
			}
			if val != nil && val != false {
				resStr, err := gsubReplacementString(vm, val)
				if err != nil {
					return nil, err
				}
				toSub = resStr
				substituted = true
			}
		case *GoFunc, *Closure:
			capMatches := matches
			if len(capMatches) > 1 { // explicit captures: pass only those, not the whole match too
				capMatches = capMatches[1:]
			}
			params := []any{}
			for _, match := range capMatches {
				params = append(params, matchValue(match))
			}
			res, err := vm.call(tval, params)
			if err != nil {
				return nil, err
			}
			if len(res) > 0 && res[0] != nil && res[0] != false {
				resStr, err := gsubReplacementString(vm, res[0])
				if err != nil {
					return nil, err
				}
				toSub = resStr
				substituted = true
			}
		}
		if substituted {
			// Declining (nil/false from a table/function replacement) keeps the
			// original text and doesn't count; an explicit replacement does, even if
			// its text happens to equal the original - Lua's own gsub always
			// allocates a fresh string once anything is actually substituted.
			changed = true
		}
		outputStr.WriteString(src[start:matches[0].Start])
		outputStr.WriteString(toSub)
		start = matches[0].End
	}
	if !changed {
		// Nothing was actually replaced (no matches, or every match kept its
		// original text): return the same string object rather than a rebuilt
		// copy, matching Lua's own gsub identity behavior.
		return []any{src, int64(count)}, nil
	}
	outputStr.WriteString(src[start:])
	return []any{outputStr.String(), int64(count)}, nil
}

func stdStringFormat(vm *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "string.format", "string"); err != nil {
		return nil, err
	}
	fmtStr, err := formatString(vm, args[0].(string), args[1:]...)
	return []any{fmtStr}, err
}

func stdStringLen(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "string.len", "string"); err != nil {
		return nil, err
	}
	return []any{int64(len(args[0].(string)))}, nil
}

func stdStringLower(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "string.lower", "string"); err != nil {
		return nil, err
	}
	return []any{strings.ToLower(args[0].(string))}, nil
}

func stdStringUpper(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "string.upper", "string"); err != nil {
		return nil, err
	}
	return []any{strings.ToUpper(args[0].(string))}, nil
}

func stdStringRep(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "string.rep", "string", "number", "~string"); err != nil {
		return nil, err
	}
	sep := ""
	if len(args) > 2 {
		sep = args[2].(string)
	}

	str := args[0].(string)
	count := toInt(args[1])
	parts := make([]string, count)
	for i := range count {
		parts[i] = str
	}

	return []any{strings.Join(parts, sep)}, nil
}

func stdStringReverse(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "string.reverse", "string"); err != nil {
		return nil, err
	}

	str := args[0].(string)
	rstr := []rune(str)
	for i, j := 0, len(str)-1; i < j; i, j = i+1, j-1 {
		rstr[i], rstr[j] = rstr[j], rstr[i]
	}

	return []any{string(rstr)}, nil
}

func stdStringSub(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "string.sub", "string", "number", "~number"); err != nil {
		return nil, err
	}
	src := args[0].(string)
	start := toInt(args[1])
	end := int64(len(src))
	if len(args) > 2 {
		end = toInt(args[2])
	}
	return []any{substring(src, start, end)}, nil
}

func stdStringPack(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "string.pack", "string"); err != nil {
		return nil, err
	}
	str, err := pack.Pack(args[0].(string), args[1:]...)
	return []any{str}, err
}

func stdStringPacksize(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "string.packsize", "string"); err != nil {
		return nil, err
	}
	size, err := pack.Packsize(args[0].(string))
	return []any{int64(size)}, err
}

func stdStringUnpack(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "string.unpack", "string", "string"); err != nil {
		return nil, err
	}
	return pack.Unpack(args[0].(string), args[1].(string))
}

func substring(str string, start, end int64) string {
	subStr := []byte(str)
	length := int64(len(subStr))

	if start == 0 && end == 0 {
		return ""
	}

	i := substringIndex(start, length+1)
	if i > length || i < 0 {
		return ""
	}

	if end == 0 {
		return ""
	}

	j := substringIndex(end, length+1)
	if j < i {
		return ""
	}

	return string(subStr[max(i-1, 0):clamp(int(j), int(i-1), int(length))])
}

func substringIndex(i, strLen int64) int64 {
	if i < 0 {
		return strLen + i
	} else if i == 0 {
		return 1
	}
	return i
}

func clamp(f, low, high int) int {
	return min(max(f, low), high)
}

// ===== Start formatting code
// A format specifier follows the form:
//
//	%[flags][width][.precision]specifier
//
// Flags:
// - - : left justify ensuring width
// - + : always show sign +/-
// - \s : (space) show sign if only -
// - # : prefix 0x for hex variables, prefix 0 for octal, show decimals with G even if 0
// - 0 : Left pad with 0 instead of space when width is suppied
// Width: Number, * is not supported in lua
// Precision: .0 ensures minimum amounts of decimals, a simple period assumes .0
// Specifiers:
// - d: int
// - i: int
// - u: uint
// - o: unsigned octal
// - x: unsigned hex int
// - X: unsigned hex int (uppercase)
// - f: float
// - e: scientific notation (3.9265e+2)
// - E: scientific notation, uppercase (3.9265E+2)
// - g: shortest representation: %e or %f
// - G: shortest representation: %E or %F
// - a: hex float
// - A: hex float uppercase (-0XC.90FEP-2)
// - c: character/rune
// - s: string
// - p: pointer address
// - q: quoted value
// - %%: %
//
// The final result of formatting will be a string with the fully filled out
// data points or an error of what does not fit.
// It largely parses string formats and passes them along to Go to make it output
// what is expected.

const (
	flagLeftJust  = 0b0000001
	flagShowSign  = 0b0000010
	flagShowMinus = 0b0000100
	flagHash      = 0b0001000
	flagZero      = 0b0010000
	flagHasWidth  = 0b0100000
	flagHasPrec   = 0b1000000
)

// formatString will format a string with a template with formatting directives. Please
// see package description for more information on the directives.
func formatString(vm *VM, tmplIn string, args ...any) (string, error) {
	var buf strings.Builder
	argIndex := 0

	tmpl := []rune(tmplIn)
	for i := 0; i < len(tmpl); i++ {
		ch := tmpl[i]
		if ch != '%' {
			buf.WriteRune(ch)
			continue
		}

		fmtSpecStart := i
		i++

		// if we have a %% we don't need an argument so we continue
		if tmpl[i] == '%' {
			if _, err := buf.WriteRune('%'); err != nil {
				return "", err
			}
			continue
		}

		if argIndex >= len(args) {
			return "", errors.New("no value")
		}

		arg := args[argIndex]
		argIndex++

		var flags uint32
		i, flags = consumeFlags(tmpl, i)

		fmtSpec := tmpl[fmtSpecStart:i]
		fmtKind := tmpl[i]
		switch tmpl[i] {
		case 'c', 'd', 'i', 'u', 'o', 'x', 'X':
			if !isNumber(arg) {
				return "", fmt.Errorf("'%c', number expected, got %T", fmtKind, arg)
			}
			switch fmtKind {
			case 'c':
				// Go's %c verb UTF-8 encodes the value as a code point; Lua's %c writes
				// the single raw byte. Formatting it as a one-byte %s keeps width/flag
				// handling (e.g. "%-16c") without going through rune encoding.
				if _, err := fmt.Fprintf(&buf, string(fmtSpec)+"s", string([]byte{byte(toInt(arg))})); err != nil {
					return "", err
				}
				continue
			case 'i':
				fmtKind = 'd'
			case 'u':
				fmtKind = 'd'
				if strings.Contains(string(fmtSpec), ".") { // go creates blank string if uint has prec
					fmtSpec = fmtSpec[:strings.Index(string(fmtSpec), ".")]
				}
				if _, err := fmt.Fprintf(&buf, string(fmtSpec)+string(fmtKind), uint64(toInt(arg))); err != nil {
					return "", err
				}
				continue
			}
			if _, err := fmt.Fprintf(&buf, string(fmtSpec)+string(fmtKind), toInt(arg)); err != nil {
				return "", err
			}
		case 'a', 'A', 'e', 'E', 'f', 'g', 'G':
			if !isNumber(arg) {
				return "", fmt.Errorf("'%c', number expected, got %T", fmtKind, arg)
			}
			switch fmtKind {
			case 'a':
				fmtKind = 'x'
			case 'A':
				fmtKind = 'X'
			case 'f', 'F':
				if flags&flagHasPrec != flagHasPrec {
					if _, frac := math.Modf(toFloat(arg)); frac == 0 {
						fmtSpec = append(fmtSpec, '.', '0')
					} else {
						digits := len(strconv.Itoa(int(frac)))
						fmtSpec = append(fmtSpec, '.')
						fmtSpec = append(fmtSpec, []rune(strconv.Itoa(digits))...)
					}
				}
			}
			if _, err := fmt.Fprintf(&buf, string(fmtSpec)+string(fmtKind), toFloat(arg)); err != nil {
				return "", err
			}
		case 'q': // lua safe string
			switch targ := arg.(type) {
			case string:
				if _, err := fmt.Fprintf(&buf, string(fmtSpec)+string(fmtKind), targ); err != nil {
					return "", err
				}
			case int64, float64, bool:
				if _, err := fmt.Fprintf(&buf, string(fmtSpec)+"s", fmt.Sprint(targ)); err != nil {
					return "", err
				}
			case nil:
				if _, err := fmt.Fprintf(&buf, string(fmtSpec)+"s", "nil"); err != nil {
					return "", err
				}
			default:
				return "", errors.New("value has no literal form")
			}
		case 's': // string
			var finalval string
			switch targ := arg.(type) {
			case string:
				finalval = targ
			case nil, fmt.Stringer:
				var err error
				finalval, err = vm.toString(targ)
				if err != nil {
					return "", err
				}
			default:
				finalval = fmt.Sprint(targ)
			}
			if _, err := fmt.Fprintf(&buf, string(fmtSpec)+string(fmtKind), finalval); err != nil {
				return "", err
			}
		case 'p': // pointer address
			// Go's %p verb only understands pointer-like types; a Lua string here is
			// a plain Go string value, so it needs unsafe.StringData to get something
			// %p can print. Reference types (tables, closures, ...) are already Go
			// pointers and fall through unchanged.
			ptrVal := arg
			if s, ok := arg.(string); ok {
				ptrVal = unsafe.StringData(s)
			}
			if _, err := fmt.Fprintf(&buf, string(fmtSpec)+string(fmtKind), ptrVal); err != nil {
				return "", err
			}
		default:
			return "", fmt.Errorf("invalid conversion %%%s", string(fmtKind))
		}
	}

	return buf.String(), nil
}

func consumeFlags(tmpl []rune, i int) (int, uint32) {
	flags := uint32(0)
flagList:
	for {
		switch tmpl[i] {
		case '-':
			flags = flags | flagLeftJust
		case '+':
			flags = flags | flagShowSign
		case ' ':
			flags = flags | flagShowMinus
		case '#':
			flags = flags | flagHash
		case '0':
			flags = flags | flagZero
		default:
			break flagList
		}
		i++
	}

	if unicode.IsDigit(tmpl[i]) {
		for unicode.IsDigit(tmpl[i]) {
			i++
		}
		flags = flags | flagHasWidth
	}

	if tmpl[i] == '.' {
		i++
		for unicode.IsDigit(tmpl[i]) {
			i++
		}
		flags = flags | flagHasPrec
	}

	return i, flags
}
