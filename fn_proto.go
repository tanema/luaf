package shine

import (
	"errors"
	"fmt"
	"strings"
)

type (
	UpIndex struct {
		fromStack bool
		name      string
		index     uint
	}
	FuncProto struct {
		sp          uint8 //stack pointer
		prev        *FuncProto
		Varargs     bool
		Arity       int
		name        string
		Constants   []Value
		Locals      []Local   // name mapped to stack index of where the local was loaded
		UpIndexes   []UpIndex // name mapped to upindex
		ByteCodes   []Bytecode
		FnTable     []int // indexes of functions in constants
		Breakable   bool
		Continuable bool
	}
)

func newFnProto(prev *FuncProto, name string, params []string, vararg bool) *FuncProto {
	locals := make([]Local, len(params))
	for i, param := range params {
		locals[i] = Local{name: param}
	}
	return &FuncProto{
		prev:      prev,
		name:      name,
		Arity:     len(params),
		Varargs:   vararg,
		sp:        uint8(len(params)),
		Locals:    locals,
		UpIndexes: []UpIndex{},
	}
}

func (fn *FuncProto) addConst(val Value) uint16 {
	if idx := findValue(fn.Constants, val); idx >= 0 {
		return uint16(idx)
	}
	fn.Constants = append(fn.Constants, val)
	index := len(fn.Constants) - 1
	if val.Type() == "function" {
		fn.FnTable = append(fn.FnTable, index)
	}
	return uint16(index)
}

func (fn *FuncProto) getConst(idx int64) (Value, error) {
	if idx < 0 || int(idx) >= len(fn.Constants) {
		return nil, errors.New("Constant address out of bounds")
	}
	return fn.Constants[idx], nil
}

func (fn *FuncProto) code(op Bytecode) {
	fn.ByteCodes = append(fn.ByteCodes, op)
}

func (fnproto *FuncProto) String() string {
	codes := make([]string, len(fnproto.ByteCodes))
	for i, bytecode := range fnproto.ByteCodes {
		codes[i] = fmt.Sprintf("[%v] %s", i, bytecode.String())
	}
	fns := make([]string, len(fnproto.FnTable))
	for i, index := range fnproto.FnTable {
		fn := fnproto.Constants[index].(*Function)
		fns[i] = fmt.Sprintf("%s", fn.val.String())
	}
	return fmt.Sprintf("Function: %v\n%v params, %v upvalue, %v locals, %v constants\n\nbytecode\n%v\n\n%v",
		fnproto.name,
		fnproto.Arity,
		len(fnproto.UpIndexes),
		len(fnproto.Locals),
		len(fnproto.Constants),
		strings.Join(codes, "\n"),
		strings.Join(fns, ""),
	)
}

func findLocal(lcl Local, name string) int {
	return strings.Compare(name, lcl.name)
}

func findUpindex(upindex UpIndex, name string) int {
	return strings.Compare(name, upindex.name)
}
