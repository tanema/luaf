package shine

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVM_Eval(t *testing.T) {
	fnproto := &FuncProto{
		Constants: []Value{
			&Integer{23},
			&Float{42.0},
			&Float{65.0},
		},
		ByteCodes: []Bytecode{
			parseOpcode("LOADK 0 0"), // r0 = 23
			parseOpcode("LOADK 1 1"), // r1 = 42
			parseOpcode("ADD 0 0 1"), // r0 = 23 + 42
			parseOpcode("LOADK 1 2"), // r1 = 65
			parseOpcode("EQ 1 0 1"),  // if r1 == r2 then div else mul and div
			parseOpcode("JMP 0 1"),
			parseOpcode("MUL 0 0 1"),
			parseOpcode("DIV 0 0 1"),
		},
	}

	vm := NewVM()
	assert.Nil(t, vm.Eval(fnproto))
	assert.Equal(t, float64(1), vm.GetStack(0).Val())
}
