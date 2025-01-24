package luaf

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"strings"
	"text/template"
)

type (
	fnDump struct {
		Signature string
		Version   string
		Format    int
		Main      *FnProto
	}
	UpIndex struct {
		FromStack bool
		Name      string
		Index     uint
	}
	local struct {
		name      string
		upvalRef  bool
		attrConst bool
		attrClose bool
	}
	LineInfo struct {
		Line   int
		Column int
	}
	FnProto struct {
		// parsing only data
		stackPointer uint8    //stack pointer
		prev         *FnProto // parent FnProto or scope
		locals       []*local // name mapped to stack index of where the local was loaded

		LineInfo
		Comment   string
		Name      string
		Filename  string
		Varargs   bool       // if the function call has varargs
		Arity     int        // parameter count
		Constants []any      // constant values to be loaded into the stack
		UpIndexes []UpIndex  // name mapped to upindex
		ByteCodes []Bytecode // bytecode for this function
		FnTable   []*FnProto // indexes of functions in constants
		Labels    map[string]int
		Gotos     map[string][]int
		LineTrace []LineInfo
	}
)

const fnProtoTemplate = `{{.Name}} <{{.Filename}}:{{.Line}}> ({{.ByteCodes | len}} instructions)
{{.Arity}}{{if .Varargs}}+{{end}} params, {{.UpIndexes | len}} upvalues, {{.Locals}} locals, {{.Constants | len}} constants, {{.FnTable | len}} functions
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
		Arity:        len(params),
		Varargs:      vararg,
		stackPointer: uint8(len(params)),
		locals:       locals,
		Labels:       map[string]int{},
		Gotos:        map[string][]int{},
	}
}

func (fn *FnProto) addFn(newfn *FnProto) uint16 {
	fn.FnTable = append(fn.FnTable, newfn)
	return uint16(len(fn.FnTable) - 1)
}

func (fn *FnProto) addLocals(names ...string) error {
	for _, lcl := range names {
		if err := fn.addLocal(lcl, false, false); err != nil {
			return err
		}
	}
	return nil
}

func (fn *FnProto) addLocal(name string, attrConst, attrClose bool) error {
	if len(fn.locals) == MAXLOCALS {
		return fmt.Errorf("local max hit while adding local %v", name)
	}
	fn.locals = append(fn.locals, &local{
		name:      name,
		attrConst: attrConst,
		attrClose: attrClose,
	})
	fn.stackPointer = uint8(len(fn.locals))
	return nil
}

func (fn *FnProto) addConst(val any) (uint16, error) {
	if i, found := search(fn.Constants, val, findConst); found {
		return uint16(i), nil
	}
	if len(fn.Constants) == MAXCONST {
		return 0, fmt.Errorf("constant max hit while adding %v", val)
	}
	fn.Constants = append(fn.Constants, val)
	return uint16(len(fn.Constants) - 1), nil
}

func (fn *FnProto) getConst(idx int64) Value {
	return ToValue(fn.Constants[idx])
}

func (fn *FnProto) addUpindex(name string, index uint, stack bool) error {
	if len(fn.UpIndexes) == MAXUPVALUES {
		return fmt.Errorf("up value max hit while adding %v", name)
	}
	fn.UpIndexes = append(fn.UpIndexes, UpIndex{FromStack: stack, Name: name, Index: index})
	return nil
}

func (fn *FnProto) code(op Bytecode, linfo LineInfo) int {
	fn.ByteCodes = append(fn.ByteCodes, op)
	fn.LineTrace = append(fn.LineTrace, linfo)
	return len(fn.ByteCodes) - 1
}

func (fnproto *FnProto) String() string {
	var buf bytes.Buffer
	tmpl := template.New("fnproto")
	tmpl.Funcs(map[string]any{
		"codeMeta": func(op Bytecode) string {
			if op.op() == LOADK {
				return fmt.Sprintf("\t%q", fnproto.getConst(op.getsBx()).String())
			} else if op.op() == LOADI {
				return fmt.Sprintf("\t%v", op.getsBx())
			} else if op.op() == LOADF {
				return fmt.Sprintf("\t%v.0", op.getsBx())
			} else if op.op() == CALL {
				return fmt.Sprintf("\t%s in %s out", optionVariable(op.getB()), optionVariable(op.getC()))
			} else if op.op() == CLOSURE {
				return fmt.Sprintf("\t%s", fnproto.FnTable[op.getB()].Name)
			} else if op.op() == TAILCALL {
				return fmt.Sprintf("\t%s in all out", optionVariable(op.getB()))
			} else if op.op() == RETURN {
				return fmt.Sprintf("\t%s out", optionVariable(op.getB()))
			}
			if op.Kind() == BytecodeTypeABC {
				b, bK := op.getBK()
				c, cK := op.getCK()
				out := []string{}
				if bK {
					out = append(out, fmt.Sprintf(`"%v"`, fnproto.getConst(b).String()))
				} else if inst := op.op(); (inst == GETTABUP || inst == SETTABUP) && b == 0 {
					out = append(out, "_ENV")
				}
				if cK {
					out = append(out, fmt.Sprintf(`"%v"`, fnproto.getConst(c).String()))
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
		FnProto: fnproto,
		Locals:  len(fnproto.locals),
	}

	if err := tmpl.Execute(&buf, data); err != nil {
		panic(err)
	}
	return buf.String()
}

// TODO, should be binary encoding but this is easier for now
func (fnproto *FnProto) Dump(strip bool) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(fnDump{
		Signature: LUA_SIGNATURE,
		Version:   LUA_VERSION,
		Format:    LUA_FORMAT,
		Main:      fnproto,
	}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func UndumpFnProto(data io.Reader) (*FnProto, error) {
	fnd := &fnDump{}
	dec := gob.NewDecoder(data)
	return fnd.Main, dec.Decode(fnd)
}

func optionVariable(param int64) string {
	narg := (param - 1)
	if narg < 0 {
		return "all"
	}
	return fmt.Sprint(narg)
}
