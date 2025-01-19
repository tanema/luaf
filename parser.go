package luaf

import (
	"fmt"
	"io"
	"reflect"
	"strings"
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

func NewParser() *Parser {
	return &Parser{
		rootfn:      newFnProto("", "env", nil, []string{"_ENV"}, false, LineInfo{}),
		breakBlocks: [][]int{},
		localsScope: []uint8{},
	}
}

func Parse(filename string, src io.Reader) (*FnProto, error) {
	p := NewParser()
	fn := newFnProto(filename, "main chunk", p.rootfn, []string{}, true, LineInfo{})
	return fn, p.Parse(filename, src, fn)
}

// Parse will reset the parser but parse the source within the context of this
// function. This allows parsing in repl and still be able to have visibility
// of locals.
func (p *Parser) Parse(filename string, src io.Reader, fn *FnProto) error {
	p.filename = filename
	p.lex = NewLexer(src)
	if err := p.block(fn); err != nil {
		return err
	}
	if err := p.next(TokenEOS); err != io.EOF {
		return err
	}
	if len(fn.ByteCodes) == 0 || fn.ByteCodes[len(fn.ByteCodes)-1].op() != RETURN {
		p.code(fn, iAB(RETURN, 0, 1))
	}
	return nil
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
		expected := make([]string, len(tt))
		for i, t := range tt {
			expected[i] = string(t)
		}
		return nil, p.parseErrf(tk, "expected %v but consumed %v", strings.Join(expected, ","), tk.Kind)
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
			_, err := p.discharge(fn, call)
			return err
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
	_, err = p.dischargeTo(fn, &exClosure{fn: fn.addFn(newFn), LineInfo: name.LineInfo}, ifn)
	return err
}

// funcstat -> FUNCTION funcname funcbody
func (p *Parser) funcstat(fn *FnProto) error {
	tk := p.mustnext(TokenFunction)
	name, hasSelf, fullname, err := p.funcname(fn)
	if err != nil {
		return err
	}
	newFn, err := p.funcbody(fn, fullname, hasSelf, tk.LineInfo)
	if err != nil {
		return err
	}
	closure := &exClosure{fn: fn.addFn(newFn), LineInfo: tk.LineInfo}
	icls, err := p.discharge(fn, closure)
	if err != nil {
		return err
	}
	return p.assignTo(fn, tk, name, icls)
}

func (p *Parser) assignTo(fn *FnProto, tk *Token, dst expression, from uint8) error {
	switch ex := dst.(type) {
	case *exVariable:
		if ex.attrConst {
			return p.parseErrf(tk, "attempt to assign to const variable '%v'", ex.name)
		} else if !ex.local {
			fn.code(iAB(SETUPVAL, ex.address, from), ex.LineInfo)
		} else {
			fn.code(iAB(MOVE, ex.address, from), ex.LineInfo)
		}
		return nil
	case *exIndex:
		ikey, keyIsConst, err := dischargeMaybeConst(fn, ex.key, fn.stackPointer)
		if err != nil {
			return err
		}
		if val, isVal := ex.table.(*exVariable); isVal {
			if val.local {
				fn.code(iABCK(SETTABLE, val.address, ikey, keyIsConst, from, false), ex.LineInfo)
			} else {
				fn.code(iABCK(SETTABUP, val.address, ikey, keyIsConst, from, false), ex.LineInfo)
			}
			return nil
		}
		err = ex.table.discharge(fn, ikey+1)
		fn.code(iABCK(SETTABLE, ikey+1, ikey, keyIsConst, from, false), ex.LineInfo)
		return err
	default:
		panic(fmt.Sprintf("unknown expression to assign to %T", dst))
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
			ident, err := p.ident()
			if err != nil {
				return nil, false, "", err
			}
			fullname += "." + ident.StringVal
			name = &exIndex{
				table:    name,
				key:      &exString{val: ident.StringVal, LineInfo: ident.LineInfo},
				LineInfo: ident.LineInfo,
			}
		case TokenColon:
			p.mustnext(TokenColon)
			ident, err := p.ident()
			if err != nil {
				return nil, false, "", err
			}
			fullname += ":" + ident.StringVal
			return &exIndex{
				table:    name,
				key:      &exString{val: ident.StringVal, LineInfo: ident.LineInfo},
				LineInfo: ident.LineInfo,
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
	if len(newFn.ByteCodes) == 0 || newFn.ByteCodes[len(newFn.ByteCodes)-1].op() != RETURN {
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
	// if we are at the end of block then there are no return vals
	if p.blockFollow(true) {
		p.code(fn, iAB(RETURN, sp0, 1))
		return nil
	}
	exprs, err := p.explist(fn)
	if err != nil {
		return err
	}
	lastExpr, err := p.dischargeAllButLast(fn, exprs)
	if err != nil {
		return err
	}
	switch expr := lastExpr.(type) {
	case *exCall:
		expr.tail = true
		if _, err := p.dischargeTo(fn, expr, sp0); err != nil {
			return err
		}
		p.code(fn, iAB(RETURN, 0, 0))
	case *exVarArgs:
		if _, err := p.discharge(fn, expr); err != nil {
			return err
		}
		p.code(fn, iAB(RETURN, sp0, 0))
	default:
		if _, err := p.discharge(fn, expr); err != nil {
			return err
		}
		p.code(fn, iAB(RETURN, sp0, uint8(len(exprs)+1)))
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
	spCondition, err := p.discharge(fn, condition)
	if err != nil {
		return err
	}
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
	defer p.popLoopBlock(fn)

	istart := int16(len(fn.ByteCodes))
	condition, err := p.expr(fn, 0)
	if err != nil {
		return err
	} else if err := p.next(TokenDo); err != nil {
		return err
	}
	spCondition, err := p.discharge(fn, condition)
	if err != nil {
		return err
	}
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
	defer p.popLoopBlock(fn)

	if exprs, err := p.explist(fn); err != nil {
		return err
	} else if len(exprs) < 2 || len(exprs) > 3 {
		return p.parseErrf(tk, "invalid for stat, expected 2-3 expressions.")
	} else if lastExpr, err := p.dischargeAllButLast(fn, exprs); err != nil {
		return err
	} else if _, err := p.discharge(fn, lastExpr); err != nil {
		return err
	} else if len(exprs) == 2 {
		if _, err := p.discharge(fn, &exInteger{val: 1}); err != nil {
			return err
		}
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
	return nil
}

// forlist -> NAME {,NAME} IN explist DO
func (p *Parser) forlist(fn *FnProto, firstName *Token) error {
	sp0 := p.pushLoopBlock(fn)
	defer p.popLoopBlock(fn)

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

	exprs, err := p.explistWant(fn, 3)
	if err != nil {
		return err
	} else if err := fn.addLocals("", "", ""); err != nil {
		return err
	} else if err := fn.addLocals(names...); err != nil {
		return err
	}

	for i, expr := range exprs {
		if _, err := p.dischargeTo(fn, expr, sp0+uint8(i)); err != nil {
			return err
		}
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
	return nil
}

func (p *Parser) repeatstat(fn *FnProto) error {
	p.mustnext(TokenRepeat)
	sp0 := p.pushLoopBlock(fn)
	defer p.popLoopBlock(fn)

	istart := len(fn.ByteCodes)
	if err := p.block(fn); err != nil {
		return err
	} else if err := p.next(TokenUntil); err != nil {
		return err
	} else if condition, err := p.expr(fn, 0); err != nil {
		return err
	} else if spCondition, err := p.discharge(fn, condition); err != nil {
		return err
	} else {
		p.code(fn, iAB(TEST, spCondition, 0))
		p.code(fn, iAsBx(JMP, sp0+1, -int16(len(fn.ByteCodes)-istart+1)))
	}
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
	names := uint8(0)
	for {
		local, err := p.ident()
		if err != nil {
			return err
		}

		name := &exVariable{
			local:    true,
			name:     local.StringVal,
			address:  lcl0 + names,
			LineInfo: local.LineInfo,
		}

		if err := p.localAttrib(name); err != nil {
			return err
		} else if err := fn.addLocal(name.name, name.attrConst, name.attrClose); err != nil {
			return err
		}

		if name.attrClose {
			p.code(fn, iAB(TBC, name.address, 0))
		}

		names++
		if p.peek().Kind != TokenComma {
			break
		}
		p.mustnext(TokenComma)
	}

	if p.peek().Kind != TokenAssign {
		_, err := p.dischargeTo(fn, &exNil{num: uint16(names - 1)}, lcl0)
		return err
	}
	p.mustnext(TokenAssign)

	fn.stackPointer = lcl0 // TODO I think this is a hack not good
	exprs, err := p.explistWant(fn, int(names))
	if err != nil {
		return err
	}
	for i, expr := range exprs {
		if _, err := p.dischargeTo(fn, expr, lcl0+uint8(i)); err != nil {
			return err
		}
	}
	return nil
}

func (p *Parser) explistWant(fn *FnProto, want int) ([]expression, error) {
	exprs, err := p.explist(fn)
	if err != nil {
		return nil, err
	} else if diff := uint8(want - len(exprs)); diff > 0 {
		switch expr := exprs[len(exprs)-1].(type) {
		case *exCall:
			expr.nret = diff + 2
		case *exVarArgs:
			expr.want = diff + 2
		default:
			exprs = append(exprs, &exNil{num: uint16(diff)})
		}
	}
	return exprs, nil
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
func (p *Parser) localAttrib(name *exVariable) error {
	if p.peek().Kind != TokenLt {
		return nil
	}

	p.mustnext(TokenLt)
	if tk, err := p._next(); err != nil {
		return err
	} else if tk.Kind != TokenIdentifier {
		return p.parseErrf(tk, "expected attrib but found %v", tk.Kind)
	} else if tk.StringVal == "const" {
		name.attrConst = true
		return p.next(TokenGt)
	} else if tk.StringVal == "close" {
		name.attrClose = true
		return p.next(TokenGt)
	} else {
		return p.parseErrf(tk, "unknown local attribute %v", tk.StringVal)
	}
}

// funcargs -> '(' [ explist ] ')' | constructor | STRING
func (p *Parser) funcargs(fn *FnProto) ([]expression, error) {
	switch p.peek().Kind {
	case TokenOpenParen:
		p.mustnext(TokenOpenParen)
		if p.peek().Kind == TokenCloseParen {
			p.mustnext(TokenCloseParen)
			return []expression{}, nil
		}
		exprs, err := p.explist(fn)
		if err != nil {
			return nil, err
		}
		return exprs, p.next(TokenCloseParen)
	case TokenOpenCurly:
		expr, err := p.constructor(fn)
		return []expression{expr}, err
	case TokenString:
		tk := p.mustnext(TokenString)
		return []expression{&exString{LineInfo: tk.LineInfo, val: tk.StringVal}}, nil
	default:
		return nil, p.parseErrf(p.peek(), "unexpected token type %v while evaluating function call", p.peek().Kind)
	}
}

// assignment -> suffixedexp { ',' suffixedexp } '=' explist
func (p *Parser) assignment(fn *FnProto, first expression) error {
	sp0 := fn.stackPointer
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
	exprs, err := p.explistWant(fn, len(names))
	if err != nil {
		return err
	}
	for i, expr := range exprs {
		if _, err := p.dischargeTo(fn, expr, sp0+uint8(i)); err != nil {
			return err
		}
	}
	for i, name := range names {
		if err := p.assignTo(fn, tk, name, sp0+uint8(i)); err != nil {
			return err
		}
	}
	return nil
}

// expr -> (simpleexp | unop expr) { binop expr }
// where 'binop' is any binary operator with a priority higher than 'limit'
func (p *Parser) expr(fn *FnProto, limit int) (desc expression, err error) {
	if tk := p.peek(); tk.isUnary() {
		if err = p.next(); err != nil {
			return nil, err
		} else if desc, err = p.expr(fn, unaryPriority); err != nil {
			return nil, err
		}
		desc = unaryExpression(fn, tk, desc)
	} else if desc, err = p.simpleexp(fn); err != nil {
		return nil, err
	}
	for op := p.peek(); op.isBinary() && binaryPriority[op.Kind][0] > limit; op = p.peek() {
		p.mustnext(op.Kind)
		rdesc, err := p.expr(fn, binaryPriority[op.Kind][1])
		if err != nil {
			return nil, err
		}
		desc = constFold(&exInfixOp{
			operand:  op.Kind,
			left:     desc,
			right:    rdesc,
			LineInfo: op.LineInfo,
		})
	}
	return desc, nil
}

func (p *Parser) dischargeAllButLast(fn *FnProto, exprs []expression) (expression, error) {
	for i := 0; i < len(exprs)-1; i++ {
		if _, err := p.discharge(fn, exprs[i]); err != nil {
			return nil, err
		}
	}
	return exprs[len(exprs)-1], nil
}

func (p *Parser) discharge(fn *FnProto, exp expression) (uint8, error) {
	return p.dischargeTo(fn, exp, fn.stackPointer)
}

func (p *Parser) dischargeTo(fn *FnProto, exp expression, dst uint8) (uint8, error) {
	err := exp.discharge(fn, dst)
	fn.stackPointer = dst + 1
	return dst, err
}

func (p *Parser) code(fn *FnProto, inst Bytecode) int {
	return fn.code(inst, p.lastTokenInfo)
}

// simpleexp -> Float | Integer | String | nil | true | false | ... | constructor | FUNCTION body | suffixedexp
func (p *Parser) simpleexp(fn *FnProto) (expression, error) {
	switch p.peek().Kind {
	case TokenFloat:
		tk := p.mustnext(TokenFloat)
		return &exFloat{LineInfo: tk.LineInfo, val: tk.FloatVal}, nil
	case TokenInteger:
		tk := p.mustnext(TokenInteger)
		return &exInteger{LineInfo: tk.LineInfo, val: tk.IntVal}, nil
	case TokenString:
		tk := p.mustnext(TokenString)
		return &exString{LineInfo: tk.LineInfo, val: tk.StringVal}, nil
	case TokenNil:
		tk := p.mustnext(TokenNil)
		return &exNil{LineInfo: tk.LineInfo, num: 1}, nil
	case TokenTrue:
		tk := p.mustnext(TokenTrue)
		return &exBool{LineInfo: tk.LineInfo, val: true}, nil
	case TokenFalse:
		tk := p.mustnext(TokenFalse)
		return &exBool{LineInfo: tk.LineInfo, val: false}, nil
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
	ch, err := p._next()
	if err != nil {
		return nil, err
	}
	switch ch.Kind {
	case TokenOpenParen:
		desc, err := p.expr(fn, nonePriority)
		if err != nil {
			return nil, err
		}
		return desc, p.next(TokenCloseParen)
	case TokenIdentifier:
		return p.name(fn, ch)
	default:
		return nil, p.parseErrf(p.peek(), "unexpected symbol %v", ch.Kind)
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
			key, err := p.ident()
			if err != nil {
				return nil, err
			}
			expr = &exIndex{
				table:    expr,
				key:      &exString{val: key.StringVal, LineInfo: key.LineInfo},
				LineInfo: key.LineInfo,
			}
		case TokenOpenBracket:
			tk := p.mustnext(TokenOpenBracket)
			key, err := p.expr(fn, nonePriority)
			if err != nil {
				return nil, err
			} else if err := p.next(TokenCloseBracket); err != nil {
				return nil, err
			}
			expr = &exIndex{
				table:    expr,
				key:      key,
				LineInfo: tk.LineInfo,
			}
		case TokenColon:
			p.mustnext(TokenColon)
			key, err := p.ident()
			if err != nil {
				return nil, err
			}
			args, err := p.funcargs(fn)
			if err != nil {
				return nil, err
			}
			fn := &exIndex{
				table:    expr,
				key:      &exString{val: key.StringVal, LineInfo: key.LineInfo},
				LineInfo: key.LineInfo,
			}
			expr = newCallExpr(fn, args, true, key.LineInfo)
		case TokenOpenParen, TokenString, TokenOpenCurly:
			tk := p.peek()
			args, err := p.funcargs(fn)
			if err != nil {
				return nil, err
			}
			expr = newCallExpr(expr, args, false, tk.LineInfo)
		default:
			fn.stackPointer = sp0
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
	expr, err := p.name(fn, &Token{StringVal: "_ENV", LineInfo: LineInfo{Line: name.Line, Column: name.Column}})
	if err != nil {
		return nil, err
	} else if _, isValue := expr.(*exVariable); !isValue {
		panic("did not find _ENV, this should never happen")
	}
	return &exIndex{
		table:    expr,
		key:      &exString{val: name.StringVal, LineInfo: name.LineInfo},
		LineInfo: name.LineInfo,
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
		return &exVariable{
			local:     true,
			name:      name.StringVal,
			address:   uint8(idx),
			lvar:      lcl,
			attrConst: lcl.attrConst,
			attrClose: lcl.attrClose,
			LineInfo:  name.LineInfo,
		}, nil
	} else if idx, ok := search(fn.UpIndexes, name.StringVal, findUpindex); ok {
		return &exVariable{
			local:    false,
			name:     name.StringVal,
			address:  uint8(idx),
			LineInfo: name.LineInfo,
		}, nil
	} else if expr, err := p.resolveVar(fn.prev, name); err != nil {
		return nil, err
	} else if expr != nil {
		if value, isValue := expr.(*exVariable); isValue && value.local {
			value.lvar.upvalRef = true
			if err := fn.addUpindex(name.StringVal, uint(value.address), true); err != nil {
				return nil, err
			}
		} else if isValue {
			if err := fn.addUpindex(name.StringVal, uint(value.address), false); err != nil {
				return nil, err
			}
		}
		return &exVariable{
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
func (p *Parser) explist(fn *FnProto) ([]expression, error) {
	list := []expression{}
	for {
		if expr, err := p.expr(fn, nonePriority); err != nil {
			return nil, err
		} else {
			list = append(list, expr)
		}
		if p.peek().Kind != TokenComma {
			break
		}
		p.mustnext(TokenComma)
	}
	return list, nil
}

// constructor -> '{' [ field { sep field } [sep] ] '}'
// sep         -> ',' | ';'
// field -> NAME = exp | '['exp']' = exp | exp
func (p *Parser) constructor(fn *FnProto) (expression, error) {
	expr := &exTable{LineInfo: p.mustnext(TokenOpenCurly).LineInfo}
	for {
		switch p.peek().Kind {
		case TokenCloseCurly:
			// do nothing, because it is an empty table
		case TokenIdentifier:
			if key, err := p.ident(); err != nil {
				return nil, err
			} else if err := p.next(TokenAssign); err != nil {
				return nil, err
			} else if val, err := p.expr(fn, 0); err != nil {
				return nil, err
			} else {
				expr.fields = append(expr.fields, tableField{key: &exString{val: key.StringVal}, val: val})
			}
		case TokenOpenBracket:
			p.mustnext(TokenOpenBracket)
			if key, err := p.expr(fn, 0); err != nil {
				return nil, err
			} else if err := p.next(TokenCloseBracket); err != nil {
				return nil, err
			} else if err := p.next(TokenAssign); err != nil {
				return nil, err
			} else if val, err := p.expr(fn, 0); err != nil {
				return nil, err
			} else {
				expr.fields = append(expr.fields, tableField{key: key, val: val})
			}
		default:
			if val, err := p.expr(fn, 0); err != nil {
				return nil, err
			} else {
				expr.array = append(expr.array, val)
			}
		}
		if tk := p.peek(); tk.Kind == TokenComma || tk.Kind == TokenSemiColon {
			p.mustnext(tk.Kind)
		} else {
			break
		}
	}
	return expr, p.next(TokenCloseCurly)
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
