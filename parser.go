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
	sp      uint16 //stack pointer
	Globals []Value
	Blocks  []*Scope
}

type Scope struct {
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
	res.Blocks = append(res.Blocks, &Scope{Locals: map[string]uint16{}})
	return blockScope(lex, res)
}

func blockScope(lex *Lexer, res *ParseResult) error {
	for {
		var err error
		switch lex.Peek().Kind {
		case TokenSemiColon:
		case TokenLocal:
			err = parseLocal(lex, res)
		case TokenFunction: //self.function_stat(lex, fp)
		case TokenIf: //self.if_stat(lex, fp)
		case TokenWhile: //self.while_stat(lex, fp)
		case TokenRepeat: //self.repeat_stat(lex, fp)
		case TokenFor: //self.for_stat(lex, fp)
		case TokenBreak: //self.break_stat(fp)
		case TokenDo: //self.do_stat(lex, fp
		case TokenDoubleColon: //self.label_stat(lex, fp, igoto)
		case TokenGoto: //self.goto_stat(lex, fp)
		case TokenReturn: //self.ret_stat(lex, fp)
		case TokenIdentifier, TokenOpenParen:
			err = parseAssignOrCall(lex, res)
		default:
			break
		}

		if err != nil {
			return err
		}
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
	scope := res.Blocks[len(res.Blocks)-1]
	if tk := lex.Peek(); tk.Kind != TokenAssign {
		scope.ByteCodes = append(scope.ByteCodes, AsBytecode(LOADNIL, res.sp, res.sp+uint16(len(names))))
		res.sp += uint16(len(names))
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
	scope := res.Blocks[len(res.Blocks)-1]
	if idx, ok := scope.Locals[name]; ok {
		return &exprDesc{kind: localExpr, idx: idx}
	} else if idx, ok := scope.UpIndexes[name]; ok {
		return &exprDesc{kind: upvalueExpr, idx: uint16(idx.Index)}
	}
	for i := len(res.Blocks) - 2; i >= 0; i-- {
		scope := res.Blocks[i]
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
	res.Blocks[0].dumpDebugInfo()
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

func parseAssignOrCall(lex *Lexer, res *ParseResult) error {
	return parsePrefixExp(lex, res)
}

func parseAssign(lex *Lexer, res *ParseResult) error {
	return nil
}

func parsePrefixExp(lex *Lexer, res *ParseResult) error {
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
		if peekTk := lex.Peek(); peekTk.Kind == TokenOpenBracket {
			// find index point
			// expression index
			if _, err := lex.Next(); err != nil {
				return err
			} else if _, err := parseExp(lex, res); err != nil {
				return err
			} else if tk, err := lex.Next(); tk.Kind != TokenOpenBracket {
				return fmt.Errorf("Expected closing bracket but found %v", tk.Kind)
			} else if err != nil {
				return err
			}
			return nil
		} else if peekTk.Kind == TokenPeriod {
			// identifier index
			// prefixexpr . Name
		} else if peekTk.Kind == TokenColon {
			// function call with self
			// prefixexpr : Name args
		} else if peekTk.Kind == TokenString || peekTk.Kind == TokenOpenCurly || peekTk.Kind == TokenOpenParen {
			// function call
			// prefixexpr"string" prefixexpr{table = "yes"} prefixexpr(args)
			parseFuncCall(lex, res, &exprDesc{kind: localExpr, value: &String{val: tk.StringVal}})
		}
		return nil // return name
	} else {
		return fmt.Errorf("expected prefix expression and found token %v", tk.Kind)
	}
}

func parseFuncCall(lex *Lexer, res *ParseResult, ident *exprDesc) error {
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
