package pattern

import (
	"errors"
	"fmt"
	"strings"
)

type (
	scanner struct{ src []byte }

	seqPattern struct {
		mustHead bool
		mustTail bool
		patterns []any
	}
	singlePattern struct{ class class }
	repeatPattern struct {
		kind  rune
		class class
	}
	capPattern    struct{ pattern any }
	numberPattern struct{ n rune }
	bracePattern  struct{ begin, end rune }

	class interface {
		fmt.Stringer
		Matches(ch rune) bool
	}
	dotClass    struct{}
	charClass   struct{ ch rune }
	singleClass struct{ class rune }
	setClass    struct {
		isNot   bool
		classes []class
	}
	rangeClass struct{ begin, end rune }
)

const EOS = -1

var (
	ErrUnexpectedEOS     = errors.New("unexpected EOS")
	ErrZeroCapture       = errors.New("invalid capture index %%0")
	ErrUnmatchedParen    = errors.New("unmatched ')'")
	ErrUnfinishedCapture = errors.New("unfinished capture")
	ErrInvalidRange      = errors.New("invalid range")
)

func (sc *scanner) Next() rune {
	if len(sc.src)-1 == 0 {
		return EOS
	}
	sc.src = sc.src[1:]
	return rune(sc.src[0])
}

func (sc *scanner) Peek() rune {
	if len(sc.src)-1 == 0 {
		return EOS
	}
	return rune(sc.src[1])
}

func parse(pattern string) (*seqPattern, error) {
	sc := &scanner{
		src: append([]byte(" "), []byte(pattern)...),
	}
	return parsePattern(sc, true)
}

func parsePattern(sc *scanner, toplevel bool) (*seqPattern, error) {
	pat := &seqPattern{}
	if toplevel {
		if sc.Peek() == '^' {
			sc.Next()
			pat.mustHead = true
		}
	}
	for {
		ch := sc.Peek()
		switch ch {
		case '%':
			sc.Next()
			switch sc.Peek() {
			case '0':
				return nil, ErrZeroCapture
			case '1', '2', '3', '4', '5', '6', '7', '8', '9':
				pat.patterns = append(pat.patterns, &numberPattern{sc.Next() - 48})
			case 'b':
				sc.Next()
				pat.patterns = append(pat.patterns, &bracePattern{sc.Next(), sc.Next()})
			default:
				pat.patterns = append(pat.patterns, &singlePattern{&singleClass{sc.Next()}})
			}
		case '.', '[', ']':
			cls, err := parseClass(sc, true)
			if err != nil {
				return nil, err
			}
			pat.patterns = append(pat.patterns, &singlePattern{cls})
		case ')':
			if toplevel {
				return nil, ErrUnmatchedParen
			}
			return pat, nil
		case '(':
			sc.Next()
			res, err := parsePattern(sc, false)
			if err != nil {
				return nil, err
			}
			ret := &capPattern{res}
			if sc.Peek() != ')' {
				return nil, ErrUnfinishedCapture
			}
			sc.Next()
			pat.patterns = append(pat.patterns, ret)
		case '*', '+', '-', '?':
			sc.Next()
			if len(pat.patterns) > 0 {
				spat, ok := pat.patterns[len(pat.patterns)-1].(*singlePattern)
				if ok {
					pat.patterns = pat.patterns[0 : len(pat.patterns)-1]
					pat.patterns = append(pat.patterns, &repeatPattern{ch, spat.class})
					continue
				}
			}
			pat.patterns = append(pat.patterns, &singlePattern{&charClass{ch}})
		case '$':
			sc.Next()
			if toplevel && sc.Peek() == EOS {
				pat.mustTail = true
			} else {
				pat.patterns = append(pat.patterns, &singlePattern{&charClass{ch}})
			}
		case EOS:
			sc.Next()
			return pat, nil
		default:
			sc.Next()
			pat.patterns = append(pat.patterns, &singlePattern{&charClass{ch}})
		}
	}
}

func parseClass(sc *scanner, allowset bool) (class, error) {
	ch := sc.Next()
	switch ch {
	case '%':
		return &singleClass{sc.Next()}, nil
	case '.':
		if allowset {
			return &dotClass{}, nil
		}
		return &charClass{ch}, nil
	case '[':
		if allowset {
			return parseClassSet(sc)
		}
		return &charClass{ch}, nil
	case EOS:
		return nil, ErrUnexpectedEOS
	default:
		return &charClass{ch}, nil
	}
}

func parseClassSet(sc *scanner) (class, error) {
	set := &setClass{false, []class{}}
	if sc.Peek() == '^' {
		set.isNot = true
		sc.Next()
	}
	isrange := false
	for {
		ch := sc.Peek()
		switch ch {
		case EOS:
			return nil, ErrUnexpectedEOS
		case ']':
			if len(set.classes) > 0 {
				sc.Next()
				if isrange {
					set.classes = append(set.classes, &charClass{'-'})
				}
				return set, nil
			}
			fallthrough
		case '-':
			if len(set.classes) > 0 {
				sc.Next()
				isrange = true
				continue
			}
			fallthrough
		default:
			cls, err := parseClass(sc, false)
			if err != nil {
				return nil, err
			}
			set.classes = append(set.classes, cls)
		}
		if isrange {
			begin := set.classes[len(set.classes)-2]
			end := set.classes[len(set.classes)-1]
			set.classes = set.classes[0 : len(set.classes)-2]
			bch, bisChar := begin.(*charClass)
			ech, eisChar := end.(*charClass)
			if !bisChar || !eisChar {
				return nil, ErrInvalidRange
			}
			set.classes = append(set.classes, &rangeClass{bch.ch, ech.ch})
			isrange = false
		}
	}
}

func (pn *dotClass) Matches(ch rune) bool { return true }
func (pn *dotClass) String() string       { return "." }

func (pn *charClass) Matches(ch rune) bool { return pn.ch == ch }
func (pn *charClass) String() string       { return string(pn.ch) }

func (pn *rangeClass) Matches(ch rune) bool { return pn.begin <= ch && ch <= pn.end }
func (pn *rangeClass) String() string       { return fmt.Sprintf("%v-%v", pn.begin, pn.end) }

func (pn *singleClass) Matches(ch rune) bool {
	ret := false
	switch pn.class {
	case 'a', 'A':
		ret = 'A' <= ch && ch <= 'Z' || 'a' <= ch && ch <= 'z'
	case 'c', 'C':
		ret = (0x00 <= ch && ch <= 0x1F) || ch == 0x7F
	case 'd', 'D':
		ret = '0' <= ch && ch <= '9'
	case 'l', 'L':
		ret = 'a' <= ch && ch <= 'z'
	case 'p', 'P':
		ret = (0x21 <= ch && ch <= 0x2f) || (0x3a <= ch && ch <= 0x40) || (0x5b <= ch && ch <= 0x60) || (0x7b <= ch && ch <= 0x7e)
	case 's', 'S':
		switch ch {
		case ' ', '\f', '\n', '\r', '\t', '\v':
			ret = true
		}
	case 'u', 'U':
		ret = 'A' <= ch && ch <= 'Z'
	case 'w', 'W':
		ret = '0' <= ch && ch <= '9' || 'A' <= ch && ch <= 'Z' || 'a' <= ch && ch <= 'z'
	case 'x', 'X':
		ret = '0' <= ch && ch <= '9' || 'a' <= ch && ch <= 'f' || 'A' <= ch && ch <= 'F'
	case 'z', 'Z':
		ret = ch == 0
	default:
		return ch == pn.class
	}
	if 'A' <= pn.class && pn.class <= 'Z' {
		return !ret
	}
	return ret
}
func (pn *singleClass) String() string { return fmt.Sprintf("%%%s", string(pn.class)) }

func (pn *setClass) Matches(ch rune) bool {
	for _, class := range pn.classes {
		if class.Matches(ch) {
			return !pn.isNot
		}
	}
	return pn.isNot
}

func (pn *setClass) String() string {
	parts := []string{}
	for _, s := range pn.classes {
		parts = append(parts, s.String())
	}
	if pn.isNot {
		return fmt.Sprintf("~[%v]", strings.Join(parts, ""))
	}
	return fmt.Sprintf("[%v]", strings.Join(parts, ""))
}
