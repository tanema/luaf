package shine

import (
	"bytes"
	"fmt"
)

type UpIndex struct {
	Local bool
	Index uint
}

type FuncProto struct {
	sp          uint8 //stack pointer
	prev        *FuncProto
	Varargs     bool
	Arity       int
	Constants   []Value
	Locals      []string           // name mapped to stack index of where the local was loaded
	UpIndexes   map[string]UpIndex // name mapped to upindex
	ByteCodes   []Bytecode
	Breakable   bool
	Continuable bool
}

func newFnProto(prev *FuncProto, params []string) *FuncProto {
	return &FuncProto{
		prev:      prev,
		Arity:     len(params),
		sp:        uint8(len(params)),
		Locals:    params,
		UpIndexes: map[string]UpIndex{},
	}
}

func (fn *FuncProto) addConst(val Value) uint16 {
	if idx := findValue(fn.Constants, val); idx >= 0 {
		return uint16(idx)
	}
	fn.Constants = append(fn.Constants, val)
	return uint16(len(fn.Constants) - 1)
}

func (fn *FuncProto) code(op Bytecode) {
	fn.ByteCodes = append(fn.ByteCodes, op)
}

func (fnproto *FuncProto) String() string {
	var out bytes.Buffer
	fmt.Fprintf(
		&out,
		"%v params, %v upvalue, %v local, %v constants\n",
		fnproto.Arity,
		len(fnproto.UpIndexes),
		len(fnproto.Locals),
		len(fnproto.Constants),
	)
	fmt.Fprintf(&out, "\nConstants:\n")
	for i, cnst := range fnproto.Constants {
		fmt.Fprintf(&out, "[%v] %v\n", i, cnst)
	}
	fmt.Fprintf(&out, "\nUpindexes:\n")
	for name, cnst := range fnproto.UpIndexes {
		fmt.Fprintf(&out, "[%v] %v local: %v\n", name, cnst.Index, cnst.Local)
	}
	fmt.Fprintf(&out, "\nBytecodes:\n")
	for i, bytecode := range fnproto.ByteCodes {
		fmt.Fprintf(&out, "[%v] %s\n", i, bytecode.String())
	}
	return out.String()
}
