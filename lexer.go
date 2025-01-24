package luaf

import (
	"bufio"
	"bytes"
	"container/list"
	"fmt"
	"io"
	"math/big"
	"strconv"
	"unicode"
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
	Lexer struct {
		LineInfo
		rdr    *bufio.Reader
		peeked list.List
	}
	LexerError struct {
		LineInfo
		err error
	}
)

func (le *LexerError) Error() string {
	return le.err.Error()
}

func NewLexer(src io.Reader) *Lexer {
	return &Lexer{
		LineInfo: LineInfo{Line: 1},
		rdr:      bufio.NewReaderSize(src, 4096),
	}
}

func (lex *Lexer) errf(msg string, data ...any) error {
	return lex.err(fmt.Errorf(msg, data...))
}

func (lex *Lexer) err(err error) error {
	if err == io.EOF {
		return err
	}
	return &LexerError{
		LineInfo: lex.LineInfo,
		err:      err,
	}
}

func (lex *Lexer) peek() rune {
	chs, _ := lex.rdr.Peek(1)
	if len(chs) == 0 {
		return 0
	}
	return rune(chs[0])
}

func (lex *Lexer) next() (rune, error) {
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

func (lex *Lexer) skip_whitespace() error {
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

func (lex *Lexer) tokenVal(tk TokenType) (*Token, error) {
	return &Token{Kind: tk, LineInfo: LineInfo{Line: lex.Line, Column: lex.Column - len(tk)}}, nil
}

func (lex *Lexer) takeTokenVal(tk TokenType) (*Token, error) {
	_, err := lex.next()
	return &Token{Kind: tk, LineInfo: LineInfo{Line: lex.Line, Column: lex.Column - len(tk)}}, err
}

// allows to reverse back if next was called. peeked is a linked list that will
// allow for FIFO stack
func (lex *Lexer) Back(tk *Token) {
	lex.peeked.PushFront(tk)
}

func (lex *Lexer) Peek() *Token {
	if lex.peeked.Len() == 0 {
		tk, err := lex.Next()
		if err != nil {
			return &Token{Kind: TokenEOS}
		}
		lex.peeked.PushBack(tk)
	}
	return lex.peeked.Front().Value.(*Token)
}

func (lex *Lexer) Next() (*Token, error) {
	if lex.peeked.Len() != 0 {
		return lex.peeked.Remove(lex.peeked.Front()).(*Token), nil
	}
	if lex.peek() == '#' && lex.Line == 1 {
		_, err := lex.next()
		if err != nil {
			return nil, err
		}
		if lex.peek() == '!' {
			if err := lex.parseShebang(); err != nil {
				return nil, err
			}
		} else {
			return lex.tokenVal(TokenLength)
		}
	}
	if err := lex.skip_whitespace(); err != nil {
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
		return lex.tokenVal(TokenMinus)
	} else if ch == '[' && (peekCh == '=' || peekCh == '[') {
		return lex.parseBracketedString()
	} else if ch == '[' {
		return lex.tokenVal(TokenOpenBracket)
	} else if ch == '=' && peekCh == '=' {
		return lex.takeTokenVal(TokenEq)
	} else if ch == '=' {
		return lex.tokenVal(TokenAssign)
	} else if ch == '<' && peekCh == '=' {
		return lex.takeTokenVal(TokenLe)
	} else if ch == '<' && peekCh == '<' {
		return lex.takeTokenVal(TokenShiftLeft)
	} else if ch == '<' {
		return lex.tokenVal(TokenLt)
	} else if ch == '>' && peekCh == '=' {
		return lex.takeTokenVal(TokenGe)
	} else if ch == '>' && peekCh == '>' {
		return lex.takeTokenVal(TokenShiftRight)
	} else if ch == '>' {
		return lex.tokenVal(TokenGt)
	} else if ch == '~' && peekCh == '=' {
		return lex.takeTokenVal(TokenNe)
	} else if ch == '~' {
		return lex.tokenVal(TokenBitwiseNotOrXOr)
	} else if ch == '/' && peekCh == '/' {
		return lex.takeTokenVal(TokenFloorDivide)
	} else if ch == '/' {
		return lex.tokenVal(TokenDivide)
	} else if ch == '.' && peekCh == '.' {
		if _, err := lex.next(); err != nil {
			return nil, err
		}
		if lex.peek() == '.' {
			return lex.takeTokenVal(TokenDots)
		}
		return lex.tokenVal(TokenConcat)
	} else if ch == '.' {
		return lex.tokenVal(TokenPeriod)
	} else if ch == '+' {
		return lex.tokenVal(TokenAdd)
	} else if ch == '*' {
		return lex.tokenVal(TokenMultiply)
	} else if ch == '%' {
		return lex.tokenVal(TokenModulo)
	} else if ch == '^' {
		return lex.tokenVal(TokenExponent)
	} else if ch == '&' {
		return lex.tokenVal(TokenBitwiseAnd)
	} else if ch == '|' {
		return lex.tokenVal(TokenBitwiseOr)
	} else if ch == ':' {
		if lex.peek() == ':' {
			return lex.takeTokenVal(TokenDoubleColon)
		}
		return lex.tokenVal(TokenColon)
	} else if ch == ',' {
		return lex.tokenVal(TokenComma)
	} else if ch == ';' {
		return lex.tokenVal(TokenSemiColon)
	} else if ch == '#' {
		return lex.tokenVal(TokenLength)
	} else if ch == '(' {
		return lex.tokenVal(TokenOpenParen)
	} else if ch == ')' {
		return lex.tokenVal(TokenCloseParen)
	} else if ch == '{' {
		return lex.tokenVal(TokenOpenCurly)
	} else if ch == '}' {
		return lex.tokenVal(TokenCloseCurly)
	} else if ch == ']' {
		return lex.tokenVal(TokenCloseBracket)
	} else if ch == '"' || ch == '\'' {
		return lex.parseString(ch)
	} else if unicode.IsDigit(ch) {
		return lex.parseNumber(ch)
	} else if unicode.IsLetter(ch) || ch == '_' {
		return lex.parseIdentifier(ch)
	}
	return nil, lex.errf("unexpected character %v", string(ch))
}

func (lex *Lexer) parseIdentifier(start rune) (*Token, error) {
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
	return &Token{
		Kind:      TokenIdentifier,
		StringVal: strVal,
		LineInfo:  linfo,
	}, nil
}

func (lex *Lexer) parseString(delimiter rune) (*Token, error) {
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
			} else {
				str.WriteRune('\\')
				str.WriteRune(ch)
			}
		} else if ch == delimiter {
			return &Token{
				Kind:      TokenString,
				StringVal: str.String(),
				LineInfo:  linfo,
			}, nil
		} else {
			str.WriteRune(ch)
		}
	}
}

func (lex *Lexer) parseNumber(start rune) (*Token, error) {
	linfo := lex.LineInfo
	var number bytes.Buffer
	if _, err := number.WriteRune(start); err != nil {
		return nil, lex.err(err)
	}

	isFloat := false
	if digits, err := lex.consumeDigits(); err != nil {
		return nil, err
	} else if _, err := number.WriteString(digits); err != nil {
		return nil, lex.err(err)
	}

	if peekCh := lex.peek(); peekCh == '.' {
		isFloat = true
		ch, err := lex.next()
		if err != nil {
			return nil, err
		}
		number.WriteRune(ch)
		if digits, err := lex.consumeDigits(); err != nil {
			return nil, err
		} else if _, err := number.WriteString(digits); err != nil {
			return nil, lex.err(err)
		}
	} else if peekCh == 'x' || peekCh == 'X' {
		return lex.parseHexidecimal()
	}

	if peekCh := lex.peek(); peekCh == 'e' || peekCh == 'E' {
		isFloat = true
		if exp, err := lex.parseExponent(); err != nil {
			return nil, err
		} else if _, err := number.WriteString(exp); err != nil {
			return nil, err
		}
	}

	return lex.formatNumber(number.String(), isFloat, linfo)
}

func (lex *Lexer) formatNumber(number string, isFloat bool, linfo LineInfo) (*Token, error) {
	if isFloat {
		fval, _, err := big.NewFloat(0).Parse(number, 0)
		if err != nil {
			return nil, lex.err(err)
		}
		num, _ := fval.Float64()
		return &Token{
			Kind:     TokenFloat,
			FloatVal: num,
			LineInfo: linfo,
		}, err
	}
	ivalue, err := strconv.ParseInt(number, 0, 64)
	if err != nil {
		return nil, lex.err(err)
	}
	return &Token{
		Kind:     TokenInteger,
		IntVal:   ivalue,
		LineInfo: linfo,
	}, nil
}

func (lex *Lexer) consumeDigits() (string, error) {
	var number bytes.Buffer
	for {
		if ch := lex.peek(); !unicode.IsDigit(ch) {
			return number.String(), nil
		} else if ch, err := lex.next(); err != nil {
			return "", err
		} else if _, err := number.WriteRune(ch); err != nil {
			return "", lex.err(err)
		}
	}
}

func (lex *Lexer) parseHexidecimal() (*Token, error) {
	linfo := lex.LineInfo
	var number bytes.Buffer
	if _, err := lex.next(); err != nil {
		return nil, err
	} else if _, err := number.WriteString("0x"); err != nil {
		return nil, lex.err(err)
	}

	for {
		if ch := lex.peek(); unicode.IsDigit(ch) || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F') {
			if ch, err := lex.next(); err != nil {
				return nil, err
			} else if _, err := number.WriteRune(ch); err != nil {
				return nil, lex.err(err)
			}
		} else {
			break
		}
	}

	isFloat := false
	if ch := lex.peek(); ch == 'p' || ch == 'P' {
		isFloat = true
		if exp, err := lex.parseExponent(); err != nil {
			return nil, err
		} else if _, err := number.WriteString(exp); err != nil {
			return nil, lex.err(err)
		}
	}

	return lex.formatNumber(number.String(), isFloat, linfo)
}

func (lex *Lexer) parseExponent() (string, error) {
	var exponent bytes.Buffer
	if ch, err := lex.next(); err != nil {
		return "", err
	} else if _, err := exponent.WriteRune(ch); err != nil {
		return "", lex.err(err)
	}
	if lex.peek() == '-' {
		if ch, err := lex.next(); err != nil {
			return "", err
		} else if _, err := exponent.WriteRune(ch); err != nil {
			return "", lex.err(err)
		}
	}

	if digits, err := lex.consumeDigits(); err != nil {
		return "", err
	} else if _, err := exponent.WriteString(digits); err != nil {
		return "", lex.err(err)
	}
	return exponent.String(), nil
}

func (lex *Lexer) parseShebang() error {
	for {
		if ch, err := lex.next(); err != nil {
			return err
		} else if ch == '\n' {
			return nil
		}
	}
}

func (lex *Lexer) parseComment() (*Token, error) {
	linfo := lex.LineInfo
	if _, err := lex.next(); err != nil {
		return nil, err
	}

	var comment bytes.Buffer
	peekCh := lex.peek()
	if ch, err := lex.next(); err != nil {
		return nil, err
	} else if ch == '[' && (peekCh == '=' || peekCh == '[') {
		str, err := lex.parseBracketed()
		return &Token{
			Kind:      TokenComment,
			StringVal: str,
			LineInfo:  linfo,
		}, err
	} else if _, err := comment.WriteRune(ch); err != nil {
		return nil, lex.err(err)
	}

	for {
		if ch, err := lex.next(); err != nil {
			return nil, err
		} else if ch == '\n' {
			return &Token{
				Kind:      TokenComment,
				StringVal: comment.String(),
				LineInfo:  linfo,
			}, nil
		} else if _, err := comment.WriteRune(ch); err != nil {
			return nil, lex.err(err)
		}
	}
}

func (lex *Lexer) parseBracketedString() (*Token, error) {
	linfo := lex.LineInfo
	str, err := lex.parseBracketed()
	return &Token{
		Kind:      TokenString,
		StringVal: str,
		LineInfo:  linfo,
	}, err
}

func (lex *Lexer) parseBracketed() (string, error) {
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
