package luaf

import (
	"bytes"
	"encoding/gob"
	"io"
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
{{.Arity}}{{if .Varargs}}+{{end}} params, {{.UpIndexes | len}} upvalues, {{.Locals | len}} locals, {{.Constants | len}} constants, {{.FnTable | len}} functions
{{- range $i, $code := .ByteCodes}}
	[{{$i}}] {{$code -}}
{{end}}

{{range .FnTable -}}
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

func (fn *FnProto) addConst(val any) uint16 {
	for i, v := range fn.Constants {
		if v == val {
			return uint16(i)
		}
	}
	fn.Constants = append(fn.Constants, val)
	return uint16(len(fn.Constants) - 1)
}

func (fn *FnProto) getConst(idx int64) Value {
	return ToValue(fn.Constants[idx])
}

func (fn *FnProto) code(op Bytecode, linfo LineInfo) int {
	fn.ByteCodes = append(fn.ByteCodes, op)
	fn.LineTrace = append(fn.LineTrace, linfo)
	return len(fn.ByteCodes) - 1
}

func (fnproto *FnProto) String() string {
	var buf bytes.Buffer
	tmpl := template.Must(template.New("fnproto").Parse(fnProtoTemplate))
	tmpl.Funcs(map[string]any{
		"codeMeta": func(op Bytecode) string {
			return ""
		},
	})
	if err := tmpl.Execute(&buf, fnproto); err != nil {
		panic(err)
	}
	return buf.String()
}

func (fnproto *FnProto) Dump() ([]byte, error) {
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
