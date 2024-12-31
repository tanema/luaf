package luaf

import (
	"fmt"
	"io"
	"math"
	"reflect"
)

type (
	Parser struct {
		filename      string
		rootfn        *FnProto
		lex           *Lexer
		breakBlocks   [][]int
		localsScope   []uint8
		lastTokenInfo LineInfo
	}
	ParserError struct {
		LineInfo
		filename string
		err      error
	}
)

func (err *ParserError) Error() string {
	return fmt.Sprintf(`Parse Error: %s:%v:%v %v`, err.filename, err.Line, err.Column, err.err)
}

func Parse(filename string, src io.Reader) (*FnProto, error) {
	p := &Parser{
		filename:    filename,
		rootfn:      newFnProto(filename, "env", nil, []string{"_ENV"}, false, LineInfo{}),
		lex:         NewLexer(src),
		breakBlocks: [][]int{},
		localsScope: []uint8{},
	}
	fn := newFnProto(filename, "main chunk", p.rootfn, []string{}, false, LineInfo{})
	err := p.block(fn)
	if err == io.EOF {
		err = nil
	}
	if len(fn.ByteCodes) > 0 && fn.ByteCodes[len(fn.ByteCodes)-1].op() != RETURN {
		p.code(fn, iAB(RETURN, 0, 1))
	}
	return fn, err
}

func (p *Parser) parseErrf(tk *Token, msg string, data ...any) error {
	return p.parseErr(tk, fmt.Errorf(msg, data...))
}

func (p *Parser) parseErr(token *Token, err error) error {
	var linfo LineInfo
	if token != nil {
		linfo = token.LineInfo
	}
	if lexErr, isLexErr := err.(*LexerError); isLexErr {
		linfo = lexErr.LineInfo
	} else if _, isParseErr := err.(*ParserError); isParseErr {
		return err
	} else if err == io.EOF {
		return err
	}
	return &ParserError{
		LineInfo: linfo,
		filename: p.filename,
		err:      err,
	}
}

func (p *Parser) peek() *Token {
	return p.lex.Peek()
}

func (p *Parser) _next(tt ...TokenType) (*Token, error) {
	tk, err := p.lex.Next()
	if err != nil {
		return nil, p.parseErr(tk, err)
	} else if len(tt) > 0 && tt[0] != tk.Kind {
		return nil, p.parseErrf(tk, "expected %v but consumed %v", tt, tk.Kind)
	}
	p.lastTokenInfo = tk.LineInfo
	return tk, nil
}

func (p *Parser) next(tt ...TokenType) error {
	_, err := p._next(tt...)
	return err
}

// This is used when the token has already been peeked but lets panic just in
// case something goes funky
func (p *Parser) mustnext(tt TokenType) *Token {
	tk, err := p._next()
	if err != nil {
		panic(err)
	} else if tk.Kind != tt {
		panic(p.parseErrf(tk, "expected %v but consumed %v", tt, tk.Kind))
	}
	return tk
}

// block -> statlist
func (p *Parser) block(fn *FnProto) error {
	return p.statList(fn)
}

// statlist -> { stat [';'] }
func (p *Parser) statList(fn *FnProto) error {
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
func (p *Parser) stat(fn *FnProto) error {
	fn.stackPointer = uint8(len(fn.locals))
	switch p.peek().Kind {
	case TokenSemiColon:
		return p.next(TokenSemiColon)
	case TokenComment:
		return p.next(TokenComment)
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
		tk := p.lex.Peek()
		expr, err := p.suffixedexp(fn)
		if err != nil {
			return err
		} else if call, isCall := expr.(*exCall); isCall {
			p.discharge(fn, call, fn.stackPointer)
			return nil
		} else if tk := p.peek(); tk.Kind == TokenAssign || tk.Kind == TokenComma {
			return p.assignment(fn, expr)
		}
		return p.parseErrf(tk, "unexpected expression %v", reflect.TypeOf(expr))
	}
}

// localstat -> LOCAL [localfunc | localassign]
func (p *Parser) localstat(fn *FnProto) error {
	p.mustnext(TokenLocal)
	if p.peek().Kind == TokenFunction {
		return p.localfunc(fn)
	}
	return p.localassign(fn)
}

// localfunc -> FUNCTION NAME funcbody
func (p *Parser) localfunc(fn *FnProto) error {
	p.mustnext(TokenFunction)
	ifn := uint8(len(fn.locals))
	name, err := p.ident()
	if err != nil {
		return err
	}
	if err := fn.addLocal(name.StringVal, false, false); err != nil {
		return err
	}
	newFn, err := p.funcbody(fn, name.StringVal, false, name.LineInfo)
	if err != nil {
		return err
	}
	expr := &exClosure{
		fn:       fn.addFn(newFn),
		LineInfo: name.LineInfo,
	}
	p.discharge(fn, expr, ifn)
	return nil
}

// funcstat -> FUNCTION funcname funcbody
func (p *Parser) funcstat(fn *FnProto) error {
	tk := p.mustnext(TokenFunction)
	sp0 := fn.stackPointer
	name, hasSelf, fullname, err := p.funcname(fn)
	if err != nil {
		return err
	} else if err := p.checkConst(tk, name); err != nil {
		return err
	}
	newFn, err := p.funcbody(fn, fullname, hasSelf, tk.LineInfo)
	if err != nil {
		return err
	}
	expr := &exClosure{
		fn:       fn.addFn(newFn),
		LineInfo: tk.LineInfo,
	}
	p.discharge(fn, expr, sp0)
	p.assignTo(fn, name, sp0)
	return nil
}

func (p *Parser) checkConst(tk *Token, dst expression) error {
	val, isVal := dst.(*exValue)
	if isVal && val.attrConst {
		return p.parseErrf(tk, "attempt to assign to const variable '%v'", val.name)
	}
	return nil
}

func (p *Parser) assignTo(fn *FnProto, dst expression, from uint8) {
	switch ex := dst.(type) {
	case *exValue:
		if !ex.local {
			p.code(fn, iAB(SETUPVAL, ex.address, from))
		} else if from != ex.address {
			p.code(fn, iAB(MOVE, ex.address, from))
		}
	case *exIndex:
		if ex.local {
			p.code(fn, iABCK(SETTABLE, ex.table, ex.key, ex.keyIsConst, from, false))
		} else {
			p.code(fn, iABCK(SETTABUP, ex.table, ex.key, ex.keyIsConst, from, false))
		}
	default:
		panic("unknown expression to assign to")
	}
}

// funcname -> NAME {fieldsel} [':' NAME]
// fieldsel     -> ['.' | ':'] NAME
func (p *Parser) funcname(fn *FnProto) (expression, bool, string, error) {
	ident, err := p.ident()
	if err != nil {
		return nil, false, "", err
	}
	name, err := p.name(fn, ident)
	if err != nil {
		return nil, false, "", err
	}
	fullname := ident.StringVal
	for {
		switch p.peek().Kind {
		case TokenPeriod:
			p.mustnext(TokenPeriod)
			itable := p.dischargeIfNeed(fn, name, fn.stackPointer)
			ident, err := p.ident()
			if err != nil {
				return nil, false, "", err
			}
			fullname += "." + ident.StringVal
			kaddr, err := fn.addConst(ident.StringVal)
			if err != nil {
				return nil, false, "", err
			}
			name = &exIndex{
				local:      true,
				table:      itable,
				key:        uint8(kaddr),
				keyIsConst: true,
				LineInfo:   ident.LineInfo,
			}
		case TokenColon:
			p.mustnext(TokenColon)
			itable := p.dischargeIfNeed(fn, name, fn.stackPointer)
			ident, err := p.ident()
			if err != nil {
				return nil, false, "", err
			}
			fullname += ":" + ident.StringVal
			kaddr, err := fn.addConst(ident.StringVal)
			if err != nil {
				return nil, false, "", err
			}
			return &exIndex{
				local:      true,
				table:      itable,
				key:        uint8(kaddr),
				keyIsConst: true,
				LineInfo:   ident.LineInfo,
			}, true, fullname, nil
		default:
			return name, false, fullname, nil
		}
	}
}

// funcbody -> parlist block END
func (p *Parser) funcbody(fn *FnProto, name string, hasSelf bool, linfo LineInfo) (*FnProto, error) {
	params, varargs, err := p.parlist()
	if err != nil {
		return nil, err
	}
	if hasSelf {
		params = append([]string{"self"}, params...)
	}
	newFn := newFnProto(p.filename, name, fn, params, varargs, linfo)
	if err := p.block(newFn); err != nil {
		return nil, err
	}
	if len(newFn.ByteCodes) > 0 && newFn.ByteCodes[len(newFn.ByteCodes)-1].op() != RETURN {
		p.code(newFn, iAB(RETURN, 0, 1))
	}
	return newFn, p.next(TokenEnd)
}

// parlist -> '(' [ {NAME ','} (NAME | '...') ] ')'
func (p *Parser) parlist() ([]string, bool, error) {
	if err := p.next(TokenOpenParen); err != nil {
		return nil, false, err
	}
	names := []string{}
	if p.peek().Kind == TokenCloseParen {
		return names, false, p.next(TokenCloseParen)
	}
	for {
		if p.peek().Kind != TokenIdentifier {
			break
		}
		name, err := p.ident()
		if err != nil {
			return nil, false, err
		}
		names = append(names, name.StringVal)
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
	return names, varargs, p.next(TokenCloseParen)
}

// retstat -> RETURN [explist] [';']
func (p *Parser) retstat(fn *FnProto) error {
	p.mustnext(TokenReturn)
	sp0 := fn.stackPointer
	if p.blockFollow(true) {
		p.code(fn, iAB(RETURN, sp0, 1))
		return nil
	}
	nret, lastExpr, lastExprDst, err := p.explist(fn)
	if err != nil {
		return err
	}
	switch expr := lastExpr.(type) {
	case *exCall:
		p.code(fn, iAB(TAILCALL, expr.fn, expr.nargs))
		p.code(fn, iAB(RETURN, 0, 0))
	case *exVarArgs:
		p.discharge(fn, expr, lastExprDst)
		p.code(fn, iAB(RETURN, sp0, 0))
	default:
		p.discharge(fn, expr, lastExprDst)
		p.code(fn, iAB(RETURN, sp0, uint8(nret+1)))
	}
	return nil
}

// dostat -> DO block END
func (p *Parser) dostat(fn *FnProto) error {
	p.mustnext(TokenDo)
	sp0 := fn.stackPointer
	if err := p.block(fn); err != nil {
		return err
	}
	p.localExpire(fn, sp0)
	return p.next(TokenEnd)
}

// ifstat -> IF exp THEN block {ELSEIF exp THEN block} [ELSE block] END
func (p *Parser) ifstat(fn *FnProto) error {
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
	return p.next(TokenEnd)
}

func (p *Parser) ifblock(fn *FnProto, jmpTbl *[]int) error {
	condition, err := p.expr(fn, 0)
	if err != nil {
		return err
	} else if err := p.next(TokenThen); err != nil {
		return err
	}
	spCondition := fn.stackPointer
	p.discharge(fn, condition, spCondition)
	p.code(fn, iAB(TEST, spCondition, 0))
	iFalseJmp := p.code(fn, iAsBx(JMP, 0, 0))
	if err := p.block(fn); err != nil {
		return err
	}
	iend := int16(len(fn.ByteCodes) - iFalseJmp)
	if tk := p.peek().Kind; tk == TokenElse || tk == TokenElseif {
		*jmpTbl = append(*jmpTbl, p.code(fn, iAsBx(JMP, 0, 0)))
		iend++
	}
	fn.ByteCodes[iFalseJmp] = iAsBx(JMP, 0, iend-1)
	return nil
}

func (p *Parser) whilestat(fn *FnProto) error {
	p.mustnext(TokenWhile)
	sp0 := p.pushLoopBlock(fn)
	istart := int16(len(fn.ByteCodes))
	condition, err := p.expr(fn, 0)
	if err != nil {
		return err
	} else if err := p.next(TokenDo); err != nil {
		return err
	}
	spCondition := fn.stackPointer
	p.discharge(fn, condition, spCondition)
	p.code(fn, iAB(TEST, spCondition, 0))
	iFalseJmp := p.code(fn, iAsBx(JMP, 0, 0))
	if err := p.block(fn); err != nil {
		return err
	} else if err := p.next(TokenEnd); err != nil {
		return err
	}
	iend := int16(len(fn.ByteCodes))
	p.code(fn, iAsBx(JMP, sp0+1, -(iend-istart)-1))
	fn.ByteCodes[iFalseJmp] = iAsBx(JMP, sp0+1, int16(iend-int16(iFalseJmp)))
	p.popLoopBlock(fn)
	return nil
}

// forstat -> FOR (fornum | forlist) END
func (p *Parser) forstat(fn *FnProto) error {
	tk := p.mustnext(TokenFor)
	name, err := p.ident()
	if err != nil {
		return err
	}
	if p.peek().Kind == TokenAssign {
		return p.fornum(fn, name)
	} else if tk := p.peek().Kind; tk == TokenComma || tk == TokenIn {
		return p.forlist(fn, name)
	}
	return p.parseErrf(tk, "malformed for statment")
}

// fornum -> NAME = exp,exp[,exp] DO
func (p *Parser) fornum(fn *FnProto, name *Token) error {
	tk := p.mustnext(TokenAssign)
	sp0 := p.pushLoopBlock(fn)
	nexprs, lastExpr, lastExprDst, err := p.explist(fn)
	if err != nil {
		return err
	}

	if nexprs < 2 || nexprs > 3 {
		return p.parseErrf(tk, "invalid for stat")
	}

	p.discharge(fn, lastExpr, lastExprDst)
	if nexprs == 2 {
		expr, err := constValToExpression(fn, int64(1), tk.LineInfo)
		if err != nil {
			return err
		}
		p.discharge(fn, expr, fn.stackPointer)
	}

	// add the iterator var, limit, step locals, the last two cannot be directly accessed
	if err := fn.addLocals(name.StringVal, "", ""); err != nil {
		return err
	}
	iforPrep := p.code(fn, iAsBx(FORPREP, sp0, 0))

	if err := p.next(TokenDo); err != nil {
		return err
	} else if err := p.block(fn); err != nil {
		return err
	} else if err := p.next(TokenEnd); err != nil {
		return err
	}

	blockSize := int16(len(fn.ByteCodes) - iforPrep - 1)
	p.code(fn, iAsBx(FORLOOP, sp0, -blockSize-1))
	fn.ByteCodes[iforPrep] = iAsBx(FORPREP, sp0, blockSize)
	p.popLoopBlock(fn)
	return nil
}

// forlist -> NAME {,NAME} IN explist DO
func (p *Parser) forlist(fn *FnProto, firstName *Token) error {
	sp0 := p.pushLoopBlock(fn)
	names := []string{firstName.StringVal}
	if p.peek().Kind == TokenComma {
		p.mustnext(TokenComma)
		name, err := p.ident()
		if err != nil {
			return err
		}
		names = append(names, name.StringVal)
	}
	if err := p.next(TokenIn); err != nil {
		return err
	}

	if err := p.explistWant(fn, 3); err != nil {
		return err
	}
	if err := fn.addLocals("", "", ""); err != nil {
		return err
	}
	if err := fn.addLocals(names...); err != nil {
		return err
	}

	ijmp := p.code(fn, iAsBx(JMP, 0, 0))

	if err := p.next(TokenDo); err != nil {
		return err
	} else if err := p.block(fn); err != nil {
		return err
	} else if err := p.next(TokenEnd); err != nil {
		return err
	}

	fn.ByteCodes[ijmp] = iAsBx(JMP, 0, int16(len(fn.ByteCodes)-ijmp-1))
	p.code(fn, iAB(TFORCALL, sp0, uint8(len(names))))
	p.code(fn, iAsBx(TFORLOOP, sp0+1, -int16(len(fn.ByteCodes)-ijmp)))
	p.popLoopBlock(fn)
	return nil
}

func (p *Parser) repeatstat(fn *FnProto) error {
	p.mustnext(TokenRepeat)
	sp0 := p.pushLoopBlock(fn)
	istart := len(fn.ByteCodes)
	if err := p.block(fn); err != nil {
		return err
	} else if err := p.next(TokenUntil); err != nil {
		return err
	}
	condition, err := p.expr(fn, 0)
	if err != nil {
		return err
	}
	spCondition := fn.stackPointer
	p.discharge(fn, condition, spCondition)
	p.code(fn, iAB(TEST, spCondition, 0))
	p.code(fn, iAsBx(JMP, sp0+1, -int16(len(fn.ByteCodes)-istart)))
	p.popLoopBlock(fn)
	return nil
}

func (p *Parser) breakstat(fn *FnProto) error {
	breakToken := p.mustnext(TokenBreak)
	if len(p.breakBlocks) == 0 {
		return p.parseErrf(breakToken, "use of a break outside of loop")
	}
	p.breakBlocks[len(p.breakBlocks)-1] = append(p.breakBlocks[len(p.breakBlocks)-1], p.code(fn, iAsBx(JMP, 0, 0)))
	return nil
}

// label -> '::' NAME '::'
func (p *Parser) labelstat(fn *FnProto) error {
	p.mustnext(TokenDoubleColon)
	name, err := p.ident()
	if err != nil {
		return err
	}
	label := name.StringVal
	if _, found := fn.Labels[label]; found {
		return p.parseErrf(name, "duplicate label defined: %v", label)
	}
	icode := len(fn.ByteCodes)
	fn.Labels[label] = icode
	if codes, hasGotos := fn.Gotos[label]; hasGotos {
		for _, jmpcode := range codes {
			fn.ByteCodes[jmpcode] = iAsBx(JMP, 0, int16(icode-jmpcode-1))
		}
		delete(fn.Gotos, label)
	}
	return p.next(TokenDoubleColon)
}

// gotostat -> 'goto' NAME
func (p *Parser) gotostat(fn *FnProto) error {
	p.mustnext(TokenGoto)
	if name, err := p.ident(); err != nil {
		return err
	} else if icode, found := fn.Labels[name.StringVal]; found {
		p.code(fn, iAsBx(JMP, 0, -int16(len(fn.ByteCodes)-icode+1)))
	} else {
		fn.Gotos[name.StringVal] = append(fn.Gotos[name.StringVal], p.code(fn, iAsBx(JMP, 0, 0)))
	}
	return nil
}

// localassign -> NAME attrib { ',' NAME attrib } ['=' explist]
func (p *Parser) localassign(fn *FnProto) error {
	lcl0 := uint8(len(fn.locals))
	names := []*exValue{}
	for {
		name, err := p.identWithAttrib(lcl0 + uint8(len(names)))
		if err != nil {
			return err
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
		if err := fn.addLocal(name.name, name.attrConst, name.attrClose); err != nil {
			return err
		}
		p.assignTo(fn, name, sp0+uint8(i))
		if name.attrClose {
			p.code(fn, iAB(TBC, name.address, 0))
		}
	}
	return nil
}

func (p *Parser) explistWant(fn *FnProto, want int) error {
	sp0 := uint8(len(fn.locals))
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
func (p *Parser) ident() (*Token, error) {
	tk, err := p._next()
	if err != nil {
		return nil, err
	} else if tk.Kind != TokenIdentifier {
		return nil, p.parseErrf(tk, "expected Name but found %v", tk.Kind)
	}
	return tk, nil
}

// NAME attrib
// attrib -> ['<' ('const' | 'close') '>']
func (p *Parser) identWithAttrib(dst uint8) (*exValue, error) {
	local, err := p.ident()
	if err != nil {
		return nil, err
	}

	name := &exValue{
		local:    true,
		name:     local.StringVal,
		address:  dst,
		LineInfo: local.LineInfo,
	}

	if p.peek().Kind == TokenLt {
		p.mustnext(TokenLt)
		if tk, err := p._next(); err != nil {
			return nil, err
		} else if tk.Kind != TokenIdentifier {
			return nil, p.parseErrf(tk, "expected attrib but found %v", tk.Kind)
		} else if tk.StringVal == "const" {
			name.attrConst = true
		} else if tk.StringVal == "close" {
			name.attrClose = true
		} else {
			return nil, p.parseErrf(tk, "unknown local attribute %v", tk.StringVal)
		}
		if err := p.next(TokenGt); err != nil {
			return nil, err
		}
	}
	return name, nil
}

// funcargs -> '(' [ explist ] ')' | constructor | STRING
func (p *Parser) funcargs(fn *FnProto) (int, error) {
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
		switch expr := lastExpr.(type) {
		case *exCall:
			expr.nret = 0 // all out
			p.discharge(fn, expr, lastExprDst)
			return -1, p.next(TokenCloseParen) // nargs all in
		case *exVarArgs:
			expr.want = 0 // var args all out
			p.discharge(fn, expr, lastExprDst)
			return -1, p.next(TokenCloseParen) // nargs all in
		}
		p.discharge(fn, lastExpr, lastExprDst)
		return nparams, p.next(TokenCloseParen)
	case TokenOpenCurly:
		_, err := p.constructor(fn)
		return 1, err
	case TokenString:
		tk := p.mustnext(TokenString)
		expr, err := constValToExpression(fn, true, tk.LineInfo)
		if err != nil {
			return 0, err
		}
		p.discharge(fn, expr, fn.stackPointer)
		return 1, nil
	default:
		return 0, p.parseErrf(p.peek(), "unexpected token type %v while evaluating function call", p.peek().Kind)
	}
}

// assignment -> suffixedexp { ',' suffixedexp } '=' explist
func (p *Parser) assignment(fn *FnProto, first expression) error {
	names := []expression{first}
	for p.peek().Kind == TokenComma {
		p.mustnext(TokenComma)
		if expr, err := p.suffixedexp(fn); err != nil {
			return err
		} else {
			names = append(names, expr)
		}
	}
	tk, err := p._next(TokenAssign)
	if err != nil {
		return err
	}
	sp0 := fn.stackPointer
	if err := p.explistWant(fn, len(names)); err != nil {
		return err
	}
	for i, name := range names {
		if err := p.checkConst(tk, name); err != nil {
			return err
		}
		p.assignTo(fn, name, sp0+uint8(i))
	}
	return nil
}

// expr -> (simpleexp | unop subexpr) { binop subexpr }
// where 'binop' is any binary operator with a priority higher than 'limit'
func (p *Parser) expr(fn *FnProto, limit int) (expression, error) {
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
		desc = tokenToUnary(tk, ival)
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
		desc = tokenToBinopExpression(op, lval, p.discharge(fn, desc, fn.stackPointer))
		op = p.peek()
	}
	return desc, nil
}

func (p *Parser) dischargeIfNeed(fn *FnProto, expr expression, dst uint8) uint8 {
	if val, isVal := expr.(*exValue); isVal && val.local {
		return val.address
	}
	return p.discharge(fn, expr, dst)
}

func (p *Parser) dischargeMaybeConst(fn *FnProto, expr expression, dst uint8) (uint8, bool) {
	if kval, isK := expr.(*exConstant); isK {
		return uint8(kval.index), true
	}
	return p.discharge(fn, expr, dst), false
}

func (p *Parser) discharge(fn *FnProto, exp expression, dst uint8) uint8 {
	if call, isCall := exp.(*exCall); isCall {
		call.discharge(fn, dst)
		return call.fn
	}
	exp.discharge(fn, dst)
	fn.stackPointer = dst + 1
	return dst
}

func (p *Parser) code(fn *FnProto, inst Bytecode) int {
	return fn.code(inst, p.lastTokenInfo)
}

// simpleexp -> Float | Integer | String | nil | true | false | ... | constructor | FUNCTION body | suffixedexp
func (p *Parser) simpleexp(fn *FnProto) (expression, error) {
	switch p.peek().Kind {
	case TokenFloat:
		tk := p.mustnext(TokenFloat)
		return constValToExpression(fn, tk.FloatVal, tk.LineInfo)
	case TokenInteger:
		tk := p.mustnext(TokenInteger)
		return constValToExpression(fn, tk.IntVal, tk.LineInfo)
	case TokenString:
		tk := p.mustnext(TokenString)
		return constValToExpression(fn, tk.StringVal, tk.LineInfo)
	case TokenNil:
		tk := p.mustnext(TokenNil)
		return constValToExpression(fn, nil, tk.LineInfo)
	case TokenTrue:
		tk := p.mustnext(TokenTrue)
		return constValToExpression(fn, true, tk.LineInfo)
	case TokenFalse:
		tk := p.mustnext(TokenFalse)
		return constValToExpression(fn, false, tk.LineInfo)
	case TokenOpenCurly:
		return p.constructor(fn)
	case TokenFunction:
		tk := p.mustnext(TokenFunction)
		newFn, err := p.funcbody(fn, "", false, tk.LineInfo)
		return &exClosure{
			fn:       fn.addFn(newFn),
			LineInfo: tk.LineInfo,
		}, err
	case TokenDots:
		tk := p.mustnext(TokenDots)
		return &exVarArgs{
			LineInfo: tk.LineInfo,
		}, nil
	default:
		return p.suffixedexp(fn)
	}
}

// primaryexp -> NAME | '(' expr ')'
func (p *Parser) primaryexp(fn *FnProto) (expression, error) {
	switch p.peek().Kind {
	case TokenOpenParen:
		p.mustnext(TokenOpenParen)
		desc, err := p.expr(fn, nonePriority)
		if err != nil {
			return nil, err
		}
		return desc, p.next(TokenCloseParen)
	case TokenIdentifier:
		return p.name(fn, p.mustnext(TokenIdentifier))
	default:
		return nil, p.parseErrf(p.peek(), "unexpected symbol %v", p.peek().Kind)
	}
}

// suffixedexp -> primaryexp { '.' NAME | '[' exp ']' | ':' NAME funcargs | funcargs }
// funccallstat -> suffixedexp funcargs
func (p *Parser) suffixedexp(fn *FnProto) (expression, error) {
	sp0 := fn.stackPointer
	expr, err := p.primaryexp(fn)
	if err != nil {
		return nil, err
	}
	for {
		switch p.peek().Kind {
		case TokenPeriod:
			p.mustnext(TokenPeriod)
			itable := p.dischargeIfNeed(fn, expr, sp0)
			key, err := p.ident()
			if err != nil {
				return nil, err
			}
			kaddr, err := fn.addConst(key.StringVal)
			if err != nil {
				return nil, err
			}
			expr = &exIndex{
				local:      true,
				table:      itable,
				key:        uint8(kaddr),
				keyIsConst: true,
				LineInfo:   key.LineInfo,
			}
		case TokenOpenBracket:
			tk := p.mustnext(TokenOpenBracket)
			itable := p.dischargeIfNeed(fn, expr, sp0)
			firstexpr, err := p.expr(fn, nonePriority)
			if err != nil {
				return nil, err
			} else if err := p.next(TokenCloseBracket); err != nil {
				return nil, err
			}
			ival, isconst := p.dischargeMaybeConst(fn, firstexpr, sp0+1)
			expr = &exIndex{
				local:      true,
				table:      itable,
				key:        ival,
				keyIsConst: isconst,
				LineInfo:   tk.LineInfo,
			}
		case TokenColon:
			p.mustnext(TokenColon)
			tblIdx := p.dischargeIfNeed(fn, expr, sp0)
			key, err := p.ident()
			if err != nil {
				return nil, err
			}
			kaddr, err := fn.addConst(key.StringVal)
			if err != nil {
				return nil, err
			}
			p.code(fn, iABCK(SELF, sp0, tblIdx, false, uint8(kaddr), true))
			fn.stackPointer++
			nargs, err := p.funcargs(fn)
			if err != nil {
				return nil, err
			}
			expr = &exCall{
				fn:       sp0,
				nret:     2,
				nargs:    uint8(nargs + 2),
				LineInfo: key.LineInfo,
			}
		case TokenOpenParen, TokenString, TokenOpenCurly:
			tk := p.peek()
			ifn := p.dischargeIfNeed(fn, expr, sp0)
			nargs, err := p.funcargs(fn)
			if err != nil {
				return nil, err
			}
			expr = &exCall{
				fn:       uint8(ifn),
				nret:     2,
				nargs:    uint8(nargs + 1),
				LineInfo: tk.LineInfo,
			}
		default:
			return expr, nil
		}
	}
}

// name is a reference to a variable that need resolution to have meaning
func (p *Parser) name(fn *FnProto, name *Token) (expression, error) {
	if expr, err := p.resolveVar(fn, name); err != nil {
		return nil, err
	} else if expr != nil {
		return expr, nil
	}
	iname, err := fn.addConst(name.StringVal)
	if err != nil {
		return nil, err
	}
	expr, err := p.name(fn, &Token{StringVal: "_ENV", LineInfo: LineInfo{Line: name.Line, Column: name.Column}})
	if err != nil {
		return nil, err
	}
	value, isValue := expr.(*exValue)
	if !isValue {
		panic("did not find _ENV, this should never happen")
	}
	return &exIndex{
		local:      value.local,
		table:      value.address,
		key:        uint8(iname),
		keyIsConst: true,
		LineInfo:   name.LineInfo,
	}, nil
}

// resolveVar will recursively look up the stack to find where the variable
// resides in the stack and then build the chain of upvars to have a referece
// to it.
func (p *Parser) resolveVar(fn *FnProto, name *Token) (expression, error) {
	if fn == nil {
		return nil, nil
	} else if idx, ok := search(fn.locals, name.StringVal, findLocal); ok {
		lcl := fn.locals[idx]
		return &exValue{
			local:     true,
			name:      name.StringVal,
			address:   uint8(idx),
			lvar:      lcl,
			attrConst: lcl.attrConst,
			attrClose: lcl.attrClose,
			LineInfo:  name.LineInfo,
		}, nil
	} else if idx, ok := search(fn.UpIndexes, name.StringVal, findUpindex); ok {
		return &exValue{
			local:    false,
			name:     name.StringVal,
			address:  uint8(idx),
			LineInfo: name.LineInfo,
		}, nil
	} else if expr, err := p.resolveVar(fn.prev, name); err != nil {
		return nil, err
	} else if expr != nil {
		if value, isValue := expr.(*exValue); isValue && value.local {
			value.lvar.upvalRef = true
			if err := fn.addUpindex(name.StringVal, uint(value.address), true); err != nil {
				return nil, err
			}
		} else if isValue {
			if err := fn.addUpindex(name.StringVal, uint(value.address), false); err != nil {
				return nil, err
			}
		}
		return &exValue{
			local:    false,
			name:     name.StringVal,
			address:  uint8(len(fn.UpIndexes) - 1),
			LineInfo: name.LineInfo,
		}, nil
	}
	return nil, nil
}

// explist -> expr { ',' expr }
// this will ensure that after evaluation, the final values are placed at
// fn.stackPointer, fn.stackPointer+1,fn.stackPointer+2......
// no matter how much of the stack was used up during computation of the expr
func (p *Parser) explist(fn *FnProto) (int, expression, uint8, error) {
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
func (p *Parser) constructor(fn *FnProto) (expression, error) {
	tk := p.mustnext(TokenOpenCurly)
	itable := fn.stackPointer
	tablecode := p.code(fn, iAB(NEWTABLE, 0, 0))
	fn.stackPointer++
	numvals, totalVals, numfields := 0, 0, 0
	tableIndex := uint64(1)

	dischargeValues := func() error {
		if tableIndex > math.MaxUint8 && tableIndex <= math.MaxUint32 {
			p.code(fn, iABC(SETLIST, itable, uint8(numvals+1), 0))
			p.code(fn, Bytecode(tableIndex))
		} else if tableIndex > math.MaxUint32 {
			return p.parseErr(tk, fmt.Errorf("table index overflow"))
		} else {
			p.code(fn, iABC(SETLIST, itable, uint8(numvals+1), uint8(tableIndex)))
		}
		tableIndex += uint64(numvals)
		numvals = 0
		fn.stackPointer = itable + 1
		return nil
	}

	for {
		switch p.peek().Kind {
		case TokenCloseCurly:
			// do nothing, because it is an empty table
		case TokenIdentifier:
			key, err := p.ident()
			if err != nil {
				return nil, err
			} else if err := p.next(TokenAssign); err != nil {
				return nil, err
			}
			desc, err := p.expr(fn, 0)
			if err != nil {
				return nil, err
			}
			kaddr, err := fn.addConst(key.StringVal)
			if err != nil {
				return nil, err
			}
			if kexp, isConst := desc.(*exConstant); isConst {
				p.code(fn, iABCK(SETTABLE, itable, uint8(kaddr), true, uint8(kexp.index), true))
			} else {
				p.code(fn, iABCK(SETTABLE, itable, uint8(kaddr), true, p.discharge(fn, desc, fn.stackPointer), false))
			}
			numfields++
			fn.stackPointer = itable + uint8(numvals) + 1
		case TokenOpenBracket:
			p.mustnext(TokenOpenBracket)
			keydesc, err := p.expr(fn, 0)
			if err != nil {
				return nil, err
			} else if err := p.next(TokenCloseBracket); err != nil {
				return nil, err
			} else if err := p.next(TokenAssign); err != nil {
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
			p.code(fn, iABCK(SETTABLE, itable, ikey, keyConst, ival, valConst))
			numfields++
			fn.stackPointer = itable + uint8(numvals) + 1
		default:
			desc, err := p.expr(fn, 0)
			if err != nil {
				return nil, err
			}
			p.discharge(fn, desc, fn.stackPointer)
			numvals++
			totalVals++
		}

		if tk := p.peek(); tk.Kind == TokenComma || tk.Kind == TokenSemiColon {
			if err := p.next(); err != nil {
				return nil, err
			}
		} else {
			break
		}
		if numvals+1 == math.MaxUint8 {
			if err := dischargeValues(); err != nil {
				return nil, err
			}
		}
	}

	if numvals > 0 {
		if err := dischargeValues(); err != nil {
			return nil, err
		}
	}
	fn.stackPointer = itable + 1
	fn.ByteCodes[tablecode] = iABC(NEWTABLE, itable, uint8(totalVals), uint8(numfields))
	return &exValue{
		local:    true,
		address:  uint8(itable),
		LineInfo: tk.LineInfo,
	}, p.next(TokenCloseCurly)
}

func (p *Parser) pushLoopBlock(fn *FnProto) uint8 {
	p.breakBlocks = append(p.breakBlocks, []int{})
	p.localsScope = append(p.localsScope, fn.stackPointer)
	return fn.stackPointer
}

func (p *Parser) popLoopBlock(fn *FnProto) {
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

func (p *Parser) localExpire(fn *FnProto, from uint8) {
	for _, local := range truncate(&fn.locals, int(from)) {
		if local.upvalRef {
			p.code(fn, iAB(CLOSE, from, 0))
			break
		}
	}
	fn.stackPointer = from
}
