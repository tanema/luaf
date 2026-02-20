package parse

import (
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/tanema/luaf/src/bytecode"
	"github.com/tanema/luaf/src/lerrors"
	"github.com/tanema/luaf/src/types"
)

type (
	// LoadMode are flags to indicate how to load/parse a chunk of data.
	LoadMode uint
	// Config not fully implemented yet. It is a configuration for how the Parser
	// behaves on each file that it parses. It may allow for the program to be more
	// strict.
	Config struct { // not implemented yet
		StringCoers bool // disallow string coersion in arith
		RequireOnly bool // require std libs instead of available by default
		EnvReadonly bool // not allowed to change _ENV
		LocalOnly   bool // not allowed to define globals only locals
		Strict      bool // type checking and throw parsing errors if types are bad
	}
	// Parser is the object that will parse a file an be able to return bytecode
	// ready for the VM.
	Parser struct {
		rootfn        *FnProto
		lex           *lexer
		filename      string
		lastComment   string
		breakBlocks   [][]int
		localsScope   []uint8
		lastTokenInfo LineInfo
		config        Config
	}
)

const (
	// ModeText implies that the chunk of text being loaded is plain text.
	ModeText LoadMode = 0b01
	// ModeBinary implies that the chunk of data being loaded is pre parsed binary.
	ModeBinary LoadMode = 0b10
)

// New creates a new parser that can parse one file at a time.
func New() *Parser {
	return &Parser{
		config:      Config{},
		rootfn:      newRootFn(),
		breakBlocks: [][]int{},
		localsScope: []uint8{},
	}
}

// File is a helper function around Parse to open and close a file automatically.
func File(path string, mode LoadMode) (*FnProto, error) {
	src, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = src.Close() }()
	return Parse(path, src, mode)
}

// Parse will, depending on the LoadMode, parse a text file and return bytecode
// or if the load mode is binary, it will undump an already parsed fnproto. If
// both modes are passed, it will try to figure out which kind of file it is parsing.
func Parse(filename string, src io.ReadSeeker, mode LoadMode) (*FnProto, error) {
	if hasLuaBinPrefix(src) && mode&ModeBinary == ModeBinary {
		return UndumpFnProto(src)
	}
	return New().Parse(filename, src)
}

// TryStat allows for trying a single statement. This is primarily for repl.
func TryStat(src string, parentFn *FnProto) (*FnProto, error) {
	filename := "<source>"
	fn := NewEmptyFnProto(filename, parentFn)
	p := New()
	p.filename = filename
	p.lex = newLexer(filename, strings.NewReader(src))
	if firsterr := p.stat(fn); firsterr != nil {
		if errors.Is(firsterr, io.EOF) {
			return nil, firsterr
		}
		p.lex = newLexer("<source>", strings.NewReader("return "+src))
		if err := p.stat(fn); err != nil {
			return nil, firsterr
		}
	}
	return fn, nil
}

// Parse will reset the parser but parse the source within the context of this
// function. This allows parsing in repl and still be able to have visibility
// of locals.
func (p *Parser) Parse(filename string, src io.Reader) (*FnProto, error) {
	fn := NewEmptyFnProto(filename, p.rootfn)
	p.filename = filename
	p.lex = newLexer(filename, src)
	if err := p.chunk(fn); err != nil && !errors.Is(err, io.EOF) {
		return fn, err
	} else if err := p.next(tokenEOS); !errors.Is(err, io.EOF) {
		return fn, err
	}
	if len(fn.ByteCodes) == 0 || bytecode.GetOp(fn.ByteCodes[len(fn.ByteCodes)-1]) != bytecode.RETURN {
		p.code(fn, bytecode.IAB(bytecode.RETURN, 0, 1))
	}
	return fn, fn.checkGotos(p)
}

func (p *Parser) parseErr(tk *token, err error) error {
	return p.fmtErr(tk, lerrors.ParserErr, err)
}

func (p *Parser) typeErr(tk *token, err error) error {
	return p.fmtErr(tk, lerrors.TypeErr, err)
}

func (p *Parser) fmtErr(tk *token, kind lerrors.ErrorKind, err error) error {
	if err == nil {
		return nil
	}
	var luaErr *lerrors.Error
	if errors.As(err, &luaErr) {
		return err
	} else if errors.Is(err, io.EOF) {
		return err
	}
	newErr := &lerrors.Error{
		Kind:     kind,
		Filename: p.filename,
		Err:      err,
	}
	if tk != nil {
		newErr.Line = tk.Line
		newErr.Column = tk.Column
	}
	return newErr
}

func (p *Parser) peek() (*token, error) {
	return p.lex.Peek()
}

func (p *Parser) consumeToken(tt tokenType) (*token, error) {
	tk, err := p.lex.Next()
	if err != nil {
		return nil, p.parseErr(tk, err)
	} else if tt != tk.Kind {
		return nil, p.parseErr(tk, fmt.Errorf("expected %q but consumed %q", tt, tk.Kind))
	}
	p.lastTokenInfo = tk.LineInfo
	return tk, nil
}

func (p *Parser) next(tt tokenType) error {
	_, err := p.consumeToken(tt)
	return err
}

// case something goes funky.
func (p *Parser) mustnext(tt tokenType) *token {
	tk, err := p.consumeToken(tt)
	if err != nil {
		panic(err)
	}
	return tk
}

func (p *Parser) beforeblock(fn *FnProto, breakable bool) {
	if breakable {
		p.breakBlocks = append(p.breakBlocks, []int{})
	}
	p.localsScope = append(p.localsScope, uint8(len(fn.Locals)))
	fn.labels = append(fn.labels, map[string]labelEntry{})
}

func (p *Parser) afterblock(fn *FnProto, breakable bool) {
	from := p.localsScope[len(p.localsScope)-1]

	if breakable {
		breaks := p.breakBlocks[len(p.breakBlocks)-1]
		endDst := len(fn.ByteCodes)
		for _, idx := range breaks {
			fn.ByteCodes[idx] = bytecode.IABx(bytecode.JMP, from+1, uint16(endDst-idx))
		}
		p.breakBlocks = p.breakBlocks[:len(p.breakBlocks)-1]
	}

	p.localsScope = p.localsScope[:len(p.localsScope)-1]

	hasUpvalRef := false
	for _, local := range fn.Locals[from:] {
		if local.upvalRef {
			hasUpvalRef = true
			break
		}
	}

	if hasUpvalRef {
		// insert close before the return if a return is already in there
		if lastCode := fn.ByteCodes[len(fn.ByteCodes)-1]; bytecode.GetOp(lastCode) == bytecode.RETURN {
			fn.ByteCodes[len(fn.ByteCodes)-1] = bytecode.IAB(bytecode.CLOSE, from, 0)
			fn.code(lastCode, p.lastTokenInfo)
		} else {
			fn.code(bytecode.IAB(bytecode.CLOSE, from, 0), p.lastTokenInfo)
		}
	}

	fn.Locals = fn.Locals[:from:from]
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

// block -> statlist.
func (p *Parser) block(fn *FnProto, breakable bool) error {
	p.beforeblock(fn, breakable)
	defer p.afterblock(fn, breakable)
	return p.statList(fn)
}

// statlist -> { stat [';'] }.
func (p *Parser) statList(fn *FnProto) error {
	if err := p.skipComments(); err != nil {
		return err
	}
	for {
		follow, err := p.blockFollow(true)
		if err != nil {
			return err
		} else if follow {
			break
		}
		ptk, err := p.peek()
		if err != nil {
			return err
		} else if ptk.Kind == tokenReturn {
			return p.stat(fn) /* 'return' must be last stat */
		} else if err := p.stat(fn); err != nil {
			return err
		}
	}
	return nil
}

// check if the next token indicates that we are still inside a block or not.
func (p *Parser) blockFollow(withuntil bool) (bool, error) {
	ptk, err := p.peek()
	if err != nil {
		return false, err
	}
	switch ptk.Kind {
	case tokenElse, tokenElseif, tokenEnd, tokenEOS:
		return true, nil
	case tokenUntil:
		return withuntil, nil
	default:
		return false, nil
	}
}

// | 'goto' NAME | funccallstat | assignment.
func (p *Parser) stat(fn *FnProto) error {
	fn.stackPointer = uint8(len(fn.Locals))
	tk, err := p.peek()
	if err != nil {
		return err
	}
	switch tk.Kind {
	case tokenSemiColon:
		return p.next(tokenSemiColon)
	case tokenComment:
		tk := p.mustnext(tokenComment)
		if strings.HasPrefix(tk.StringVal, "!") {
			return p.configComment(tk)
		}
		p.lastComment = tk.StringVal
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
	case tokenLabel:
		return p.labelstat(fn)
	case tokenBreak:
		return p.breakstat(fn)
	case tokenGoto:
		return p.gotostat(fn)
	case tokenTypeDef:
		return p.typedefstat(fn, false)
	default:
		tk, err := p.lex.Peek()
		if err != nil {
			return err
		}
		expr, err := p.suffixedexp(fn)
		if err != nil {
			return err
		} else if call, isCall := expr.(*exCall); isCall {
			_, err := p.discharge(fn, tk, call)
			return err
		} else if tk, err := p.peek(); err != nil {
			return err
		} else if tk.Kind == tokenAssign || tk.Kind == tokenComma {
			return p.assignment(fn, expr)
		}
		return p.parseErr(tk, fmt.Errorf("unexpected expression %v", reflect.TypeOf(expr)))
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

// localstat -> local [localfunc | localassign | typedef ].
func (p *Parser) localstat(fn *FnProto) error {
	tk := p.mustnext(tokenLocal)
	ptk, err := p.peek()
	if err != nil {
		return err
	} else if ptk.Kind == tokenFunction {
		return p.localfunc(fn)
	} else if ptk.Kind == tokenTypeDef {
		return p.typedefstat(fn, true)
	}
	return p.localassign(fn, tk)
}

// localfunc -> FUNCTION NAME funcbody.
func (p *Parser) localfunc(fn *FnProto) error {
	tk := p.mustnext(tokenFunction)
	ifn := uint8(len(fn.Locals))
	name, err := p.consumeToken(tokenIdentifier)
	if err != nil {
		return err
	}
	// TODO definition
	if err := fn.addLocal(&Local{
		name:     name.StringVal,
		typeDefn: &types.Function{},
	}); err != nil {
		return err
	}
	newFn, err := p.funcbody(fn, name.StringVal, false, name.LineInfo)
	if err != nil {
		return err
	}

	expr := &exClosure{
		fn:       fn.addFn(newFn),
		fnproto:  newFn,
		LineInfo: name.LineInfo,
	}

	_, err = p.dischargeTo(fn, tk, expr, ifn)
	return err
}

// funcstat -> FUNCTION funcname funcbody.
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
	closure := &exClosure{
		fn:       fn.addFn(newFn),
		fnproto:  newFn,
		LineInfo: tk.LineInfo,
	}
	icls, err := p.discharge(fn, tk, closure)
	if err != nil {
		return p.parseErr(tk, err)
	}
	return p.assignTo(fn, tk, name, icls, closure)
}

func (p *Parser) assignTo(fn *FnProto, tk *token, dst expression, from uint8, value expression) error {
	valKind, err := value.inferType()
	if err != nil {
		return err
	}

	switch ex := dst.(type) {
	case *exVariable:
		if p.config.Strict && !ex.typeDefn.Check(valKind) {
			return p.typeErr(tk, fmt.Errorf("expected %s, but received %s", ex.typeDefn, valKind))
		} else if ex.attrConst {
			return p.parseErr(tk, fmt.Errorf("attempt to assign to const variable '%v'", ex.name))
		} else if !ex.local {
			fn.code(bytecode.IAB(bytecode.SETUPVAL, from, ex.address), ex.LineInfo)
		} else {
			fn.code(bytecode.IAB(bytecode.MOVE, ex.address, from), ex.LineInfo)
		}
		return nil
	case *exIndex:
		if val, isVal := ex.table.(*exVariable); isVal {
			ikey, keyIsConst, err := dischargeMaybeConst(fn, ex.key, fn.stackPointer)
			if err != nil {
				return err
			}
			if val.local {
				fn.code(bytecode.IABCK(bytecode.SETTABLE, val.address, ikey, keyIsConst, from, false), ex.LineInfo)
			} else {
				fn.code(bytecode.IABCK(bytecode.SETTABUP, val.address, ikey, keyIsConst, from, false), ex.LineInfo)
			}
			return nil
		}
		itable, err := p.discharge(fn, tk, ex.table)
		if err != nil {
			return err
		}
		ikey, keyIsConst, err := dischargeMaybeConst(fn, ex.key, fn.stackPointer)
		if err != nil {
			return err
		}
		fn.code(bytecode.IABCK(bytecode.SETTABLE, itable, ikey, keyIsConst, from, false), ex.LineInfo)
		return nil
	default:
		panic(fmt.Sprintf("unknown expression to assign to %T", dst))
	}
}

// fieldsel     -> ['.' | ':'] NAME.
func (p *Parser) funcname(fn *FnProto) (expression, bool, string, error) {
	ident, err := p.consumeToken(tokenIdentifier)
	if err != nil {
		return nil, false, "", err
	}
	name, err := p.name(fn, ident)
	if err != nil {
		return nil, false, "", err
	}
	fullname := ident.StringVal
	for {
		ptk, err := p.peek()
		if err != nil {
			return nil, false, "", err
		}

		switch ptk.Kind {
		case tokenPeriod:
			p.mustnext(tokenPeriod)
			ident, err := p.consumeToken(tokenIdentifier)
			if err != nil {
				return nil, false, "", err
			}
			fullname += "." + ident.StringVal
			name = &exIndex{
				table:    name,
				key:      &exString{val: ident.StringVal, LineInfo: ident.LineInfo},
				typeDefn: &types.Function{},
				LineInfo: ident.LineInfo,
			}
		case tokenColon:
			p.mustnext(tokenColon)
			ident, err := p.consumeToken(tokenIdentifier)
			if err != nil {
				return nil, false, "", err
			}
			fullname += ":" + ident.StringVal
			return &exIndex{
				table:    name,
				key:      &exString{val: ident.StringVal, LineInfo: ident.LineInfo},
				typeDefn: &types.Function{},
				LineInfo: ident.LineInfo,
			}, true, fullname, nil
		default:
			return name, false, fullname, nil
		}
	}
}

// funcbody -> parlist block END.
func (p *Parser) funcbody(fn *FnProto, name string, hasSelf bool, linfo LineInfo) (*FnProto, error) {
	params, varargs, err := p.parlist()
	if err != nil {
		return nil, err
	}
	if hasSelf {
		// TODO self will have a type def so use it instead of freeform
		params = append([]types.NamedPair{{Name: "self", Defn: types.NewTable()}}, params...)
	}

	defn := &types.Function{
		Params: params,
	}

	ptk, err := p.peek()
	if err != nil {
		return nil, err
	}
	if ptk.Kind == tokenColon {
		defn.Return, err = p.retlist(fn)
		if err != nil {
			return nil, err
		}
	}

	localParams := make([]*Local, len(params))
	for i, p := range params {
		localParams[i] = &Local{name: p.Name, typeDefn: types.Any}
	}

	newFn := NewFnProto(p.filename, name, fn, localParams, varargs, defn, linfo)
	newFn.Comment = p.lastComment
	p.lastComment = ""
	if err := p.block(newFn, false); err != nil {
		return nil, err
	}
	if len(newFn.ByteCodes) == 0 || bytecode.GetOp(newFn.ByteCodes[len(newFn.ByteCodes)-1]) != bytecode.RETURN {
		p.code(newFn, bytecode.IAB(bytecode.RETURN, 0, 1))
	}
	if err := fn.checkGotos(p); err != nil {
		return nil, err
	}
	return newFn, p.next(tokenEnd)
}

// parlist -> '(' [ {NAME ','} (NAME | '...') ] ')'.
func (p *Parser) parlist() ([]types.NamedPair, bool, error) {
	if err := p.next(tokenOpenParen); err != nil {
		return nil, false, err
	}
	names := []types.NamedPair{}
	ptk, err := p.peek()
	if err != nil {
		return nil, false, err
	} else if ptk.Kind == tokenCloseParen {
		return names, false, p.next(tokenCloseParen)
	}
	ptk, err = p.peek()
	if err != nil {
		return nil, false, err
	}
	for ptk.Kind == tokenIdentifier {
		name, err := p.consumeToken(tokenIdentifier)
		if err != nil {
			return nil, false, err
		}
		defn := types.NamedPair{
			Name: name.StringVal,
			Defn: types.Any,
		}

		names = append(names, defn)
		ptk, err = p.peek()
		if err != nil {
			return nil, false, err
		} else if ptk.Kind != tokenComma {
			break
		}
		p.mustnext(tokenComma)

		ptk, err = p.peek()
		if err != nil {
			return nil, false, err
		}
	}

	varargs := false
	ptk, err = p.peek()
	if err != nil {
		return nil, false, err
	} else if ptk.Kind == tokenDots {
		p.mustnext(tokenDots)
		varargs = true
	}
	return names, varargs, p.next(tokenCloseParen)
}

// retlist ":" '(' [ {NAME ','} (NAME | '...') ] ')'.
func (p *Parser) retlist(fn *FnProto) ([]types.Definition, error) {
	p.mustnext(tokenColon)
	ptk, err := p.peek()
	if err != nil {
		return nil, err
	} else if ptk.Kind != tokenOpenParen {
		defn, err := p.simpletype(fn)
		return []types.Definition{defn}, err
	}

	defns := []types.Definition{}
	for {
		defn, err := p.simpletype(fn)
		if err != nil {
			return nil, err
		}
		defns = append(defns, defn)
		if ptk, err := p.peek(); err != nil {
			return nil, err
		} else if ptk.Kind == tokenComma {
			p.mustnext(tokenComma)
		} else {
			break
		}
	}

	return defns, p.next(tokenCloseParen)
}

// retstat -> RETURN [explist] [';'].
func (p *Parser) retstat(fn *FnProto) error {
	tk := p.mustnext(tokenReturn)
	sp0 := fn.stackPointer
	// if we are at the end of block then there are no return vals
	follow, err := p.blockFollow(true)
	if err != nil {
		return err
	} else if follow {
		p.code(fn, bytecode.IAB(bytecode.RETURN, sp0, 1))
		return nil
	}

	exprs, err := p.explist(fn)
	if err != nil {
		return err
	}
	lastExpr, err := p.dischargeAllButLast(fn, tk, exprs)
	if err != nil {
		return err
	}
	switch expr := lastExpr.(type) {
	case *exCall:
		expr.tail = true
		if _, err := p.dischargeTo(fn, tk, expr, sp0); err != nil {
			return err
		}
	case *exVarArgs:
		if _, err := p.discharge(fn, tk, expr); err != nil {
			return err
		}
		p.code(fn, bytecode.IAB(bytecode.RETURN, sp0, 0))
	default:
		if _, err := p.discharge(fn, tk, expr); err != nil {
			return err
		}
		p.code(fn, bytecode.IAB(bytecode.RETURN, sp0, uint8(len(exprs)+1)))
	}
	// consume any comments after the return because the parse is pretty strict about
	// the return being the end of the block, but comments should be allowed
	return p.skipComments()
}

func (p *Parser) skipComments() error {
	for {
		if ptk, err := p.peek(); err != nil || ptk.Kind != tokenComment {
			return err
		}
		p.mustnext(tokenComment)
	}
}

// dostat -> DO block END.
func (p *Parser) dostat(fn *FnProto) error {
	p.mustnext(tokenDo)
	if err := p.block(fn, false); err != nil {
		return err
	}
	return p.next(tokenEnd)
}

// ifstat -> IF exp THEN block {ELSEIF exp THEN block} [ELSE block] END.
func (p *Parser) ifstat(fn *FnProto) error {
	tk := p.mustnext(tokenIf)
	jmpTbl := []int{} // index of opcode that jump to the end of the block

	if err := p.ifblock(fn, tk, &jmpTbl); err != nil {
		return err
	}

	for {
		if ptk, err := p.peek(); err != nil {
			return err
		} else if ptk.Kind != tokenElseif {
			break
		}
		tk = p.mustnext(tokenElseif)
		if err := p.ifblock(fn, tk, &jmpTbl); err != nil {
			return err
		}
	}

	if ptk, err := p.peek(); err != nil {
		return err
	} else if ptk.Kind == tokenElse {
		p.mustnext(tokenElse)
		if err := p.block(fn, false); err != nil {
			return err
		}
	}

	iend := len(fn.ByteCodes) - 1
	for _, idx := range jmpTbl {
		fn.ByteCodes[idx] = bytecode.IABx(bytecode.JMP, 0, uint16(iend-idx))
	}
	return p.next(tokenEnd)
}

func (p *Parser) ifblock(fn *FnProto, tk *token, jmpTbl *[]int) error {
	condition, err := p.expression(fn)
	if err != nil {
		return err
	} else if err := p.next(tokenThen); err != nil {
		return err
	}
	// todo if condition is false just skip block and raise no errors
	spCondition, err := p.discharge(fn, tk, condition)
	if err != nil {
		return err
	}
	p.code(fn, bytecode.IAB(bytecode.TEST, spCondition, 0))
	iFalseJmp := p.code(fn, bytecode.IAsBx(bytecode.JMP, 0, 0))
	if err := p.block(fn, false); err != nil {
		return err
	}
	iend := int16(len(fn.ByteCodes) - iFalseJmp)
	if ptk, err := p.peek(); err != nil {
		return err
	} else if tk := ptk.Kind; tk == tokenElse || tk == tokenElseif {
		*jmpTbl = append(*jmpTbl, p.code(fn, bytecode.IAsBx(bytecode.JMP, 0, 0)))
		iend++
	}
	fn.ByteCodes[iFalseJmp] = bytecode.IAsBx(bytecode.JMP, 0, iend-1)
	return nil
}

func (p *Parser) whilestat(fn *FnProto) error {
	tk := p.mustnext(tokenWhile)
	sp0 := fn.stackPointer
	istart := int16(len(fn.ByteCodes))
	condition, err := p.expression(fn)
	if err != nil {
		return err
	} else if err := p.next(tokenDo); err != nil {
		return err
	}
	spCondition, err := p.discharge(fn, tk, condition)
	if err != nil {
		return err
	}
	p.code(fn, bytecode.IAB(bytecode.TEST, spCondition, 0))
	iFalseJmp := p.code(fn, bytecode.IAsBx(bytecode.JMP, 0, 0))
	if err := p.block(fn, true); err != nil {
		return err
	} else if err := p.next(tokenEnd); err != nil {
		return err
	}
	iend := int16(len(fn.ByteCodes))
	p.code(fn, bytecode.IAsBx(bytecode.JMP, sp0+1, -(iend-istart)-1))
	fn.ByteCodes[iFalseJmp] = bytecode.IAsBx(bytecode.JMP, sp0+1, iend-int16(iFalseJmp))
	return nil
}

// forstat -> FOR (fornum | forlist) END.
func (p *Parser) forstat(fn *FnProto) error {
	tk := p.mustnext(tokenFor)
	name, err := p.consumeToken(tokenIdentifier)
	if err != nil {
		return err
	}
	if ptk, err := p.peek(); err != nil {
		return err
	} else if ptk.Kind == tokenAssign {
		return p.fornum(fn, name)
	} else if ptk.Kind == tokenComma || ptk.Kind == tokenIn {
		return p.forlist(fn, name)
	}
	return p.parseErr(tk, errors.New("malformed for statement"))
}

// fornum -> NAME = exp,exp[,exp] DO.
func (p *Parser) fornum(fn *FnProto, name *token) error {
	tk := p.mustnext(tokenAssign)
	sp0 := fn.stackPointer

	if exprs, err := p.explist(fn); err != nil {
		return err
	} else if len(exprs) < 2 || len(exprs) > 3 {
		return p.parseErr(tk, errors.New("invalid for stat, expected 2-3 expressions"))
	} else if lastExpr, err := p.dischargeAllButLast(fn, tk, exprs); err != nil {
		return err
	} else if _, err := p.discharge(fn, tk, lastExpr); err != nil {
		return err
	} else if len(exprs) == 2 {
		if _, err := p.discharge(fn, tk, &exInteger{val: 1}); err != nil {
			return err
		}
	}

	p.beforeblock(fn, true)
	defer p.afterblock(fn, true)

	// add the iterator var, limit, step locals, the last two cannot be directly accessed
	if err := fn.addLocal(&Local{name: name.StringVal, typeDefn: types.Number}); err != nil {
		return err
	} else if err := fn.addLocal(&Local{name: "", typeDefn: types.Number}); err != nil {
		return err
	} else if err := fn.addLocal(&Local{name: "", typeDefn: types.Number}); err != nil {
		return err
	}

	iforPrep := p.code(fn, bytecode.IAsBx(bytecode.FORPREP, sp0, 0))

	if err := p.next(tokenDo); err != nil {
		return err
	} else if err := p.statList(fn); err != nil {
		return err
	} else if err := p.next(tokenEnd); err != nil {
		return err
	}

	blockSize := int16(len(fn.ByteCodes) - iforPrep - 1)
	p.code(fn, bytecode.IAsBx(bytecode.FORLOOP, sp0, -blockSize-1))
	fn.ByteCodes[iforPrep] = bytecode.IAsBx(bytecode.FORPREP, sp0, blockSize)
	return nil
}

// forlist -> NAME {,NAME} IN explist DO.
func (p *Parser) forlist(fn *FnProto, firstName *token) error {
	sp0 := fn.stackPointer

	names := []string{firstName.StringVal}
	if ptk, err := p.peek(); err != nil {
		return err
	} else if ptk.Kind == tokenComma {
		for {
			p.mustnext(tokenComma)
			name, err := p.consumeToken(tokenIdentifier)
			if err != nil {
				return err
			}
			names = append(names, name.StringVal)
			if ptk, err := p.peek(); err != nil {
				return err
			} else if ptk.Kind != tokenComma {
				break
			}
		}
	}
	if err := p.next(tokenIn); err != nil {
		return err
	}

	p.beforeblock(fn, true)
	defer p.afterblock(fn, true)
	lcl0 := uint8(len(fn.Locals))
	exprs, err := p.explistWant(fn, 3)
	if err != nil {
		return err
	} else if err := fn.addLocal(&Local{name: "", typeDefn: &types.Function{}}); err != nil {
		return err
	} else if err := fn.addLocal(&Local{name: "", typeDefn: types.NewTable()}); err != nil {
		return err
	} else if err := fn.addLocal(&Local{name: "", typeDefn: types.Number}); err != nil {
		return err
	}

	for _, name := range names {
		if err := fn.addLocal(&Local{name: name, typeDefn: types.Any}); err != nil {
			return err
		}
	}

	for i, expr := range exprs {
		if _, err := p.dischargeTo(fn, firstName, expr, lcl0+uint8(i)); err != nil {
			return err
		}
	}

	ijmp := p.code(fn, bytecode.IAsBx(bytecode.JMP, 0, 0))
	if err := p.next(tokenDo); err != nil {
		return err
	} else if err := p.statList(fn); err != nil {
		return err
	} else if err := p.next(tokenEnd); err != nil {
		return err
	}

	fn.ByteCodes[ijmp] = bytecode.IAsBx(bytecode.JMP, 0, int16(len(fn.ByteCodes)-ijmp-1))
	p.code(fn, bytecode.IAB(bytecode.TFORCALL, sp0, uint8(len(names))))
	p.code(fn, bytecode.IAsBx(bytecode.TFORLOOP, sp0+1, -int16(len(fn.ByteCodes)-ijmp)))
	return nil
}

func (p *Parser) repeatstat(fn *FnProto) error {
	tk := p.mustnext(tokenRepeat)
	sp0 := uint8(len(fn.Locals))

	istart := len(fn.ByteCodes)
	if err := p.block(fn, true); err != nil {
		return err
	} else if err := p.next(tokenUntil); err != nil {
		return err
	}
	condition, err := p.expression(fn)
	if err != nil {
		return err
	}
	spCondition, err := p.discharge(fn, tk, condition)
	if err != nil {
		return err
	}
	p.code(fn, bytecode.IAB(bytecode.TEST, spCondition, 0))
	p.code(fn, bytecode.IAsBx(bytecode.JMP, sp0+1, -int16(len(fn.ByteCodes)-istart+1)))
	fn.stackPointer = sp0
	return nil
}

func (p *Parser) breakstat(fn *FnProto) error {
	breakToken := p.mustnext(tokenBreak)
	if len(p.breakBlocks) == 0 {
		return p.parseErr(breakToken, errors.New("use of a break outside of loop"))
	}
	p.breakBlocks[len(p.breakBlocks)-1] = append(p.breakBlocks[len(p.breakBlocks)-1],
		p.code(fn, bytecode.IAsBx(bytecode.JMP, 0, 0)))
	return nil
}

// label -> '::' NAME '::'.
func (p *Parser) labelstat(fn *FnProto) error {
	label := p.mustnext(tokenLabel)
	name := label.StringVal
	if entry := fn.findLabel(name); entry != nil {
		return p.parseErr(label, fmt.Errorf("label '%s' already defined on line %v", label, entry.token.Line))
	}
	icode := len(fn.ByteCodes)
	level := len(fn.labels) - 1
	fn.labels[len(fn.labels)-1][name] = labelEntry{token: label, label: name, pc: icode}
	if gotos, hasGotos := fn.gotos[name]; hasGotos {
		finalGotos := []gotoEntry{}
		for _, entry := range gotos {
			if entry.level >= level {
				fn.ByteCodes[entry.pc] = bytecode.IAsBx(bytecode.JMP, 0, int16(icode-entry.pc-1))
			} else {
				finalGotos = append(finalGotos, entry)
			}
		}
		if len(finalGotos) == 0 {
			delete(fn.gotos, name)
		} else {
			fn.gotos[name] = finalGotos
		}
	}
	return nil
}

// gotostat -> 'goto' NAME.
func (p *Parser) gotostat(fn *FnProto) error {
	tk := p.mustnext(tokenGoto)
	if name, err := p.consumeToken(tokenIdentifier); err != nil {
		return err
	} else if label := fn.findLabel(name.StringVal); label != nil {
		p.code(fn, bytecode.IAsBx(bytecode.JMP, 0, -int16(len(fn.ByteCodes)-label.pc+1)))
	} else {
		fn.gotos[name.StringVal] = append(fn.gotos[name.StringVal], gotoEntry{
			token: tk,
			label: name.StringVal,
			level: len(fn.labels) - 1,
			pc:    p.code(fn, bytecode.IAsBx(bytecode.JMP, 0, 0)),
		})
	}
	return nil
}

// <typedef> ::= "type" <name> ("<" <gtypelistwithdefaults> ">")? <type>.
func (p *Parser) typedefstat(fn *FnProto, isLocal bool) error {
	p.mustnext(tokenTypeDef)
	name := p.mustnext(tokenIdentifier)
	typeDefn, err := p.typestat(fn)
	if err != nil {
		return err
	}
	return fn.addType(name.StringVal, typeDefn, isLocal)
}

// <type> ::= <simpletype> "?"? ("|" <simpletype> "?"?)* | <simpletype> ("&" <simpletype>)*.
func (p *Parser) typestat(fn *FnProto) (types.Definition, error) {
	for {
		stype, err := p.simpletype(fn)
		if err != nil {
			return nil, err
		}
		defn := stype
		tk, err := p.peek()
		if err != nil {
			return nil, err
		} else if tk.Kind == tokenOptional {
			defn = &types.Union{Defn: []types.Definition{stype, types.Nil}}
			p.mustnext(tokenOptional)
			tk, err = p.peek()
			if err != nil {
				return nil, err
			}
		}
		switch tk.Kind {
		case tokenBitwiseAnd:
			return p.unionTypeStat(fn, defn)
		case tokenBitwiseOrUnion:
			return p.intersectionTypeStat(fn, defn)
		default:
			return defn, nil
		}
	}
}

func (p *Parser) unionTypeStat(fn *FnProto, defn types.Definition) (types.Definition, error) {
	p.mustnext(tokenBitwiseAnd)
	union := &types.Union{Defn: []types.Definition{defn}}
	for {
		stype, err := p.simpletype(fn)
		if err != nil {
			return nil, err
		}
		union.Defn = append(union.Defn, stype)
		if tk, err := p.peek(); err != nil {
			return nil, err
		} else if tk.Kind != tokenBitwiseAnd {
			return union, nil
		}
	}
}

func (p *Parser) intersectionTypeStat(fn *FnProto, defn types.Definition) (types.Definition, error) {
	p.mustnext(tokenBitwiseOrUnion)
	inter := &types.Intersection{Defn: []types.Definition{defn}}
	for {
		stype, err := p.simpletype(fn)
		if err != nil {
			return nil, err
		}
		inter.Defn = append(inter.Defn, stype)
		if tk, err := p.peek(); err != nil {
			return nil, err
		} else if tk.Kind != tokenBitwiseOrUnion {
			return inter, nil
		}
	}
}

// <simpletype> ::= <name> ("." <name>)* ("<" <typeparamlist> ">") |
// "typeof" "(" <expr> ")" | <tbltype> | <fntype> | "(" <type> ")".
func (p *Parser) simpletype(fn *FnProto) (types.Definition, error) {
	tk, err := p.peek()
	if err != nil {
		return nil, err
	}
	switch tk.Kind {
	case tokenOpenParen:
		p.mustnext(tokenOpenParen)
		defn, err := p.typestat(fn)
		if err != nil {
			return nil, err
		}
		return defn, p.next(tokenCloseParen)
	case tokenOpenCurly:
		return p.tbltypedef(fn)
	case tokenFunction:
		return p.fntypedef(fn)
	case tokenIdentifier:
		tk := p.mustnext(tokenIdentifier)
		if tk.StringVal == "typeof" {
			return p.typeofDefStat(fn)
		}
		// TODO consume namespacing and type params
		return fn.resolveType(tk.StringVal)
	default:
		return nil, p.parseErr(tk, fmt.Errorf("type declaration expected definition found %s", tk))
	}
}

func (p *Parser) typeofDefStat(_ *FnProto) (types.Definition, error) {
	return nil, errors.New("typeof typedef not yet implemented")
}

// "function" <params> ":" <returnTypes>.
func (p *Parser) fntypedef(_ *FnProto) (types.Definition, error) {
	p.mustnext(tokenFunction)
	return &types.Function{}, errors.New("function type not implemented yet")
}

// <tbltype>      ::= "{" "[" <type> "]" | "[" <type> "]" ":" <type> | <proplist> "}"
// <proplist>     ::= <name> ":" <type> (<sep> <proplist>)*.
func (p *Parser) tbltypedef(fn *FnProto) (types.Definition, error) {
	p.mustnext(tokenOpenCurly)
	ptk, err := p.peek()
	if err != nil {
		return nil, err
	}
	switch ptk.Kind {
	case tokenCloseCurly:
		return types.NewTable(), nil
	case tokenIdentifier:
		return p.structTypeDef(fn)
	case tokenOpenBracket:
		return p.tblSubTypeDef(fn)
	default:
		return nil, p.parseErr(ptk, fmt.Errorf("unexpected token %v, while parsing type declaration", ptk.Kind))
	}
}

func (p *Parser) structTypeDef(fn *FnProto) (types.Definition, error) {
	tblDefn := &types.Table{
		Hint:      types.TblStruct,
		FieldDefn: map[string]types.Definition{},
	}
	for {
		tk, err := p.peek()
		if err != nil {
			return nil, err
		} else if tk.Kind == tokenComment {
			p.mustnext(tokenComment)
			continue
		}

		if err := p.next(tokenIdentifier); err != nil {
			return nil, err
		} else if err := p.next(tokenAssign); err != nil {
			return nil, err
		}

		valDefn, err := p.typestat(fn)
		if err != nil {
			return nil, err
		}

		tblDefn.FieldDefn[tk.StringVal] = valDefn

		if tk, err := p.peek(); err != nil {
			return nil, err
		} else if tk.Kind == tokenComma || tk.Kind == tokenSemiColon {
			p.mustnext(tk.Kind)
		} else {
			break
		}
	}
	return tblDefn, p.next(tokenCloseCurly)
}

func (p *Parser) tblSubTypeDef(fn *FnProto) (types.Definition, error) {
	p.mustnext(tokenOpenBracket)
	keyDefn, err := p.typestat(fn)
	if err != nil {
		return nil, err
	} else if err := p.next(tokenCloseBracket); err != nil {
		return nil, err
	}
	if ptk, err := p.peek(); err != nil {
		return nil, err
	} else if ptk.Kind != tokenColon {
		return &types.Table{
			Hint:    types.TblArray,
			KeyDefn: types.Int,
			ValDefn: keyDefn,
		}, nil
	}
	p.mustnext(tokenColon)
	valDefn, err := p.typestat(fn)
	if err != nil {
		return nil, err
	}
	return &types.Table{
		Hint:    types.TblMap,
		KeyDefn: keyDefn,
		ValDefn: valDefn,
	}, p.next(tokenCloseCurly)
}

func (p *Parser) localassign(fn *FnProto, decl *token) error {
	lcl0 := uint8(len(fn.Locals))
	names := []*Local{}
	for {
		ident, err := p.consumeToken(tokenIdentifier)
		if err != nil {
			return err
		}

		lcl := &Local{
			name:     ident.StringVal,
			typeDefn: types.Any,
		}

		if ptk, err := p.peek(); err != nil {
			return err
		} else if ptk.Kind == tokenColon { // type declaration
			p.mustnext(tokenColon)
			tk, err := p.consumeToken(tokenIdentifier)
			if err != nil {
				return err
			}
			typeDefn, err := fn.resolveType(tk.StringVal)
			if err != nil {
				return err
			}
			lcl.typeDefn = typeDefn
		}

		if ptk, err := p.peek(); err != nil {
			return err
		} else if ptk.Kind == tokenLt { // lua 1.4 const/close declarations
			p.mustnext(tokenLt)
			if tk, err := p.consumeToken(tokenIdentifier); err != nil {
				return err
			} else if tokenType(tk.StringVal) == "close" {
				lcl.attrClose = true
			} else if tokenType(tk.StringVal) == "const" {
				lcl.attrConst = true
			}
			if err := p.next(tokenGt); err != nil {
				return err
			}
		}

		names = append(names, lcl)
		if ptk, err := p.peek(); err != nil {
			return err
		} else if ptk.Kind != tokenComma {
			break
		}
		p.mustnext(tokenComma)
	}

	if ptk, err := p.peek(); err != nil {
		return err
	} else if ptk.Kind != tokenAssign {
		for _, lcl := range names {
			if err := fn.addLocal(lcl); err != nil {
				return err
			}
		}
		_, err := p.dischargeTo(fn, decl, &exNil{num: uint16(len(names) - 1)}, lcl0)
		return err
	}
	p.mustnext(tokenAssign)

	exprs, err := p.explistWant(fn, len(names))
	if err != nil {
		return err
	}
	for i, lcl := range names {
		if i < len(exprs) {
			if _, err := p.dischargeTo(fn, decl, exprs[i], lcl0+uint8(i)); err != nil {
				return err
			}
			defn, err := exprs[i].inferType()
			if err != nil {
				return err
			}
			// generalize numbers
			if defn == types.Int || defn == types.Float {
				defn = types.Number
			}
			lcl.typeDefn = defn
		}
		if err := fn.addLocal(lcl); err != nil {
			return err
		} else if lcl.attrClose {
			p.code(fn, bytecode.IAB(bytecode.TBC, lcl0+uint8(i), 0))
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

// funcargs -> '(' [ explist ] ')' | constructor | STRING.
func (p *Parser) funcargs(fn *FnProto) ([]expression, error) {
	ptk, err := p.peek()
	if err != nil {
		return nil, err
	}

	switch ptk.Kind {
	case tokenOpenParen:
		p.mustnext(tokenOpenParen)
		if ptk, err := p.peek(); err != nil {
			return nil, err
		} else if ptk.Kind == tokenCloseParen {
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
		return nil, p.parseErr(ptk, fmt.Errorf("unexpected token type %v while evaluating function call", ptk.Kind))
	}
}

// assignment -> suffixedexp { ',' suffixedexp } '=' explist.
func (p *Parser) assignment(fn *FnProto, first expression) error {
	sp0 := fn.stackPointer
	names := []expression{first}
	for {
		ptk, err := p.peek()
		if err != nil {
			return err
		} else if ptk.Kind != tokenComma {
			break
		}
		p.mustnext(tokenComma)
		expr, err := p.suffixedexp(fn)
		if err != nil {
			return err
		}
		names = append(names, expr)
	}
	tk, err := p.consumeToken(tokenAssign)
	if err != nil {
		return err
	}
	exprs, err := p.explistWant(fn, len(names))
	if err != nil {
		return err
	}
	for i, expr := range exprs {
		if _, err := p.dischargeTo(fn, tk, expr, sp0+uint8(i)); err != nil {
			return err
		}
	}
	// Pre-evaluate table and key operands of indexed LHS targets to temporary
	// registers. This prevents conflicts when a variable appearing as a table or
	// key in one target is reassigned by an earlier target in the same statement.
	// Lua semantics require all LHS targets to be resolved before any assignment.
	if len(names) > 1 {
		for _, name := range names {
			idx, ok := name.(*exIndex)
			if !ok {
				continue
			}
			if v, isVar := idx.table.(*exVariable); isVar {
				tmp := fn.stackPointer
				if err := v.discharge(fn, tmp); err != nil {
					return err
				}
				fn.stackPointer = tmp + 1
				idx.table = &exVariable{local: true, address: tmp, name: v.name, LineInfo: v.LineInfo}
			}
			if _, isConst := exIsConst(idx.key); !isConst {
				tmp := fn.stackPointer
				if err := idx.key.discharge(fn, tmp); err != nil {
					return err
				}
				fn.stackPointer = tmp + 1
				idx.key = &exVariable{local: true, address: tmp, LineInfo: idx.LineInfo}
			}
		}
	}
	for i, name := range names {
		var val expression = &exNil{}
		if i < len(exprs) {
			val = exprs[i]
		}
		if err := p.assignTo(fn, tk, name, sp0+uint8(i), val); err != nil {
			return err
		}
	}
	return nil
}

func (p *Parser) expression(fn *FnProto) (expression, error) {
	return p.expr(fn, 0)
}

// where 'binop' is any binary operator with a priority higher than 'limit'.
func (p *Parser) expr(fn *FnProto, limit int) (expression, error) {
	var desc expression
	if tk, err := p.peek(); err != nil {
		return nil, err
	} else if tk.isUnary() {
		if err = p.next(tk.Kind); err != nil {
			return nil, err
		} else if desc, err = p.expr(fn, unaryPriority); err != nil {
			return nil, err
		}
		desc = unaryExpression(tk, desc)
	} else if desc, err = p.simpleexp(fn); err != nil {
		return nil, err
	}
	op, err := p.peek()
	if err != nil {
		return nil, err
	}
	for op.isBinary() && binaryPriority[op.Kind][0] > limit {
		p.mustnext(op.Kind)
		rdesc, err := p.expr(fn, binaryPriority[op.Kind][1])
		if err != nil {
			return nil, err
		}
		desc = newInfixExpr(op, desc, rdesc)
		op, err = p.peek()
		if err != nil {
			return nil, err
		}
	}
	return desc, nil
}

func (p *Parser) dischargeAllButLast(fn *FnProto, tk *token, exprs []expression) (expression, error) {
	for i := range len(exprs) - 1 {
		if _, err := p.discharge(fn, tk, exprs[i]); err != nil {
			return nil, err
		}
	}
	return exprs[len(exprs)-1], nil
}

func (p *Parser) discharge(fn *FnProto, tk *token, exp expression) (uint8, error) {
	return p.dischargeTo(fn, tk, exp, fn.stackPointer)
}

func (p *Parser) dischargeTo(fn *FnProto, tk *token, exp expression, dst uint8) (uint8, error) {
	err := exp.discharge(fn, dst)
	fn.stackPointer = dst + 1
	return dst, p.parseErr(tk, err)
}

func (p *Parser) code(fn *FnProto, inst uint32) int {
	return fn.code(inst, p.lastTokenInfo)
}

// simpleexp -> Float | Integer | String | nil | true | false | ... | constructor | FUNCTION body | suffixedexp.
func (p *Parser) simpleexp(fn *FnProto) (expression, error) {
	ptk, err := p.peek()
	if err != nil {
		return nil, err
	}
	switch ptk.Kind {
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
			fnproto:  newFn,
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

// primaryexp -> NAME | '(' expr ')'.
func (p *Parser) primaryexp(fn *FnProto) (expression, error) {
	tk, err := p.peek()
	if err != nil {
		return nil, err
	}
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
		return nil, p.parseErr(tk, fmt.Errorf("unexpected symbol %v", tk.Kind))
	}
}

// funccallstat -> suffixedexp funcargs.
func (p *Parser) suffixedexp(fn *FnProto) (expression, error) {
	sp0 := fn.stackPointer
	expr, err := p.primaryexp(fn)
	if err != nil {
		return nil, err
	}
	for {
		ptk, err := p.peek()
		if err != nil {
			return nil, err
		}
		switch ptk.Kind {
		case tokenPeriod:
			p.mustnext(tokenPeriod)
			key, err := p.consumeToken(tokenIdentifier)
			if err != nil {
				return nil, err
			}
			expr = &exIndex{
				table:    expr,
				key:      &exString{val: key.StringVal, LineInfo: key.LineInfo},
				typeDefn: types.Any,
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
				typeDefn: types.Any,
				LineInfo: tk.LineInfo,
			}
		case tokenColon:
			p.mustnext(tokenColon)
			key, err := p.consumeToken(tokenIdentifier)
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
				typeDefn: &types.Function{},
				LineInfo: key.LineInfo,
			}
			expr = newCallExpr(fn, args, true, key.LineInfo)
		case tokenOpenParen, tokenString, tokenOpenCurly:
			tk, err := p.peek()
			if err != nil {
				return nil, err
			}
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

// name is a reference to a variable that need resolution to have meaning.
func (p *Parser) name(fn *FnProto, name *token) (expression, error) {
	if expr, err := p.resolveVar(fn, name); err != nil {
		return nil, err
	} else if expr != nil {
		return expr, nil
	}
	expr, err := p.name(fn, &token{StringVal: "_ENV", LineInfo: LineInfo{Line: name.Line, Column: name.Column}})
	if err != nil {
		return nil, err
	}
	varexpr, isValue := expr.(*exVariable)
	if !isValue {
		panic("did not find _ENV, this should never happen")
	}
	return &exIndex{
		table:    expr,
		key:      &exString{val: name.StringVal, LineInfo: name.LineInfo},
		typeDefn: varexpr.typeDefn,
		LineInfo: name.LineInfo,
	}, nil
}

// resolveVar will recursively look up the stack to find where the variable
// resides in the stack and then build the chain of upvars to have a referece
// to it.
func (p *Parser) resolveVar(fn *FnProto, name *token) (*exVariable, error) {
	if fn == nil {
		return nil, nil
	} else if idx, ok := search(fn.Locals, name.StringVal, findLocal); ok {
		lcl := fn.Locals[idx]
		return &exVariable{
			local:     true,
			name:      name.StringVal,
			address:   uint8(idx),
			lvar:      lcl,
			attrConst: lcl.attrConst,
			attrClose: lcl.attrClose,
			typeDefn:  lcl.typeDefn,
			LineInfo:  name.LineInfo,
		}, nil
	} else if idx, ok := search(fn.UpIndexes, name.StringVal, findUpindex); ok {
		return &exVariable{
			local:    false,
			name:     name.StringVal,
			address:  uint8(idx),
			typeDefn: fn.UpIndexes[idx].typeDefn,
			LineInfo: name.LineInfo,
		}, nil
	} else if value, err := p.resolveVar(fn.prev, name); err != nil {
		return nil, err
	} else if value != nil {
		if value.local {
			value.lvar.upvalRef = true
		}
		if err := fn.addUpindex(name.StringVal, value.address, value.local, value.typeDefn); err != nil {
			return nil, err
		}
		return &exVariable{
			local:    false,
			name:     name.StringVal,
			address:  uint8(len(fn.UpIndexes) - 1),
			typeDefn: value.typeDefn,
			LineInfo: name.LineInfo,
		}, nil
	}
	return nil, nil
}

// no matter how much of the stack was used up during computation of the expr.
func (p *Parser) explist(fn *FnProto) ([]expression, error) {
	list := []expression{}
	for {
		if err := p.skipComments(); err != nil {
			return nil, err
		}
		expr, err := p.expression(fn)
		if err != nil {
			return nil, err
		}
		list = append(list, expr)
		if err := p.skipComments(); err != nil {
			return nil, err
		}
		if ptk, err := p.peek(); err != nil {
			return nil, err
		} else if ptk.Kind != tokenComma {
			break
		}
		p.mustnext(tokenComma)
	}
	return list, nil
}

// field -> NAME = exp | '['exp']' = exp | exp.
func (p *Parser) constructor(fn *FnProto) (expression, error) {
	expr := &exTable{LineInfo: p.mustnext(tokenOpenCurly).LineInfo}
	for {
		ptk, err := p.peek()
		if err != nil {
			return nil, err
		}
		switch ptk.Kind {
		case tokenCloseCurly:
			// do nothing, because it is an empty table
		case tokenComment:
			p.mustnext(tokenComment)
			continue
		case tokenIdentifier:
			tk := p.mustnext(tokenIdentifier)
			ptk, err := p.peek()
			if err != nil {
				return nil, err
			}
			if ptk.Kind == tokenAssign {
				p.mustnext(tokenAssign)
				val, err := p.expr(fn, 0)
				if err != nil {
					return nil, err
				}
				expr.keys = append(expr.keys, &exString{val: tk.StringVal})
				expr.vals = append(expr.vals, val)
			} else {
				p.lex.back(tk)
				val, err := p.expr(fn, 0)
				if err != nil {
					return nil, err
				}
				expr.array = append(expr.array, val)
			}
		case tokenOpenBracket:
			p.mustnext(tokenOpenBracket)
			key, err := p.expr(fn, 0)
			if err != nil {
				return nil, err
			} else if err := p.next(tokenCloseBracket); err != nil {
				return nil, err
			} else if err := p.next(tokenAssign); err != nil {
				return nil, err
			}
			val, err := p.expr(fn, 0)
			if err != nil {
				return nil, err
			}
			expr.keys = append(expr.keys, key)
			expr.vals = append(expr.vals, val)
		default:
			val, err := p.expr(fn, 0)
			if err != nil {
				return nil, err
			}
			expr.array = append(expr.array, val)
		}
		ptk, err = p.peek()
		if err != nil {
			return nil, err
		}
		if ptk.Kind == tokenComma || ptk.Kind == tokenSemiColon {
			p.mustnext(ptk.Kind)
		} else {
			break
		}
	}
	return expr, p.next(tokenCloseCurly)
}

func toString(val any) string {
	switch tin := val.(type) {
	case nil:
		return "nil"
	case string:
		return tin
	case bool:
		return strconv.FormatBool(tin)
	case int64:
		return strconv.FormatInt(tin, 10)
	case float64:
		return fmt.Sprintf("%v", tin)
	default:
		return fmt.Sprintf("Unknown value type: %v", val)
	}
}

// this is good for slices of non-simple datatypes.
func search[S ~[]E, E, T any](x S, target T, cmp func(E, T) bool) (int, bool) {
	for i := range x {
		if cmp(x[i], target) {
			return i, true
		}
	}
	return -1, false
}

func findLocal(lcl *Local, name string) bool   { return name == lcl.name }
func findConst(k, name any) bool               { return k == name }
func findUpindex(ui Upindex, name string) bool { return name == ui.Name }
