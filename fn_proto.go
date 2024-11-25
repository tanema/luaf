package luaf

import (
	"bytes"
	"text/template"
)

type (
	UpIndex struct {
		fromStack bool
		name      string
		index     uint
	}
	Local struct {
		name      string
		upvalRef  bool
		attrConst bool
		attrClose bool
	}
	FnProto struct {
		Name         string
		Filename     string
		Line         int
		stackPointer uint8      //stack pointer
		prev         *FnProto   // parent FnProto or scope
		Varargs      bool       // if the function call has varargs
		Arity        int        // parameter count
		Constants    []any      // constant values to be loaded into the stack
		Locals       []*Local   // name mapped to stack index of where the local was loaded
		UpIndexes    []UpIndex  // name mapped to upindex
		ByteCodes    []Bytecode // bytecode for this function
		FnTable      []*FnProto // indexes of functions in constants
		Labels       map[string]int
		Gotos        map[string][]int
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

func newFnProto(filename, name string, prev *FnProto, params []string, vararg bool, line int) *FnProto {
	locals := make([]*Local, len(params))
	for i, p := range params {
		locals[i] = &Local{name: p}
	}
	return &FnProto{
		Filename:     filename,
		Name:         name,
		Line:         line,
		prev:         prev,
		Arity:        len(params),
		Varargs:      vararg,
		stackPointer: uint8(len(params)),
		Locals:       locals,
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

func (fn *FnProto) code(op Bytecode) int {
	fn.ByteCodes = append(fn.ByteCodes, op)
	return len(fn.ByteCodes) - 1
}

func (fnproto *FnProto) String() string {
	var buf bytes.Buffer
	if err := template.Must(template.New("fnproto").Parse(fnProtoTemplate)).Execute(&buf, fnproto); err != nil {
		panic(err)
	}
	return buf.String()
}
