package parse

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"maps"
	"strconv"
	"strings"
	"text/template"

	"github.com/tanema/luaf/internal/bytecode"
	"github.com/tanema/luaf/internal/conf"
	"github.com/tanema/luaf/internal/types"
)

const _ENVName = "_ENV"

type (
	// Upindex captures an upvalue position for fetching them during runtime.
	Upindex struct {
		Name      string
		FromStack bool
		typeDefn  types.Definition
		Index     uint8
	}
	// Local is a local variable refence.
	Local struct {
		name      string
		upvalRef  bool
		attrConst bool
		attrClose bool
		typeDefn  types.Definition
	}
	labelEntry struct {
		token *token
		label string
		pc    int
	}
	gotoEntry struct {
		token *token
		label string
		pc    int
		level int
	}
	// LineInfo is a shared struct that is used for tracking where the behviour
	// originated from in the sourcecode.
	LineInfo struct {
		Line   int64
		Column int64
	}
	// FnProto is a construct that captures a function scope that can be called.
	// it is not always a function, even the main scope of a file outside of a function
	// is a FnProto.
	FnProto struct {
		prev  *FnProto // parent FnProto or scope
		gotos map[string][]gotoEntry

		Name      string
		Filename  string
		Comment   string
		Locals    []*Local // name mapped to stack index of where the local was loaded
		labels    []map[string]labelEntry
		Constants []any      // constant values to be loaded into the stack
		UpIndexes []Upindex  // name mapped to upindex
		ByteCodes []uint32   // bytecode for this function
		FnTable   []*FnProto // indexes of functions in constants
		LineTrace []LineInfo

		defn     *types.Function
		typeDefs map[string]types.Definition

		LineInfo
		Arity int64 // parameter count
		// parsing only data
		stackPointer uint8 // stack pointer
		Varargs      bool  // if the function call has varargs
	}
)

const fnProtoTemplate = `{{.Name}} <{{.Filename}}:{{.Line}}> ({{.ByteCodes | len}} instructions)
{{.Arity}}{{if .Varargs}}+{{end}} params, {{.UpIndexes | len}} upvalues,
{{- .Locals}} locals,
{{- .Constants | len}} constants, {{.FnTable | len}} functions
{{- range $i, $code := .ByteCodes}}
	{{$i}}	[{{with $li := index $.LineTrace $i}}{{$li.Line}}{{end}}]	{{$code | bctostr}}; {{$code | codeMeta -}}
{{end}}
{{range .FnTable}}
{{. -}}
{{end}}`

// NewFnProto creates a new FnProto for parsing. It is the result from parsing that
// contains the bytecode and debugging information for if an error happens.
func NewFnProto(
	filename, name string, prev *FnProto, params []*Local, vararg bool, defn *types.Function, linfo LineInfo,
) *FnProto {
	return &FnProto{
		Filename:     filename,
		Name:         name,
		LineInfo:     linfo,
		prev:         prev,
		Arity:        int64(len(params)),
		Varargs:      vararg,
		stackPointer: uint8(len(params)),
		Locals:       params,
		labels:       []map[string]labelEntry{},
		gotos:        map[string][]gotoEntry{},
		defn:         defn,
		typeDefs:     map[string]types.Definition{},
	}
}

func newRootFn() *FnProto {
	params := []*Local{{name: _ENVName, typeDefn: types.NewTable()}}
	typeDefs := map[string]types.Definition{}
	maps.Copy(typeDefs, types.DefaultDefns)
	return &FnProto{
		Name:         "env",
		Arity:        int64(len(params)),
		stackPointer: uint8(len(params)),
		Locals:       params,
		defn: &types.Function{
			Params: []types.NamedPair{{Name: _ENVName, Defn: types.NewTable()}},
			Return: []types.Definition{types.Any},
		},
		typeDefs: typeDefs,
	}
}

// NewEmptyFnProto creates a new fnproto without any parsing. This is mainly used
// by the runtime package for running a repl.
func NewEmptyFnProto(name string, rootFn *FnProto) *FnProto {
	if rootFn == nil {
		rootFn = newRootFn()
	}
	return NewFnProto(
		name,
		"main",
		rootFn,
		[]*Local{},
		true,
		&types.Function{
			Params: []types.NamedPair{},
			Return: []types.Definition{types.Any},
		},
		LineInfo{},
	)
}

// NewFnProtoFrom creates a new FnProto from another, used for repl.
func NewFnProtoFrom(fn *FnProto) *FnProto {
	return &FnProto{
		Filename:     fn.Filename,
		Name:         fn.Name,
		LineInfo:     fn.LineInfo,
		prev:         fn.prev,
		Arity:        fn.Arity,
		Varargs:      fn.Varargs,
		Locals:       fn.Locals,
		Constants:    fn.Constants,
		FnTable:      fn.FnTable,
		UpIndexes:    fn.UpIndexes,
		stackPointer: fn.stackPointer,
		defn:         fn.defn,
		typeDefs:     map[string]types.Definition{},
	}
}

func (fn *FnProto) addFn(newfn *FnProto) uint16 {
	fn.FnTable = append(fn.FnTable, newfn)
	return uint16(len(fn.FnTable) - 1)
}

func (fn *FnProto) addLocal(lcl *Local) error {
	if len(fn.Locals) == conf.MAXLOCALS {
		return fmt.Errorf("local overflow while adding local %v", lcl.name)
	}
	fn.Locals = append(fn.Locals, lcl)
	fn.stackPointer = uint8(len(fn.Locals))
	return nil
}

func (fn *FnProto) addConst(val any) (uint16, error) {
	if i, found := search(fn.Constants, val, findConst); found {
		return uint16(i), nil
	}
	if len(fn.Constants) == conf.MAXCONST {
		return 0, fmt.Errorf("constant overflow while adding %v", val)
	}
	fn.Constants = append(fn.Constants, val)
	return uint16(len(fn.Constants) - 1), nil
}

func (fn *FnProto) addType(name string, defn types.Definition, local bool) error {
	if _, found := fn.typeDefs[name]; found {
		return fmt.Errorf("type %s already defined", name)
	} else if !local && fn.prev != nil {
		return fn.prev.addType(name, defn, local)
	}
	fn.typeDefs[name] = defn
	return nil
}

func (fn *FnProto) resolveType(name string) (types.Definition, error) {
	if defn, found := fn.typeDefs[name]; found {
		return defn, nil
	} else if fn.prev != nil {
		return fn.prev.resolveType(name)
	}
	return nil, fmt.Errorf("unknown type definition %s", name)
}

// GetConst gets a constant from predefined constants in the fn.
func (fn *FnProto) GetConst(idx int64) any {
	if idx < 0 || int(idx) >= len(fn.Constants) {
		return nil
	}
	return fn.Constants[idx]
}

func (fn *FnProto) addUpindex(name string, index uint8, stack bool, defn types.Definition) error {
	if len(fn.UpIndexes) == conf.MAXUPVALUES {
		return fmt.Errorf("up value overflow while adding %v", name)
	}
	fn.UpIndexes = append(fn.UpIndexes, Upindex{FromStack: stack, Name: name, Index: index, typeDefn: defn})
	return nil
}

func (fn *FnProto) findLabel(label string) *labelEntry {
	for i := len(fn.labels) - 1; i >= 0; i-- {
		if entry, found := fn.labels[i][label]; found {
			return &entry
		}
	}
	return nil
}

func (fn *FnProto) checkGotos(p *Parser) error {
	if len(fn.gotos) > 0 {
		for label := range fn.gotos {
			for _, entry := range fn.gotos[label] {
				return p.parseErr(entry.token, fmt.Errorf("no visible label '%s' for <goto>", entry.label))
			}
		}
	}
	return nil
}

// finalize is the final step that does the following:
// - ensures that the function ends with a return statement (simplifies VM).
// - validates that all gotos have a destination.
// future:
// - optimizes the generated bytecode.
//   - JMP threading: unconditional jumps are folded.
//   - Null op: usesless loads and/or math operations that do nothing like multiplication by 1
//   - Specialize Ops Add -> AddI
//   - Duplicate load values, LoadI 0, 1, LOADI 1, 1, ADD 0, 0, 1 => LoadI 0, 1, ADD 0, 0, 0
func (fn *FnProto) finalize(p *Parser) error {
	if len(fn.ByteCodes) == 0 || !bytecode.IsReturn(fn.ByteCodes[len(fn.ByteCodes)-1]) {
		p.code(fn, bytecode.Return(0, 0))
	}

	return fn.checkGotos(p)
}

func (fn *FnProto) code(op uint32, linfo LineInfo) int {
	fn.ByteCodes = append(fn.ByteCodes, op)
	fn.LineTrace = append(fn.LineTrace, linfo)
	return len(fn.ByteCodes) - 1
}

func (fn *FnProto) String() string {
	var buf bytes.Buffer
	tmpl := template.New("fnproto")
	tmpl.Funcs(map[string]any{
		"bctostr": func(op uint32) string {
			return bytecode.ToString(op)
		},
		"codeMeta": func(op uint32) string {
			switch bytecode.GetOp(op) {
			case bytecode.NEWTABLE:
				return fmt.Sprintf("\tindexed: %d, keyed: %d", bytecode.GetvB(op), bytecode.GetvC(op))
			case bytecode.LOADK, bytecode.LOADKX:
				return fmt.Sprintf("\tconstant: %q", toString(fn.GetConst(bytecode.GetBx(op))))
			case bytecode.LOADI:
				return fmt.Sprintf("\tconstant: %v", bytecode.GetsBx(op))
			case bytecode.LOADF:
				return fmt.Sprintf("\tconstant: %v.0", bytecode.GetsBx(op))
			case bytecode.CALL:
				return fmt.Sprintf("\t%s params, %s returns", optionVariable(bytecode.GetB(op)), optionVariable(bytecode.GetC(op)))
			case bytecode.CLOSURE:
				return "\t" + fn.FnTable[bytecode.GetB(op)].Name
			case bytecode.TAILCALL:
				return fmt.Sprintf("\t%s params, variable returns", optionVariable(bytecode.GetB(op)))
			case bytecode.RETURN:
				return fmt.Sprintf("\t%s return values", optionVariable(bytecode.GetB(op)))
			case bytecode.VARARG:
				return fmt.Sprintf("\t%s values", optionVariable(bytecode.GetB(op)))
			case bytecode.SETLIST:
				return fmt.Sprintf("\t%s values in from stack index %v", optionVariable(bytecode.GetvB(op)), bytecode.GetvC(op))
			}
			if bytecode.Kind(op) == bytecode.TypeABC {
				b := bytecode.GetB(op)
				c := bytecode.GetC(op)
				out := []string{}
				if inst := bytecode.GetOp(op); (inst == bytecode.GETTABUP || inst == bytecode.SETTABUP) && b == 0 {
					out = append(out, _ENVName)
				}
				if bytecode.GetK(op) {
					switch bytecode.GetOp(op) {
					case bytecode.SELF, bytecode.GETTABLE, bytecode.GETTABUP, bytecode.SETTABLE,
						bytecode.SETTABUP, bytecode.SETI, bytecode.SETFIELD:
						out = append(out, fmt.Sprintf(`"%v"`, toString(fn.GetConst(c))))
					case bytecode.NOT, bytecode.LEN:
						out = append(out, fmt.Sprintf(`"%v"`, toString(fn.GetConst(b))))
					}
				}
				return "\t" + strings.Join(out, " ")
			}
			return ""
		},
	})
	tmpl = template.Must(tmpl.Parse(fnProtoTemplate))
	data := struct {
		*FnProto
		Locals int
	}{
		FnProto: fn,
		Locals:  len(fn.Locals),
	}

	if err := tmpl.Execute(&buf, data); err != nil {
		panic(err)
	}
	return buf.String()
}

func optionVariable(param int64) string {
	narg := (param - 1)
	if narg < 0 {
		return "all"
	}
	return strconv.FormatInt(narg, 10)
}

// Dump will serialize fnproto data into a byte array for writing out to a file.
func (fn *FnProto) Dump(_ bool) ([]byte, error) {
	var end binary.ByteOrder = binary.NativeEndian
	buf := []byte{}
	return buf, anyerr([]error{
		dumpHeader(&buf, end),
		dumpFn(&buf, end, fn),
	})
}

func hasLuaBinPrefix(src io.ReadSeeker) bool {
	prefix := make([]byte, 256)
	n, err := src.Read(prefix)
	if _, seekErr := src.Seek(0, io.SeekStart); seekErr != nil {
		return false
	} else if err != nil && !errors.Is(err, io.EOF) {
		return false
	}
	return strings.HasPrefix(string(prefix[:n]), conf.LUASIGNATURE)
}

// UndumpFnProto will deserialize fnproto data into a new fnproto ready for interpreting.
func UndumpFnProto(buf io.Reader) (*FnProto, error) {
	var end binary.ByteOrder = binary.NativeEndian
	fn := &FnProto{}
	return fn, anyerr([]error{
		undumpHeader(buf, end),
		undumpFn(buf, end, fn),
	})
}

func dumpHeader(buf *[]byte, end binary.ByteOrder) error {
	return anyerr([]error{
		dump(buf, end, conf.LUASIGNATURE),
		dump(buf, end, conf.LUAVERSION),
		dump(buf, end, int8(conf.LUAFORMAT)),
	})
}

func undumpHeader(buf io.Reader, end binary.ByteOrder) error {
	var signature, version string
	var format int8
	if err := anyerr([]error{
		undump(buf, end, &signature),
		undump(buf, end, &version),
		undump(buf, end, &format),
	}); err != nil {
		return err
	}
	if signature != conf.LUASIGNATURE {
		return errors.New("invalid signature")
	} else if version != conf.LUAVERSION {
		return fmt.Errorf("unsupported version, current %v, found %v", conf.LUAVERSION, version)
	} else if format != conf.LUAFORMAT {
		return fmt.Errorf("unsupported format, current %v, found %v", conf.LUAFORMAT, format)
	}
	return nil
}

func dumpFn(buf *[]byte, end binary.ByteOrder, fn *FnProto) error {
	return anyerr([]error{
		dump(buf, end, fn.Line),
		dump(buf, end, fn.Arity),
		dumpByteCodes(buf, end, fn),
		dumpConstants(buf, end, fn),
		dumpUpvals(buf, end, fn),
		dumpFnTable(buf, end, fn),
	})
}

func undumpFn(buf io.Reader, end binary.ByteOrder, fn *FnProto) error {
	return anyerr([]error{
		undump(buf, end, &fn.Line),
		undump(buf, end, &fn.Arity),
		undumpByteCodes(buf, end, fn),
		undumpConstants(buf, end, fn),
		undumpUpvals(buf, end, fn),
		undumpFnTable(buf, end, fn),
	})
}

func dumpByteCodes(buf *[]byte, end binary.ByteOrder, fn *FnProto) error {
	if err := dump(buf, end, int64(len(fn.ByteCodes))); err != nil {
		return fmt.Errorf("dumpByteCodes: %w", err)
	}
	for _, code := range fn.ByteCodes {
		if err := dump(buf, end, code); err != nil {
			return err
		}
	}
	return nil
}

func undumpByteCodes(buf io.Reader, end binary.ByteOrder, fn *FnProto) error {
	var size int64
	if err := undump(buf, end, &size); err != nil {
		return fmt.Errorf("undumpFnTable: %w", err)
	}
	fn.ByteCodes = make([]uint32, size)
	for i := range size {
		var code uint32
		if err := undump(buf, end, &code); err != nil {
			return err
		}
		fn.ByteCodes[i] = code
	}
	return nil
}

func dumpConstants(buf *[]byte, end binary.ByteOrder, fn *FnProto) error {
	if err := dump(buf, end, int64(len(fn.Constants))); err != nil {
		return fmt.Errorf("dumpConstants: %w", err)
	}
	for _, konst := range fn.Constants {
		switch konst.(type) {
		case string:
			if err := dump(buf, end, 's'); err != nil {
				return err
			}
		case float64:
			if err := dump(buf, end, 'f'); err != nil {
				return err
			}
		case int64:
			if err := dump(buf, end, 'i'); err != nil {
				return err
			}
		}
		if err := dump(buf, end, konst); err != nil {
			return err
		}
	}
	return nil
}

func undumpConstants(buf io.Reader, end binary.ByteOrder, fn *FnProto) error {
	var size int64
	if err := undump(buf, end, &size); err != nil {
		return fmt.Errorf("undumpConstants: %w", err)
	}
	fn.Constants = make([]any, size)
	for i := range size {
		var kind rune
		if err := undump(buf, end, &kind); err != nil {
			return err
		}
		switch kind {
		case 's':
			var val string
			if err := undump(buf, end, &val); err != nil {
				return err
			}
			fn.Constants[i] = val
		case 'f':
			var val float64
			if err := undump(buf, end, &val); err != nil {
				return err
			}
			fn.Constants[i] = val
		case 'i':
			var val int64
			if err := undump(buf, end, &val); err != nil {
				return err
			}
			fn.Constants[i] = val
		}
	}
	return nil
}

func dumpUpvals(buf *[]byte, end binary.ByteOrder, fn *FnProto) error {
	if err := dump(buf, end, int64(len(fn.UpIndexes))); err != nil {
		return fmt.Errorf("dumpUpvals: %w", err)
	}
	for _, index := range fn.UpIndexes {
		if err := anyerr([]error{
			dump(buf, end, index.FromStack),
			dump(buf, end, index.Index),
			dump(buf, end, index.Name),
		}); err != nil {
			return err
		}
	}
	return nil
}

func undumpUpvals(buf io.Reader, end binary.ByteOrder, fn *FnProto) error {
	var size int64
	if err := undump(buf, end, &size); err != nil {
		return fmt.Errorf("undumpUpvals: %w", err)
	}
	fn.UpIndexes = make([]Upindex, size)
	for i := range size {
		index := Upindex{}
		if err := anyerr([]error{
			undump(buf, end, &index.FromStack),
			undump(buf, end, &index.Index),
			undump(buf, end, &index.Name),
		}); err != nil {
			return err
		}
		fn.UpIndexes[i] = index
	}
	return nil
}

func dumpFnTable(buf *[]byte, end binary.ByteOrder, fn *FnProto) error {
	if err := dump(buf, end, int64(len(fn.FnTable))); err != nil {
		return fmt.Errorf("dumpFnTable: %w", err)
	}
	for _, proto := range fn.FnTable {
		if err := dumpFn(buf, end, proto); err != nil {
			return err
		}
	}
	return nil
}

func undumpFnTable(buf io.Reader, end binary.ByteOrder, fn *FnProto) error {
	var size int64
	if err := undump(buf, end, &size); err != nil {
		return fmt.Errorf("undumpFnTable: %w", err)
	}
	fn.FnTable = make([]*FnProto, size)
	for i := range size {
		proto := &FnProto{}
		if err := undumpFn(buf, end, proto); err != nil {
			return err
		}
		fn.FnTable[i] = proto
	}
	return nil
}

func dump(buf *[]byte, end binary.ByteOrder, val any) error {
	var err error
	switch tval := val.(type) {
	case bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64,
		float32, float64, []byte:
		*buf, err = binary.Append(*buf, end, tval)
	case string:
		*buf, err = binary.Append(*buf, end, fmt.Appendf(nil, "%s\000", val))
	default:
		return fmt.Errorf("dump: unsupported type %T", val)
	}
	if err != nil {
		return fmt.Errorf("dump: %w", err)
	}
	return nil
}

func undump(buf io.Reader, end binary.ByteOrder, val any) error {
	switch tval := val.(type) {
	case *string:
		strBuf := []byte{}
		for {
			var b byte
			if err := binary.Read(buf, end, &b); err != nil {
				return fmt.Errorf("undump string: %w", err)
			} else if b == '\000' {
				break
			}
			strBuf = append(strBuf, b)
		}
		*tval = string(strBuf)
		return nil
	default:
		if err := binary.Read(buf, end, val); err != nil {
			return fmt.Errorf("undump: %w", err)
		}
		return nil
	}
}

func anyerr(errs []error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}
