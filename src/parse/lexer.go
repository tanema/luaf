package parse

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"math/big"
	"strconv"
	"strings"
	"unicode"

	"github.com/tanema/luaf/src/lerrors"
)

var escapeCodes = map[rune]rune{
	'a':  '\x07', // bell
	'b':  '\x08', // backspace
	'f':  '\x0C', // form feed
	'n':  '\n',   // newline
	'r':  '\r',   // carriage return
	't':  '\t',   // tab
	'v':  '\x0B', // vertical tab
	'\\': '\\',   // backslach
	'"':  '"',    // quote
	'\'': '\'',   // apostrophe
	'[':  '[',    // brackets for bracketed strings
	']':  ']',
}

type (
	lexer struct {
		filename string
		rdr      *bufio.Reader
		peeked   []*token
		LineInfo
	}
)

func newLexer(filename string, src io.Reader) *lexer {
	return &lexer{
		filename: filename,
		LineInfo: LineInfo{Line: 1},
		rdr:      bufio.NewReaderSize(src, 4096),
		peeked:   []*token{},
	}
}

func (lex *lexer) errf(msg string, data ...any) error {
	return lex.err(fmt.Errorf(msg, data...))
}

func (lex *lexer) err(err error) error {
	if errors.Is(err, io.EOF) {
		return err
	}
	return &lerrors.Error{
		Filename: lex.filename,
		Kind:     lerrors.LexerErr,
		Line:     lex.Line,
		Column:   lex.Column,
		Err:      err,
	}
}

func (lex *lexer) peek() rune {
	chs, _ := lex.rdr.Peek(1)
	if len(chs) == 0 {
		return 0
	}
	return rune(chs[0])
}

func (lex *lexer) next() (rune, error) {
	ch, _, err := lex.rdr.ReadRune()
	if err != nil {
		return ch, lex.err(err)
	}
	if ch == '\n' || ch == '\r' {
		lex.Line++
		lex.Column = 0
	}
	lex.Column++
	return ch, err
}

func (lex *lexer) mustNext(expected rune) error {
	ch, err := lex.next()
	if err != nil {
		return err
	} else if ch != expected {
		return lex.err(fmt.Errorf("expected rune %v but found %v", string(expected), string(ch)))
	}
	return nil
}

func (lex *lexer) skipWhitespace() error {
	for {
		if tk := lex.peek(); tk == ' ' || tk == '\t' || tk == '\n' || tk == '\r' {
			if _, err := lex.next(); err != nil {
				return err
			}
			continue
		}
		return nil
	}
}

func (lex *lexer) tokenVal(tk tokenType) (*token, error) {
	return &token{Kind: tk, LineInfo: LineInfo{Line: lex.Line, Column: lex.Column - int64(len(tk))}}, nil
}

func (lex *lexer) takeTokenVal(tk tokenType) (*token, error) {
	_, err := lex.next()
	return &token{Kind: tk, LineInfo: LineInfo{Line: lex.Line, Column: lex.Column - int64(len(tk))}}, err
}

// allow for FIFO stack.
func (lex *lexer) back(tk *token) {
	lex.peeked = append(lex.peeked, tk)
}

func (lex *lexer) Peek() (*token, error) {
	if len(lex.peeked) == 0 {
		tk, err := lex.Next()
		if err != nil && !errors.Is(err, io.EOF) {
			return &token{Kind: tokenEOS}, err
		} else if err != nil && errors.Is(err, io.EOF) {
			return &token{Kind: tokenEOS}, nil
		}
		lex.peeked = append(lex.peeked, tk)
	}
	return lex.peeked[len(lex.peeked)-1], nil
}

func (lex *lexer) Next() (*token, error) {
	if len(lex.peeked) != 0 {
		top := lex.peeked[len(lex.peeked)-1]
		lex.peeked = lex.peeked[:len(lex.peeked)-1]
		return top, nil
	}
	if lex.peek() == '#' && lex.Line == 1 && lex.Column == 0 {
		if err := lex.parseShebang(); err != nil {
			return nil, err
		}
	}
	if err := lex.skipWhitespace(); err != nil {
		return nil, err
	}
	ch, err := lex.next()
	if err != nil {
		return nil, err
	}
	peekCh := lex.peek()
	if ch == '-' && peekCh == '-' {
		return lex.parseComment()
	} else if ch == '-' {
		return lex.tokenVal(tokenMinus)
	} else if ch == '[' && (peekCh == '=' || peekCh == '[') {
		return lex.parseBracketedString()
	} else if ch == '[' {
		return lex.tokenVal(tokenOpenBracket)
	} else if ch == '=' && peekCh == '=' {
		return lex.takeTokenVal(tokenEq)
	} else if ch == '=' {
		return lex.tokenVal(tokenAssign)
	} else if ch == '<' && peekCh == '=' {
		return lex.takeTokenVal(tokenLe)
	} else if ch == '<' && peekCh == '<' {
		return lex.takeTokenVal(tokenShiftLeft)
	} else if ch == '<' {
		return lex.tokenVal(tokenLt)
	} else if ch == '>' && peekCh == '=' {
		return lex.takeTokenVal(tokenGe)
	} else if ch == '>' && peekCh == '>' {
		return lex.takeTokenVal(tokenShiftRight)
	} else if ch == '>' {
		return lex.tokenVal(tokenGt)
	} else if ch == '~' && peekCh == '=' {
		return lex.takeTokenVal(tokenNe)
	} else if ch == '~' {
		return lex.tokenVal(tokenBitwiseNotOrXOr)
	} else if ch == '/' && peekCh == '/' {
		return lex.takeTokenVal(tokenFloorDivide)
	} else if ch == '/' {
		return lex.tokenVal(tokenDivide)
	} else if ch == '.' {
		if unicode.IsDigit(peekCh) {
			return lex.parseNumber(ch)
		} else if peekCh == '.' {
			if _, err := lex.next(); err != nil {
				return nil, err
			}
			if lex.peek() == '.' {
				return lex.takeTokenVal(tokenDots)
			}
			return lex.tokenVal(tokenConcat)
		}
		return lex.tokenVal(tokenPeriod)
	} else if ch == '+' {
		return lex.tokenVal(tokenAdd)
	} else if ch == '*' {
		return lex.tokenVal(tokenMultiply)
	} else if ch == '%' {
		return lex.tokenVal(tokenModulo)
	} else if ch == '^' {
		return lex.tokenVal(tokenExponent)
	} else if ch == '&' {
		return lex.tokenVal(tokenBitwiseAnd)
	} else if ch == '|' {
		return lex.tokenVal(tokenBitwiseOrUnion)
	} else if ch == ':' {
		if lex.peek() == ':' {
			return lex.parseLabel()
		}
		return lex.tokenVal(tokenColon)
	} else if ch == ',' {
		return lex.tokenVal(tokenComma)
	} else if ch == ';' {
		return lex.tokenVal(tokenSemiColon)
	} else if ch == '#' {
		return lex.tokenVal(tokenLength)
	} else if ch == '(' {
		return lex.tokenVal(tokenOpenParen)
	} else if ch == ')' {
		return lex.tokenVal(tokenCloseParen)
	} else if ch == '{' {
		return lex.tokenVal(tokenOpenCurly)
	} else if ch == '}' {
		return lex.tokenVal(tokenCloseCurly)
	} else if ch == ']' {
		return lex.tokenVal(tokenCloseBracket)
	} else if ch == '"' || ch == '\'' {
		return lex.parseString(ch)
	} else if unicode.IsDigit(ch) {
		return lex.parseNumber(ch)
	} else if unicode.IsLetter(ch) || ch == '_' {
		return lex.parseIdentifier(ch)
	}
	return nil, lex.errf("unexpected character %v", string(ch))
}

func (lex *lexer) parseLabel() (*token, error) {
	linfo := lex.LineInfo
	if err := lex.mustNext(':'); err != nil {
		return nil, err
	}

	ch, err := lex.next()
	if err != nil {
		return nil, err
	}

	tk, err := lex.parseIdentifier(ch)
	if err != nil {
		return nil, err
	}

	if err := lex.mustNext(':'); err != nil {
		return nil, lex.err(errors.New("unexpected character while parsing label"))
	} else if err := lex.mustNext(':'); err != nil {
		return nil, lex.err(errors.New("unexpected character while parsing label"))
	}

	return &token{
		Kind:      tokenLabel,
		StringVal: tk.StringVal,
		LineInfo:  linfo,
	}, nil
}

func (lex *lexer) parseIdentifier(start rune) (*token, error) {
	linfo := lex.LineInfo
	var ident bytes.Buffer
	if _, err := ident.WriteRune(start); err != nil {
		return nil, err
	}

	for {
		if peekCh := lex.peek(); unicode.IsLetter(peekCh) || unicode.IsDigit(peekCh) || peekCh == '_' {
			if ch, err := lex.next(); err != nil {
				return nil, err
			} else if _, err := ident.WriteRune(ch); err != nil {
				return nil, err
			}
		} else {
			break
		}
	}

	strVal := ident.String()
	if kw, ok := keywords[strVal]; ok {
		return lex.tokenVal(kw)
	}
	return &token{
		Kind:      tokenIdentifier,
		StringVal: strVal,
		LineInfo:  linfo,
	}, nil
}

/*
A short literal string can be delimited by matching single or double quotes, and
can contain the following C-like escape sequences:

'\a'      (bell)
'\b'      (backspace)
'\f'      (form feed)
'\n'      (newline)
'\r'      (carriage return)
'\t'      (horizontal tab)
'\v'      (vertical tab)
'\\'      (backslash)
'\"'      (quotation mark [double quote])
'\”      (apostrophe [single quote])
'\z'      skips the following span of white-space characters, including line breaks
\xXX      where XX is a sequence of exactly two hexadecimal digits specifies any byte
\ddd      where ddd is a sequence of up to three decimal digits
\u{XXX}   where XXX is a sequence of one or more hexadecimal digits representing the character code point

For convenience, when the opening long bracket is immediately followed by a newline,
the newline is not included in the string.
*/
func (lex *lexer) parseString(delimiter rune) (*token, error) {
	linfo := lex.LineInfo
	var str bytes.Buffer
	for {
		if ch, err := lex.next(); err != nil {
			return nil, err
		} else if ch == '\\' {
			if ch, err := lex.next(); err != nil {
				return nil, err
			} else if esc, ok := escapeCodes[ch]; ok {
				str.WriteRune(esc)
			} else if ch == 'z' { // remove spaces
				peekCh := lex.peek()
				for peekCh == ' ' {
					if _, err := lex.next(); err != nil {
						return nil, err
					}
					peekCh = lex.peek()
				}
			} else if ch == 'u' { // utf8 unicode character
				if err := lex.mustNext('{'); err != nil {
					return nil, err
				}

				var hexNumber bytes.Buffer
				if err := lex.consumeDigits(&hexNumber, true); err != nil {
					return nil, err
				}

				ivalue, err := strconv.ParseInt(hexNumber.String(), 16, 64)
				if err != nil {
					return nil, lex.err(fmt.Errorf("parse int: %w", errors.Unwrap(err)))
				}
				str.WriteRune(rune(ivalue))

				if err := lex.mustNext('}'); err != nil {
					return nil, err
				}
			} else if ch == 'x' { // hex char code
				var hexNumber bytes.Buffer
				firstCh := lex.peek()
				if isHexDigit(firstCh) {
					if err := lex.writeNext(&hexNumber); err != nil {
						return nil, err
					}
				} else {
					return nil, lex.err(fmt.Errorf("hexadecimal digit expected near %q", `\x`))
				}

				if isHexDigit(lex.peek()) {
					if err := lex.writeNext(&hexNumber); err != nil {
						return nil, err
					}
				} else {
					return nil, lex.err(fmt.Errorf("hexadecimal digit expected near %q", `\x`+string(firstCh)))
				}

				ivalue, err := strconv.ParseInt(hexNumber.String(), 16, 64)
				if err != nil {
					return nil, lex.err(fmt.Errorf("parse int: %w", errors.Unwrap(err)))
				}

				str.WriteRune(rune(ivalue))
			} else if unicode.IsDigit(ch) {
				var number bytes.Buffer
				if _, err := number.WriteRune(ch); err != nil {
					return nil, lex.err(err)
				}

				for range 2 {
					peekCh := lex.peek()
					if !unicode.IsDigit(peekCh) {
						break
					}
					if err := lex.writeNext(&number); err != nil {
						return nil, err
					}
				}

				ivalue, err := strconv.ParseInt(number.String(), 10, 64)
				if err != nil {
					return nil, lex.err(fmt.Errorf("parse int: %w", errors.Unwrap(err)))
				}

				str.WriteRune(rune(ivalue))
			} else {
				return nil, lex.err(fmt.Errorf("unexpected escape code \\%s", string(ch)))
			}
		} else if ch == delimiter {
			return &token{
				Kind:      tokenString,
				StringVal: str.String(),
				LineInfo:  linfo,
			}, nil
		} else {
			str.WriteRune(ch)
		}
	}
}

func (lex *lexer) parseNumber(start rune) (*token, error) {
	linfo := lex.LineInfo
	var number bytes.Buffer
	isHex, isFloat := false, false

	if start != '.' {
		if _, err := number.WriteRune(start); err != nil {
			return nil, lex.err(err)
		}

		if err := lex.consumeDigits(&number, isHex); err != nil {
			return nil, err
		}

		if peekCh := lex.peek(); peekCh == 'x' || peekCh == 'X' {
			isHex = true
			if err := lex.writeNext(&number); err != nil {
				return nil, err
			} else if err := lex.consumeDigits(&number, isHex); err != nil {
				return nil, err
			}
		}
		if peekCh := lex.peek(); peekCh == '.' {
			isFloat = true
			if err := lex.writeNext(&number); err != nil {
				return nil, err
			} else if err := lex.consumeDigits(&number, isHex); err != nil {
				return nil, err
			}
		}
	} else {
		number.WriteRune('0')
		number.WriteRune('.')
		isFloat = true
		if err := lex.writeNext(&number); err != nil {
			return nil, err
		} else if err := lex.consumeDigits(&number, isHex); err != nil {
			return nil, err
		}
	}

	if peekCh := lex.peek(); peekCh == 'e' || peekCh == 'E' {
		isFloat = true
		if err := lex.parseExponent(&number, isHex); err != nil {
			return nil, err
		}
	}

	if ch := lex.peek(); isHex && (ch == 'p' || ch == 'P') {
		isFloat = true
		if err := lex.parseExponent(&number, isHex); err != nil {
			return nil, err
		}
	}

	if isFloat {
		fval, _, err := big.ParseFloat(number.String(), 0, 0, big.ToNearestEven)
		if err != nil {
			return nil, lex.err(fmt.Errorf("parse float: %w", errors.Unwrap(err)))
		}
		num, _ := fval.Float64()
		return &token{
			Kind:     tokenFloat,
			FloatVal: num,
			LineInfo: linfo,
		}, err
	}

	strNum := number.String()
	if !isHex {
		strNum = strings.TrimLeft(strNum, "0")
		if len(strNum) == 0 {
			return &token{Kind: tokenInteger, IntVal: 0, LineInfo: linfo}, nil
		}
	}

	ivalue, err := strconv.ParseInt(strNum, 0, 64)
	if err != nil {
		return nil, lex.err(fmt.Errorf("parse int: %w", errors.Unwrap(err)))
	}
	return &token{
		Kind:     tokenInteger,
		IntVal:   ivalue,
		LineInfo: linfo,
	}, nil
}

func (lex *lexer) consumeDigits(number *bytes.Buffer, withHex bool) error {
	for {
		ch := lex.peek()
		if !unicode.IsDigit(ch) && (!withHex || !isHexDigit(ch)) {
			return nil
		} else if err := lex.writeNext(number); err != nil {
			return err
		}
	}
}

func (lex *lexer) parseExponent(number *bytes.Buffer, withHex bool) error {
	if err := lex.writeNext(number); err != nil {
		return err
	}
	if tk := lex.peek(); tk == '-' || tk == '+' {
		if err := lex.writeNext(number); err != nil {
			return err
		}
	}
	return lex.consumeDigits(number, withHex)
}

func (lex *lexer) writeNext(number *bytes.Buffer) error {
	if ch, err := lex.next(); err != nil {
		return err
	} else if _, err := number.WriteRune(ch); err != nil {
		return lex.err(err)
	}
	return nil
}

func (lex *lexer) parseShebang() error {
	for {
		if ch, err := lex.next(); err != nil {
			return err
		} else if ch == '\n' {
			return nil
		}
	}
}

func (lex *lexer) parseComment() (*token, error) {
	linfo := lex.LineInfo
	if _, err := lex.next(); err != nil {
		return nil, err
	}

	var comment bytes.Buffer
	peekCh := lex.peek()
	if ch, err := lex.next(); err != nil {
		return nil, err
	} else if ch == '[' && (peekCh == '=' || peekCh == '[') {
		str, _ := lex.parseBracketed()
		return &token{
			Kind:      tokenComment,
			StringVal: str,
			LineInfo:  linfo,
		}, nil
	} else if _, err := comment.WriteRune(ch); err != nil {
		return nil, lex.err(err)
	}

	for {
		ch, err := lex.next()
		if err != nil && !errors.Is(err, io.EOF) {
			return nil, err
		} else if ch == '\n' || errors.Is(err, io.EOF) {
			return &token{
				Kind:      tokenComment,
				StringVal: comment.String(),
				LineInfo:  linfo,
			}, nil
		} else if _, err := comment.WriteRune(ch); err != nil {
			return nil, lex.err(err)
		}
	}
}

func (lex *lexer) parseBracketedString() (*token, error) {
	linfo := lex.LineInfo
	str, err := lex.parseBracketed()
	return &token{
		Kind:      tokenString,
		StringVal: str,
		LineInfo:  linfo,
	}, err
}

func (lex *lexer) parseBracketed() (string, error) {
	var start bytes.Buffer
	if _, err := start.WriteRune(']'); err != nil {
		return "", err
	}

	for {
		if ch, err := lex.next(); err != nil {
			return "", err
		} else if ch == '=' {
			if _, err := start.WriteRune('='); err != nil {
				return "", err
			}
		} else if ch == '[' {
			if _, err := start.WriteRune(']'); err != nil {
				return "", err
			}
			break
		} else {
			return "", lex.errf("malformed bracketed string, expected [ or = and found %v", string(ch))
		}
	}

	if err := lex.skipWhitespace(); err != nil {
		return "", err
	}

	var str bytes.Buffer
	expected := start.String()
	var endPart bytes.Buffer
	for {
		if ch, err := lex.next(); err != nil {
			return "", err
		} else if ch == ']' && endPart.Len() > 0 {
			if endPart.String()+"]" == expected {
				return str.String(), nil
			}
			str.WriteString(endPart.String())
			endPart.Reset()
			endPart.WriteRune(']')
		} else if ch == ']' && endPart.Len() == 0 {
			endPart.WriteRune(']')
		} else if ch == '=' && endPart.Len() > 0 {
			endPart.WriteRune('=')
		} else if str.Len() == 0 && ch == '\n' {
			continue
		} else {
			if endPart.Len() > 0 {
				str.WriteString(endPart.String())
				endPart.Reset()
			}
			str.WriteRune(ch)
		}
	}
}

func isHexDigit(ch rune) bool {
	return unicode.IsDigit(ch) || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')
}
