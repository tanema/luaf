package shine

import (
	"bytes"
	"fmt"
)

type FuncProto struct {
	sp          uint8 //stack pointer
	Varargs     bool
	Arity       int
	Constants   []Value
	Locals      map[string]uint16  // name mapped to stack index of where the local was loaded
	UpIndexes   map[string]UpIndex // name mapped to upindex
	ByteCodes   []Bytecode
	Breakable   bool
	Continuable bool
}

func newFnProto(params []string) *FuncProto {
	locals := map[string]uint16{}
	for i, param := range params {
		locals[param] = uint16(i)
	}
	return &FuncProto{
		Arity:     len(params),
		Locals:    locals,
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

func (fn *FuncProto) findVar(name string) *exprDesc {
	if idx, ok := fn.Locals[name]; ok {
		return &exprDesc{kind: localExpr, a: idx}
	} else if idx, ok := fn.UpIndexes[name]; ok {
		return &exprDesc{kind: upvalueExpr, a: uint16(idx.Index)}
	}
	return nil
}

func (fn *FuncProto) code(op Bytecode) {
	fn.ByteCodes = append(fn.ByteCodes, op)
}

func (fnproto *FuncProto) String() string {
	var out bytes.Buffer
	fmt.Println("Constants:")
	for i, cnst := range fnproto.Constants {
		fmt.Fprintf(&out, "%v\t%v\n", i, cnst)
	}
	fmt.Fprintf(
		&out,
		"%v params, %v upvalue, %v local, %v constants\n",
		fnproto.Arity,
		len(fnproto.UpIndexes),
		len(fnproto.Locals),
		len(fnproto.Constants),
	)
	for i, bytecode := range fnproto.ByteCodes {
		fmt.Fprintf(&out, "%v\t%s\n", i, bytecode.String())
	}
	return out.String()
}
