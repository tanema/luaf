package shine

import (
	"fmt"
	"io"
	"slices"
)

type (
	Local struct {
		name      string
		attrConst bool
		attrClose bool
	}
	Parser struct {
		rootfn *FuncProto
		lex    *Lexer
	}
)

func Parse(filename string, src io.Reader) (*FuncProto, error) {
	p := &Parser{
		rootfn: newFnProto(nil, "env", []string{"_ENV"}, false),
		lex:    NewLexer(src),
	}
	fn, err := p.block(newFnProto(p.rootfn, "main", []string{}, false))
	if err == io.EOF {
		err = nil
	}
	return fn, err
}

func (p *Parser) peek() *Token {
	return p.lex.Peek()
}

func (p *Parser) next() error {
	_, err := p.lex.Next()
	return err
}

func (p *Parser) assertNext(tt TokenType) error {
	tk, err := p.lex.Next()
	if err != nil {
		return err
	} else if tk.Kind != tt {
		return fmt.Errorf("expected %v but consumed %v", tt, tk.Kind)
	}
	return nil
}

func (p *Parser) block(fn *FuncProto) (*FuncProto, error) {
	err := p.statList(fn)
	fn.code(iABC(RETURN, 0, 0, false, 0, false))
	return fn, err
}

func (p *Parser) statList(fn *FuncProto) error {
	for !p.blockFollow(true) {
		if p.peek().Kind == TokenReturn {
			return p.statement(fn) /* 'return' must be last statement */
		}
		if err := p.statement(fn); err != nil {
			return err
		}
	}
	return nil
}

func (p *Parser) blockFollow(withuntil bool) bool {
	switch p.peek().Kind {
	case TokenElse, TokenElseif, TokenEnd, TokenEOS:
		return true
	case TokenUntil:
		return withuntil
	default:
		return false
	}
}

func (p *Parser) statement(fn *FuncProto) error {
	switch p.peek().Kind {
	case TokenSemiColon:
		return p.assertNext(TokenSemiColon)
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
		} else if tk := p.peek(); tk.Kind == TokenAssign || tk.Kind == TokenComma {
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
	if err := p.assertNext(TokenLocal); err != nil {
		return err
	} else if p.peek().Kind == TokenFunction {
		return p.localFunc(fn)
	}
	return p.localAssignment(fn)
}

func (p *Parser) localFunc(fn *FuncProto) error {
	if err := p.assertNext(TokenFunction); err != nil {
		return err
	}
	name, err := p.name()
	if err != nil {
		return err
	}
	params, varargs, err := p.parlist()
	if err != nil {
		return err
	}

	newFnProto(fn, name.name, params, varargs)
	// block
	return nil
}

func (p *Parser) parlist() ([]string, bool, error) {
	if err := p.assertNext(TokenOpenParen); err != nil {
		return nil, false, err
	}
	names := []string{}
	for {
		name, err := p.name()
		if err != nil {
			return nil, false, err
		}
		names = append(names, name.name)
		if p.peek().Kind != TokenComma {
			break
		} else if err := p.assertNext(TokenComma); err != nil {
			return nil, false, err
		}
	}
	varargs := false
	if p.peek().Kind == TokenDots {
		varargs = true
		if err := p.assertNext(TokenDots); err != nil {
			return nil, false, err
		}
	}
	return names, varargs, p.assertNext(TokenCloseParen)
}

func (p *Parser) localAssignment(fn *FuncProto) error {
	names := []Local{}
	for {
		lcl, err := p.nameAttrib()
		if err != nil {
			return err
		}
		names = append(names, *lcl)
		if p.peek().Kind != TokenComma {
			break
		} else if err := p.next(); err != nil {
			return err
		}
	}
	fn.Locals = append(fn.Locals, names...)
	if p.peek().Kind != TokenAssign {
		fn.code(iABC(LOADNIL, fn.sp, uint8(len(names)-1), false, 0, false))
	} else if err := p.assertNext(TokenAssign); err != nil {
		return err
	} else if err := p.exprList(fn, len(names)); err != nil {
		return err
	}
	fn.sp += uint8(len(names))
	return nil
}

func (p *Parser) name() (*Local, error) {
	tk, err := p.lex.Next()
	if err != nil {
		return nil, err
	} else if tk.Kind != TokenIdentifier {
		return nil, fmt.Errorf("expected Name but found %v", tk.Kind)
	}
	return &Local{name: tk.StringVal}, nil
}

func (p *Parser) nameAttrib() (*Local, error) {
	local, err := p.name()
	if err != nil {
		return nil, err
	}
	if p.peek().Kind == TokenLt {
		if err := p.assertNext(TokenLt); err != nil {
			return nil, err
		} else if tk, err := p.lex.Next(); err != nil {
			return nil, err
		} else if tk.Kind != TokenIdentifier {
			return nil, fmt.Errorf("expected attrib but found %v", tk.Kind)
		} else if tk.StringVal == "const" {
			local.attrConst = true
		} else if tk.StringVal == "close" {
			local.attrClose = true
		} else {
			return nil, fmt.Errorf("unknown local attribute %v", tk.StringVal)
		}
		if err := p.assertNext(TokenGt); err != nil {
			return nil, err
		}
	}
	return local, nil
}

// assignment -> suffixedexp { ',' suffixedexp } '=' explist
func (p *Parser) assignment(fn *FuncProto, first *exprDesc) error {
	names := []*exprDesc{first}
	for p.peek().Kind == TokenComma {
		if err := p.assertNext(TokenComma); err != nil {
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
		return fmt.Errorf("expected '=' but found %v", p.peek().Kind)
	}

	sp0 := fn.sp
	if err := p.exprList(fn, len(names)); err != nil {
		return err
	}
	for i, name := range names {
		switch name.kind {
		case localExpr:
			fn.code(iABC(MOVE, uint8(name.a), sp0+uint8(i), false, 0, false))
		case upvalueExpr:
			fn.code(iABC(SETUPVAL, uint8(name.a), sp0+uint8(i), false, 0, false))
		case indexExpr:
			fn.code(iABC(SETTABLE, uint8(name.a), uint8(name.b), false, sp0+uint8(i), false))
		case indexUpFieldExpr:
			fn.code(iABC(SETTABUP, uint8(name.a), uint8(name.b), true, sp0+uint8(i), false))
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
	if tk := p.peek(); tk.isUnary() {
		if err = p.next(); err != nil {
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
	op = p.peek()
	for op.isBinary() && binaryPriority[op.Kind][0] > limit {
		if err = p.next(); err != nil {
			return err
		} else if err := p.expr(fn, binaryPriority[op.Kind][1]); err != nil {
			return err
		}
		p.dischargeBinop(fn, op, fn.sp-2, fn.sp-2, fn.sp-1)
		op = p.peek()
	}
	return nil
}

func (p *Parser) discharge(fn *FuncProto, dst uint8, exp *exprDesc) {
	var code Bytecode
	switch exp.kind {
	case constExpr:
		code = iABx(LOADK, dst, exp.b)
	case nilExpr:
		code = iABx(LOADNIL, dst, 1)
	case boolExpr:
		code = iABC(LOADBOOL, dst, uint8(exp.b), false, 0, false)
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
		code = iABC(EQ, 1, c, false, b, false)
	case TokenNe:
		code = iABC(EQ, 0, c, false, b, false)
	case TokenLt:
		code = iABC(LT, dst, b, false, c, false)
	case TokenLe:
		code = iABC(LE, dst, b, false, c, false)
	case TokenGt:
		code = iABC(LT, dst, c, false, b, false)
	case TokenGe:
		code = iABC(LE, dst, c, false, b, false)
	case TokenBitwiseOr:
		code = iABC(BOR, dst, b, false, c, false)
	case TokenBitwiseNotOrXOr:
		code = iABC(BXOR, dst, b, false, c, false)
	case TokenBitwiseAnd:
		code = iABC(BAND, dst, b, false, c, false)
	case TokenShiftLeft:
		code = iABC(SHL, dst, b, false, c, false)
	case TokenShiftRight:
		code = iABC(SHR, dst, b, false, c, false)
	case TokenConcat:
		code = iABC(CONCAT, dst, b, false, c, false)
	case TokenAdd:
		code = iABC(ADD, dst, b, false, c, false)
	case TokenMinus:
		code = iABC(SUB, dst, b, false, c, false)
	case TokenMultiply:
		code = iABC(MUL, dst, b, false, c, false)
	case TokenModulo:
		code = iABC(MOD, dst, b, false, c, false)
	case TokenDivide:
		code = iABC(DIV, dst, b, false, c, false)
	case TokenFloorDivide:
		code = iABC(IDIV, dst, b, false, c, false)
	case TokenExponent:
		code = iABC(POW, dst, b, false, c, false)
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
		if err := p.assertNext(TokenOpenParen); err != nil {
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
	if expr := p.singlevar(fn, "_ENV"); expr.kind == localExpr {
		return &exprDesc{kind: indexExpr, a: expr.a, b: iname, bConst: true}
	} else if expr.kind == upvalueExpr {
		return &exprDesc{kind: indexUpFieldExpr, a: expr.a, b: iname, bConst: true}
	}
	panic(fmt.Sprintf("this should never happen cannot find _ENV while locating %v", name))
}

func (p *Parser) resolveVar(fn *FuncProto, name string) *exprDesc {
	if fn == nil {
		return nil
	} else if idx, ok := slices.BinarySearchFunc(fn.Locals, name, findLocal); ok {
		return &exprDesc{kind: localExpr, a: uint16(idx)}
	} else if idx, ok := slices.BinarySearchFunc(fn.UpIndexes, name, findUpindex); ok {
		return &exprDesc{kind: upvalueExpr, a: uint16(idx)}
	} else if expr := p.resolveVar(fn.prev, name); expr != nil {
		if expr.kind == localExpr {
			fn.UpIndexes = append(fn.UpIndexes, UpIndex{fromStack: true, name: name, index: uint(expr.a)})
		} else {
			fn.UpIndexes = append(fn.UpIndexes, UpIndex{fromStack: false, name: name, index: uint(expr.a)})
		}
		return &exprDesc{kind: upvalueExpr, name: name, a: uint16(len(fn.UpIndexes) - 1)}
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
		if p.peek().Kind != TokenComma {
			break
		} else if err := p.assertNext(TokenComma); err != nil {
			return err
		}
	}
	if numExprs > want { // discarg extra values
		fn.sp -= uint8(numExprs) - uint8(want)
	} else if numExprs < want { // pad stack with nil
		fn.code(iABx(LOADNIL, fn.sp, uint16(want)-uint16(numExprs)))
	}
	return nil
}
