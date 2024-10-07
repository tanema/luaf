package shine

import (
	"fmt"
	"io"
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
		rootfn: newFnProto(nil, []string{"_ENV"}, false),
		lex:    NewLexer(src),
	}
	fn := newFnProto(p.rootfn, []string{}, false)
	err := p.block(fn)
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

// This is used when the token has already been peeked but lets panic just in
// case something goes funky
func (p *Parser) mustnext(tt TokenType) *Token {
	tk, err := p.lex.Next()
	if err != nil {
		panic(err)
	} else if tk.Kind != tt {
		panic(fmt.Sprintf("expected %v but consumed %v", tt, tk.Kind))
	}
	return tk
}

// block -> statlist
func (p *Parser) block(fn *FuncProto) error {
	hasReturn, err := p.statList(fn)
	if err != nil {
		return err
	} else if !hasReturn {
		fn.code(iABCK(RETURN, 0, 0, false, 0, false))
	}
	return nil
}

// statlist -> { stat [';'] }
func (p *Parser) statList(fn *FuncProto) (bool, error) {
	for !p.blockFollow(true) {
		if p.peek().Kind == TokenReturn {
			return true, p.statement(fn) /* 'return' must be last statement */
		}
		if err := p.statement(fn); err != nil {
			return false, err
		}
	}
	return false, nil
}

// check if the next token indicates that we are still inside a block or not
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

// stat -> ';' | ifstat | whilestat | DO block END | forstat | repeatstat | funcstat | localstat | label | retstat | 'break' | 'goto' NAME | funccallstat | assignment
func (p *Parser) statement(fn *FuncProto) error {
	switch p.peek().Kind {
	case TokenSemiColon:
		return p.assertNext(TokenSemiColon)
	case TokenLocal:
		return p.localstat(fn)
	case TokenFunction:
		return p.funcstat(fn)
	case TokenIf: //self.if_stat()
	case TokenWhile: //self.while_stat()
	case TokenDo: //self.do_stat()
	case TokenFor: //self.for_stat()
	case TokenRepeat: //self.repeat_stat()
	case TokenDoubleColon: //self.label_stat()
	case TokenReturn: //self.ret_stat()
	case TokenBreak: //self.break_stat()
	case TokenGoto: //self.goto_stat()
	default:
		if expr, err := p.suffixedexp(fn); err != nil {
			return err
		} else if call, isCall := expr.(*exCall); isCall { // fncall
			call.discharge(fn, fn.sp)
		} else if tk := p.peek(); tk.Kind == TokenAssign || tk.Kind == TokenComma {
			return p.assignment(fn, expr)
		}
	}
	return nil
}

// localstat -> LOCAL [localfunc | localassign]
func (p *Parser) localstat(fn *FuncProto) error {
	p.mustnext(TokenLocal)
	if p.peek().Kind == TokenFunction {
		return p.localfunc(fn)
	}
	return p.localAssignment(fn)
}

// localfunc -> FUNCTION NAME funcbody
func (p *Parser) localfunc(fn *FuncProto) error {
	p.mustnext(TokenFunction)
	name, err := p.ident()
	if err != nil {
		return err
	}
	newFn, err := p.funcbody(fn)
	if err != nil {
		return err
	}
	fn.addFn(newFn)
	fn.code(iABx(CLOSURE, fn.sp, uint16(len(fn.FnTable)-1)))
	local := &exValue{local: true, name: name.name, address: uint8(len(fn.Locals))}
	local.assignTo(fn, fn.sp, false)
	fn.Locals = append(fn.Locals, Local{name: name.name})
	fn.sp++
	return p.assertNext(TokenEnd)
}

// funcstat -> FUNCTION funcname funcbody
func (p *Parser) funcstat(fn *FuncProto) error {
	p.mustnext(TokenFunction)
	name, err := p.funcname(fn)
	if err != nil {
		return err
	}
	newFn, err := p.funcbody(fn)
	if err != nil {
		return err
	}
	ifn := fn.sp
	cls := &exClosure{fn: uint16(len(fn.FnTable))}
	fn.addFn(newFn)
	p.discharge(fn, cls, ifn)
	name.(assignable).assignTo(fn, ifn, false)
	fn.sp++
	return p.assertNext(TokenEnd)
}

// funcname -> NAME {fieldsel} [':' NAME]
// fieldsel     -> ['.' | ':'] NAME
func (p *Parser) funcname(fn *FuncProto) (expression, error) {
	name, err := p.identName(fn)
	if err != nil {
		return nil, err
	}
	for {
		switch p.peek().Kind {
		case TokenPeriod:
			p.mustnext(TokenPeriod)
			itable := fn.sp
			p.discharge(fn, name, itable)

			key, err := p.identName(fn)
			if err != nil {
				return nil, err
			}
			ikey := fn.sp
			p.discharge(fn, key, ikey)
			name = &exIndex{local: true, table: itable, key: ikey, keyIsConst: false}
		case TokenColon:
			p.mustnext(TokenColon)
			itable := fn.sp
			p.discharge(fn, name, itable)

			key, err := p.identName(fn)
			if err != nil {
				return nil, err
			}
			ikey := fn.sp
			p.discharge(fn, key, ikey)
			return &exIndex{local: true, table: itable, key: ikey}, nil
		default:
			return name, nil
		}
	}
}

// funcbody -> parlist block END
func (p *Parser) funcbody(fn *FuncProto) (*FuncProto, error) {
	params, varargs, err := p.parlist()
	if err != nil {
		return nil, err
	}
	newFn := newFnProto(fn, params, varargs)
	return newFn, p.block(newFn)
}

// parlist -> '(' [ {NAME ','} (NAME | '...') ] ')'
func (p *Parser) parlist() ([]string, bool, error) {
	if err := p.assertNext(TokenOpenParen); err != nil {
		return nil, false, err
	}
	names := []string{}
	if p.peek().Kind == TokenCloseParen {
		return names, false, p.assertNext(TokenCloseParen)
	}
	for {
		name, err := p.ident()
		if err != nil {
			return nil, false, err
		}
		names = append(names, name.name)
		if p.peek().Kind != TokenComma {
			break
		}
		p.mustnext(TokenComma)
	}
	varargs := false
	if p.peek().Kind == TokenDots {
		p.mustnext(TokenDots)
		varargs = true
	}
	return names, varargs, p.assertNext(TokenCloseParen)
}

// localassign -> NAME attrib { ',' NAME attrib } ['=' explist]
func (p *Parser) localAssignment(fn *FuncProto) error {
	names := []Local{}
	for {
		lcl, err := p.identWithAttrib()
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
		fn.code(iAB(LOADNIL, fn.sp, uint8(len(names)-1)))
	}
	p.mustnext(TokenAssign)
	_, err := p.explist(fn, len(names))
	fn.sp += uint8(len(names))
	return err
}

// ident is a simple identifier that will be needed for later use as a var
func (p *Parser) ident() (*Local, error) {
	tk, err := p.lex.Next()
	if err != nil {
		return nil, err
	} else if tk.Kind != TokenIdentifier {
		return nil, fmt.Errorf("expected Name but found %v", tk.Kind)
	}
	return &Local{name: tk.StringVal}, nil
}

// NAME attrib
// attrib -> ['<' ('const' | 'close') '>']
func (p *Parser) identWithAttrib() (*Local, error) {
	local, err := p.ident()
	if err != nil {
		return nil, err
	}
	if p.peek().Kind == TokenLt {
		p.mustnext(TokenLt)
		if tk, err := p.lex.Next(); err != nil {
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

// funcargs -> '(' [ explist ] ')' | constructor | STRING
func (p *Parser) funcargs(fn *FuncProto) (int, error) {
	switch p.peek().Kind {
	case TokenOpenParen:
		p.mustnext(TokenOpenParen)
		if p.peek().Kind == TokenCloseParen {
			p.mustnext(TokenCloseParen)
			return 0, nil
		}
		nparams, err := p.explist(fn, 0)
		if err != nil {
			return -1, err
		}
		return nparams, p.assertNext(TokenCloseParen)
	case TokenOpenCurly:
		_, err := p.constructor(fn)
		return 1, err
	case TokenString:
		p.discharge(fn, &exConstant{index: fn.addConst(p.mustnext(TokenString).StringVal)}, fn.sp)
		return 1, nil
	default:
		return 0, fmt.Errorf("unexpected token type %v while evaluating function call", p.peek().Kind)
	}
}

// assignment -> suffixedexp { ',' suffixedexp } '=' explist
func (p *Parser) assignment(fn *FuncProto, first expression) error {
	names := []expression{first}
	for p.peek().Kind == TokenComma {
		p.mustnext(TokenComma)
		if expr, err := p.suffixedexp(fn); err != nil {
			return err
		} else {
			names = append(names, expr)
		}
	}
	if err := p.assertNext(TokenAssign); err != nil {
		return err
	}

	sp0 := fn.sp
	if _, err := p.explist(fn, len(names)); err != nil {
		return err
	}
	for i, name := range names {
		name.(assignable).assignTo(fn, sp0+uint8(i), false)
	}
	return nil
}

// expr -> (simpleexp | unop subexpr) { binop subexpr }
// where 'binop' is any binary operator with a priority higher than 'limit'
func (p *Parser) expr(fn *FuncProto, limit int) (expression, error) {
	var desc expression
	var err error
	if tk := p.peek(); tk.isUnary() {
		if err := p.next(); err != nil {
			return nil, err
		} else if desc, err = p.expr(fn, unaryPriority); err != nil {
			return nil, err
		}
		p.dischargeUnaryOp(fn, tk, fn.sp-1, fn.sp-1)
	} else {
		desc, err = p.simpleexp(fn)
		if err != nil {
			return nil, err
		}
	}
	op := p.peek()
	for op.isBinary() && binaryPriority[op.Kind][0] > limit {
		if err := p.next(); err != nil {
			return nil, err
		} else if desc, err = p.expr(fn, binaryPriority[op.Kind][1]); err != nil {
			return nil, err
		}
		p.dischargeBinop(fn, op, fn.sp-2, fn.sp-2, fn.sp-1)
		op = p.peek()
	}
	return desc, nil
}

func (p *Parser) discharge(fn *FuncProto, exp expression, dst uint8) {
	exp.discharge(fn, dst)
	fn.sp = dst + 1
}

// dischargeBinop will add the bytecode to execute the binop
func (p *Parser) dischargeBinop(fn *FuncProto, op *Token, dst, b, c uint8) {
	switch op.Kind {
	case TokenBitwiseOr:
		fn.code(iABC(BOR, dst, b, c))
	case TokenBitwiseNotOrXOr:
		fn.code(iABC(BXOR, dst, b, c))
	case TokenBitwiseAnd:
		fn.code(iABC(BAND, dst, b, c))
	case TokenShiftLeft:
		fn.code(iABC(SHL, dst, b, c))
	case TokenShiftRight:
		fn.code(iABC(SHR, dst, b, c))
	case TokenConcat:
		fn.code(iABC(CONCAT, dst, b, c))
	case TokenAdd:
		fn.code(iABC(ADD, dst, b, c))
	case TokenMinus:
		fn.code(iABC(SUB, dst, b, c))
	case TokenMultiply:
		fn.code(iABC(MUL, dst, b, c))
	case TokenModulo:
		fn.code(iABC(MOD, dst, b, c))
	case TokenDivide:
		fn.code(iABC(DIV, dst, b, c))
	case TokenFloorDivide:
		fn.code(iABC(IDIV, dst, b, c))
	case TokenExponent:
		fn.code(iABC(POW, dst, b, c))

	case TokenLt:
		fn.code(iABC(LT, 1, b, c))
	case TokenLe:
		fn.code(iABC(LE, 1, b, c))
	case TokenGt:
		fn.code(iABC(LT, 1, c, b))
	case TokenGe:
		fn.code(iABC(LE, 1, c, b))
	case TokenEq:
		fn.code(iABC(EQ, 1, c, b))
	case TokenNe:
		fn.code(iABC(EQ, 0, c, b))
	case TokenOr, TokenAnd:
		panic("or and not implemented yet")
	default:
		panic("unknown binop")
	}
	fn.sp = dst + 1
}

// dischargeUnaryOp will add the bytecode to execute the unary op
func (p *Parser) dischargeUnaryOp(fn *FuncProto, op *Token, dst, b uint8) {
	switch op.Kind {
	case TokenNot:
		fn.code(iAB(NOT, dst, b))
	case TokenLength:
		fn.code(iAB(LEN, dst, b))
	case TokenMinus:
		fn.code(iAB(UNM, dst, b))
	case TokenBitwiseNotOrXOr:
		fn.code(iAB(BNOT, dst, b))
	default:
		panic("unknown unary")
	}
	fn.sp = dst + 1
}

// simpleexp -> Float | Integer | String | nil | true | false | ... | constructor | FUNCTION body | suffixedexp
func (p *Parser) simpleexp(fn *FuncProto) (expression, error) {
	switch p.peek().Kind {
	case TokenFloat:
		return &exConstant{index: fn.addConst(p.mustnext(TokenFloat).FloatVal)}, nil
	case TokenInteger:
		return &exConstant{index: fn.addConst(p.mustnext(TokenInteger).IntVal)}, nil
	case TokenString:
		return &exConstant{index: fn.addConst(p.mustnext(TokenString).StringVal)}, nil
	case TokenNil:
		p.mustnext(TokenNil)
		return &exNil{}, nil
	case TokenTrue:
		p.mustnext(TokenTrue)
		return &exBool{value: true, skipnext: false}, nil
	case TokenFalse:
		p.mustnext(TokenFalse)
		return &exBool{value: false, skipnext: false}, nil
	case TokenOpenCurly:
		return p.constructor(fn)
	case TokenFunction:
		p.mustnext(TokenFunction)
		newFn, err := p.funcbody(fn)
		return &exClosure{fn: fn.addFn(newFn)}, err
	case TokenDots:
		return &exVarArgs{}, nil
	default:
		return p.suffixedexp(fn)
	}
}

// primaryexp -> NAME | '(' expr ')'
func (p *Parser) primaryexp(fn *FuncProto) (expression, error) {
	switch p.peek().Kind {
	case TokenOpenParen:
		p.mustnext(TokenOpenParen)
		desc, err := p.expr(fn, nonePriority)
		if err != nil {
			return nil, err
		}
		return desc, p.assertNext(TokenCloseParen)
	case TokenIdentifier:
		return p.name(fn, p.mustnext(TokenIdentifier).StringVal), nil
	default:
		return nil, fmt.Errorf("unexpected symbol %v", p.peek().Kind)
	}
}

// suffixedexp -> primaryexp { '.' NAME | '[' exp ']' | ':' NAME funcargs | funcargs }
// funccallstat -> suffixedexp funcargs
func (p *Parser) suffixedexp(fn *FuncProto) (expression, error) {
	expr, err := p.primaryexp(fn)
	if err != nil {
		return nil, err
	}
	for {
		switch p.peek().Kind {
		case TokenPeriod:
			p.mustnext(TokenPeriod)
			itable := fn.sp
			p.discharge(fn, expr, itable)
			key, err := p.ident()
			if err != nil {
				return nil, err
			}
			expr = &exIndex{local: true, table: itable, key: uint8(fn.addConst(key.name)), keyIsConst: true}
		case TokenOpenBracket:
			p.mustnext(TokenOpenBracket)
			itable := fn.sp
			p.discharge(fn, expr, itable)
			expr, err := p.expr(fn, nonePriority)
			if err != nil {
				return nil, err
			} else if err := p.assertNext(TokenCloseBracket); err != nil {
				return nil, err
			}
			ival := fn.sp
			p.discharge(fn, expr, ival)
			expr = &exIndex{local: true, table: itable, key: ival}
		case TokenColon:
			p.mustnext(TokenColon)
			p.discharge(fn, expr, fn.sp)
			key, err := p.ident()
			if err != nil {
				return nil, err
			}
			ifn := fn.sp
			fn.code(iABCK(SELF, fn.sp, fn.sp-1, false, uint8(fn.addConst(key.name)), true))
			fn.sp++
			nargs, err := p.funcargs(fn)
			if err != nil {
				return nil, err
			}
			expr = &exCall{fn: uint8(ifn), nargs: uint8(nargs + 1)}
		case TokenOpenParen, TokenString, TokenOpenCurly:
			ifn := fn.sp
			p.discharge(fn, expr, fn.sp)
			nargs, err := p.funcargs(fn)
			if err != nil {
				return nil, err
			}
			expr = &exCall{fn: uint8(ifn), nargs: uint8(nargs + 1)}
		default:
			return expr, nil
		}
	}
}

func (p *Parser) identName(fn *FuncProto) (expression, error) {
	ident, err := p.ident()
	if err != nil {
		return nil, err
	}
	return p.name(fn, ident.name), nil
}

// name is a reference to a variable that need resolution to have meaning
func (p *Parser) name(fn *FuncProto, name string) expression {
	if expr := p.resolveVar(fn, name); expr != nil {
		return expr
	}
	iname := fn.addConst(name)
	expr := p.name(fn, "_ENV")
	value, isValue := expr.(*exValue)
	if !isValue {
		panic("did not find _ENV, this should never happen")
	}
	return &exIndex{local: value.local, table: value.address, key: uint8(iname), keyIsConst: true}
}

// resolveVar will recursively look up the stack to find where the variable
// resides in the stack and then build the chain of upvars to have a referece
// to it.
func (p *Parser) resolveVar(fn *FuncProto, name string) expression {
	if fn == nil {
		return nil
	} else if idx, ok := search(fn.Locals, name, findLocal); ok {
		return &exValue{local: true, name: name, address: uint8(idx)}
	} else if idx, ok := search(fn.UpIndexes, name, findUpindex); ok {
		return &exValue{local: false, name: name, address: uint8(idx)}
	} else if expr := p.resolveVar(fn.prev, name); expr != nil {
		if value, isValue := expr.(*exValue); !isValue {
		} else if value.local {
			fn.UpIndexes = append(fn.UpIndexes, UpIndex{fromStack: true, name: name, index: uint(value.address)})
		} else {
			fn.UpIndexes = append(fn.UpIndexes, UpIndex{fromStack: false, name: name, index: uint(value.address)})
		}
		return &exValue{local: false, name: name, address: uint8(len(fn.UpIndexes) - 1)}
	}
	return nil
}

// explist -> expr { ',' expr }
func (p *Parser) explist(fn *FuncProto, want int) (int, error) {
	numExprs := 0
	for {
		if _, err := p.expr(fn, nonePriority); err != nil {
			return -1, err
		}
		numExprs++
		if p.peek().Kind != TokenComma {
			break
		}
		p.mustnext(TokenComma)
	}
	if want > 0 {
		if numExprs > want { // discarg extra values
			fn.sp -= uint8(numExprs) - uint8(want)
		} else if numExprs < want { // pad stack with nil
			fn.code(iABx(LOADNIL, fn.sp, uint16(want)-uint16(numExprs)))
		}
		return want, nil
	}
	return numExprs, nil
}

// constructor -> '{' [ field { sep field } [sep] ] '}'
// sep         -> ',' | ';'
// field -> NAME = exp | '['exp']' = exp | exp
func (p *Parser) constructor(fn *FuncProto) (expression, error) {
	p.mustnext(TokenOpenCurly)
	itable := fn.sp
	tablecode := fn.code(iAB(NEWTABLE, fn.sp, 0))
	fn.sp++
	numvals, numfields := 0, 0
	for {
		switch p.peek().Kind {
		case TokenIdentifier:
			key, err := p.ident()
			if err != nil {
				return nil, err
			} else if err := p.assertNext(TokenAssign); err != nil {
				return nil, err
			}
			assignExpr := &exIndex{
				local:      true,
				table:      itable,
				key:        uint8(fn.addConst(key.name)),
				keyIsConst: true,
			}
			desc, err := p.expr(fn, 0)
			if err != nil {
				return nil, err
			}
			ival := fn.sp
			p.discharge(fn, desc, ival)
			assignExpr.assignTo(fn, ival, false)
			numfields++
		case TokenOpenBracket:
			p.mustnext(TokenOpenBracket)
			keydesc, err := p.expr(fn, 0)
			if err != nil {
				return nil, err
			} else if err := p.assertNext(TokenCloseBracket); err != nil {
				return nil, err
			} else if err := p.assertNext(TokenAssign); err != nil {
				return nil, err
			}
			ikey := fn.sp
			p.discharge(fn, keydesc, ikey)
			assignExpr := &exIndex{
				local:      true,
				table:      itable,
				key:        ikey,
				keyIsConst: false,
			}
			valdesc, err := p.expr(fn, 0)
			if err != nil {
				return nil, err
			}
			ival := fn.sp
			p.discharge(fn, valdesc, ival)
			assignExpr.assignTo(fn, ival, false)
			numfields++
		default:
			desc, err := p.expr(fn, 0)
			if err != nil {
				return nil, err
			}
			p.discharge(fn, desc, fn.sp)
			numvals++
		}
		if tk := p.peek(); tk.Kind == TokenComma || tk.Kind == TokenSemiColon {
			p.next()
		} else {
			break
		}
	}

	if numvals > 0 {
		fn.code(iAB(SETLIST, itable, uint8(numvals+1)))
	}
	fn.sp -= uint8(numvals)
	fn.ByteCodes[tablecode] = iABC(NEWTABLE, itable, uint8(numvals), uint8(numfields))
	return &exValue{local: true, address: uint8(itable)}, p.assertNext(TokenCloseCurly)
}
