package luaf

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"math/big"
	"strconv"
	"unicode"
)

var escapeCodes = map[rune]rune{
	'a':  '\x07',
	'b':  '\x08',
	'f':  '\x0C',
	'n':  '\n',
	'r':  '\r',
	't':  '\t',
	'v':  '\x0B',
	'\\': '\\',
	'"':  '"',
	'\'': '\'',
	'[':  '[',
	']':  ']',
}

type Lexer struct {
	rdr         *bufio.Reader
	peeked      rune
	peekedToken *Token
}

func NewLexer(src io.Reader) *Lexer {
	return &Lexer{
		rdr: bufio.NewReader(src),
	}
}

func (lex *Lexer) peek() rune {
	if lex.peeked != 0 {
		return lex.peeked
	}
	lex.peeked, _, _ = lex.rdr.ReadRune()
	return lex.peeked
}

func (lex *Lexer) next() (rune, error) {
	if lex.peeked != 0 {
		ch := lex.peeked
		lex.peeked = 0
		return ch, nil
	}
	ch, _, err := lex.rdr.ReadRune()
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
	return &Token{Kind: tk}, nil
}

func (lex *Lexer) takeTokenVal(tk TokenType) (*Token, error) {
	_, err := lex.next()
	return &Token{Kind: tk}, err
}

func (lex *Lexer) Peek() *Token {
	if lex.peekedToken == nil {
		tk, err := lex.Next()
		if err != nil {
			return &Token{Kind: TokenEOS}
		}
		lex.peekedToken = tk
	}
	return lex.peekedToken
}

func (lex *Lexer) Next() (*Token, error) {
	if lex.peekedToken != nil {
		token := lex.peekedToken
		lex.peekedToken = nil
		return token, nil
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
	return nil, fmt.Errorf("Unexpected character %v", string(ch))
}

func (lex *Lexer) parseIdentifier(start rune) (*Token, error) {
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
	}, nil
}

func (lex *Lexer) parseString(delimiter rune) (*Token, error) {
	var str bytes.Buffer
	for {
		if ch, err := lex.next(); err != nil {
			return nil, err
		} else if ch == '\\' {
			if ch, err := lex.next(); err != nil {
				return nil, err
			} else if esc, ok := escapeCodes[ch]; !ok {
				return nil, fmt.Errorf("Invalid escape code \\%v", ch)
			} else if _, err := str.WriteRune(esc); err != nil {
				return nil, err
			}
		} else if ch == delimiter {
			return &Token{
				Kind:      TokenString,
				StringVal: str.String(),
			}, nil
		} else {
			str.WriteRune(ch)
		}
	}
}

func (lex *Lexer) parseNumber(start rune) (*Token, error) {
	var number bytes.Buffer
	if _, err := number.WriteRune(start); err != nil {
		return nil, err
	}

	isFloat := false
	if digits, err := lex.consumeDigits(); err != nil {
		return nil, err
	} else if _, err := number.WriteString(digits); err != nil {
		return nil, err
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
			return nil, err
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

	return formatNumber(number.String(), isFloat)
}

func formatNumber(number string, isFloat bool) (*Token, error) {
	if isFloat {
		fval, _, err := big.NewFloat(0).Parse(number, 0)
		if err != nil {
			return nil, err
		}
		num, _ := fval.Float64()
		return &Token{
			Kind:     TokenFloat,
			FloatVal: num,
		}, err
	}
	ivalue, err := strconv.ParseInt(number, 0, 64)
	return &Token{
		Kind:   TokenInteger,
		IntVal: ivalue,
	}, err
}

func (lex *Lexer) consumeDigits() (string, error) {
	var number bytes.Buffer
	for {
		if ch := lex.peek(); !unicode.IsDigit(ch) {
			return number.String(), nil
		} else if ch, err := lex.next(); err != nil {
			return "", err
		} else if _, err := number.WriteRune(ch); err != nil {
			return "", err
		}
	}
}

func (lex *Lexer) parseHexidecimal() (*Token, error) {
	var number bytes.Buffer
	if _, err := lex.next(); err != nil {
		return nil, err
	} else if _, err := number.WriteString("0x"); err != nil {
		return nil, err
	}

	for {
		if ch := lex.peek(); unicode.IsDigit(ch) || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F') {
			if ch, err := lex.next(); err != nil {
				return nil, err
			} else if _, err := number.WriteRune(ch); err != nil {
				return nil, err
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
			return nil, err
		}
	}

	return formatNumber(number.String(), isFloat)
}

func (lex *Lexer) parseExponent() (string, error) {
	var exponent bytes.Buffer
	if ch, err := lex.next(); err != nil {
		return "", err
	} else if _, err := exponent.WriteRune(ch); err != nil {
		return "", err
	}
	if lex.peek() == '-' {
		if ch, err := lex.next(); err != nil {
			return "", err
		} else if _, err := exponent.WriteRune(ch); err != nil {
			return "", err
		}
	}

	if digits, err := lex.consumeDigits(); err != nil {
		return "", err
	} else if _, err := exponent.WriteString(digits); err != nil {
		return "", err
	}
	return exponent.String(), nil
}

func (lex *Lexer) parseComment() (*Token, error) {
	if _, err := lex.next(); err != nil {
		return nil, err
	}

	var comment bytes.Buffer
	if ch, err := lex.next(); err != nil {
		return nil, err
	} else if ch == '[' {
		str, err := lex.parseBracketed()
		return &Token{
			Kind:      TokenComment,
			StringVal: str,
		}, err
	} else if _, err := comment.WriteRune(ch); err != nil {
		return nil, err
	}

	for {
		if ch, err := lex.next(); err != nil {
			return nil, err
		} else if ch == '\n' {
			return &Token{
				Kind:      TokenComment,
				StringVal: comment.String(),
			}, nil
		} else if _, err := comment.WriteRune(ch); err != nil {
			return nil, err
		}
	}
}

func (lex *Lexer) parseBracketedString() (*Token, error) {
	str, err := lex.parseBracketed()
	return &Token{
		Kind:      TokenString,
		StringVal: str,
	}, err
}

func (lex *Lexer) parseBracketed() (string, error) {
	startLen := 0
	for {
		if ch, err := lex.next(); err != nil {
			return "", err
		} else if ch == '=' {
			startLen += 1
		} else if ch == '[' {
			break
		}
	}

	var str bytes.Buffer
	for {
		if ch, err := lex.next(); err != nil {
			return "", err
		} else if ch == ']' {
			var endPart bytes.Buffer
			endLen := 0
			if _, err := endPart.WriteRune(ch); err != nil {
				return "", err
			}
			for {
				if ch, err := lex.next(); err != nil {
					return "", err
				} else if ch == '=' {
					if _, err := endPart.WriteRune(ch); err != nil {
						return "", err
					}
					endLen += 1
				} else if ch == ']' && startLen == endLen {
					return str.String(), nil
				} else {
					if _, err := endPart.WriteRune(ch); err != nil {
						return "", err
					}
					str.WriteString(endPart.String())
				}
			}
		} else if str.Len() == 0 && ch == '\n' {
			continue
		} else if _, err := str.WriteRune(ch); err != nil {
			return "", err
		}
	}
}
