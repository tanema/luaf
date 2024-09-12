package shine

import "fmt"

type (
	TokenType string
	Token     struct {
		Kind      TokenType
		Ident     string
		StringVal string
		FloatVal  float64
		IntVal    int64
	}
)

const (
	TokenAdd             TokenType = "+"
	TokenMinus           TokenType = "-"
	TokenMultiply        TokenType = "*"
	TokenDivide          TokenType = "/"
	TokenFloorDivide     TokenType = "//"
	TokenModulo          TokenType = "%"
	TokenExponent        TokenType = "^"
	TokenBitwiseAnd      TokenType = "&"
	TokenBitwiseOr       TokenType = "||"
	TokenBitwiseNotOrXOr TokenType = "~"
	TokenShiftLeft       TokenType = "<<"
	TokenShiftRight      TokenType = ">>"
	TokenAssign          TokenType = "="
	TokenColon           TokenType = ":"
	TokenComma           TokenType = ","
	TokenPeriod          TokenType = "."
	TokenSemiColon       TokenType = ";"
	TokenLength          TokenType = "#"
	TokenOpenParen       TokenType = "("
	TokenCloseParen      TokenType = ")"
	TokenOpenCurly       TokenType = "{"
	TokenCloseCurly      TokenType = "}"
	TokenOpenBracket     TokenType = "["
	TokenCloseBracket    TokenType = "]"
	TokenAnd             TokenType = "and"
	TokenBreak           TokenType = "break"
	TokenDo              TokenType = "do"
	TokenElse            TokenType = "else"
	TokenElseif          TokenType = "elseif"
	TokenEnd             TokenType = "end"
	TokenFalse           TokenType = "false"
	TokenFor             TokenType = "for"
	TokenFunction        TokenType = "function"
	TokenGoto            TokenType = "goto"
	TokenIf              TokenType = "if"
	TokenIn              TokenType = "in"
	TokenLocal           TokenType = "local"
	TokenNil             TokenType = "nil"
	TokenNot             TokenType = "not"
	TokenOr              TokenType = "or"
	TokenRepeat          TokenType = "repeat"
	TokenReturn          TokenType = "return"
	TokenThen            TokenType = "then"
	TokenTrue            TokenType = "true"
	TokenUntil           TokenType = "until"
	TokenWhile           TokenType = "while"
	TokenConcat          TokenType = ".."
	TokenDots            TokenType = "..."
	TokenEq              TokenType = "=="
	TokenGe              TokenType = ">="
	TokenGt              TokenType = ">"
	TokenLe              TokenType = "<="
	TokenLt              TokenType = "<"
	TokenNe              TokenType = "~="
	TokenDoubleColon     TokenType = "::"
	TokenFloat           TokenType = "float"
	TokenInteger         TokenType = "integer"
	TokenIdentifier      TokenType = "identifier"
	TokenString          TokenType = "string"
	TokenComment         TokenType = "comment"
	TokenEOS             TokenType = "<EOS>"
)

var keywords = map[string]TokenType{
	string(TokenAnd):      TokenAnd,
	string(TokenTrue):     TokenTrue,
	string(TokenFalse):    TokenFalse,
	string(TokenNil):      TokenNil,
	string(TokenBreak):    TokenBreak,
	string(TokenDo):       TokenDo,
	string(TokenElse):     TokenElse,
	string(TokenElseif):   TokenElseif,
	string(TokenEnd):      TokenEnd,
	string(TokenFor):      TokenFor,
	string(TokenFunction): TokenFunction,
	string(TokenGoto):     TokenGoto,
	string(TokenIf):       TokenIf,
	string(TokenIn):       TokenIn,
	string(TokenLocal):    TokenLocal,
	string(TokenNot):      TokenNot,
	string(TokenOr):       TokenOr,
	string(TokenRepeat):   TokenRepeat,
	string(TokenReturn):   TokenReturn,
	string(TokenThen):     TokenThen,
	string(TokenUntil):    TokenUntil,
	string(TokenWhile):    TokenWhile,
}

func (tk *Token) String() string {
	switch tk.Kind {
	case TokenFloat:
		return fmt.Sprintf("f%v", tk.FloatVal)
	case TokenInteger:
		return fmt.Sprintf("i%v", tk.IntVal)
	case TokenIdentifier:
		return fmt.Sprintf("<%v>", tk.StringVal)
	case TokenString:
		return fmt.Sprintf("\"%v\"", tk.StringVal)
	case TokenComment:
		return fmt.Sprintf("// %v", tk.StringVal)
	default:
		return string(tk.Kind)
	}
}

func (tk *Token) isUnary() bool {
	switch tk.Kind {
	case TokenNot, TokenLength, TokenMinus, TokenBitwiseNotOrXOr:
		return true
	default:
		return false
	}
}

func (tk *Token) isBinary() bool {
	switch tk.Kind {
	case TokenAdd, TokenDivide, TokenMinus, TokenBitwiseNotOrXOr, TokenMultiply,
		TokenOr, TokenLt, TokenLe, TokenGt, TokenGe, TokenEq, TokenAssign, TokenNe,
		TokenShiftRight, TokenShiftLeft, TokenConcat:
		return true
	default:
		return false
	}
}

func (tk *Token) isKeyword() bool {
	_, ok := keywords[string(tk.Kind)]
	return ok
}
