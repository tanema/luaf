// Package format contains the behaviour for formatting a string based on formatting
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
// what is expected.
package format

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"
)

const (
	flagLeftJust  = 0b0000001
	flagShowSign  = 0b0000010
	flagShowMinus = 0b0000100
	flagHash      = 0b0001000
	flagZero      = 0b0010000
	flagHasWidth  = 0b0100000
	flagHasPrec   = 0b1000000
)

// String will format a string with a template with formatting directives. Please
// see package description for more information on the directives.
func String(tmplIn string, args ...any) (string, error) {
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
			finalval, ok := toInt(arg)
			if !ok {
				return "", fmt.Errorf("'%c', number expected, got %T", fmtKind, arg)
			}
			switch fmtKind {
			case 'i':
				fmtKind = 'd'
			case 'u':
				fmtKind = 'd'
				if strings.Contains(string(fmtSpec), ".") { // go creates blank string if uint has prec
					fmtSpec = fmtSpec[:strings.Index(string(fmtSpec), ".")]
				}
				if _, err := buf.WriteString(fmt.Sprintf(string(fmtSpec)+string(fmtKind), uint64(finalval))); err != nil {
					return "", err
				}
				continue
			}
			if _, err := buf.WriteString(fmt.Sprintf(string(fmtSpec)+string(fmtKind), finalval)); err != nil {
				return "", err
			}
		case 'a', 'A', 'e', 'E', 'f', 'g', 'G':
			finalval, ok := toFloat(arg)
			if !ok {
				return "", fmt.Errorf("'%c', number expected, got %T", fmtKind, arg)
			}
			switch fmtKind {
			case 'a':
				fmtKind = 'x'
			case 'A':
				fmtKind = 'X'
			case 'f', 'F':
				if flags&flagHasPrec != flagHasPrec {
					if _, frac := math.Modf(finalval); frac == 0 {
						fmtSpec = append(fmtSpec, '.', '0')
					} else {
						digits := len(strconv.Itoa(int(frac)))
						fmtSpec = append(fmtSpec, '.')
						fmtSpec = append(fmtSpec, []rune(strconv.Itoa(digits))...)
					}
				}
			}
			if _, err := buf.WriteString(fmt.Sprintf(string(fmtSpec)+string(fmtKind), finalval)); err != nil {
				return "", err
			}
		case 'q': // lua safe string
			switch targ := arg.(type) {
			case string:
				if _, err := buf.WriteString(fmt.Sprintf(string(fmtSpec)+string(fmtKind), targ)); err != nil {
					return "", err
				}
			case int64, float64, bool:
				if _, err := buf.WriteString(fmt.Sprintf(string(fmtSpec)+"s", fmt.Sprint(targ))); err != nil {
					return "", err
				}
			case nil:
				if _, err := buf.WriteString(fmt.Sprintf(string(fmtSpec)+"s", "nil")); err != nil {
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
			case nil:
				finalval = "nil"
			case fmt.Stringer:
				finalval = targ.String()
			default:
				finalval = fmt.Sprint(targ)
			}
			if _, err := buf.WriteString(fmt.Sprintf(string(fmtSpec)+string(fmtKind), finalval)); err != nil {
				return "", err
			}
		case 'p': // pointer address
			if _, err := buf.WriteString(fmt.Sprintf(string(fmtSpec)+string(fmtKind), arg)); err != nil {
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
