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
	fn := newFnProto(p.rootfn, "main", []string{}, false)
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
		fn.code(iABC(RETURN, 0, 0, false, 0, false))
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
		expr, err := p.suffixedexp(fn)
		if err != nil {
			return err
		} else if tk := p.peek(); tk.Kind == TokenAssign || tk.Kind == TokenComma {
			return p.assignment(fn, expr)
		}
		return p.funccallstat(fn, expr)
	}
	return nil
}

// localstat -> LOCAL [localfunc | localassign]
func (p *Parser) localstat(fn *FuncProto) error {
	p.mustnext(TokenLocal)
	if p.peek().Kind == TokenFunction {
		return p.localFunc(fn)
	}
	return p.localAssignment(fn)
}

// localfunc -> FUNCTION NAME funcbody
func (p *Parser) localFunc(fn *FuncProto) error {
	p.mustnext(TokenFunction)
	name, err := p.ident()
	if err != nil {
		return err
	}
	newFn, err := p.funcbody(fn, name.name)
	if err != nil {
		return err
	}
	fn.Locals = append(fn.Locals, Local{name: name.name})
	fn.code(iABx(LOADK, fn.sp, fn.addFn(newFn)))
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
	newFn, err := p.funcbody(fn, name.name)
	if err != nil {
		return err
	}
	fn.code(iABx(LOADK, fn.sp, fn.addConst(&Function{val: newFn})))
	fn.sp++
	return p.assertNext(TokenEnd)
}

// funcname -> NAME {fieldsel} [':' NAME]
// fieldsel     -> ['.' | ':'] NAME
func (p *Parser) funcname(fn *FuncProto) (*exprDesc, error) {
	panic("funcname not impl yet")
	return nil, nil
}

// funcbody -> parlist block END
func (p *Parser) funcbody(fn *FuncProto, name string) (*FuncProto, error) {
	params, varargs, err := p.parlist()
	if err != nil {
		return nil, err
	}
	newFn := newFnProto(fn, name, params, varargs)
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
		fn.code(iABC(LOADNIL, fn.sp, uint8(len(names)-1), false, 0, false))
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

// funccallstat -> suffixedexp funcargs
func (p *Parser) funccallstat(fn *FuncProto, funcDef *exprDesc) error {
	ifn := fn.sp
	p.discharge(fn, funcDef)
	nargs, err := p.funcargs(fn)
	if err != nil {
		return err
	}
	fn.code(iABC(CALL, ifn, uint8(nargs+1), false, 0, false))
	return nil
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
		tableexpr, err := p.constructor(fn)
		if err != nil {
			return -1, err
		}
		p.discharge(fn, tableexpr)
		return 1, nil
	case TokenString:
		tk, err := p.lex.Next()
		if err != nil {
			return -1, err
		}
		str := &String{val: tk.StringVal}
		p.discharge(fn, &exprDesc{kind: constExpr, b: fn.addConst(str)})
		return 1, nil
	default:
		return 0, fmt.Errorf("unexpected token type while evaluating function call")
	}
}

// assignment -> suffixedexp { ',' suffixedexp } '=' explist
func (p *Parser) assignment(fn *FuncProto, first *exprDesc) error {
	names := []*exprDesc{first}
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
	if tk := p.peek(); tk.isUnary() {
		if err := p.next(); err != nil {
			return err
		} else if err := p.expr(fn, unaryPriority); err != nil {
			return err
		}
	} else {
		expr, err := p.simpleexp(fn)
		if err != nil {
			return err
		}
		p.discharge(fn, expr)
	}
	op := p.peek()
	for op.isBinary() && binaryPriority[op.Kind][0] > limit {
		if err := p.next(); err != nil {
			return err
		} else if err := p.expr(fn, binaryPriority[op.Kind][1]); err != nil {
			return err
		}
		p.dischargeBinop(fn, op, fn.sp-2, fn.sp-2, fn.sp-1)
		op = p.peek()
	}
	return nil
}

// load a single value onto the stack at the current stack pointer for later reference
func (p *Parser) discharge(fn *FuncProto, exp *exprDesc) {
	dst := fn.sp
	var todoname string
	var code Bytecode
	switch exp.kind {
	case constExpr, functionExpr:
		code = iABx(LOADK, dst, exp.b)
	case nilExpr:
		code = iABx(LOADNIL, dst, 1)
	case boolExpr:
		code = iABC(LOADBOOL, dst, uint8(exp.b), false, 0, false)
	case localExpr:
		code = iABC(MOVE, dst, uint8(exp.a), false, 0, false)
	case upvalueExpr:
		code = iABC(GETUPVAL, dst, uint8(exp.a), false, 0, false)
	case indexExpr:
		code = iABC(GETTABLE, dst, uint8(exp.a), false, uint8(exp.b), exp.bConst)
	case indexUpFieldExpr:
		code = iABC(GETTABUP, dst, uint8(exp.a), false, uint8(exp.b), exp.bConst)
	case closureExpr:
		code = iABx(CLOSURE, dst, exp.b)
	case callExpr:
		code = iABC(CALL, dst, uint8(exp.b), exp.bConst, uint8(exp.c), exp.cConst)
	case varArgsExpr:
		todoname = "varArgsExpr"
	case unaryOpExpr:
		todoname = "unaryOpExpr"
	case binaryOpExpr:
		todoname = "binaryOpExpr"
	case testExpr:
		todoname = "testExpr"
	case compareExpr:
		todoname = "compareExpr"
	default:
		panic("unknown expression to discharge")
	}
	if todoname != "" {
		panic(fmt.Sprintf("unexpected exprdesc kind %v", todoname))
	}
	fn.code(code)
	fn.sp++
}

// dischargeBinop will add the bytecode to execute the binop
func (p *Parser) dischargeBinop(fn *FuncProto, op *Token, dst, b, c uint8) {
	var code Bytecode
	switch op.Kind {
	case TokenOr, TokenAnd:
		panic("or and not implemented yet")
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
		panic("unknown binop")
	}
	fn.code(code)
	fn.sp = dst + 1
}

// simpleexp -> Float | Integer | String | nil | true | false | ... | constructor | FUNCTION body | suffixedexp
func (p *Parser) simpleexp(fn *FuncProto) (*exprDesc, error) {
	switch p.peek().Kind {
	case TokenFloat:
		value := &Float{val: p.mustnext(TokenFloat).FloatVal}
		idx := fn.addConst(value)
		return &exprDesc{kind: constExpr, b: idx}, nil
	case TokenInteger:
		value := &Integer{val: p.mustnext(TokenInteger).IntVal}
		idx := fn.addConst(value)
		return &exprDesc{kind: constExpr, b: idx}, nil
	case TokenString:
		value := &String{val: p.mustnext(TokenString).StringVal}
		idx := fn.addConst(value)
		return &exprDesc{kind: constExpr, b: idx}, nil
	case TokenNil:
		p.mustnext(TokenNil)
		return &exprDesc{kind: nilExpr}, nil
	case TokenTrue:
		p.mustnext(TokenTrue)
		return &exprDesc{kind: boolExpr, b: 1}, nil
	case TokenFalse:
		p.mustnext(TokenFalse)
		return &exprDesc{kind: boolExpr, b: 0}, nil
	case TokenOpenCurly:
		return p.constructor(fn)
	case TokenFunction: // function
		p.mustnext(TokenFunction)
		newFn, err := p.funcbody(fn, "anon")
		return &exprDesc{kind: constExpr, b: fn.addConst(&Function{val: newFn})}, err
	case TokenDots: // varargs
		panic("tokendots still doesnt work")
	default:
		return p.suffixedexp(fn)
	}
}

// primaryexp -> NAME | '(' expr ')'
func (p *Parser) primaryexp(fn *FuncProto) (*exprDesc, error) {
	switch p.peek().Kind {
	case TokenOpenParen:
		p.mustnext(TokenOpenParen)
		if err := p.expr(fn, nonePriority); err != nil {
			return nil, err
		}
		return nil, p.assertNext(TokenCloseParen)
	case TokenIdentifier:
		return p.name(fn, p.mustnext(TokenIdentifier).StringVal), nil
	default:
		return nil, fmt.Errorf("unexpected symbol %v", p.peek().Kind)
	}
}

// suffixedexp -> primaryexp { '.' NAME | '[' exp ']' | ':' NAME funcargs | funcargs }
func (p *Parser) suffixedexp(fn *FuncProto) (*exprDesc, error) {
	return p.primaryexp(fn)
}

// name is a reference to a variable that need resolution to have meaning
func (p *Parser) name(fn *FuncProto, name string) *exprDesc {
	if expr := p.resolveVar(fn, name); expr != nil {
		return expr
	}
	iname := fn.addConst(&String{val: name})
	if expr := p.name(fn, "_ENV"); expr.kind == localExpr {
		return &exprDesc{kind: indexExpr, a: expr.a, b: iname, bConst: true}
	} else if expr.kind == upvalueExpr {
		return &exprDesc{kind: indexUpFieldExpr, a: expr.a, b: iname, bConst: true}
	}
	panic(fmt.Sprintf("this should never happen cannot find _ENV while locating %v", name))
}

// resolveVar will recursively look up the stack to find where the variable
// resides in the stack and then build the chain of upvars to have a referece
// to it.
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

// explist -> expr { ',' expr }
func (p *Parser) explist(fn *FuncProto, want int) (int, error) {
	numExprs := 0
	for {
		if err := p.expr(fn, nonePriority); err != nil {
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
func (p *Parser) constructor(fn *FuncProto) (*exprDesc, error) {
	p.mustnext(TokenOpenCurly)
	itable := fn.sp
	tablecode := fn.code(iABC(NEWTABLE, fn.sp, 0, false, 0, false))
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
			} else if err := p.expr(fn, 0); err != nil {
				return nil, err
			}
			ikey := fn.addConst(&String{val: key.name})
			fn.code(iABC(SETTABLE, itable, uint8(ikey), true, fn.sp-1, false))
			fn.sp--
			numfields++
		case TokenOpenBracket:
			p.mustnext(TokenOpenBracket)
			if err := p.expr(fn, 0); err != nil {
				return nil, err
			} else if err := p.assertNext(TokenCloseBracket); err != nil {
				return nil, err
			} else if err := p.assertNext(TokenAssign); err != nil {
				return nil, err
			} else if err := p.expr(fn, 0); err != nil {
				return nil, err
			}
			fn.code(iABC(SETTABLE, itable, fn.sp-2, false, fn.sp-1, false))
			fn.sp -= 2
			numfields++
		default:
			if err := p.expr(fn, 0); err != nil {
				return nil, err
			}
			numvals++
		}

		if tk := p.peek(); tk.Kind == TokenComma || tk.Kind == TokenSemiColon {
			p.next()
		} else {
			break
		}
	}
	if numvals > 0 {
		fn.code(iABC(SETLIST, itable, uint8(numvals+1), false, 0, false))
	}
	fn.ByteCodes[tablecode] = iABC(NEWTABLE, itable, uint8(numvals), false, uint8(numfields), false)
	return &exprDesc{kind: constExpr, b: uint16(itable)}, p.assertNext(TokenCloseCurly)
}
