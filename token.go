package luaf

import "fmt"

type (
	tokenType string
	token     struct {
		LineInfo
		Kind      tokenType
		Ident     string
		StringVal string
		FloatVal  float64
		IntVal    int64
	}
)

const (
	tokenAdd             tokenType = "+"
	tokenMinus           tokenType = "-"
	tokenMultiply        tokenType = "*"
	tokenDivide          tokenType = "/"
	tokenFloorDivide     tokenType = "//"
	tokenModulo          tokenType = "%"
	tokenExponent        tokenType = "^"
	tokenBitwiseAnd      tokenType = "&"
	tokenBitwiseOr       tokenType = "||"
	tokenBitwiseNotOrXOr tokenType = "~"
	tokenShiftLeft       tokenType = "<<"
	tokenShiftRight      tokenType = ">>"
	tokenAssign          tokenType = "="
	tokenColon           tokenType = ":"
	tokenComma           tokenType = ","
	tokenPeriod          tokenType = "."
	tokenSemiColon       tokenType = ";"
	tokenLength          tokenType = "#"
	tokenOpenParen       tokenType = "("
	tokenCloseParen      tokenType = ")"
	tokenOpenCurly       tokenType = "{"
	tokenCloseCurly      tokenType = "}"
	tokenOpenBracket     tokenType = "["
	tokenCloseBracket    tokenType = "]"
	tokenAnd             tokenType = "and"
	tokenBreak           tokenType = "break"
	tokenDo              tokenType = "do"
	tokenElse            tokenType = "else"
	tokenElseif          tokenType = "elseif"
	tokenEnd             tokenType = "end"
	tokenFalse           tokenType = "false"
	tokenFor             tokenType = "for"
	tokenFunction        tokenType = "function"
	tokenGoto            tokenType = "goto"
	tokenIf              tokenType = "if"
	tokenIn              tokenType = "in"
	tokenLocal           tokenType = "local"
	tokenNil             tokenType = "nil"
	tokenNot             tokenType = "not"
	tokenOr              tokenType = "or"
	tokenRepeat          tokenType = "repeat"
	tokenReturn          tokenType = "return"
	tokenThen            tokenType = "then"
	tokenTrue            tokenType = "true"
	tokenUntil           tokenType = "until"
	tokenWhile           tokenType = "while"
	tokenConcat          tokenType = ".."
	tokenDots            tokenType = "..."
	tokenEq              tokenType = "=="
	tokenGe              tokenType = ">="
	tokenGt              tokenType = ">"
	tokenLe              tokenType = "<="
	tokenLt              tokenType = "<"
	tokenNe              tokenType = "~="
	tokenDoubleColon     tokenType = "::"
	tokenFloat           tokenType = "float"
	tokenInteger         tokenType = "integer"
	tokenIdentifier      tokenType = "identifier"
	tokenString          tokenType = "string"
	tokenComment         tokenType = "comment"
	tokenEOS             tokenType = "<EOS>"
)

const unaryPriority = 12

// left, right priority for binary ops
var (
	binaryPriority = map[tokenType][2]int{
		tokenOr:              {1, 1},
		tokenAnd:             {2, 2},
		tokenEq:              {3, 3},
		tokenLt:              {3, 3},
		tokenLe:              {3, 3},
		tokenGt:              {3, 3},
		tokenGe:              {3, 3},
		tokenNe:              {3, 3},
		tokenBitwiseOr:       {4, 4},
		tokenBitwiseNotOrXOr: {5, 5},
		tokenBitwiseAnd:      {6, 6},
		tokenShiftLeft:       {7, 7},
		tokenShiftRight:      {7, 7},
		tokenConcat:          {9, 8},
		tokenAdd:             {10, 10},
		tokenMinus:           {10, 10},
		tokenMultiply:        {11, 11},
		tokenModulo:          {11, 11},
		tokenDivide:          {11, 11},
		tokenFloorDivide:     {11, 11},
		tokenExponent:        {14, 13},
	}
	keywords = map[string]tokenType{
		string(tokenAnd):      tokenAnd,
		string(tokenTrue):     tokenTrue,
		string(tokenFalse):    tokenFalse,
		string(tokenNil):      tokenNil,
		string(tokenBreak):    tokenBreak,
		string(tokenDo):       tokenDo,
		string(tokenElse):     tokenElse,
		string(tokenElseif):   tokenElseif,
		string(tokenEnd):      tokenEnd,
		string(tokenFor):      tokenFor,
		string(tokenFunction): tokenFunction,
		string(tokenGoto):     tokenGoto,
		string(tokenIf):       tokenIf,
		string(tokenIn):       tokenIn,
		string(tokenLocal):    tokenLocal,
		string(tokenNot):      tokenNot,
		string(tokenOr):       tokenOr,
		string(tokenRepeat):   tokenRepeat,
		string(tokenReturn):   tokenReturn,
		string(tokenThen):     tokenThen,
		string(tokenUntil):    tokenUntil,
		string(tokenWhile):    tokenWhile,
	}
	tokenToBytecodeOp = map[tokenType]BytecodeOp{
		tokenEq:              EQ,
		tokenLt:              LT,
		tokenLe:              LE,
		tokenBitwiseOr:       BOR,
		tokenBitwiseNotOrXOr: BXOR,
		tokenBitwiseAnd:      BAND,
		tokenShiftLeft:       SHL,
		tokenShiftRight:      SHR,
		tokenConcat:          CONCAT,
		tokenAdd:             ADD,
		tokenMinus:           SUB,
		tokenMultiply:        MUL,
		tokenModulo:          MOD,
		tokenDivide:          DIV,
		tokenFloorDivide:     IDIV,
		tokenExponent:        POW,
	}
	tokenToMetaMethod = map[tokenType]metaMethod{
		tokenAdd:             metaAdd,
		tokenMinus:           metaSub,
		tokenMultiply:        metaMul,
		tokenDivide:          metaDiv,
		tokenFloorDivide:     metaIDiv,
		tokenModulo:          metaMod,
		tokenBitwiseAnd:      metaBAnd,
		tokenBitwiseOr:       metaBOr,
		tokenBitwiseNotOrXOr: metaBXOr,
		tokenShiftLeft:       metaShl,
		tokenShiftRight:      metaShr,
		tokenExponent:        metaPow,
	}
)

func (tk *token) String() string {
	switch tk.Kind {
	case tokenFloat:
		return fmt.Sprintf("f%v", tk.FloatVal)
	case tokenInteger:
		return fmt.Sprintf("i%v", tk.IntVal)
	case tokenIdentifier:
		return fmt.Sprintf("<%v>", tk.StringVal)
	case tokenString:
		return fmt.Sprintf("\"%v\"", tk.StringVal)
	case tokenComment:
		return fmt.Sprintf("// %v", tk.StringVal)
	default:
		return string(tk.Kind)
	}
}

func (tk *token) isUnary() bool {
	switch tk.Kind {
	case tokenNot, tokenLength, tokenMinus, tokenBitwiseNotOrXOr:
		return true
	default:
		return false
	}
}

func (tk *token) isBinary() bool {
	_, ok := binaryPriority[tk.Kind]
	return ok
}
