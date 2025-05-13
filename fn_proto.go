package luaf

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"strconv"
	"strings"
	"text/template"

	"github.com/pkg/errors"
)

type (
	upindex struct {
		Name      string
		FromStack bool
		Index     uint8
	}
	local struct {
		name      string
		upvalRef  bool
		attrConst bool
		attrClose bool
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
	// LineInfo is a shared info for keeping track of position for parsed artifacts.
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
		locals    []*local // name mapped to stack index of where the local was loaded
		labels    []map[string]labelEntry
		Constants []any      // constant values to be loaded into the stack
		UpIndexes []upindex  // name mapped to upindex
		ByteCodes []Bytecode // bytecode for this function
		FnTable   []*FnProto // indexes of functions in constants
		LineTrace []LineInfo

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
	{{$i}}	[{{with $li := index $.LineTrace $i}}{{$li.Line}}{{end}}]	{{$code}} ; {{$code | codeMeta -}}
{{end}}
{{range .FnTable}}
{{. -}}
{{end}}`

func newFnProto(filename, name string, prev *FnProto, params []string, vararg bool, linfo LineInfo) *FnProto {
	locals := make([]*local, len(params))
	for i, p := range params {
		locals[i] = &local{name: p}
	}
	return &FnProto{
		Filename:     filename,
		Name:         name,
		LineInfo:     linfo,
		prev:         prev,
		Arity:        int64(len(params)),
		Varargs:      vararg,
		stackPointer: uint8(len(params)),
		locals:       locals,
		labels:       []map[string]labelEntry{},
		gotos:        map[string][]gotoEntry{},
	}
}

// used for repl.
func newFnProtoFrom(fn *FnProto) *FnProto {
	return &FnProto{
		Filename:     fn.Filename,
		Name:         fn.Name,
		LineInfo:     fn.LineInfo,
		prev:         fn.prev,
		Arity:        fn.Arity,
		Varargs:      fn.Varargs,
		locals:       fn.locals,
		Constants:    fn.Constants,
		FnTable:      fn.FnTable,
		UpIndexes:    fn.UpIndexes,
		stackPointer: fn.stackPointer,
	}
}

func (fn *FnProto) addFn(newfn *FnProto) uint16 {
	fn.FnTable = append(fn.FnTable, newfn)
	return uint16(len(fn.FnTable) - 1)
}

func (fn *FnProto) addLocals(names ...string) error {
	for _, lcl := range names {
		if err := fn.addLocal(&local{name: lcl}); err != nil {
			return err
		}
	}
	return nil
}

func (fn *FnProto) addLocal(lcl *local) error {
	if len(fn.locals) == MAXLOCALS {
		return fmt.Errorf("local overflow while adding local %v", lcl.name)
	}
	fn.locals = append(fn.locals, lcl)
	fn.stackPointer = uint8(len(fn.locals))
	return nil
}

func (fn *FnProto) addConst(val any) (uint16, error) {
	if i, found := search(fn.Constants, val, findConst); found {
		return uint16(i), nil
	}
	if len(fn.Constants) == MAXCONST {
		return 0, fmt.Errorf("constant overflow while adding %v", val)
	}
	fn.Constants = append(fn.Constants, val)
	return uint16(len(fn.Constants) - 1), nil
}

func (fn *FnProto) getConst(idx int64) any {
	return fn.Constants[idx]
}

func (fn *FnProto) addUpindex(name string, index uint8, stack bool) error {
	if len(fn.UpIndexes) == MAXUPVALUES {
		return fmt.Errorf("up value overflow while adding %v", name)
	}
	fn.UpIndexes = append(fn.UpIndexes, upindex{FromStack: stack, Name: name, Index: index})
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
				return p.parseErrf(entry.token, "no visible label '%s' for <goto>", entry.label)
			}
		}
	}
	return nil
}

func (fn *FnProto) code(op Bytecode, linfo LineInfo) int {
	fn.ByteCodes = append(fn.ByteCodes, op)
	fn.LineTrace = append(fn.LineTrace, linfo)
	return len(fn.ByteCodes) - 1
}

func (fn *FnProto) String() string {
	var buf bytes.Buffer
	tmpl := template.New("fnproto")
	tmpl.Funcs(map[string]any{
		"codeMeta": func(op Bytecode) string {
			if op.op() == LOADK {
				return fmt.Sprintf("\t%q", ToString(fn.getConst(op.getsBx())))
			} else if op.op() == LOADI {
				return fmt.Sprintf("\t%v", op.getsBx())
			} else if op.op() == LOADF {
				return fmt.Sprintf("\t%v.0", op.getsBx())
			} else if op.op() == CALL {
				return fmt.Sprintf("\t%s in %s out", optionVariable(op.getB()), optionVariable(op.getC()))
			} else if op.op() == CLOSURE {
				return "\t" + fn.FnTable[op.getB()].Name
			} else if op.op() == TAILCALL {
				return fmt.Sprintf("\t%s in all out", optionVariable(op.getB()))
			} else if op.op() == RETURN {
				return fmt.Sprintf("\t%s out", optionVariable(op.getB()))
			} else if op.op() == VARARG {
				return fmt.Sprintf("\t%s in", optionVariable(op.getB()))
			} else if op.op() == SETLIST {
				return fmt.Sprintf("\t%s in at index %v", optionVariable(op.getB()), op.getC())
			}
			if op.kind() == bytecodeTypeABC {
				b, bK := op.getBK()
				c, cK := op.getCK()
				out := []string{}
				if bK {
					out = append(out, fmt.Sprintf(`"%v"`, ToString(fn.getConst(b))))
				} else if inst := op.op(); (inst == GETTABUP || inst == SETTABUP) && b == 0 {
					out = append(out, "_ENV")
				}
				if cK {
					out = append(out, fmt.Sprintf(`"%v"`, ToString(fn.getConst(c))))
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
		Locals:  len(fn.locals),
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
	if _, err := src.Read(prefix); err != nil {
		return false
	} else if strings.HasPrefix(string(prefix), LUASIGNATURE) {
		return true
	} else if _, err := src.Seek(0, io.SeekStart); err != nil {
		return false
	}
	return false
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
		dump(buf, end, LUASIGNATURE),
		dump(buf, end, LUAVERSION),
		dump(buf, end, int8(LUAFORMAT)),
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
	if signature != LUASIGNATURE {
		return errors.New("invalid signature")
	} else if version != LUAVERSION {
		return fmt.Errorf("unsupported version, current %v, found %v", LUAVERSION, version)
	} else if format != LUAFORMAT {
		return fmt.Errorf("unsupported format, current %v, found %v", LUAFORMAT, format)
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
		return errors.Wrap(err, "dumpByteCodes")
	}
	for _, code := range fn.ByteCodes {
		if err := dump(buf, end, uint32(code)); err != nil {
			return err
		}
	}
	return nil
}

func undumpByteCodes(buf io.Reader, end binary.ByteOrder, fn *FnProto) error {
	var size int64
	if err := undump(buf, end, &size); err != nil {
		return errors.Wrap(err, "undumpFnTable")
	}
	fn.ByteCodes = make([]Bytecode, size)
	for i := range size {
		var code uint32
		if err := undump(buf, end, &code); err != nil {
			return err
		}
		fn.ByteCodes[i] = Bytecode(code)
	}
	return nil
}

func dumpConstants(buf *[]byte, end binary.ByteOrder, fn *FnProto) error {
	if err := dump(buf, end, int64(len(fn.Constants))); err != nil {
		return errors.Wrap(err, "dumpConstants")
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
		return errors.Wrap(err, "undumpConstants")
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
		return errors.Wrap(err, "dumpUpvals")
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
		return errors.Wrap(err, "undumpUpvals: ")
	}
	fn.UpIndexes = make([]upindex, size)
	for i := range size {
		index := upindex{}
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
		return errors.Wrap(err, "dumpFnTable: ")
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
		return errors.Wrap(err, "undumpFnTable")
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
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64,
		float32, float64, []byte:
		*buf, err = binary.Append(*buf, end, tval)
	case string:
		*buf, err = binary.Append(*buf, end, []byte(fmt.Sprintf("%s\000", val)))
	}
	return errors.Wrap(err, "dump: ")
}

func undump(buf io.Reader, end binary.ByteOrder, val any) error {
	switch tval := val.(type) {
	case *string:
		strBuf := []byte{}
		for {
			var b byte
			if err := binary.Read(buf, end, &b); err != nil {
				return errors.Wrap(err, "undump string: ")
			} else if b == '\000' {
				break
			}
			strBuf = append(strBuf, b)
		}
		*tval = string(strBuf)
		return nil
	default:
		if err := binary.Read(buf, end, val); err != nil {
			return errors.Wrap(err, "undump: ")
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
