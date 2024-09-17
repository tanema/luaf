package shine

import (
	"fmt"
	"io"
)

type UpIndex struct {
	Local bool
	Index uint
}

type ParseResult struct {
	blocks []*Scope
}

type Scope struct {
	sp          uint16 //stack pointer
	Varargs     bool
	Nparam      uint
	Constants   []Value
	Locals      map[string]uint16  // name mapped to stack index of where the local was loaded
	UpIndexes   map[string]UpIndex // name mapped to upindex
	ByteCodes   []Bytecode
	Breakable   bool
	Continuable bool
}

func Parse(filename string, src io.Reader) (*ParseResult, error) {
	lex := NewLexer(src)
	res := &ParseResult{}
	err := chunk(lex, res)
	res.dumpDebugInfo()
	if err == io.EOF {
		return res, nil
	}
	return res, err
}

func chunk(lex *Lexer, res *ParseResult) error {
	res.blocks = append(res.blocks, &Scope{Locals: map[string]uint16{}})
	//return blockScope(lex, res)
	return nil
}

func parseStatList(lex *Lexer, res *ParseResult) error {
	for !blockFollow(lex, true) {
		if lex.Peek().Kind == TokenReturn {
			return parseStatement(lex, res) /* 'return' must be last statement */
		}
		if err := parseStatement(lex, res); err != nil {
			return err
		}
	}
	return nil
}

func blockFollow(lex *Lexer, withuntil bool) bool {
	switch lex.Peek().Kind {
	case TokenElse, TokenElseif, TokenEnd, TokenEOS:
		return true
	case TokenUntil:
		return withuntil
	default:
		return false
	}
}

func parseStatement(lex *Lexer, res *ParseResult) error {
	switch lex.Peek().Kind {
	case TokenSemiColon:
		_, err := lex.Next()
		return err
	case TokenIf: //self.if_stat(lex, fp)
	case TokenWhile: //self.while_stat(lex, fp)
	case TokenDo: //self.do_stat(lex, fp
	case TokenFor: //self.for_stat(lex, fp)
	case TokenRepeat: //self.repeat_stat(lex, fp)
	case TokenFunction: //self.function_stat(lex, fp)
	case TokenLocal:
		return parseLocal(lex, res)
	case TokenDoubleColon: //self.label_stat(lex, fp, igoto)
	case TokenReturn: //self.ret_stat(lex, fp)
	case TokenBreak: //self.break_stat(fp)
	case TokenGoto: //self.goto_stat(lex, fp)
	default:
		return expr(lex, res)
	}
	return nil
}

func parseLocal(lex *Lexer, res *ParseResult) error {
	if _, err := lex.Next(); err != nil {
		return err
	}

	if lex.Peek().Kind == TokenFunction {
		return nil
	} else if lex.Peek().Kind != TokenIdentifier {
		return fmt.Errorf("unexpected token after local keyword")
	}
	names, err := parseNameList(lex)
	if err != nil {
		return err
	}
	//scope := res.blocks[len(res.blocks)-1]
	if tk := lex.Peek(); tk.Kind != TokenAssign {
		//scope.ByteCodes = append(scope.ByteCodes, IABC(LOADNIL, res.sp, res.sp+uint16(len(names)), 0))
		//scope.sp += uint16(len(names))
		return nil
	} else if _, err := lex.Next(); err != nil {
		return err
	}

	vals, err := parseExpList(lex, res)
	if err != nil {
		return err
	}
	matches := min(len(names), len(vals))
	for i := 0; i < matches; i++ {
		vals[i].load(names[i], res)
	}
	return nil
}

func (res *ParseResult) findVal(name string) *exprDesc {
	scope := res.blocks[len(res.blocks)-1]
	if idx, ok := scope.Locals[name]; ok {
		return &exprDesc{kind: localExpr, idx: idx}
	} else if idx, ok := scope.UpIndexes[name]; ok {
		return &exprDesc{kind: upvalueExpr, idx: uint16(idx.Index)}
	}
	for i := len(res.blocks) - 2; i >= 0; i-- {
		scope := res.blocks[i]
		if idx, ok := scope.Locals[name]; ok {
			return &exprDesc{kind: localExpr, idx: idx}
		} else if idx, ok := scope.UpIndexes[name]; ok {
			return &exprDesc{kind: upvalueExpr, idx: uint16(idx.Index)}
		}
	}
	// addConst
	res.findVal("_ENV")

	return nil

}

func (scope *Scope) addConst(val Value) uint16 {
	if idx := findValue(scope.Constants, val); idx >= 0 {
		return uint16(idx)
	}
	scope.Constants = append(scope.Constants, val)
	return uint16(len(scope.Constants) - 1)
}

func (res *ParseResult) dumpDebugInfo() {
	res.blocks[0].dumpDebugInfo()
}

func (scope *Scope) dumpDebugInfo() {
	fmt.Printf("%v params, %v upvalue, %v local, %v constants\n", scope.Nparam, len(scope.UpIndexes), len(scope.Locals), len(scope.Constants))
	for i, bytecode := range scope.ByteCodes {
		fmt.Printf("%v\t%s\n", i, bytecode.String())
	}
}

func findValue(all []Value, item Value) int {
	for i, v := range all {
		if v.Val() == item.Val() {
			return i
		}
	}
	return -1
}

func parseNameList(lex *Lexer) ([]string, error) {
	names := []string{}
	for {
		tk, err := lex.Next()
		if err != nil {
			return nil, err
		} else if tk.Kind != TokenIdentifier {
			return nil, fmt.Errorf("expected Name but found %v", tk.Kind)
		}
		names = append(names, tk.StringVal)
		if lex.Peek().Kind != TokenComma {
			break
		} else if _, err := lex.Next(); err != nil {
			return nil, err
		}
	}
	return names, nil
}

func expr(lex *Lexer, res *ParseResult) error {
	return subexpr(lex, res, 0)
}

// subexpr -> (simpleexp | unop subexpr) { binop subexpr }
// where 'binop' is any binary operator with a priority higher than 'limit'
func subexpr(lex *Lexer, res *ParseResult, limit int) error {
	if tk, err := lex.Next(); err != nil {
		return err
	} else if tk.isUnary() {
		if err := subexpr(lex, res, 12); err != nil {
			return err
		}
	} else {
	}
	return nil
}

// simpleexp -> Float | Integer | String | nil | true | false | ... | constructor | FUNCTION body | suffixedexp
func simpleexp(lex *Lexer, res *ParseResult, tk *Token) error {
	switch tk.Kind {
	case TokenFloat:
	case TokenInteger:
	case TokenString:
	case TokenNil:
	case TokenTrue, TokenFalse:
	case TokenDots:
	case TokenOpenCurly:
	case TokenFunction:
	default:
		return suffixedexp(lex, res)
	}
	return nil
}

// primaryexp -> NAME | '(' expr ')'
func primaryexp(lex *Lexer, res *ParseResult) error {
	tk, err := lex.Next()
	if err != nil {
		return err
	}
	switch tk.Kind {
	case TokenOpenParen:
		if _, err := lex.Next(); err != nil {
			return err
		} else if _, err := parseExp(lex, res); err != nil {
			return err
		} else if lastCh, err := lex.Next(); err != nil {
			return err
		} else if lastCh.Kind != TokenCloseParen {
			return fmt.Errorf("unmatched paren")
		}
		return nil // return expression
	case TokenIdentifier:
		return singlevar(lex, res)
	default:
		return fmt.Errorf("unexpected symbol %v", tk.Kind)
	}
}

// suffixedexp -> primaryexp { '.' NAME | '[' exp ']' | ':' NAME funcargs | funcargs }
func suffixedexp(lex *Lexer, res *ParseResult) error {
	if tk, err := lex.Next(); err != nil {
		return err
	} else if tk.Kind == TokenOpenParen {
		if _, err := lex.Next(); err != nil {
			return err
		} else if _, err := parseExp(lex, res); err != nil {
			return err
		} else if lastCh, err := lex.Next(); err != nil {
			return err
		} else if lastCh.Kind != TokenCloseParen {
			return fmt.Errorf("unmatched paren")
		}
		return nil // return expression
	} else if tk.Kind == TokenIdentifier {
		switch lex.Peek().Kind {
		case TokenPeriod: // field index
		case TokenOpenBracket: // index
		case TokenColon: // fn call with self
		case TokenString, TokenOpenCurly, TokenOpenParen: // fn call
		}
		return nil // return name
	} else {
		return fmt.Errorf("expected prefix expression and found token %v", tk.Kind)
	}
}

func singlevar(lex *Lexer, res *ParseResult) error {
	return nil
}

func parseExpList(lex *Lexer, res *ParseResult) ([]*exprDesc, error) {
	exprs := []*exprDesc{}
	for {
		desc, err := parseExp(lex, res)
		if err != nil {
			return nil, err
		}
		exprs = append(exprs, desc)
		if lex.Peek().Kind != TokenComma {
			break
		} else if _, err := lex.Next(); err != nil {
			return nil, err
		}
	}
	return exprs, nil
}

func parseExp(lex *Lexer, res *ParseResult) (*exprDesc, error) {
	tk, err := lex.Next()
	if err != nil {
		return nil, err
	}
	switch tk.Kind {
	case TokenNil:
		return &exprDesc{kind: nilExpr}, nil
	case TokenFalse, TokenTrue:
		return &exprDesc{kind: booleanExpr, value: &Boolean{tk.Kind == TokenTrue}}, nil
	case TokenInteger:
		return &exprDesc{kind: integerExpr, value: &Integer{tk.IntVal}}, nil
	case TokenFloat:
		return &exprDesc{kind: floatExpr, value: &Float{tk.FloatVal}}, nil
	case TokenString:
		return &exprDesc{kind: stringExpr, value: &String{tk.StringVal}}, nil
	case TokenOpenCurly:
		return parseTableConstructor(lex, res)
	case TokenFunction:
		return parseClosure(lex, res)
	case TokenNot, TokenMinus, TokenLength:
		return parseUnary(lex, res)
	default:
		return nil, nil
	}
}

func parseClosure(lex *Lexer, res *ParseResult) (*exprDesc, error) {
	return nil, nil
}

func parseUnary(lex *Lexer, res *ParseResult) (*exprDesc, error) {
	return nil, nil
}

func parseTableConstructor(lex *Lexer, res *ParseResult) (*exprDesc, error) {
	return nil, nil
}

func min(x, y int) int {
	if x >= y {
		return x
	}
	return y
}
