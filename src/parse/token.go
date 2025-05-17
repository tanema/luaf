package parse

import (
	"fmt"

	"github.com/tanema/luaf/src/bytecode"
)

type (
	// MetaMethod is the enum of valid meta methods.
	MetaMethod string
	tokenType  string
	token      struct {
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

	// MetaAdd is the __add metamethod.
	MetaAdd MetaMethod = "__add"
	// MetaSub is the __sub metamethod.
	MetaSub MetaMethod = "__sub"
	// MetaMul is the __mul metamethod.
	MetaMul MetaMethod = "__mul"
	// MetaDiv is the __div metamethod.
	MetaDiv MetaMethod = "__div"
	// MetaMod is the __mod methamethod.
	MetaMod MetaMethod = "__mod"
	// MetaPow is the __pow methamethod.
	MetaPow MetaMethod = "__pow"
	// MetaUNM is the __unm methamethod.
	MetaUNM MetaMethod = "__unm"
	// MetaIDiv is the __idiv methamethod.
	MetaIDiv MetaMethod = "__idiv"
	// MetaBAnd is the __band methamethod.
	MetaBAnd MetaMethod = "__band"
	// MetaBOr is the __bor methamethod.
	MetaBOr MetaMethod = "__bor"
	// MetaBXOr is the __bxor methamethod.
	MetaBXOr MetaMethod = "__bxor"
	// MetaBNot is the __bnot methamethod.
	MetaBNot MetaMethod = "__bnot"
	// MetaShl is the __shl methamethod.
	MetaShl MetaMethod = "__shl"
	// MetaShr is the __shr methamethod.
	MetaShr MetaMethod = "__shr"
	// MetaConcat is the __concat methamethod.
	MetaConcat MetaMethod = "__concat"
	// MetaLen is the __len methamethod.
	MetaLen MetaMethod = "__len"
	// MetaEq is the __eq methamethod.
	MetaEq MetaMethod = "__eq"
	// MetaLt is the __lt methamethod.
	MetaLt MetaMethod = "__lt"
	// MetaLe is the __le methamethod.
	MetaLe MetaMethod = "__le"
	// MetaIndex is the __index methamethod.
	MetaIndex MetaMethod = "__index"
	// MetaNewIndex is the __newindex methamethod.
	MetaNewIndex MetaMethod = "__newindex"
	// MetaCall is the __call methamethod.
	MetaCall MetaMethod = "__call"
	// MetaClose is the __close methamethod.
	MetaClose MetaMethod = "__close"
	// MetaToString is the __tostring methamethod.
	MetaToString MetaMethod = "__tostring"
	// MetaName is the __name methamethod.
	MetaName MetaMethod = "__name"
	// MetaPairs is the __pairs methamethod.
	MetaPairs MetaMethod = "__pairs"
	// MetaMeta is the __metatable methamethod.
	MetaMeta MetaMethod = "__metatable"
	// MetaGC is the __gc methamethod.
	MetaGC MetaMethod = "__gc"
)

const unaryPriority = 12

// left, right priority for binary ops.
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
	tokenToBytecodeOp = map[tokenType]bytecode.Op{
		tokenEq:              bytecode.EQ,
		tokenLt:              bytecode.LT,
		tokenLe:              bytecode.LE,
		tokenBitwiseOr:       bytecode.BOR,
		tokenBitwiseNotOrXOr: bytecode.BXOR,
		tokenBitwiseAnd:      bytecode.BAND,
		tokenShiftLeft:       bytecode.SHL,
		tokenShiftRight:      bytecode.SHR,
		tokenConcat:          bytecode.CONCAT,
		tokenAdd:             bytecode.ADD,
		tokenMinus:           bytecode.SUB,
		tokenMultiply:        bytecode.MUL,
		tokenModulo:          bytecode.MOD,
		tokenDivide:          bytecode.DIV,
		tokenFloorDivide:     bytecode.IDIV,
		tokenExponent:        bytecode.POW,
	}
	tokenToMetaMethod = map[tokenType]MetaMethod{
		tokenAdd:             MetaAdd,
		tokenMinus:           MetaSub,
		tokenMultiply:        MetaMul,
		tokenDivide:          MetaDiv,
		tokenFloorDivide:     MetaIDiv,
		tokenModulo:          MetaMod,
		tokenBitwiseAnd:      MetaBAnd,
		tokenBitwiseOr:       MetaBOr,
		tokenBitwiseNotOrXOr: MetaBXOr,
		tokenShiftLeft:       MetaShl,
		tokenShiftRight:      MetaShr,
		tokenExponent:        MetaPow,
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
