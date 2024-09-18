package shine

import (
	"fmt"
	"io"
)

type UpIndex struct {
	Local bool
	Index uint
}

type Parser struct {
	blocks []*FuncProto
	lex    *Lexer
}

type FuncProto struct {
	sp          uint16 //stack pointer
	Varargs     bool
	Nparam      uint64
	Constants   []Value
	Locals      map[string]uint16  // name mapped to stack index of where the local was loaded
	UpIndexes   map[string]UpIndex // name mapped to upindex
	ByteCodes   []Bytecode
	Breakable   bool
	Continuable bool
}

func Parse(filename string, src io.Reader) (*FuncProto, error) {
	p := &Parser{
		lex: NewLexer(src),
	}
	err := p.chunk()
	p.dumpDebugInfo()
	if err == io.EOF {
		return p.blocks[0], nil
	}
	return p.blocks[0], err
}

func (p *Parser) chunk() error {
	p.blocks = append(p.blocks, &FuncProto{Locals: map[string]uint16{}})
	//return p.blockScope()
	return p.statList()
}

func (p *Parser) statList() error {
	for !p.blockFollow(true) {
		if p.lex.Peek().Kind == TokenReturn {
			return p.statement() /* 'return' must be last statement */
		}
		if err := p.statement(); err != nil {
			return err
		}
	}
	return nil
}

func (p *Parser) blockFollow(withuntil bool) bool {
	switch p.lex.Peek().Kind {
	case TokenElse, TokenElseif, TokenEnd, TokenEOS:
		return true
	case TokenUntil:
		return withuntil
	default:
		return false
	}
}

func (p *Parser) statement() error {
	switch p.lex.Peek().Kind {
	case TokenSemiColon:
		_, err := p.lex.Next()
		return err
	case TokenIf: //self.if_stat()
	case TokenWhile: //self.while_stat()
	case TokenDo: //self.do_stat()
	case TokenFor: //self.for_stat()
	case TokenRepeat: //self.repeat_stat()
	case TokenFunction: //self.function_stat()
	case TokenLocal:
		return p.local()
	case TokenDoubleColon: //self.label_stat()
	case TokenReturn: //self.ret_stat()
	case TokenBreak: //self.break_stat()
	case TokenGoto: //self.goto_stat()
	default:
		return p.expr()
	}
	return nil
}

func (p *Parser) local() error {
	if _, err := p.lex.Next(); err != nil {
		return err
	}

	if p.lex.Peek().Kind == TokenFunction {
		return nil
	} else if p.lex.Peek().Kind != TokenIdentifier {
		return fmt.Errorf("unexpected token after local keyword")
	}
	names, err := p.nameList()
	if err != nil {
		return err
	}
	//fnproto := p.blocks[len(res.blocks)-1]
	if tk := p.lex.Peek(); tk.Kind != TokenAssign {
		//fnproto.ByteCodes = append(fnproto.ByteCodes, IABC(LOADNIL, p.sp, p.sp+uint16(len(names)), 0))
		//fnproto.sp += uint16(len(names))
		return nil
	} else if _, err := p.lex.Next(); err != nil {
		return err
	}

	vals, err := p.expList()
	if err != nil {
		return err
	}
	matches := min(len(names), len(vals))
	for i := 0; i < matches; i++ {
		//vals[i].load(names[i])
	}
	return nil
}

func (p *Parser) findVal(name string) *exprDesc {
	fnproto := p.blocks[len(p.blocks)-1]
	if idx, ok := fnproto.Locals[name]; ok {
		return &exprDesc{kind: localExpr, idx: idx}
	} else if idx, ok := fnproto.UpIndexes[name]; ok {
		return &exprDesc{kind: upvalueExpr, idx: uint16(idx.Index)}
	}
	for i := len(p.blocks) - 2; i >= 0; i-- {
		fnproto := p.blocks[i]
		if idx, ok := fnproto.Locals[name]; ok {
			return &exprDesc{kind: localExpr, idx: idx}
		} else if idx, ok := fnproto.UpIndexes[name]; ok {
			return &exprDesc{kind: upvalueExpr, idx: uint16(idx.Index)}
		}
	}
	// addConst
	p.findVal("_ENV")

	return nil

}

func (fnproto *FuncProto) addConst(val Value) uint16 {
	if idx := findValue(fnproto.Constants, val); idx >= 0 {
		return uint16(idx)
	}
	fnproto.Constants = append(fnproto.Constants, val)
	return uint16(len(fnproto.Constants) - 1)
}

func (p *Parser) dumpDebugInfo() {
	p.blocks[0].dumpDebugInfo()
}

func (fnproto *FuncProto) dumpDebugInfo() {
	fmt.Printf("%v params, %v upvalue, %v local, %v constants\n", fnproto.Nparam, len(fnproto.UpIndexes), len(fnproto.Locals), len(fnproto.Constants))
	for i, bytecode := range fnproto.ByteCodes {
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

func (p *Parser) nameList() ([]string, error) {
	names := []string{}
	for {
		tk, err := p.lex.Next()
		if err != nil {
			return nil, err
		} else if tk.Kind != TokenIdentifier {
			return nil, fmt.Errorf("expected Name but found %v", tk.Kind)
		}
		names = append(names, tk.StringVal)
		if p.lex.Peek().Kind != TokenComma {
			break
		} else if _, err := p.lex.Next(); err != nil {
			return nil, err
		}
	}
	return names, nil
}

func (p *Parser) expr() error {
	return p.subexpr(0)
}

// subexpr -> (simpleexp | unop subexpr) { binop subexpr }
// where 'binop' is any binary operator with a priority higher than 'limit'
func (p *Parser) subexpr(limit int) error {
	if tk, err := p.lex.Next(); err != nil {
		return err
	} else if tk.isUnary() {
		if err := p.subexpr(12); err != nil {
			return err
		}
	} else {
	}
	return nil
}

// simpleexp -> Float | Integer | String | nil | true | false | ... | constructor | FUNCTION body | suffixedexp
func (p *Parser) simpleexp(tk *Token) error {
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
		return p.suffixedexp()
	}
	return nil
}

// primaryexp -> NAME | '(' expr ')'
func (p *Parser) primaryexp() error {
	tk, err := p.lex.Next()
	if err != nil {
		return err
	}
	switch tk.Kind {
	case TokenOpenParen:
		if _, err := p.lex.Next(); err != nil {
			return err
		} else if _, err := p.exp(); err != nil {
			return err
		} else if lastCh, err := p.lex.Next(); err != nil {
			return err
		} else if lastCh.Kind != TokenCloseParen {
			return fmt.Errorf("unmatched paren")
		}
		return nil // return expression
	case TokenIdentifier:
		return p.singlevar()
	default:
		return fmt.Errorf("unexpected symbol %v", tk.Kind)
	}
}

// suffixedexp -> primaryexp { '.' NAME | '[' exp ']' | ':' NAME funcargs | funcargs }
func (p *Parser) suffixedexp() error {
	if tk, err := p.lex.Next(); err != nil {
		return err
	} else if tk.Kind == TokenOpenParen {
		if _, err := p.lex.Next(); err != nil {
			return err
		} else if _, err := p.exp(); err != nil {
			return err
		} else if lastCh, err := p.lex.Next(); err != nil {
			return err
		} else if lastCh.Kind != TokenCloseParen {
			return fmt.Errorf("unmatched paren")
		}
		return nil // return expression
	} else if tk.Kind == TokenIdentifier {
		switch p.lex.Peek().Kind {
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

func (p *Parser) singlevar() error {
	return nil
}

func (p *Parser) expList() ([]*exprDesc, error) {
	exprs := []*exprDesc{}
	for {
		desc, err := p.exp()
		if err != nil {
			return nil, err
		}
		exprs = append(exprs, desc)
		if p.lex.Peek().Kind != TokenComma {
			break
		} else if _, err := p.lex.Next(); err != nil {
			return nil, err
		}
	}
	return exprs, nil
}

func (p *Parser) exp() (*exprDesc, error) {
	tk, err := p.lex.Next()
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
		return p.tableConstructor()
	case TokenFunction:
		return p.closure()
	case TokenNot, TokenMinus, TokenLength:
		return p.unary()
	default:
		return nil, nil
	}
}

func (p *Parser) closure() (*exprDesc, error) {
	return nil, nil
}

func (p *Parser) unary() (*exprDesc, error) {
	return nil, nil
}

func (p *Parser) tableConstructor() (*exprDesc, error) {
	return nil, nil
}

func min(x, y int) int {
	if x >= y {
		return x
	}
	return y
}
