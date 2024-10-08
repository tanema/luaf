package shine

import (
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
		stackPointer uint8        //stack pointer
		prev         *FuncProto   // parent FuncProto or scope
		Varargs      bool         // if the function call has varargs
		Arity        int          // parameter count
		Constants    []any        // constant values to be loaded into the stack
		Locals       []string     // name mapped to stack index of where the local was loaded
		UpIndexes    []UpIndex    // name mapped to upindex
		ByteCodes    []Bytecode   // bytecode for this function
		FnTable      []*FuncProto // indexes of functions in constants
		Breakable    bool         // is this scope one that break can be called
	}
)

func newFnProto(prev *FuncProto, params []string, vararg bool) *FuncProto {
	return &FuncProto{
		prev:         prev,
		Arity:        len(params),
		Varargs:      vararg,
		stackPointer: uint8(len(params)),
		Locals:       params,
	}
}

func (fn *FuncProto) addFn(newfn *FuncProto) uint16 {
	fn.FnTable = append(fn.FnTable, newfn)
	return uint16(len(fn.FnTable) - 1)
}

func (fn *FuncProto) addConst(val any) uint16 {
	for i, v := range fn.Constants {
		if v == val {
			return uint16(i)
		}
	}
	fn.Constants = append(fn.Constants, val)
	return uint16(len(fn.Constants) - 1)
}

func (fn *FuncProto) getConst(idx int64) Value {
	return ToValue(fn.Constants[idx])
}

func (fn *FuncProto) code(op Bytecode) int {
	fn.ByteCodes = append(fn.ByteCodes, op)
	return len(fn.ByteCodes) - 1
}

func (fnproto *FuncProto) String() string {
	codes := make([]string, len(fnproto.ByteCodes))
	for i, bytecode := range fnproto.ByteCodes {
		codes[i] = fmt.Sprintf("[%v] %s", i, bytecode.String())
	}
	fns := make([]string, len(fnproto.FnTable))
	for i, fn := range fnproto.FnTable {
		fns[i] = fmt.Sprintf("\n\n%s", fn.String())
	}
	vararg := ""
	if fnproto.Varargs {
		vararg = "+"
	}

	return fmt.Sprintf(`function: (%v instructions)
%v%v params, %v upvalue, %v locals, %v constants, %v functions
%v%v`,
		len(fnproto.ByteCodes),
		fnproto.Arity,
		vararg,
		len(fnproto.UpIndexes),
		len(fnproto.Locals),
		len(fnproto.Constants),
		len(fnproto.FnTable),
		strings.Join(codes, "\n"),
		strings.Join(fns, ""),
	)
}

func search[S ~[]E, E, T any](x S, target T, cmp func(E, T) bool) (int, bool) {
	for i := range x {
		if cmp(x[i], target) {
			return i, true
		}
	}
	return -1, false
}

func findLocal(lcl string, name string) bool {
	return name == lcl
}

func findUpindex(upindex UpIndex, name string) bool {
	return name == upindex.name
}

func findBroker(b Broker, idx int) bool {
	return idx == b.index
}
