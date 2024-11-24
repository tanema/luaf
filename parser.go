package luaf

import (
	"fmt"
	"io"
	"reflect"
)

type (
	Parser struct {
		filename    string
		rootfn      *FuncProto
		lex         *Lexer
		breakBlocks [][]int
		localsScope []uint8
	}
	ParserError struct {
		fn       *FuncProto
		row, col int
		err      error
	}
)

func (err *ParserError) Error() string {
	return fmt.Sprintf(`Parse Error: %s:%v:%v %v`, err.fn.Filename, err.row, err.col, err.err)
}

func Parse(filename string, src io.Reader) (*FuncProto, error) {
	p := &Parser{
		filename:    filename,
		rootfn:      newFnProto(filename, "env", nil, []string{"_ENV"}, false, 0),
		lex:         NewLexer(src),
		breakBlocks: [][]int{},
		localsScope: []uint8{},
	}
	fn := newFnProto(filename, "main", p.rootfn, []string{}, false, 0)
	err := p.block(fn)
	if err == io.EOF {
		err = nil
	}
	if len(fn.ByteCodes) > 0 && fn.ByteCodes[len(fn.ByteCodes)-1].op() != RETURN {
		fn.code(iAB(RETURN, 0, 1))
	}
	return fn, err
}

func (p *Parser) parseErr(fn *FuncProto, token *Token, err error) error {
	return &ParserError{
		fn:  fn,
		row: token.Row,
		col: token.Column,
		err: err,
	}
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
	return p.statList(fn)
}

// statlist -> { stat [';'] }
func (p *Parser) statList(fn *FuncProto) error {
	for !p.blockFollow(true) {
		if p.peek().Kind == TokenReturn {
			return p.stat(fn) /* 'return' must be last stat */
		} else if err := p.stat(fn); err != nil {
			return err
		}
	}
	return nil
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

// stat -> ';' | ifstat | whilestat | dostat
// | forstat | repeatstat | funcstat
// | localstat | label | retstat | 'break'
// | 'goto' NAME | funccallstat | assignment
func (p *Parser) stat(fn *FuncProto) error {
	fn.stackPointer = uint8(len(fn.Locals))
	switch p.peek().Kind {
	case TokenSemiColon:
		return p.assertNext(TokenSemiColon)
	case TokenComment:
		return p.assertNext(TokenComment)
	case TokenLocal:
		return p.localstat(fn)
	case TokenFunction:
		return p.funcstat(fn)
	case TokenReturn:
		return p.retstat(fn)
	case TokenDo:
		return p.dostat(fn)
	case TokenIf:
		return p.ifstat(fn)
	case TokenWhile:
		return p.whilestat(fn)
	case TokenFor:
		return p.forstat(fn)
	case TokenRepeat:
		return p.repeatstat(fn)
	case TokenDoubleColon:
		return p.labelstat(fn)
	case TokenBreak:
		return p.breakstat(fn)
	case TokenGoto:
		return p.gotostat(fn)
	default:
		expr, err := p.suffixedexp(fn)
		if err != nil {
			return err
		} else if call, isCall := expr.(*exCall); isCall {
			p.discharge(fn, call, fn.stackPointer)
			return nil
		} else if tk := p.peek(); tk.Kind == TokenAssign || tk.Kind == TokenComma {
			return p.assignment(fn, expr)
		}
		return p.parseErr(fn, p.peek(), fmt.Errorf("unexpected expression %v", reflect.TypeOf(expr)))
	}
}

// localstat -> LOCAL [localfunc | localassign]
func (p *Parser) localstat(fn *FuncProto) error {
	p.mustnext(TokenLocal)
	if p.peek().Kind == TokenFunction {
		return p.localfunc(fn)
	}
	return p.localassign(fn)
}

// localfunc -> FUNCTION NAME funcbody
func (p *Parser) localfunc(fn *FuncProto) error {
	tk := p.mustnext(TokenFunction)
	ifn := uint8(len(fn.Locals))
	name, err := p.ident(fn)
	if err != nil {
		return err
	}
	p.addLocal(fn, name, false, false)
	newFn, err := p.funcbody(fn, name, tk.Row)
	if err != nil {
		return err
	}
	p.discharge(fn, &exClosure{fn: fn.addFn(newFn)}, ifn)
	return nil
}

// funcstat -> FUNCTION funcname funcbody
func (p *Parser) funcstat(fn *FuncProto) error {
	tk := p.mustnext(TokenFunction)
	sp0 := fn.stackPointer
	name, fullname, err := p.funcname(fn)
	if err != nil {
		return err
	} else if err := p.checkConst(name); err != nil {
		return err
	}
	newFn, err := p.funcbody(fn, fullname, tk.Row)
	if err != nil {
		return err
	}
	p.discharge(fn, &exClosure{fn: fn.addFn(newFn)}, sp0)
	p.assignTo(fn, name, sp0)
	return nil
}

func (p *Parser) checkConst(dst expression) error {
	val, isVal := dst.(*exValue)
	if isVal && val.attrConst {
		return fmt.Errorf("attempt to assign to const variable '%v'", val.name)
	}
	return nil
}

func (p *Parser) assignTo(fn *FuncProto, dst expression, from uint8) {
	switch ex := dst.(type) {
	case *exValue:
		if !ex.local {
			fn.code(iAB(SETUPVAL, ex.address, from))
		} else if from != ex.address {
			fn.code(iAB(MOVE, ex.address, from))
		}
	case *exIndex:
		if ex.local {
			fn.code(iABCK(SETTABLE, ex.table, ex.key, ex.keyIsConst, from, false))
		} else {
			fn.code(iABCK(SETTABUP, ex.table, ex.key, ex.keyIsConst, from, false))
		}
	default:
		panic("unknown expression to assign to")
	}
}

// funcname -> NAME {fieldsel} [':' NAME]
// fieldsel     -> ['.' | ':'] NAME
func (p *Parser) funcname(fn *FuncProto) (expression, string, error) {
	ident, err := p.ident(fn)
	if err != nil {
		return nil, "", err
	}
	name := p.name(fn, ident)
	fullname := ident
	for {
		switch p.peek().Kind {
		case TokenPeriod:
			p.mustnext(TokenPeriod)
			itable := fn.stackPointer
			p.discharge(fn, name, itable)

			ident, err := p.ident(fn)
			if err != nil {
				return nil, "", err
			}
			key := p.name(fn, ident)
			ikey := fn.stackPointer
			p.discharge(fn, key, ikey)
			fullname += "." + ident
			name = &exIndex{local: true, table: itable, key: ikey, keyIsConst: false}
		case TokenColon:
			p.mustnext(TokenColon)
			itable := fn.stackPointer
			p.discharge(fn, name, itable)

			ident, err := p.ident(fn)
			if err != nil {
				return nil, "", err
			}
			key := p.name(fn, ident)
			ikey := fn.stackPointer
			p.discharge(fn, key, ikey)
			fullname += ":" + ident
			return &exIndex{local: true, table: itable, key: ikey}, fullname, nil
		default:
			return name, fullname, nil
		}
	}
}

// funcbody -> parlist block END
func (p *Parser) funcbody(fn *FuncProto, name string, row int) (*FuncProto, error) {
	params, varargs, err := p.parlist(fn)
	if err != nil {
		return nil, err
	}
	newFn := newFnProto(p.filename, name, fn, params, varargs, row)
	if err := p.block(newFn); err != nil {
		return nil, err
	}
	if len(newFn.ByteCodes) > 0 && newFn.ByteCodes[len(newFn.ByteCodes)-1].op() != RETURN {
		newFn.code(iAB(RETURN, 0, 1))
	}
	return newFn, p.assertNext(TokenEnd)
}

// parlist -> '(' [ {NAME ','} (NAME | '...') ] ')'
func (p *Parser) parlist(fn *FuncProto) ([]string, bool, error) {
	if err := p.assertNext(TokenOpenParen); err != nil {
		return nil, false, err
	}
	names := []string{}
	if p.peek().Kind == TokenCloseParen {
		return names, false, p.assertNext(TokenCloseParen)
	}
	for {
		if p.peek().Kind != TokenIdentifier {
			break
		}
		name, err := p.ident(fn)
		if err != nil {
			return nil, false, err
		}
		names = append(names, name)
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

// retstat -> RETURN [explist] [';']
func (p *Parser) retstat(fn *FuncProto) error {
	p.mustnext(TokenReturn)
	sp0 := fn.stackPointer
	if p.blockFollow(true) {
		fn.code(iAB(RETURN, sp0, 1))
		return nil
	}
	nret, lastExpr, lastExprDst, err := p.explist(fn)
	if err != nil {
		return err
	}
	switch expr := lastExpr.(type) {
	case *exCall:
		fn.code(iAB(TAILCALL, expr.fn, expr.nargs))
		fn.code(iAB(RETURN, 0, 0))
	case *exVarArgs:
		p.discharge(fn, expr, lastExprDst)
		fn.code(iAB(RETURN, sp0, 0))
	default:
		p.discharge(fn, expr, lastExprDst)
		fn.code(iAB(RETURN, sp0, uint8(nret+1)))
	}
	return nil
}

// dostat -> DO block END
func (p *Parser) dostat(fn *FuncProto) error {
	p.mustnext(TokenDo)
	sp0 := fn.stackPointer
	if err := p.block(fn); err != nil {
		return err
	}
	p.localExpire(fn, sp0)
	return p.assertNext(TokenEnd)
}

// ifstat -> IF exp THEN block {ELSEIF exp THEN block} [ELSE block] END
func (p *Parser) ifstat(fn *FuncProto) error {
	p.mustnext(TokenIf)
	jmpTbl := []int{} // index of opcode that jump to the end of the block

	if err := p.ifblock(fn, &jmpTbl); err != nil {
		return err
	}

	for p.peek().Kind == TokenElseif {
		p.mustnext(TokenElseif)
		if err := p.ifblock(fn, &jmpTbl); err != nil {
			return err
		}
	}

	if p.peek().Kind == TokenElse {
		p.mustnext(TokenElse)
		if err := p.block(fn); err != nil {
			return err
		}
	}

	iend := len(fn.ByteCodes) - 1
	for _, idx := range jmpTbl {
		fn.ByteCodes[idx] = iABx(JMP, 0, uint16(iend-idx))
	}
	return p.assertNext(TokenEnd)
}

func (p *Parser) ifblock(fn *FuncProto, jmpTbl *[]int) error {
	condition, err := p.expr(fn, 0)
	if err != nil {
		return err
	} else if err := p.assertNext(TokenThen); err != nil {
		return err
	}
	spCondition := fn.stackPointer
	p.discharge(fn, condition, spCondition)
	fn.code(iAB(TEST, spCondition, 0))
	iFalseJmp := fn.code(iAsBx(JMP, 0, 0))
	if err := p.block(fn); err != nil {
		return err
	}
	iend := int16(len(fn.ByteCodes) - iFalseJmp)
	if tk := p.peek().Kind; tk == TokenElse || tk == TokenElseif {
		*jmpTbl = append(*jmpTbl, fn.code(iAsBx(JMP, 0, 0)))
		iend++
	}
	fn.ByteCodes[iFalseJmp] = iAsBx(JMP, 0, iend-1)
	return nil
}

func (p *Parser) whilestat(fn *FuncProto) error {
	p.mustnext(TokenWhile)
	sp0 := p.pushLoopBlock(fn)
	istart := int16(len(fn.ByteCodes))
	condition, err := p.expr(fn, 0)
	if err != nil {
		return err
	} else if err := p.assertNext(TokenDo); err != nil {
		return err
	}
	spCondition := fn.stackPointer
	p.discharge(fn, condition, spCondition)
	fn.code(iAB(TEST, spCondition, 0))
	iFalseJmp := fn.code(iAsBx(JMP, 0, 0))
	if err := p.block(fn); err != nil {
		return err
	} else if err := p.assertNext(TokenEnd); err != nil {
		return err
	}
	iend := int16(len(fn.ByteCodes))
	fn.code(iAsBx(JMP, sp0+1, -(iend-istart)-1))
	fn.ByteCodes[iFalseJmp] = iAsBx(JMP, sp0+1, int16(iend-int16(iFalseJmp)))
	p.popLoopBlock(fn)
	return nil
}

// forstat -> FOR (fornum | forlist) END
func (p *Parser) forstat(fn *FuncProto) error {
	p.mustnext(TokenFor)
	name, err := p.ident(fn)
	if err != nil {
		return err
	}
	if p.peek().Kind == TokenAssign {
		return p.fornum(fn, name)
	} else if tk := p.peek().Kind; tk == TokenComma || tk == TokenIn {
		return p.forlist(fn, name)
	}
	return p.parseErr(fn, p.peek(), fmt.Errorf("malformed for statment"))
}

// fornum -> NAME = exp,exp[,exp] DO
func (p *Parser) fornum(fn *FuncProto, name string) error {
	p.mustnext(TokenAssign)
	sp0 := p.pushLoopBlock(fn)
	nexprs, lastExpr, lastExprDst, err := p.explist(fn)
	if err != nil {
		return err
	}

	if nexprs < 2 || nexprs > 3 {
		return p.parseErr(fn, p.peek(), fmt.Errorf("invalid for stat"))
	}

	p.discharge(fn, lastExpr, lastExprDst)
	if nexprs == 2 {
		p.discharge(fn, &exConstant{index: fn.addConst(1)}, fn.stackPointer)
	}

	// add the iterator var, limit, step locals, the last two cannot be directly accessed
	p.addLocal(fn, name, false, false)
	p.addLocal(fn, "", false, false)
	p.addLocal(fn, "", false, false)
	iforPrep := fn.code(iAsBx(FORPREP, sp0, 0))

	if err := p.assertNext(TokenDo); err != nil {
		return err
	} else if err := p.block(fn); err != nil {
		return err
	} else if err := p.assertNext(TokenEnd); err != nil {
		return err
	}

	blockSize := int16(len(fn.ByteCodes) - iforPrep - 1)
	fn.code(iAsBx(FORLOOP, sp0, -blockSize-1))
	fn.ByteCodes[iforPrep] = iAsBx(FORPREP, sp0, blockSize)
	p.popLoopBlock(fn)
	return nil
}

// forlist -> NAME {,NAME} IN explist DO
func (p *Parser) forlist(fn *FuncProto, firstName string) error {
	sp0 := p.pushLoopBlock(fn)
	names := []string{firstName}
	if p.peek().Kind == TokenComma {
		p.mustnext(TokenComma)
		name, err := p.ident(fn)
		if err != nil {
			return err
		}
		names = append(names, name)
	}
	if err := p.assertNext(TokenIn); err != nil {
		return err
	}

	if err := p.explistWant(fn, 3); err != nil {
		return err
	}
	p.addLocal(fn, "", false, false)
	p.addLocal(fn, "", false, false)
	p.addLocal(fn, "", false, false)
	for _, name := range names {
		p.addLocal(fn, name, false, false)
	}

	ijmp := fn.code(iAsBx(JMP, 0, 0))

	if err := p.assertNext(TokenDo); err != nil {
		return err
	} else if err := p.block(fn); err != nil {
		return err
	} else if err := p.assertNext(TokenEnd); err != nil {
		return err
	}

	fn.ByteCodes[ijmp] = iAsBx(JMP, 0, int16(len(fn.ByteCodes)-ijmp-1))
	fn.code(iAB(TFORCALL, sp0, uint8(len(names))))
	fn.code(iAsBx(TFORLOOP, sp0+1, -int16(len(fn.ByteCodes)-ijmp)))
	p.popLoopBlock(fn)
	return nil
}

func (p *Parser) repeatstat(fn *FuncProto) error {
	p.mustnext(TokenRepeat)
	sp0 := p.pushLoopBlock(fn)
	istart := len(fn.ByteCodes)
	if err := p.block(fn); err != nil {
		return err
	} else if err := p.assertNext(TokenUntil); err != nil {
		return err
	}
	condition, err := p.expr(fn, 0)
	if err != nil {
		return err
	}
	spCondition := fn.stackPointer
	p.discharge(fn, condition, spCondition)
	fn.code(iAB(TEST, spCondition, 0))
	fn.code(iAsBx(JMP, sp0+1, -int16(len(fn.ByteCodes)-istart)))
	p.popLoopBlock(fn)
	return nil
}

func (p *Parser) breakstat(fn *FuncProto) error {
	breakToken := p.mustnext(TokenBreak)
	if len(p.breakBlocks) == 0 {
		return p.parseErr(fn, breakToken, fmt.Errorf("use of a break outside of loop"))
	}
	p.breakBlocks[len(p.breakBlocks)-1] = append(p.breakBlocks[len(p.breakBlocks)-1], fn.code(iAsBx(JMP, 0, 0)))
	return nil
}

// label -> '::' NAME '::'
func (p *Parser) labelstat(fn *FuncProto) error {
	lableToken := p.mustnext(TokenDoubleColon)
	name, err := p.ident(fn)
	if err != nil {
		return err
	}
	if _, found := fn.Labels[name]; found {
		return p.parseErr(fn, lableToken, fmt.Errorf("duplicate label defined: %v", name))
	}
	icode := len(fn.ByteCodes)
	fn.Labels[name] = icode
	if codes, hasGotos := fn.Gotos[name]; hasGotos {
		for _, jmpcode := range codes {
			fn.ByteCodes[jmpcode] = iAsBx(JMP, 0, int16(icode-jmpcode-1))
		}
		delete(fn.Gotos, name)
	}
	return p.assertNext(TokenDoubleColon)
}

// gotostat -> 'goto' NAME
func (p *Parser) gotostat(fn *FuncProto) error {
	p.mustnext(TokenGoto)
	if name, err := p.ident(fn); err != nil {
		return err
	} else if icode, found := fn.Labels[name]; found {
		fn.code(iAsBx(JMP, 0, -int16(len(fn.ByteCodes)-icode+1)))
	} else {
		fn.Gotos[name] = append(fn.Gotos[name], fn.code(iAsBx(JMP, 0, 0)))
	}
	return nil
}

// localassign -> NAME attrib { ',' NAME attrib } ['=' explist]
func (p *Parser) localassign(fn *FuncProto) error {
	lcl0 := uint8(len(fn.Locals))
	names := []*exValue{}
	for {
		lcl, attrConst, attrClose, err := p.identWithAttrib(fn)
		if err != nil {
			return err
		}
		name := &exValue{
			local:     true,
			name:      lcl,
			attrConst: attrConst,
			attrClose: attrClose,
			address:   lcl0 + uint8(len(names)),
		}
		names = append(names, name)
		if p.peek().Kind != TokenComma {
			break
		} else if err := p.next(); err != nil {
			return err
		}
	}
	if p.peek().Kind != TokenAssign {
		p.discharge(fn, &exNil{num: uint16(len(names) - 1)}, lcl0)
		return nil
	}
	p.mustnext(TokenAssign)
	sp0 := fn.stackPointer
	if err := p.explistWant(fn, len(names)); err != nil {
		return err
	}
	for i, name := range names {
		p.addLocal(fn, name.name, name.attrConst, name.attrClose)
		p.assignTo(fn, name, sp0+uint8(i))
		if name.attrClose {
			fn.code(iAB(TBC, name.address, 0))
		}
	}
	return nil
}

func (p *Parser) explistWant(fn *FuncProto, want int) error {
	sp0 := uint8(len(fn.Locals))
	numExprs, lastExpr, lastExprDst, err := p.explist(fn)
	if err != nil {
		return err
	}
	diff := uint8(want - numExprs)
	switch expr := lastExpr.(type) {
	case *exCall:
		if diff > 0 {
			expr.nret = diff + 2
		}
		p.discharge(fn, expr, lastExprDst)
	case *exVarArgs:
		if diff > 0 {
			expr.want = diff + 2
		}
		p.discharge(fn, expr, lastExprDst)
	default:
		p.discharge(fn, expr, lastExprDst)
		if diff > 0 {
			p.discharge(fn, &exNil{num: uint16(diff)}, fn.stackPointer)
		}
	}
	fn.stackPointer = sp0 + uint8(want)
	return nil
}

// ident is a simple identifier that will be needed for later use as a var
func (p *Parser) ident(fn *FuncProto) (string, error) {
	tk, err := p.lex.Next()
	if err != nil {
		return "", err
	} else if tk.Kind != TokenIdentifier {
		return "", p.parseErr(fn, tk, fmt.Errorf("expected Name but found %v", tk.Kind))
	}
	return tk.StringVal, nil
}

// NAME attrib
// attrib -> ['<' ('const' | 'close') '>']
func (p *Parser) identWithAttrib(fn *FuncProto) (string, bool, bool, error) {
	attrConst, attrClose := false, false
	local, err := p.ident(fn)
	if err != nil {
		return "", false, false, err
	}
	if p.peek().Kind == TokenLt {
		p.mustnext(TokenLt)
		if tk, err := p.lex.Next(); err != nil {
			return "", false, false, err
		} else if tk.Kind != TokenIdentifier {
			return "", false, false, fmt.Errorf("expected attrib but found %v", tk.Kind)
		} else if tk.StringVal == "const" {
			attrConst = true
		} else if tk.StringVal == "close" {
			attrClose = true
		} else {
			return "", false, false, fmt.Errorf("unknown local attribute %v", tk.StringVal)
		}
		if err := p.assertNext(TokenGt); err != nil {
			return "", false, false, err
		}
	}
	return local, attrConst, attrClose, nil
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
		nparams, lastExpr, lastExprDst, err := p.explist(fn)
		if err != nil {
			return 0, err
		}
		p.discharge(fn, lastExpr, lastExprDst)
		switch lastExpr.(type) {
		case *exCall, *exVarArgs:
			return -1, p.assertNext(TokenCloseParen)
		default:
			return nparams, p.assertNext(TokenCloseParen)
		}
	case TokenOpenCurly:
		_, err := p.constructor(fn)
		return 1, err
	case TokenString:
		p.discharge(fn, &exConstant{index: fn.addConst(p.mustnext(TokenString).StringVal)}, fn.stackPointer)
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
	sp0 := fn.stackPointer
	if err := p.explistWant(fn, len(names)); err != nil {
		return err
	}
	for i, name := range names {
		if err := p.checkConst(name); err != nil {
			return err
		}
		p.assignTo(fn, name, sp0+uint8(i))
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
		ival := fn.stackPointer
		p.discharge(fn, desc, ival)
		desc = tokenToUnary(tk.Kind, ival)
	} else {
		desc, err = p.simpleexp(fn)
		if err != nil {
			return nil, err
		}
	}
	op := p.peek()
	for op.isBinary() && binaryPriority[op.Kind][0] > limit {
		lval := p.discharge(fn, desc, fn.stackPointer)
		if err := p.next(); err != nil {
			return nil, err
		}
		desc, err = p.expr(fn, binaryPriority[op.Kind][1])
		if err != nil {
			return nil, err
		}
		desc = tokenToBinopExpression(op.Kind, lval, p.discharge(fn, desc, fn.stackPointer))
		op = p.peek()
	}
	return desc, nil
}

func (p *Parser) discharge(fn *FuncProto, exp expression, dst uint8) uint8 {
	if call, isCall := exp.(*exCall); isCall {
		call.discharge(fn, dst)
		return call.fn
	}
	exp.discharge(fn, dst)
	fn.stackPointer = dst + 1
	return dst
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
		return &exNil{num: 1}, p.assertNext(TokenNil)
	case TokenTrue:
		return &exBool{val: true, skip: false}, p.assertNext(TokenTrue)
	case TokenFalse:
		return &exBool{val: false, skip: false}, p.assertNext(TokenFalse)
	case TokenOpenCurly:
		return p.constructor(fn)
	case TokenFunction:
		tk := p.mustnext(TokenFunction)
		newFn, err := p.funcbody(fn, "", tk.Row)
		return &exClosure{fn: fn.addFn(newFn)}, err
	case TokenDots:
		p.mustnext(TokenDots)
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
			itable := p.discharge(fn, expr, fn.stackPointer)
			key, err := p.ident(fn)
			if err != nil {
				return nil, err
			}
			expr = &exIndex{local: true, table: itable, key: uint8(fn.addConst(key)), keyIsConst: true}
		case TokenOpenBracket:
			p.mustnext(TokenOpenBracket)
			itable := p.discharge(fn, expr, fn.stackPointer)
			firstexpr, err := p.expr(fn, nonePriority)
			if err != nil {
				return nil, err
			} else if err := p.assertNext(TokenCloseBracket); err != nil {
				return nil, err
			}
			ival := p.discharge(fn, firstexpr, fn.stackPointer)
			expr = &exIndex{local: true, table: itable, key: ival}
		case TokenColon:
			p.mustnext(TokenColon)
			p.discharge(fn, expr, fn.stackPointer)
			key, err := p.ident(fn)
			if err != nil {
				return nil, err
			}
			ifn := fn.stackPointer
			fn.code(iABCK(SELF, fn.stackPointer, fn.stackPointer-1, false, uint8(fn.addConst(key)), true))
			fn.stackPointer++
			nargs, err := p.funcargs(fn)
			if err != nil {
				return nil, err
			}
			expr = &exCall{fn: ifn, nret: 2, nargs: uint8(nargs + 1)}
		case TokenOpenParen, TokenString, TokenOpenCurly:
			ifn := p.discharge(fn, expr, fn.stackPointer)
			nargs, err := p.funcargs(fn)
			if err != nil {
				return nil, err
			}
			expr = &exCall{fn: uint8(ifn), nret: 2, nargs: uint8(nargs + 1)}
		default:
			return expr, nil
		}
	}
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
		lcl := fn.Locals[idx]
		return &exValue{local: true, name: name, address: uint8(idx), lvar: lcl, attrConst: lcl.attrConst, attrClose: lcl.attrClose}
	} else if idx, ok := search(fn.UpIndexes, name, findUpindex); ok {
		return &exValue{local: false, name: name, address: uint8(idx)}
	} else if expr := p.resolveVar(fn.prev, name); expr != nil {
		if value, isValue := expr.(*exValue); isValue && value.local {
			value.lvar.upvalRef = true
			fn.UpIndexes = append(fn.UpIndexes, UpIndex{fromStack: true, name: name, index: uint(value.address)})
		} else if isValue {
			fn.UpIndexes = append(fn.UpIndexes, UpIndex{fromStack: false, name: name, index: uint(value.address)})
		}
		return &exValue{local: false, name: name, address: uint8(len(fn.UpIndexes) - 1)}
	}
	return nil
}

// explist -> expr { ',' expr }
// this will ensure that after evaluation, the final values are placed at
// fn.stackPointer, fn.stackPointer+1,fn.stackPointer+2......
// no matter how much of the stack was used up during computation of the expr
func (p *Parser) explist(fn *FuncProto) (int, expression, uint8, error) {
	sp0 := fn.stackPointer
	numExprs := 0
	for {
		expr, err := p.expr(fn, nonePriority)
		if err != nil {
			return -1, nil, 0, err
		}
		if p.peek().Kind != TokenComma {
			return numExprs + 1, expr, sp0 + uint8(numExprs), nil
		}
		p.discharge(fn, expr, sp0+uint8(numExprs))
		numExprs++
		p.mustnext(TokenComma)
	}
}

// constructor -> '{' [ field { sep field } [sep] ] '}'
// sep         -> ',' | ';'
// field -> NAME = exp | '['exp']' = exp | exp
func (p *Parser) constructor(fn *FuncProto) (expression, error) {
	p.mustnext(TokenOpenCurly)
	itable := fn.stackPointer
	tablecode := fn.code(iAB(NEWTABLE, 0, 0))
	fn.stackPointer++
	numvals, numfields := 0, 0
	for {
		switch p.peek().Kind {
		case TokenCloseCurly:
			// do nothing, because it is an empty table
		case TokenIdentifier:
			key, err := p.ident(fn)
			if err != nil {
				return nil, err
			} else if err := p.assertNext(TokenAssign); err != nil {
				return nil, err
			}
			desc, err := p.expr(fn, 0)
			if err != nil {
				return nil, err
			}
			if kexp, isConst := desc.(*exConstant); isConst {
				fn.code(iABCK(SETTABLE, itable, uint8(fn.addConst(key)), true, uint8(kexp.index), true))
			} else {
				fn.code(iABCK(SETTABLE, itable, uint8(fn.addConst(key)), true, p.discharge(fn, desc, fn.stackPointer), false))
			}
			numfields++
			fn.stackPointer = itable + uint8(numvals) + 1
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
			ikey := fn.stackPointer
			keyConst := false
			if kexp, isConst := keydesc.(*exConstant); isConst {
				ikey = uint8(kexp.index)
				keyConst = true
			} else {
				p.discharge(fn, keydesc, ikey)
			}
			valdesc, err := p.expr(fn, 0)
			if err != nil {
				return nil, err
			}
			ival := fn.stackPointer
			valConst := false
			if vexp, isConst := valdesc.(*exConstant); isConst {
				ival = uint8(vexp.index)
				valConst = true
			} else {
				p.discharge(fn, valdesc, ival)
			}
			fn.code(iABCK(SETTABLE, itable, ikey, keyConst, ival, valConst))
			numfields++
			fn.stackPointer = itable + uint8(numvals) + 1
		default:
			desc, err := p.expr(fn, 0)
			if err != nil {
				return nil, err
			}
			p.discharge(fn, desc, fn.stackPointer)
			numvals++
		}

		if tk := p.peek(); tk.Kind == TokenComma || tk.Kind == TokenSemiColon {
			if err := p.next(); err != nil {
				return nil, err
			}
		} else {
			break
		}
	}

	if numvals > 0 {
		fn.code(iABC(SETLIST, itable, uint8(numvals+1), 1))
	}
	fn.stackPointer = itable + 1
	fn.ByteCodes[tablecode] = iABC(NEWTABLE, itable, uint8(numvals), uint8(numfields))
	return &exValue{local: true, address: uint8(itable)}, p.assertNext(TokenCloseCurly)
}

func (p *Parser) pushLoopBlock(fn *FuncProto) uint8 {
	p.breakBlocks = append(p.breakBlocks, []int{})
	p.localsScope = append(p.localsScope, fn.stackPointer)
	return fn.stackPointer
}

func (p *Parser) popLoopBlock(fn *FuncProto) {
	if len(p.breakBlocks) == 0 {
		return
	}
	from := p.localsScope[len(p.localsScope)-1]
	breaks := p.breakBlocks[len(p.breakBlocks)-1]
	endDst := len(fn.ByteCodes)
	for _, idx := range breaks {
		fn.ByteCodes[idx] = iABx(JMP, from+1, uint16(endDst-idx-1))
	}
	p.breakBlocks = p.breakBlocks[:len(p.breakBlocks)-1]
	p.localsScope = p.localsScope[:len(p.localsScope)-1]
	p.localExpire(fn, from)
}

func (p *Parser) addLocal(fn *FuncProto, name string, attrConst, attrClose bool) {
	fn.Locals = append(fn.Locals, &Local{
		name:      name,
		attrConst: attrConst,
		attrClose: attrClose,
	})
	fn.stackPointer = uint8(len(fn.Locals))
}

func (p *Parser) localExpire(fn *FuncProto, from uint8) {
	for _, local := range truncate(&fn.Locals, int(from)) {
		if local.upvalRef {
			fn.code(iAB(CLOSE, from, 0))
			break
		}
	}
	fn.stackPointer = from
}
