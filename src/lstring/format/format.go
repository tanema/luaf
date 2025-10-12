// package format contains the behaviour for formatting a string based on formatting
// specifiers in a string.
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
// what is expected
package format

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

const (
	FlagLeftJust  = 0b00001
	FlagShowSign  = 0b00010
	FlagShowMinus = 0b00100
	FlagHash      = 0b01000
	FlagZero      = 0b10000
)

func String(tmplIn string, args ...any) (string, error) {
	var buf strings.Builder
	var err error
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
		var width, precision int64
		i, flags = consumeFlags(tmpl, i)
		if i, width, err = consumeNum(tmpl, i); err != nil {
			return "", err
		}

		if tmpl[i] == '.' {
			i++
			if i, precision, err = consumeNum(tmpl, i); err != nil {
				return "", err
			}
		}

		fmtSpec := tmpl[fmtSpecStart : i+1]
		fmt.Println(string(fmtSpec))
		if fmtArg, err := formatArg(arg, tmpl[i], flags, width, precision); err != nil {
			return "", err
		} else if _, err := buf.WriteString(fmtArg); err != nil {
			return "", err
		}
	}

	return buf.String(), nil
}

func padStr(str string, width int64, flags uint32) string {
	leftJust := flags&FlagLeftJust == FlagLeftJust
	showSign := flags&FlagShowSign == FlagShowSign
	showMinus := flags&FlagShowMinus == FlagShowMinus

	if flags&FlagShowSign == FlagShowSign || flags&FlagShowMinus == FlagShowMinus {
		width--
	}

	fmtTmpl := "%"
	if leftJust {
		fmtTmpl += "-"
	}
	if showSign {
		fmtTmpl += "+"
	} else if showMinus {
		fmtTmpl += " "
	}
	if leftJust {
		fmtTmpl += "-"
	}
	if flags&FlagZero == FlagZero {
		fmtTmpl += "0"
	}

	fmtTmpl += fmt.Sprint(width) + "s"
	return fmt.Sprintf(fmtTmpl, str)
}

func formatArg(arg any, fmtKind rune, flags uint32, width, precision int64) (string, error) {
	switch fmtKind {
	case 'c', 'd', 'i', 'u', 'o', 'x', 'X':
		return formatInt(arg, byte(fmtKind), flags, width)
	case 'a', 'A', 'e', 'E', 'f', 'g', 'G':
		return formatFloat(arg, byte(fmtKind), flags, width, precision)
	case 's', 'q': // string
		return formatString(arg, byte(fmtKind), flags, width)
	case 'p': // pointer address
		return fmt.Sprintf("%p", arg), nil
	default:
		return "", fmt.Errorf("invalid conversion %%%s", string(fmtKind))
	}
}

func formatString(val any, fmtKind byte, flags uint32, width int64) (string, error) {
	switch fmtKind {
	case 's':
		return fmt.Sprint(val), nil
	case 'q':
		// TODO
	}
	return "", nil
}

func formatInt(val any, fmtKind byte, flags uint32, width int64) (string, error) {
	finalval, ok := toInt(val)
	if !ok {
		return "", fmt.Errorf("'%c', number expected, got %T", fmtKind, val)
	}

	switch fmtKind {
	case 'c':
		return fmt.Sprintf("%c", finalval), nil
	case 'd', 'i':
		return fmt.Sprintf("%d", finalval), nil
	case 'u':
		return fmt.Sprintf("%d", uint64(finalval)), nil
	case 'o':
		if flags&FlagHash == FlagHash {
			return fmt.Sprintf("%O", finalval), nil
		}
		return fmt.Sprintf("%o", finalval), nil
	case 'x':
		return fmt.Sprintf("%x", finalval), nil
	case 'X':
		return fmt.Sprintf("%X", finalval), nil
	}

	return "", nil
}

func formatFloat(val any, fmtKind byte, flags uint32, width, precision int64) (string, error) {
	finalval, ok := toFloat(val)
	if !ok {
		return "", fmt.Errorf("'%c', number expected, got %T", fmtKind, val)
	}

	switch fmtKind {
	case 'a':
		fmtKind = 'x'
	case 'A':
		fmtKind = 'X'
	}

	return strconv.FormatFloat(finalval, fmtKind, int(precision), 64), nil
}

func consumeFlags(tmpl []rune, i int) (int, uint32) {
	flags := uint32(0)
	for {
		switch tmpl[i] {
		case '-':
			flags = flags | FlagLeftJust
		case '+':
			flags = flags | FlagShowSign
		case ' ':
			flags = flags | FlagShowMinus
		case '#':
			flags = flags | FlagHash
		case '0':
			flags = flags | FlagZero
		default:
			return i, flags
		}
		i++
	}
}

func consumeNum(tmpl []rune, i int) (int, int64, error) {
	var str strings.Builder
	for unicode.IsDigit(tmpl[i]) {
		if _, err := str.WriteRune(tmpl[i]); err != nil {
			return -1, -1, err
		}
		i++
	}
	if str.Len() == 0 {
		return i, 0, nil
	}
	num, err := strconv.ParseInt(str.String(), 10, 64)
	return i, num, err
}
