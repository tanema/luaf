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

func Parse(filename string, src io.Reader) (*FuncProto, error) {
	p := &Parser{lex: NewLexer(src)}
	fn, err := p.block()
	if err == io.EOF {
		err = nil
	}
	return fn, err
}

func (p *Parser) block() (*FuncProto, error) {
	fn := newFnProto([]string{"_ENV"})
	p.blocks = append(p.blocks, fn)
	err := p.statList()
	p.blocks = p.blocks[:len(p.blocks)-1]
	return fn, err
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
	case TokenLocal: // return p.local()
	case TokenDoubleColon: //self.label_stat()
	case TokenReturn: //self.ret_stat()
	case TokenBreak: //self.break_stat()
	case TokenGoto: //self.goto_stat()
	default:
		expr, err := p.suffixedexp()
		if err != nil {
			return err
		} else if tk := p.lex.Peek(); tk.Kind == TokenAssign || tk.Kind == TokenComma {
			if err := p.assignment(expr); err != nil {
				return err
			}
		} else {
			fmt.Println("function call")
		}
	}
	return nil
}

// assignment -> suffixedexp { ',' suffixedexp } '=' explist
func (p *Parser) assignment(first *exprDesc) error {
	names := []*exprDesc{first}
	for p.lex.Peek().Kind == TokenComma {
		if _, err := p.lex.Next(); err != nil {
			return err
		} else if expr, err := p.suffixedexp(); err != nil {
			return err
		} else {
			names = append(names, expr)
		}
	}
	if tk, err := p.lex.Next(); err != nil {
		return err
	} else if tk.Kind != TokenAssign {
		return fmt.Errorf("expected '=' but found %v", p.lex.Peek().Kind)
	}

	fn := p.fn()
	sp0 := fn.sp
	if err := p.exprList(len(names)); err != nil {
		return err
	}
	fmt.Println("sp0", sp0, "sp", fn.sp-sp0, "names", len(names))
	return nil
}

// expr -> (simpleexp | unop subexpr) { binop subexpr }
// where 'binop' is any binary operator with a priority higher than 'limit'
func (p *Parser) expr(limit int) error {
	fn := p.fn()
	var err error
	var op *Token
	if tk := p.lex.Peek(); tk.isUnary() {
		if _, err = p.lex.Next(); err != nil {
			return err
		} else if err = p.expr(unaryPriority); err != nil {
			return err
		}
	} else {
		expr, err := p.simpleexp()
		if err != nil {
			return err
		}
		p.discharge(fn.sp, expr)
	}
	if err != nil {
		return err
	}
	op = p.lex.Peek()
	for op.isBinary() && binaryPriority[op.Kind][0] > limit {
		if _, err = p.lex.Next(); err != nil {
			return err
		}
		err := p.expr(binaryPriority[op.Kind][1])
		if err != nil {
			return err
		}
		p.dischargeBinop(op, fn.sp-2, fn.sp-2, fn.sp-1)
		op = p.lex.Peek()
	}
	return nil
}

func (p *Parser) discharge(dst uint8, exp *exprDesc) {
	fn := p.fn()
	var code Bytecode
	switch exp.kind {
	case constExpr:
		code = IABx(LOADK, dst, exp.b)
	case nilExpr:
		code = IABx(LOADNIL, dst, 1)
	case boolExpr:
		code = IABC(LOADBOOL, dst, uint8(exp.b), 0)
	default:
		panic("unexpected exprdesc kind")
	}
	fn.code(code)
	fn.sp++
}

func (p *Parser) dischargeBinop(op *Token, dst, b, c uint8) {
	var code Bytecode
	switch op.Kind {
	case TokenOr, TokenAnd:
		return
	case TokenEq:
		code = IABC(EQ, 1, c, b)
	case TokenNe:
		code = IABC(EQ, 0, c, b)
	case TokenLt:
		code = IABC(LT, dst, b, c)
	case TokenLe:
		code = IABC(LE, dst, b, c)
	case TokenGt:
		code = IABC(LT, dst, c, b)
	case TokenGe:
		code = IABC(LE, dst, c, b)
	case TokenBitwiseOr:
		code = IABC(BOR, dst, b, c)
	case TokenBitwiseNotOrXOr:
		code = IABC(BXOR, dst, b, c)
	case TokenBitwiseAnd:
		code = IABC(BAND, dst, b, c)
	case TokenShiftLeft:
		code = IABC(SHL, dst, b, c)
	case TokenShiftRight:
		code = IABC(SHR, dst, b, c)
	case TokenConcat:
		code = IABC(CONCAT, dst, b, c)
	case TokenAdd:
		code = IABC(ADD, dst, b, c)
	case TokenMinus:
		code = IABC(SUB, dst, b, c)
	case TokenMultiply:
		code = IABC(MUL, dst, b, c)
	case TokenModulo:
		code = IABC(MOD, dst, b, c)
	case TokenDivide:
		code = IABC(DIV, dst, b, c)
	case TokenFloorDivide:
		code = IABC(IDIV, dst, b, c)
	case TokenExponent:
		code = IABC(POW, dst, b, c)
	default:
		return
	}
	fn := p.fn()
	fn.code(code)
	fn.sp = dst + 1
}

// simpleexp -> Float | Integer | String | nil | true | false | ... | constructor | FUNCTION body | suffixedexp
func (p *Parser) simpleexp() (*exprDesc, error) {
	tk, err := p.lex.Next()
	if err != nil {
		return nil, err
	}
	switch tk.Kind {
	case TokenFloat:
		value := &Float{val: tk.FloatVal}
		idx := p.fn().addConst(value)
		return &exprDesc{kind: constExpr, b: idx}, nil
	case TokenInteger:
		value := &Integer{val: tk.IntVal}
		idx := p.fn().addConst(value)
		return &exprDesc{kind: constExpr, b: idx}, nil
	case TokenString:
		value := &String{val: tk.StringVal}
		idx := p.fn().addConst(value)
		return &exprDesc{kind: constExpr, b: idx}, nil
	case TokenNil:
		return &exprDesc{kind: nilExpr}, nil
	case TokenTrue:
		return &exprDesc{kind: boolExpr, b: 1}, nil
	case TokenFalse:
		return &exprDesc{kind: boolExpr, b: 0}, nil
	case TokenDots: // varargs
	case TokenOpenCurly: //table constructor
	case TokenFunction: // function
	default:
		return p.suffixedexp()
	}
	return nil, nil
}

func (p *Parser) fn() *FuncProto {
	return p.blocks[len(p.blocks)-1]
}

// primaryexp -> NAME | '(' expr ')'
func (p *Parser) primaryexp() (*exprDesc, error) {
	tk, err := p.lex.Next()
	if err != nil {
		return nil, err
	}
	switch tk.Kind {
	case TokenOpenParen:
		if _, err := p.lex.Next(); err != nil {
			return nil, err
		} else if err := p.expr(nonePriority); err != nil {
			return nil, err
		} else if lastCh, err := p.lex.Next(); err != nil {
			return nil, err
		} else if lastCh.Kind != TokenCloseParen {
			return nil, fmt.Errorf("unmatched paren")
		}
		return nil, nil // return expression
	case TokenIdentifier:
		return p.singlevar(tk.StringVal)
	default:
		return nil, fmt.Errorf("unexpected symbol %v", tk.Kind)
	}
}

// suffixedexp -> primaryexp { '.' NAME | '[' exp ']' | ':' NAME funcargs | funcargs }
func (p *Parser) suffixedexp() (*exprDesc, error) {
	return p.primaryexp()
}

func (p *Parser) singlevar(name string) (*exprDesc, error) {
	fn := p.blocks[len(p.blocks)-1]
	if expr := fn.findVar(name); expr != nil {
		return expr, nil
	}
	for i := len(p.blocks) - 2; i >= 0; i-- {
		if expr := p.blocks[i].findVar(name); expr != nil {
			return expr, nil
		}
	}
	iname := fn.addConst(&String{val: name})
	expr, err := p.singlevar("_ENV")
	if err != nil {
		return nil, err
	} else if expr.kind == localExpr {
		return &exprDesc{kind: indexFieldExpr, a: expr.a, b: iname}, nil
	} else if expr.kind == upvalueExpr {
		return &exprDesc{kind: indexUpFieldExpr, a: expr.a, b: iname}, nil
	}
	return nil, fmt.Errorf("cannot find _ENV")
}

func (p *Parser) exprList(want int) error {
	numExprs := 0
	for {
		if err := p.expr(nonePriority); err != nil {
			return err
		}
		numExprs++
		if p.lex.Peek().Kind != TokenComma {
			break
		} else if _, err := p.lex.Next(); err != nil {
			return err
		}
	}
	fn := p.fn()
	if numExprs > want { // discarg extra values
		fn.sp -= uint8(numExprs) - uint8(want)
	} else if numExprs < want { // pad stack with nil
		fn.code(IABx(LOADNIL, fn.sp, uint16(want)-uint16(numExprs)))
	}
	return nil
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
