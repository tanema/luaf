package luaf

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
)

type (
	ParseConfig struct { // not implemented yet
		StringCoers bool // disallow string coersion in arith
		RequireOnly bool // require std libs instead of available by default
		EnvReadonly bool // not allowed to change _ENV
		LocalOnly   bool // not allowed to define globals only locals
		Strict      bool // stringer type checking and throw parsing errors if types are bad
	}
	Parser struct {
		config        ParseConfig
		filename      string
		lastComment   string
		rootfn        *FnProto
		lex           *lexer
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
		config:      ParseConfig{},
		rootfn:      newFnProto("", "env", nil, []string{"_ENV"}, false, LineInfo{}),
		breakBlocks: [][]int{},
		localsScope: []uint8{},
	}
}

func ParseFile(path string, mode LoadMode) (*FnProto, error) {
	src, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer src.Close()
	return Parse(path, src, mode)
}

func Parse(filename string, src io.ReadSeeker, mode LoadMode) (*FnProto, error) {
	if hasLuaBinPrefix(src) && mode&ModeBinary == ModeBinary {
		return UndumpFnProto(src)
	}
	p := NewParser()
	fn := newFnProto(filename, "<main>", p.rootfn, []string{}, true, LineInfo{})
	return fn, p.Parse(filename, src, fn)
}

// Parse will reset the parser but parse the source within the context of this
// function. This allows parsing in repl and still be able to have visibility
// of locals.
func (p *Parser) Parse(filename string, src io.Reader, fn *FnProto) error {
	p.filename = filename
	p.lex = newLexer(src)
	if err := p.chunk(fn); err != nil {
		return err
	} else if err := p.next(tokenEOS); err != io.EOF {
		return err
	}
	if len(fn.ByteCodes) == 0 || fn.ByteCodes[len(fn.ByteCodes)-1].op() != RETURN {
		p.code(fn, iAB(RETURN, 0, 1))
	}
	return fn.checkGotos(p)
}

func (p *Parser) parseErrf(tk *token, msg string, data ...any) error {
	return p.parseErr(tk, fmt.Errorf(msg, data...))
}

func (p *Parser) parseErr(tk *token, err error) error {
	var linfo LineInfo
	if tk != nil {
		linfo = tk.LineInfo
	}
	if lexErr, isLexErr := err.(*lexerError); isLexErr {
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

func (p *Parser) peek() *token {
	return p.lex.Peek()
}

func (p *Parser) _next(tt tokenType) (*token, error) {
	tk, err := p.lex.Next()
	if err != nil {
		return nil, p.parseErr(tk, err)
	} else if tt != tk.Kind {
		return nil, p.parseErrf(tk, "expected %v but consumed %v", tt, tk.Kind)
	}
	p.lastTokenInfo = tk.LineInfo
	return tk, nil
}

func (p *Parser) next(tt tokenType) error {
	_, err := p._next(tt)
	return err
}

// This is used when the token has already been peeked but lets panic just in
// case something goes funky
func (p *Parser) mustnext(tt tokenType) *token {
	tk, err := p._next(tt)
	if err != nil {
		panic(err)
	}
	return tk
}

func (p *Parser) beforeblock(fn *FnProto, breakable bool) {
	if breakable {
		p.breakBlocks = append(p.breakBlocks, []int{})
	}
	p.localsScope = append(p.localsScope, uint8(len(fn.locals)))
	fn.labels = append(fn.labels, map[string]labelEntry{})
}

func (p *Parser) afterblock(fn *FnProto, breakable bool) {
	from := p.localsScope[len(p.localsScope)-1]

	if breakable {
		breaks := p.breakBlocks[len(p.breakBlocks)-1]
		endDst := len(fn.ByteCodes)
		for _, idx := range breaks {
			fn.ByteCodes[idx] = iABx(JMP, from+1, uint16(endDst-idx))
		}
		p.breakBlocks = p.breakBlocks[:len(p.breakBlocks)-1]
	}

	p.localsScope = p.localsScope[:len(p.localsScope)-1]
	expired := truncate(&fn.locals, int(from))
	for _, local := range expired {
		if local.upvalRef {
			p.code(fn, iAB(CLOSE, from, 0))
			break
		}
	}
	fn.stackPointer = from
	fn.labels = fn.labels[:len(fn.labels)-1]
}

func (p *Parser) chunk(fn *FnProto) error {
	fn.labels = append(fn.labels, map[string]labelEntry{})
	defer func() {
		fn.labels = fn.labels[:len(fn.labels)-1]
	}()
	return p.statList(fn)
}

// block -> statlist
func (p *Parser) block(fn *FnProto, breakable bool) error {
	p.beforeblock(fn, breakable)
	defer p.afterblock(fn, breakable)
	return p.statList(fn)
}

// statlist -> { stat [';'] }
func (p *Parser) statList(fn *FnProto) error {
	p.skipComments()
	for !p.blockFollow(true) {
		if p.peek().Kind == tokenReturn {
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
	case tokenElse, tokenElseif, tokenEnd, tokenEOS:
		return true
	case tokenUntil:
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
	tk := p.peek()
	switch tk.Kind {
	case tokenSemiColon:
		return p.next(tokenSemiColon)
	case tokenComment:
		if tk := p.mustnext(tokenComment); strings.HasPrefix(tk.StringVal, "!") {
			return p.configComment(tk)
		} else {
			p.lastComment = tk.StringVal
		}
		return nil
	case tokenLocal:
		return p.localstat(fn)
	case tokenFunction:
		return p.funcstat(fn)
	case tokenReturn:
		return p.retstat(fn)
	case tokenDo:
		return p.dostat(fn)
	case tokenIf:
		return p.ifstat(fn)
	case tokenWhile:
		return p.whilestat(fn)
	case tokenFor:
		return p.forstat(fn)
	case tokenRepeat:
		return p.repeatstat(fn)
	case tokenDoubleColon:
		return p.labelstat(fn)
	case tokenBreak:
		return p.breakstat(fn)
	case tokenGoto:
		return p.gotostat(fn)
	default:
		tk := p.lex.Peek()
		expr, err := p.suffixedexp(fn)
		if err != nil {
			return err
		} else if call, isCall := expr.(*exCall); isCall {
			_, err := p.discharge(fn, call)
			return err
		} else if tk := p.peek(); tk.Kind == tokenAssign || tk.Kind == tokenComma {
			return p.assignment(fn, expr)
		}
		return p.parseErrf(tk, "unexpected expression %v", reflect.TypeOf(expr))
	}
}

func (p *Parser) configComment(comment *token) error {
	config := strings.TrimPrefix(comment.StringVal, "!")
	for _, cfg := range strings.Split(config, ",") {
		cfg = strings.TrimSpace(cfg)
		enabled := !strings.HasPrefix(cfg, "no")
		switch strings.TrimPrefix(cfg, "no") {
		case "stringCoers":
			p.config.StringCoers = enabled
		case "requireOnly":
			p.config.RequireOnly = enabled
		case "envReadonly":
			p.config.EnvReadonly = enabled
		case "localOnly":
			p.config.LocalOnly = enabled
		case "strict":
			p.config.Strict = enabled
		}
	}
	return nil
}

// localstat -> LOCAL [localfunc | localassign]
func (p *Parser) localstat(fn *FnProto) error {
	p.mustnext(tokenLocal)
	if p.peek().Kind == tokenFunction {
		return p.localfunc(fn)
	}
	return p.localassign(fn)
}

// localfunc -> FUNCTION NAME funcbody
func (p *Parser) localfunc(fn *FnProto) error {
	p.mustnext(tokenFunction)
	ifn := uint8(len(fn.locals))
	name, err := p._next(tokenIdentifier)
	if err != nil {
		return err
	}
	if err := fn.addLocal(&local{name: name.StringVal}); err != nil {
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
	tk := p.mustnext(tokenFunction)
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

func (p *Parser) assignTo(fn *FnProto, tk *token, dst expression, from uint8) error {
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
		itable, err := p.discharge(fn, ex.table)
		fn.code(iABCK(SETTABLE, itable, ikey, keyIsConst, from, false), ex.LineInfo)
		return err
	default:
		panic(fmt.Sprintf("unknown expression to assign to %T", dst))
	}
}

// funcname -> NAME {fieldsel} [':' NAME]
// fieldsel     -> ['.' | ':'] NAME
func (p *Parser) funcname(fn *FnProto) (expression, bool, string, error) {
	ident, err := p._next(tokenIdentifier)
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
		case tokenPeriod:
			p.mustnext(tokenPeriod)
			ident, err := p._next(tokenIdentifier)
			if err != nil {
				return nil, false, "", err
			}
			fullname += "." + ident.StringVal
			name = &exIndex{
				table:    name,
				key:      &exString{val: ident.StringVal, LineInfo: ident.LineInfo},
				LineInfo: ident.LineInfo,
			}
		case tokenColon:
			p.mustnext(tokenColon)
			ident, err := p._next(tokenIdentifier)
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
	newFn.Comment = p.lastComment
	p.lastComment = ""
	if err := p.block(newFn, false); err != nil {
		return nil, err
	}
	if len(newFn.ByteCodes) == 0 || newFn.ByteCodes[len(newFn.ByteCodes)-1].op() != RETURN {
		p.code(newFn, iAB(RETURN, 0, 1))
	}
	if err := fn.checkGotos(p); err != nil {
		return nil, err
	}
	return newFn, p.next(tokenEnd)
}

// parlist -> '(' [ {NAME ','} (NAME | '...') ] ')'
func (p *Parser) parlist() ([]string, bool, error) {
	if err := p.next(tokenOpenParen); err != nil {
		return nil, false, err
	}
	names := []string{}
	if p.peek().Kind == tokenCloseParen {
		return names, false, p.next(tokenCloseParen)
	}
	for {
		if p.peek().Kind != tokenIdentifier {
			break
		}
		name, err := p._next(tokenIdentifier)
		if err != nil {
			return nil, false, err
		}
		names = append(names, name.StringVal)
		if p.peek().Kind != tokenComma {
			break
		}
		p.mustnext(tokenComma)
	}
	varargs := false
	if p.peek().Kind == tokenDots {
		p.mustnext(tokenDots)
		varargs = true
	}
	return names, varargs, p.next(tokenCloseParen)
}

// retstat -> RETURN [explist] [';']
func (p *Parser) retstat(fn *FnProto) error {
	p.mustnext(tokenReturn)
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
	// consume any comments after the return because the parse is pretty strict about
	// the return being the end of the block, but comments should be allowed
	p.skipComments()
	return nil
}

func (p *Parser) skipComments() {
	for p.peek().Kind == tokenComment {
		p.mustnext(tokenComment)
	}
}

// dostat -> DO block END
func (p *Parser) dostat(fn *FnProto) error {
	p.mustnext(tokenDo)
	if err := p.block(fn, false); err != nil {
		return err
	}
	return p.next(tokenEnd)
}

// ifstat -> IF exp THEN block {ELSEIF exp THEN block} [ELSE block] END
func (p *Parser) ifstat(fn *FnProto) error {
	p.mustnext(tokenIf)
	jmpTbl := []int{} // index of opcode that jump to the end of the block

	if err := p.ifblock(fn, &jmpTbl); err != nil {
		return err
	}

	for p.peek().Kind == tokenElseif {
		p.mustnext(tokenElseif)
		if err := p.ifblock(fn, &jmpTbl); err != nil {
			return err
		}
	}

	if p.peek().Kind == tokenElse {
		p.mustnext(tokenElse)
		if err := p.block(fn, false); err != nil {
			return err
		}
	}

	iend := len(fn.ByteCodes) - 1
	for _, idx := range jmpTbl {
		fn.ByteCodes[idx] = iABx(JMP, 0, uint16(iend-idx))
	}
	return p.next(tokenEnd)
}

func (p *Parser) ifblock(fn *FnProto, jmpTbl *[]int) error {
	condition, err := p.expression(fn)
	if err != nil {
		return err
	} else if err := p.next(tokenThen); err != nil {
		return err
	}
	spCondition, err := p.discharge(fn, condition)
	if err != nil {
		return err
	}
	p.code(fn, iAB(TEST, spCondition, 0))
	iFalseJmp := p.code(fn, iAsBx(JMP, 0, 0))
	if err := p.block(fn, false); err != nil {
		return err
	}
	iend := int16(len(fn.ByteCodes) - iFalseJmp)
	if tk := p.peek().Kind; tk == tokenElse || tk == tokenElseif {
		*jmpTbl = append(*jmpTbl, p.code(fn, iAsBx(JMP, 0, 0)))
		iend++
	}
	fn.ByteCodes[iFalseJmp] = iAsBx(JMP, 0, iend-1)
	return nil
}

func (p *Parser) whilestat(fn *FnProto) error {
	p.mustnext(tokenWhile)
	sp0 := fn.stackPointer
	istart := int16(len(fn.ByteCodes))
	condition, err := p.expression(fn)
	if err != nil {
		return err
	} else if err := p.next(tokenDo); err != nil {
		return err
	}
	spCondition, err := p.discharge(fn, condition)
	if err != nil {
		return err
	}
	p.code(fn, iAB(TEST, spCondition, 0))
	iFalseJmp := p.code(fn, iAsBx(JMP, 0, 0))
	if err := p.block(fn, true); err != nil {
		return err
	} else if err := p.next(tokenEnd); err != nil {
		return err
	}
	iend := int16(len(fn.ByteCodes))
	p.code(fn, iAsBx(JMP, sp0+1, -(iend-istart)-1))
	fn.ByteCodes[iFalseJmp] = iAsBx(JMP, sp0+1, int16(iend-int16(iFalseJmp)))
	return nil
}

// forstat -> FOR (fornum | forlist) END
func (p *Parser) forstat(fn *FnProto) error {
	tk := p.mustnext(tokenFor)
	name, err := p._next(tokenIdentifier)
	if err != nil {
		return err
	}
	if p.peek().Kind == tokenAssign {
		return p.fornum(fn, name)
	} else if tk := p.peek().Kind; tk == tokenComma || tk == tokenIn {
		return p.forlist(fn, name)
	}
	return p.parseErrf(tk, "malformed for statment")
}

// fornum -> NAME = exp,exp[,exp] DO
func (p *Parser) fornum(fn *FnProto, name *token) error {
	tk := p.mustnext(tokenAssign)
	sp0 := fn.stackPointer

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

	p.beforeblock(fn, true)
	defer p.afterblock(fn, true)
	// add the iterator var, limit, step locals, the last two cannot be directly accessed
	if err := fn.addLocals(name.StringVal, "", ""); err != nil {
		return err
	}
	iforPrep := p.code(fn, iAsBx(FORPREP, sp0, 0))

	if err := p.next(tokenDo); err != nil {
		return err
	} else if err := p.statList(fn); err != nil {
		return err
	} else if err := p.next(tokenEnd); err != nil {
		return err
	}

	blockSize := int16(len(fn.ByteCodes) - iforPrep - 1)
	p.code(fn, iAsBx(FORLOOP, sp0, -blockSize-1))
	fn.ByteCodes[iforPrep] = iAsBx(FORPREP, sp0, blockSize)
	return nil
}

// forlist -> NAME {,NAME} IN explist DO
func (p *Parser) forlist(fn *FnProto, firstName *token) error {
	sp0 := fn.stackPointer

	names := []string{firstName.StringVal}
	if p.peek().Kind == tokenComma {
		for {
			p.mustnext(tokenComma)
			name, err := p._next(tokenIdentifier)
			if err != nil {
				return err
			}
			names = append(names, name.StringVal)
			if p.peek().Kind != tokenComma {
				break
			}
		}
	}
	if err := p.next(tokenIn); err != nil {
		return err
	}

	p.beforeblock(fn, true)
	defer p.afterblock(fn, true)
	lcl0 := uint8(len(fn.locals))
	exprs, err := p.explistWant(fn, 3)
	if err != nil {
		return err
	} else if err := fn.addLocals("", "", ""); err != nil {
		return err
	} else if err := fn.addLocals(names...); err != nil {
		return err
	}

	for i, expr := range exprs {
		if _, err := p.dischargeTo(fn, expr, lcl0+uint8(i)); err != nil {
			return err
		}
	}

	ijmp := p.code(fn, iAsBx(JMP, 0, 0))
	if err := p.next(tokenDo); err != nil {
		return err
	} else if err := p.statList(fn); err != nil {
		return err
	} else if err := p.next(tokenEnd); err != nil {
		return err
	}

	fn.ByteCodes[ijmp] = iAsBx(JMP, 0, int16(len(fn.ByteCodes)-ijmp-1))
	p.code(fn, iAB(TFORCALL, sp0, uint8(len(names))))
	p.code(fn, iAsBx(TFORLOOP, sp0+1, -int16(len(fn.ByteCodes)-ijmp)))
	return nil
}

func (p *Parser) repeatstat(fn *FnProto) error {
	p.mustnext(tokenRepeat)
	sp0 := uint8(len(fn.locals))

	istart := len(fn.ByteCodes)
	if err := p.block(fn, true); err != nil {
		return err
	} else if err := p.next(tokenUntil); err != nil {
		return err
	} else if condition, err := p.expression(fn); err != nil {
		return err
	} else if spCondition, err := p.discharge(fn, condition); err != nil {
		return err
	} else {
		p.code(fn, iAB(TEST, spCondition, 0))
		p.code(fn, iAsBx(JMP, sp0+1, -int16(len(fn.ByteCodes)-istart+1)))
	}
	fn.stackPointer = sp0
	return nil
}

func (p *Parser) breakstat(fn *FnProto) error {
	breakToken := p.mustnext(tokenBreak)
	if len(p.breakBlocks) == 0 {
		return p.parseErrf(breakToken, "use of a break outside of loop")
	}
	p.breakBlocks[len(p.breakBlocks)-1] = append(p.breakBlocks[len(p.breakBlocks)-1], p.code(fn, iAsBx(JMP, 0, 0)))
	return nil
}

// label -> '::' NAME '::'
func (p *Parser) labelstat(fn *FnProto) error {
	p.mustnext(tokenDoubleColon)
	name, err := p._next(tokenIdentifier)
	if err != nil {
		return err
	}
	label := name.StringVal
	if entry := fn.findLabel(label); entry != nil {
		return p.parseErrf(name, "label '%s' already defined on line %v", label, entry.token.Line)
	}
	icode := len(fn.ByteCodes)
	level := len(fn.labels) - 1
	fn.labels[len(fn.labels)-1][label] = labelEntry{token: name, label: label, pc: icode}
	if gotos, hasGotos := fn.gotos[label]; hasGotos {
		finalGotos := []gotoEntry{}
		for _, entry := range gotos {
			if entry.level >= level {
				fn.ByteCodes[entry.pc] = iAsBx(JMP, 0, int16(icode-entry.pc-1))
			} else {
				finalGotos = append(finalGotos, entry)
			}
		}
		if len(finalGotos) == 0 {
			delete(fn.gotos, label)
		} else {
			fn.gotos[label] = finalGotos
		}
	}
	return p.next(tokenDoubleColon)
}

// gotostat -> 'goto' NAME
func (p *Parser) gotostat(fn *FnProto) error {
	tk := p.mustnext(tokenGoto)
	if name, err := p._next(tokenIdentifier); err != nil {
		return err
	} else if label := fn.findLabel(name.StringVal); label != nil {
		p.code(fn, iAsBx(JMP, 0, -int16(len(fn.ByteCodes)-label.pc+1)))
	} else {
		fn.gotos[name.StringVal] = append(fn.gotos[name.StringVal], gotoEntry{
			token: tk,
			label: name.StringVal,
			level: len(fn.labels) - 1,
			pc:    p.code(fn, iAsBx(JMP, 0, 0)),
		})
	}
	return nil
}

// localassign -> NAME attrib { ',' NAME attrib } ['=' explist]
// attrib -> ['<' ('const' | 'close') '>']
func (p *Parser) localassign(fn *FnProto) error {
	lcl0 := uint8(len(fn.locals))
	names := []*local{}
	for {
		ident, err := p._next(tokenIdentifier)
		if err != nil {
			return err
		}

		lcl := &local{name: ident.StringVal}
		if p.peek().Kind == tokenLt {
			p.mustnext(tokenLt)
			if tk, err := p._next(tokenIdentifier); err != nil {
				return err
			} else if tk.StringVal == "const" {
				lcl.attrConst = true
			} else if tk.StringVal == "close" {
				lcl.attrClose = true
			} else {
				return p.parseErrf(tk, "unknown local attribute %v", tk.StringVal)
			}
			if err := p.next(tokenGt); err != nil {
				return err
			}
		}
		names = append(names, lcl)
		if p.peek().Kind != tokenComma {
			break
		}
		p.mustnext(tokenComma)
	}

	if p.peek().Kind != tokenAssign {
		_, err := p.dischargeTo(fn, &exNil{num: uint16(len(names) - 1)}, lcl0)
		return err
	}
	p.mustnext(tokenAssign)

	exprs, err := p.explistWant(fn, len(names))
	if err != nil {
		return err
	}
	for i, lcl := range names {
		if i < len(exprs) {
			if _, err := p.dischargeTo(fn, exprs[i], lcl0+uint8(i)); err != nil {
				return err
			}
		}
		if err := fn.addLocal(lcl); err != nil {
			return err
		} else if lcl.attrClose {
			p.code(fn, iAB(TBC, lcl0+uint8(i), 0))
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

// funcargs -> '(' [ explist ] ')' | constructor | STRING
func (p *Parser) funcargs(fn *FnProto) ([]expression, error) {
	switch p.peek().Kind {
	case tokenOpenParen:
		p.mustnext(tokenOpenParen)
		if p.peek().Kind == tokenCloseParen {
			p.mustnext(tokenCloseParen)
			return []expression{}, nil
		}
		exprs, err := p.explist(fn)
		if err != nil {
			return nil, err
		}
		return exprs, p.next(tokenCloseParen)
	case tokenOpenCurly:
		expr, err := p.constructor(fn)
		return []expression{expr}, err
	case tokenString:
		tk := p.mustnext(tokenString)
		return []expression{&exString{LineInfo: tk.LineInfo, val: tk.StringVal}}, nil
	default:
		return nil, p.parseErrf(p.peek(), "unexpected token type %v while evaluating function call", p.peek().Kind)
	}
}

// assignment -> suffixedexp { ',' suffixedexp } '=' explist
func (p *Parser) assignment(fn *FnProto, first expression) error {
	sp0 := fn.stackPointer
	names := []expression{first}
	for p.peek().Kind == tokenComma {
		p.mustnext(tokenComma)
		if expr, err := p.suffixedexp(fn); err != nil {
			return err
		} else {
			names = append(names, expr)
		}
	}
	tk, err := p._next(tokenAssign)
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

func (p *Parser) expression(fn *FnProto) (desc expression, err error) {
	return p.expr(fn, 0)
}

// expr -> (simpleexp | unop expr) { binop expr }
// where 'binop' is any binary operator with a priority higher than 'limit'
func (p *Parser) expr(fn *FnProto, limit int) (desc expression, err error) {
	if tk := p.peek(); tk.isUnary() {
		if err = p.next(tk.Kind); err != nil {
			return nil, err
		} else if desc, err = p.expr(fn, unaryPriority); err != nil {
			return nil, err
		}
		desc = unaryExpression(tk, desc)
	} else if desc, err = p.simpleexp(fn); err != nil {
		return nil, err
	}
	for op := p.peek(); op.isBinary() && binaryPriority[op.Kind][0] > limit; op = p.peek() {
		p.mustnext(op.Kind)
		rdesc, err := p.expr(fn, binaryPriority[op.Kind][1])
		if err != nil {
			return nil, err
		}
		desc = newInfixExpr(op, desc, rdesc)
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
	case tokenFloat:
		tk := p.mustnext(tokenFloat)
		return &exFloat{LineInfo: tk.LineInfo, val: tk.FloatVal}, nil
	case tokenInteger:
		tk := p.mustnext(tokenInteger)
		return &exInteger{LineInfo: tk.LineInfo, val: tk.IntVal}, nil
	case tokenString:
		tk := p.mustnext(tokenString)
		return &exString{LineInfo: tk.LineInfo, val: tk.StringVal}, nil
	case tokenNil:
		tk := p.mustnext(tokenNil)
		return &exNil{LineInfo: tk.LineInfo, num: 1}, nil
	case tokenTrue:
		tk := p.mustnext(tokenTrue)
		return &exBool{LineInfo: tk.LineInfo, val: true}, nil
	case tokenFalse:
		tk := p.mustnext(tokenFalse)
		return &exBool{LineInfo: tk.LineInfo, val: false}, nil
	case tokenOpenCurly:
		return p.constructor(fn)
	case tokenFunction:
		tk := p.mustnext(tokenFunction)
		newFn, err := p.funcbody(fn, "", false, tk.LineInfo)
		return &exClosure{
			fn:       fn.addFn(newFn),
			LineInfo: tk.LineInfo,
		}, err
	case tokenDots:
		tk := p.mustnext(tokenDots)
		return &exVarArgs{
			LineInfo: tk.LineInfo,
		}, nil
	default:
		return p.suffixedexp(fn)
	}
}

// primaryexp -> NAME | '(' expr ')'
func (p *Parser) primaryexp(fn *FnProto) (expression, error) {
	tk := p.peek()
	switch tk.Kind {
	case tokenOpenParen:
		p.mustnext(tokenOpenParen)
		desc, err := p.expression(fn)
		if err != nil {
			return nil, err
		}
		return desc, p.next(tokenCloseParen)
	case tokenIdentifier:
		return p.name(fn, p.mustnext(tokenIdentifier))
	default:
		return nil, p.parseErrf(p.peek(), "unexpected symbol %v", tk.Kind)
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
		case tokenPeriod:
			p.mustnext(tokenPeriod)
			key, err := p._next(tokenIdentifier)
			if err != nil {
				return nil, err
			}
			expr = &exIndex{
				table:    expr,
				key:      &exString{val: key.StringVal, LineInfo: key.LineInfo},
				LineInfo: key.LineInfo,
			}
		case tokenOpenBracket:
			tk := p.mustnext(tokenOpenBracket)
			key, err := p.expression(fn)
			if err != nil {
				return nil, err
			} else if err := p.next(tokenCloseBracket); err != nil {
				return nil, err
			}
			expr = &exIndex{
				table:    expr,
				key:      key,
				LineInfo: tk.LineInfo,
			}
		case tokenColon:
			p.mustnext(tokenColon)
			key, err := p._next(tokenIdentifier)
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
		case tokenOpenParen, tokenString, tokenOpenCurly:
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
func (p *Parser) name(fn *FnProto, name *token) (expression, error) {
	if expr, err := p.resolveVar(fn, name); err != nil {
		return nil, err
	} else if expr != nil {
		return expr, nil
	}
	expr, err := p.name(fn, &token{StringVal: "_ENV", LineInfo: LineInfo{Line: name.Line, Column: name.Column}})
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
func (p *Parser) resolveVar(fn *FnProto, name *token) (*exVariable, error) {
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
	} else if value, err := p.resolveVar(fn.prev, name); err != nil {
		return nil, err
	} else if value != nil {
		if value.local {
			value.lvar.upvalRef = true
		}
		if err := fn.addUpindex(name.StringVal, value.address, value.local); err != nil {
			return nil, err
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
		p.skipComments()
		if expr, err := p.expression(fn); err != nil {
			return nil, err
		} else {
			list = append(list, expr)
		}
		p.skipComments()
		if p.peek().Kind != tokenComma {
			break
		}
		p.mustnext(tokenComma)
	}
	return list, nil
}

// constructor -> '{' [ field { sep field } [sep] ] '}'
// sep         -> ',' | ';'
// field -> NAME = exp | '['exp']' = exp | exp
func (p *Parser) constructor(fn *FnProto) (expression, error) {
	expr := &exTable{LineInfo: p.mustnext(tokenOpenCurly).LineInfo}
	for {
		switch p.peek().Kind {
		case tokenCloseCurly:
			// do nothing, because it is an empty table
		case tokenComment:
			p.mustnext(tokenComment)
			continue
		case tokenIdentifier:
			tk := p.mustnext(tokenIdentifier)
			if p.peek().Kind == tokenAssign {
				p.mustnext(tokenAssign)
				if val, err := p.expr(fn, 0); err != nil {
					return nil, err
				} else {
					expr.fields = append(expr.fields, tableField{key: &exString{val: tk.StringVal}, val: val})
				}
			} else {
				p.lex.back(tk)
				if val, err := p.expr(fn, 0); err != nil {
					return nil, err
				} else {
					expr.array = append(expr.array, val)
				}
			}
		case tokenOpenBracket:
			p.mustnext(tokenOpenBracket)
			if key, err := p.expr(fn, 0); err != nil {
				return nil, err
			} else if err := p.next(tokenCloseBracket); err != nil {
				return nil, err
			} else if err := p.next(tokenAssign); err != nil {
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
		if tk := p.peek(); tk.Kind == tokenComma || tk.Kind == tokenSemiColon {
			p.mustnext(tk.Kind)
		} else {
			break
		}
	}
	return expr, p.next(tokenCloseCurly)
}
