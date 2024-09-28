package shine

import (
	"fmt"
	"io"
	"slices"
)

type Parser struct {
	lex *Lexer
}

func Parse(filename string, src io.Reader) (*FuncProto, error) {
	p := &Parser{
		lex: NewLexer(src),
	}
	fn, err := p.block(newFnProto(nil, []string{"_ENV"}))
	if err == io.EOF {
		err = nil
	}
	return fn, err
}

func (p *Parser) block(fn *FuncProto) (*FuncProto, error) {
	newfn := newFnProto(fn, []string{})
	err := p.statList(newfn)
	newfn.code(IABC(RETURN, 0, 0, 0))
	return newfn, err
}

func (p *Parser) statList(fn *FuncProto) error {
	for !p.blockFollow(true) {
		if p.lex.Peek().Kind == TokenReturn {
			return p.statement(fn) /* 'return' must be last statement */
		}
		if err := p.statement(fn); err != nil {
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

func (p *Parser) statement(fn *FuncProto) error {
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
		return p.localstat(fn)
	case TokenDoubleColon: //self.label_stat()
	case TokenReturn: //self.ret_stat()
	case TokenBreak: //self.break_stat()
	case TokenGoto: //self.goto_stat()
	default:
		expr, err := p.suffixedexp(fn)
		if err != nil {
			return err
		} else if tk := p.lex.Peek(); tk.Kind == TokenAssign || tk.Kind == TokenComma {
			if err := p.assignment(fn, expr); err != nil {
				return err
			}
		} else {
			fmt.Println("function call")
		}
	}
	return nil
}

func (p *Parser) localstat(fn *FuncProto) error {
	if _, err := p.lex.Next(); err != nil {
		return err
	}
	if p.lex.Peek().Kind == TokenFunction {
	}
	return nil
}

// assignment -> suffixedexp { ',' suffixedexp } '=' explist
func (p *Parser) assignment(fn *FuncProto, first *exprDesc) error {
	names := []*exprDesc{first}
	for p.lex.Peek().Kind == TokenComma {
		if _, err := p.lex.Next(); err != nil {
			return err
		} else if expr, err := p.suffixedexp(fn); err != nil {
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

	sp0 := fn.sp
	if err := p.exprList(fn, len(names)); err != nil {
		return err
	}
	for i, name := range names {
		switch name.kind {
		case localExpr:
			fn.code(IABC(MOVE, uint8(name.a), sp0+uint8(i), 0))
		case upvalueExpr:
			fn.code(IABC(SETUPVAL, uint8(name.a), sp0+uint8(i), 0))
		case indexExpr:
			fn.code(IABC(SETTABLE, uint8(name.a), uint8(name.b), sp0+uint8(i)))
		case indexUpFieldExpr:
			fn.code(IABC(SETTABUP, uint8(name.a), uint8(name.b), sp0+uint8(i)))
		default:
			return fmt.Errorf("assignment to %v invalid", name.kind)
		}
	}
	return nil
}

// expr -> (simpleexp | unop subexpr) { binop subexpr }
// where 'binop' is any binary operator with a priority higher than 'limit'
func (p *Parser) expr(fn *FuncProto, limit int) error {
	var err error
	var op *Token
	if tk := p.lex.Peek(); tk.isUnary() {
		if _, err = p.lex.Next(); err != nil {
			return err
		} else if err = p.expr(fn, unaryPriority); err != nil {
			return err
		}
	} else {
		expr, err := p.simpleexp(fn)
		if err != nil {
			return err
		}
		p.discharge(fn, fn.sp, expr)
	}
	if err != nil {
		return err
	}
	op = p.lex.Peek()
	for op.isBinary() && binaryPriority[op.Kind][0] > limit {
		if _, err = p.lex.Next(); err != nil {
			return err
		}
		err := p.expr(fn, binaryPriority[op.Kind][1])
		if err != nil {
			return err
		}
		p.dischargeBinop(fn, op, fn.sp-2, fn.sp-2, fn.sp-1)
		op = p.lex.Peek()
	}
	return nil
}

func (p *Parser) discharge(fn *FuncProto, dst uint8, exp *exprDesc) {
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

func (p *Parser) dischargeBinop(fn *FuncProto, op *Token, dst, b, c uint8) {
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
	fn.code(code)
	fn.sp = dst + 1
}

// simpleexp -> Float | Integer | String | nil | true | false | ... | constructor | FUNCTION body | suffixedexp
func (p *Parser) simpleexp(fn *FuncProto) (*exprDesc, error) {
	tk, err := p.lex.Next()
	if err != nil {
		return nil, err
	}
	switch tk.Kind {
	case TokenFloat:
		value := &Float{val: tk.FloatVal}
		idx := fn.addConst(value)
		return &exprDesc{kind: constExpr, b: idx}, nil
	case TokenInteger:
		value := &Integer{val: tk.IntVal}
		idx := fn.addConst(value)
		return &exprDesc{kind: constExpr, b: idx}, nil
	case TokenString:
		value := &String{val: tk.StringVal}
		idx := fn.addConst(value)
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
		return p.suffixedexp(fn)
	}
	return nil, nil
}

// primaryexp -> NAME | '(' expr ')'
func (p *Parser) primaryexp(fn *FuncProto) (*exprDesc, error) {
	tk, err := p.lex.Next()
	if err != nil {
		return nil, err
	}
	switch tk.Kind {
	case TokenOpenParen:
		if _, err := p.lex.Next(); err != nil {
			return nil, err
		} else if err := p.expr(fn, nonePriority); err != nil {
			return nil, err
		} else if lastCh, err := p.lex.Next(); err != nil {
			return nil, err
		} else if lastCh.Kind != TokenCloseParen {
			return nil, fmt.Errorf("unmatched paren")
		}
		return nil, nil // return expression
	case TokenIdentifier:
		return p.singlevar(fn, tk.StringVal), nil
	default:
		return nil, fmt.Errorf("unexpected symbol %v", tk.Kind)
	}
}

// suffixedexp -> primaryexp { '.' NAME | '[' exp ']' | ':' NAME funcargs | funcargs }
func (p *Parser) suffixedexp(fn *FuncProto) (*exprDesc, error) {
	return p.primaryexp(fn)
}

func (p *Parser) singlevar(fn *FuncProto, name string) *exprDesc {
	if expr := p.resolveVar(fn, name); expr != nil {
		return expr
	}
	iname := fn.addConst(&String{val: name})
	expr := p.singlevar(fn, "_ENV")
	if expr.kind == localExpr {
		return &exprDesc{kind: indexExpr, a: expr.a, b: iname}
	} else if expr.kind == upvalueExpr {
		return &exprDesc{kind: indexUpFieldExpr, a: expr.a, b: iname}
	}
	panic(fmt.Sprintf("cannot find _ENV while locating %v", name))
}

func (p *Parser) resolveVar(fn *FuncProto, name string) *exprDesc {
	if fn == nil {
		return nil
	} else if idx, ok := slices.BinarySearch(fn.Locals, name); ok {
		return &exprDesc{kind: localExpr, a: uint16(idx)}
	} else if idx, ok := fn.UpIndexes[name]; ok {
		return &exprDesc{kind: upvalueExpr, a: uint16(idx.Index)}
	} else if expr := p.resolveVar(fn.prev, name); expr != nil {
		if expr.kind == localExpr {
			fn.prev.UpIndexes[name] = UpIndex{Local: true, Index: uint(expr.a)}
		}
		fn.UpIndexes[name] = UpIndex{Local: false}
		return &exprDesc{kind: upvalueExpr, name: name}
	}
	return nil
}

func (p *Parser) exprList(fn *FuncProto, want int) error {
	numExprs := 0
	for {
		if err := p.expr(fn, nonePriority); err != nil {
			return err
		}
		numExprs++
		if p.lex.Peek().Kind != TokenComma {
			break
		} else if _, err := p.lex.Next(); err != nil {
			return err
		}
	}
	if numExprs > want { // discarg extra values
		fn.sp -= uint8(numExprs) - uint8(want)
	} else if numExprs < want { // pad stack with nil
		fn.code(IABx(LOADNIL, fn.sp, uint16(want)-uint16(numExprs)))
	}
	return nil
}
