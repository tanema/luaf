package luaf

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVM_Eval(t *testing.T) {
	t.Run("MOVE", func(t *testing.T) {
		fnproto := &FuncProto{
			Constants: []any{int64(23)},
			ByteCodes: []Bytecode{iABx(LOADK, 0, 0), iAB(MOVE, 1, 0)},
		}
		vm := NewVM()
		value, programCounter, err := vm.eval(fnproto, nil)
		assert.NoError(t, err)
		assert.Nil(t, value)
		assert.Equal(t, int64(len(fnproto.ByteCodes)), programCounter)
		assert.Equal(t, &Integer{val: 23}, vm.Stack[1])
		assert.Equal(t, &Integer{val: 23}, vm.Stack[2])
	})

	t.Run("LOADK", func(t *testing.T) {
		fnproto := &FuncProto{
			Constants: []any{int64(23)},
			ByteCodes: []Bytecode{iABx(LOADK, 0, 0)},
		}
		vm := NewVM()
		value, programCounter, err := vm.eval(fnproto, nil)
		assert.NoError(t, err)
		assert.Nil(t, value)
		assert.Equal(t, int64(len(fnproto.ByteCodes)), programCounter)
		assert.Equal(t, &Integer{val: 23}, vm.Stack[1])
	})

	t.Run("LOADBOOL", func(t *testing.T) {
		fnproto := &FuncProto{ByteCodes: []Bytecode{iABx(LOADBOOL, 0, 1), iABC(LOADBOOL, 1, 0, 1)}}
		vm := NewVM()
		value, programCounter, err := vm.eval(fnproto, nil)
		assert.NoError(t, err)
		assert.Nil(t, value)
		assert.Equal(t, int64(len(fnproto.ByteCodes)+1), programCounter)
		assert.Equal(t, &Boolean{val: true}, vm.Stack[1])
		assert.Equal(t, &Boolean{val: false}, vm.Stack[2])
	})

	t.Run("LOADI", func(t *testing.T) {
		fnproto := &FuncProto{ByteCodes: []Bytecode{iABx(LOADI, 0, 1274)}}
		vm := NewVM()
		value, programCounter, err := vm.eval(fnproto, nil)
		assert.NoError(t, err)
		assert.Nil(t, value)
		assert.Equal(t, int64(len(fnproto.ByteCodes)), programCounter)
		assert.Equal(t, &Integer{val: 1274}, vm.Stack[1])
	})

	t.Run("LOADI EXTAARG", func(t *testing.T) {
		t.Skip("TODO")
	})

	t.Run("LOADNil", func(t *testing.T) {
		fnproto := &FuncProto{ByteCodes: []Bytecode{iABx(LOADNIL, 0, 8)}}
		vm := NewVM()
		value, programCounter, err := vm.eval(fnproto, nil)
		assert.NoError(t, err)
		assert.Nil(t, value)
		assert.Equal(t, int64(len(fnproto.ByteCodes)), programCounter)
		assert.Equal(t, &Nil{}, vm.Stack[1])
		assert.Equal(t, &Nil{}, vm.Stack[2])
		assert.Equal(t, &Nil{}, vm.Stack[3])
		assert.Equal(t, &Nil{}, vm.Stack[4])
		assert.Equal(t, &Nil{}, vm.Stack[5])
		assert.Equal(t, &Nil{}, vm.Stack[6])
		assert.Equal(t, &Nil{}, vm.Stack[7])
		assert.Equal(t, &Nil{}, vm.Stack[8])
	})

	t.Run("ADD", func(t *testing.T) {
		fnproto := &FuncProto{
			Constants: []any{float64(32), float64(112), "Don't touch me"},
			ByteCodes: []Bytecode{
				iABx(LOADI, 0, 1274), iABx(LOADI, 1, 72), iABC(ADD, 0, 0, 1),
				iABx(LOADK, 1, 0), iABx(LOADK, 2, 1), iABC(ADD, 1, 1, 2),
				iABx(LOADI, 2, 42), iABx(LOADK, 3, 0), iABC(ADD, 2, 2, 3),
				iABx(LOADK, 3, 0), iABx(LOADI, 4, 99), iABC(ADD, 3, 3, 4),
				iABx(LOADK, 4, 2), iABx(LOADI, 5, 0), iABC(ADD, 4, 4, 5),
			},
		}
		vm := NewVM()
		value, programCounter, err := vm.eval(fnproto, nil)
		assert.Error(t, err)
		assert.Nil(t, value)
		assert.Equal(t, int64(len(fnproto.ByteCodes)-1), programCounter)
		assert.Equal(t, &Integer{val: 1346}, vm.Stack[1])
		assert.Equal(t, &Float{val: 144}, vm.Stack[2])
		assert.Equal(t, &Float{val: 74}, vm.Stack[3])
		assert.Equal(t, &Float{val: 131}, vm.Stack[4])
	})

	t.Run("SUB", func(t *testing.T) {
		fnproto := &FuncProto{
			Constants: []any{float64(32), float64(112), "Don't touch me"},
			ByteCodes: []Bytecode{
				iABx(LOADI, 0, 1274), iABx(LOADI, 1, 72), iABC(SUB, 0, 0, 1),
				iABx(LOADK, 1, 0), iABx(LOADK, 2, 1), iABC(SUB, 1, 1, 2),
				iABx(LOADI, 2, 42), iABx(LOADK, 3, 0), iABC(SUB, 2, 2, 3),
				iABx(LOADK, 3, 0), iABx(LOADI, 4, 99), iABC(SUB, 3, 3, 4),
				iABx(LOADK, 4, 2), iABx(LOADI, 5, 0), iABC(SUB, 4, 4, 5),
			},
		}
		vm := NewVM()
		value, programCounter, err := vm.eval(fnproto, nil)
		assert.Error(t, err)
		assert.Nil(t, value)
		assert.Equal(t, int64(len(fnproto.ByteCodes)-1), programCounter)
		assert.Equal(t, &Integer{val: 1202}, vm.Stack[1])
		assert.Equal(t, &Float{val: -80}, vm.Stack[2])
		assert.Equal(t, &Float{val: 10}, vm.Stack[3])
		assert.Equal(t, &Float{val: -67}, vm.Stack[4])
	})

	t.Run("MUL", func(t *testing.T) {
		fnproto := &FuncProto{
			Constants: []any{float64(32), float64(112), "Don't touch me"},
			ByteCodes: []Bytecode{
				iABx(LOADI, 0, 1274), iABx(LOADI, 1, 72), iABC(MUL, 0, 0, 1),
				iABx(LOADK, 1, 0), iABx(LOADK, 2, 1), iABC(MUL, 1, 1, 2),
				iABx(LOADI, 2, 42), iABx(LOADK, 3, 0), iABC(MUL, 2, 2, 3),
				iABx(LOADK, 3, 0), iABx(LOADI, 4, 99), iABC(MUL, 3, 3, 4),
				iABx(LOADK, 4, 2), iABx(LOADI, 5, 0), iABC(MUL, 4, 4, 5),
			},
		}
		vm := NewVM()
		value, programCounter, err := vm.eval(fnproto, nil)
		assert.Error(t, err)
		assert.Nil(t, value)
		assert.Equal(t, int64(len(fnproto.ByteCodes)-1), programCounter)
		assert.Equal(t, &Integer{val: 91728}, vm.Stack[1])
		assert.Equal(t, &Float{val: 3584}, vm.Stack[2])
		assert.Equal(t, &Float{val: 1344}, vm.Stack[3])
		assert.Equal(t, &Float{val: 3168}, vm.Stack[4])
	})

	t.Run("DIV", func(t *testing.T) {
		fnproto := &FuncProto{
			Constants: []any{float64(112), float64(32), "Don't touch me"},
			ByteCodes: []Bytecode{
				iABx(LOADI, 0, 1274), iABx(LOADI, 1, 10), iABC(DIV, 0, 0, 1),
				iABx(LOADK, 1, 0), iABx(LOADK, 2, 1), iABC(DIV, 1, 1, 2),
				iABx(LOADI, 2, 42), iABx(LOADK, 3, 0), iABC(DIV, 2, 2, 3),
				iABx(LOADK, 3, 0), iABx(LOADI, 4, 1), iABC(DIV, 3, 3, 4),
				iABx(LOADK, 4, 2), iABx(LOADI, 5, 0), iABC(DIV, 4, 4, 5),
			},
		}
		vm := NewVM()
		value, programCounter, err := vm.eval(fnproto, nil)
		assert.Error(t, err)
		assert.Nil(t, value)
		assert.Equal(t, int64(len(fnproto.ByteCodes)-1), programCounter)
		assert.Equal(t, &Float{val: 127.4}, vm.Stack[1])
		assert.Equal(t, &Float{val: 3.5}, vm.Stack[2])
		assert.Equal(t, &Float{val: 0.375}, vm.Stack[3])
		assert.Equal(t, &Float{val: 112}, vm.Stack[4])
	})

	t.Run("MOD", func(t *testing.T) {
		fnproto := &FuncProto{
			Constants: []any{float64(112), float64(32), "Don't touch me"},
			ByteCodes: []Bytecode{
				iABx(LOADI, 0, 1274), iABx(LOADI, 1, 72), iABC(MOD, 0, 0, 1),
				iABx(LOADK, 1, 0), iABx(LOADK, 2, 1), iABC(MOD, 1, 1, 2),
				iABx(LOADI, 2, 42), iABx(LOADK, 3, 0), iABC(MOD, 2, 2, 3),
				iABx(LOADK, 3, 0), iABx(LOADI, 4, 1), iABC(MOD, 3, 3, 4),
				iABx(LOADK, 4, 2), iABx(LOADI, 5, 0), iABC(MOD, 4, 4, 5),
			},
		}
		vm := NewVM()
		value, programCounter, err := vm.eval(fnproto, nil)
		assert.Error(t, err)
		assert.Nil(t, value)
		assert.Equal(t, int64(len(fnproto.ByteCodes)-1), programCounter)
		assert.Equal(t, &Integer{val: 50}, vm.Stack[1])
		assert.Equal(t, &Float{val: 16}, vm.Stack[2])
		assert.Equal(t, &Float{val: 42}, vm.Stack[3])
		assert.Equal(t, &Float{val: 0}, vm.Stack[4])
	})

	t.Run("POW", func(t *testing.T) {
		fnproto := &FuncProto{
			Constants: []any{float64(2), float64(3), "Don't touch me"},
			ByteCodes: []Bytecode{
				iABx(LOADI, 0, 2), iABx(LOADI, 1, 4), iABC(POW, 0, 0, 1),
				iABx(LOADK, 1, 0), iABx(LOADK, 2, 1), iABC(POW, 1, 1, 2),
				iABx(LOADI, 2, 2), iABx(LOADK, 3, 0), iABC(POW, 2, 2, 3),
				iABx(LOADK, 3, 0), iABx(LOADI, 4, 1), iABC(POW, 3, 3, 4),
				iABx(LOADK, 4, 2), iABx(LOADI, 5, 0), iABC(POW, 4, 4, 5),
			},
		}
		vm := NewVM()
		value, programCounter, err := vm.eval(fnproto, nil)
		assert.Error(t, err)
		assert.Nil(t, value)
		assert.Equal(t, int64(len(fnproto.ByteCodes)-1), programCounter)
		assert.Equal(t, &Float{val: 16}, vm.Stack[1])
		assert.Equal(t, &Float{val: 8}, vm.Stack[2])
		assert.Equal(t, &Float{val: 4}, vm.Stack[3])
		assert.Equal(t, &Float{val: 2}, vm.Stack[4])
	})

	t.Run("IDIV", func(t *testing.T) {
		fnproto := &FuncProto{
			Constants: []any{float64(112), float64(32), "Don't touch me"},
			ByteCodes: []Bytecode{
				iABx(LOADI, 0, 1274), iABx(LOADI, 1, 72), iABC(IDIV, 0, 0, 1),
				iABx(LOADK, 1, 0), iABx(LOADK, 2, 1), iABC(IDIV, 1, 1, 2),
				iABx(LOADI, 2, 42), iABx(LOADK, 3, 0), iABC(IDIV, 2, 2, 3),
				iABx(LOADK, 3, 0), iABx(LOADI, 4, 1), iABC(IDIV, 3, 3, 4),
				iABx(LOADK, 4, 2), iABx(LOADI, 5, 0), iABC(IDIV, 4, 4, 5),
			},
		}
		vm := NewVM()
		value, programCounter, err := vm.eval(fnproto, nil)
		assert.Error(t, err)
		assert.Nil(t, value)
		assert.Equal(t, int64(len(fnproto.ByteCodes)-1), programCounter)
		assert.Equal(t, &Integer{val: 17}, vm.Stack[1])
		assert.Equal(t, &Float{val: 3}, vm.Stack[2])
		assert.Equal(t, &Float{val: 0}, vm.Stack[3])
		assert.Equal(t, &Float{val: 112}, vm.Stack[4])
	})

	t.Run("BAND", func(t *testing.T) {
		fnproto := &FuncProto{
			Constants: []any{float64(2), float64(3), "Don't touch me"},
			ByteCodes: []Bytecode{
				iABx(LOADI, 0, 2), iABx(LOADI, 1, 4), iABC(BAND, 0, 0, 1),
				iABx(LOADK, 1, 0), iABx(LOADK, 2, 1), iABC(BAND, 1, 1, 2),
				iABx(LOADI, 2, 2), iABx(LOADK, 3, 0), iABC(BAND, 2, 2, 3),
				iABx(LOADK, 3, 0), iABx(LOADI, 4, 1), iABC(BAND, 3, 3, 4),
				iABx(LOADK, 4, 2), iABx(LOADI, 5, 0), iABC(BAND, 4, 4, 5),
			},
		}
		vm := NewVM()
		value, programCounter, err := vm.eval(fnproto, nil)
		assert.Error(t, err)
		assert.Nil(t, value)
		assert.Equal(t, int64(len(fnproto.ByteCodes)-1), programCounter)
		assert.Equal(t, &Integer{val: 0}, vm.Stack[1])
		assert.Equal(t, &Integer{val: 2}, vm.Stack[2])
		assert.Equal(t, &Integer{val: 2}, vm.Stack[3])
		assert.Equal(t, &Integer{val: 0}, vm.Stack[4])
	})

	t.Run("BOR", func(t *testing.T) {
		fnproto := &FuncProto{
			Constants: []any{float64(2), float64(3), "Don't touch me"},
			ByteCodes: []Bytecode{
				iABx(LOADI, 0, 2), iABx(LOADI, 1, 4), iABC(BOR, 0, 0, 1),
				iABx(LOADK, 1, 0), iABx(LOADK, 2, 1), iABC(BOR, 1, 1, 2),
				iABx(LOADI, 2, 2), iABx(LOADK, 3, 0), iABC(BOR, 2, 2, 3),
				iABx(LOADK, 3, 0), iABx(LOADI, 4, 1), iABC(BOR, 3, 3, 4),
				iABx(LOADK, 4, 2), iABx(LOADI, 5, 0), iABC(BOR, 4, 4, 5),
			},
		}
		vm := NewVM()
		value, programCounter, err := vm.eval(fnproto, nil)
		assert.Error(t, err)
		assert.Nil(t, value)
		assert.Equal(t, int64(len(fnproto.ByteCodes)-1), programCounter)
		assert.Equal(t, &Integer{val: 6}, vm.Stack[1])
		assert.Equal(t, &Integer{val: 3}, vm.Stack[2])
		assert.Equal(t, &Integer{val: 2}, vm.Stack[3])
		assert.Equal(t, &Integer{val: 3}, vm.Stack[4])
	})

	t.Run("BXOR", func(t *testing.T) {
		fnproto := &FuncProto{
			Constants: []any{float64(2), float64(3), "Don't touch me"},
			ByteCodes: []Bytecode{
				iABx(LOADI, 0, 2), iABx(LOADI, 1, 4), iABC(BXOR, 0, 0, 1),
				iABx(LOADK, 1, 0), iABx(LOADK, 2, 1), iABC(BXOR, 1, 1, 2),
				iABx(LOADI, 2, 2), iABx(LOADK, 3, 0), iABC(BXOR, 2, 2, 3),
				iABx(LOADK, 3, 0), iABx(LOADI, 4, 1), iABC(BXOR, 3, 3, 4),
				iABx(LOADK, 4, 2), iABx(LOADI, 5, 0), iABC(BXOR, 4, 4, 5),
			},
		}
		vm := NewVM()
		value, programCounter, err := vm.eval(fnproto, nil)
		assert.Error(t, err)
		assert.Nil(t, value)
		assert.Equal(t, int64(len(fnproto.ByteCodes)-1), programCounter)
		assert.Equal(t, &Integer{val: 6}, vm.Stack[1])
		assert.Equal(t, &Integer{val: 3}, vm.Stack[2])
		assert.Equal(t, &Integer{val: 2}, vm.Stack[3])
		assert.Equal(t, &Integer{val: 3}, vm.Stack[4])
	})

	t.Run("SHL", func(t *testing.T) {
		fnproto := &FuncProto{
			Constants: []any{float64(2), float64(3), "Don't touch me"},
			ByteCodes: []Bytecode{
				iABx(LOADI, 0, 2), iABx(LOADI, 1, 4), iABC(SHL, 0, 0, 1),
				iABx(LOADK, 1, 0), iABx(LOADK, 2, 1), iABC(SHL, 1, 1, 2),
				iABx(LOADI, 2, 2), iABx(LOADK, 3, 0), iABC(SHL, 2, 2, 3),
				iABx(LOADK, 3, 0), iABx(LOADI, 4, 1), iABC(SHL, 3, 3, 4),
				iABx(LOADK, 4, 2), iABx(LOADI, 5, 0), iABC(SHL, 4, 4, 5),
			},
		}
		vm := NewVM()
		value, programCounter, err := vm.eval(fnproto, nil)
		assert.Error(t, err)
		assert.Nil(t, value)
		assert.Equal(t, int64(len(fnproto.ByteCodes)-1), programCounter)
		assert.Equal(t, &Integer{val: 32}, vm.Stack[1])
		assert.Equal(t, &Integer{val: 16}, vm.Stack[2])
		assert.Equal(t, &Integer{val: 8}, vm.Stack[3])
		assert.Equal(t, &Integer{val: 4}, vm.Stack[4])
	})

	t.Run("SHR", func(t *testing.T) {
		fnproto := &FuncProto{
			Constants: []any{float64(100), float64(1), "Don't touch me"},
			ByteCodes: []Bytecode{
				iABx(LOADI, 0, 100), iABx(LOADI, 1, 1), iABC(SHR, 0, 0, 1),
				iABx(LOADK, 1, 0), iABx(LOADK, 2, 1), iABC(SHR, 1, 1, 2),
				iABx(LOADI, 2, 500), iABx(LOADK, 3, 1), iABC(SHR, 2, 2, 3),
				iABx(LOADK, 3, 0), iABx(LOADI, 4, 1), iABC(SHR, 3, 3, 4),
				iABx(LOADK, 4, 2), iABx(LOADI, 5, 0), iABC(SHR, 4, 4, 5),
			},
		}
		vm := NewVM()
		value, programCounter, err := vm.eval(fnproto, nil)
		assert.Error(t, err)
		assert.Nil(t, value)
		assert.Equal(t, int64(len(fnproto.ByteCodes)-1), programCounter)
		assert.Equal(t, &Integer{val: 50}, vm.Stack[1])
		assert.Equal(t, &Integer{val: 50}, vm.Stack[2])
		assert.Equal(t, &Integer{val: 250}, vm.Stack[3])
		assert.Equal(t, &Integer{val: 50}, vm.Stack[4])
	})

	t.Run("UNM", func(t *testing.T) {
		fnproto := &FuncProto{
			Constants: []any{float64(200), "Don't touch me"},
			ByteCodes: []Bytecode{
				iABx(LOADI, 0, 100), iAB(UNM, 0, 0),
				iABx(LOADK, 1, 0), iAB(UNM, 1, 1),
				iABx(LOADK, 2, 1), iAB(UNM, 2, 2),
			},
		}
		vm := NewVM()
		value, programCounter, err := vm.eval(fnproto, nil)
		assert.Error(t, err)
		assert.Nil(t, value)
		assert.Equal(t, int64(len(fnproto.ByteCodes)-1), programCounter)
		assert.Equal(t, &Integer{val: -100}, vm.Stack[1])
		assert.Equal(t, &Float{val: -200}, vm.Stack[2])
	})

	t.Run("BNOT", func(t *testing.T) {
		fnproto := &FuncProto{
			Constants: []any{float64(100), "Don't touch me"},
			ByteCodes: []Bytecode{
				iABx(LOADI, 0, 100), iAB(BNOT, 0, 0),
				iABx(LOADK, 1, 0), iAB(BNOT, 1, 1),
				iABx(LOADK, 2, 1), iAB(BNOT, 2, 2),
			},
		}
		vm := NewVM()
		value, programCounter, err := vm.eval(fnproto, nil)
		assert.Error(t, err)
		assert.Nil(t, value)
		assert.Equal(t, int64(len(fnproto.ByteCodes)-1), programCounter)
		assert.Equal(t, &Integer{val: -101}, vm.Stack[1])
		assert.Equal(t, &Integer{val: -101}, vm.Stack[2])
	})

	t.Run("NOT", func(t *testing.T) {
		fnproto := &FuncProto{
			Constants: []any{float64(0), float64(1), "Don't touch me"},
			ByteCodes: []Bytecode{
				iABx(LOADI, 0, 0), iAB(NOT, 0, 0), // integer == 0
				iABx(LOADI, 1, 1), iAB(NOT, 1, 1), // integer != 0
				iABx(LOADK, 2, 0), iAB(NOT, 2, 2), // float == 0
				iABx(LOADK, 3, 1), iAB(NOT, 3, 3), // float != 0
				iABx(LOADNIL, 4, 1), iAB(NOT, 4, 4), // nil
				iAB(LOADBOOL, 5, 1), iAB(NOT, 5, 5), // true
				iAB(LOADBOOL, 6, 0), iAB(NOT, 6, 6), // false
				iAB(LOADK, 7, 2), iAB(NOT, 7, 7), // string
			},
		}
		vm := NewVM()
		value, programCounter, err := vm.eval(fnproto, nil)
		assert.NoError(t, err)
		assert.Nil(t, value)
		assert.Equal(t, int64(len(fnproto.ByteCodes)), programCounter)
		assert.Equal(t, &Boolean{val: true}, vm.Stack[1])
		assert.Equal(t, &Boolean{val: false}, vm.Stack[2])
		assert.Equal(t, &Boolean{val: true}, vm.Stack[3])
		assert.Equal(t, &Boolean{val: false}, vm.Stack[4])
		assert.Equal(t, &Boolean{val: true}, vm.Stack[5])
		assert.Equal(t, &Boolean{val: false}, vm.Stack[6])
		assert.Equal(t, &Boolean{val: true}, vm.Stack[7])
		assert.Equal(t, &Boolean{val: false}, vm.Stack[8])
	})

	t.Run("CONCAT", func(t *testing.T) {
		fnproto := &FuncProto{
			Constants: []any{float64(200), "Don't touch me"},
			ByteCodes: []Bytecode{
				iABx(LOADI, 0, 100), iABx(LOADK, 1, 0), iABx(LOADK, 2, 1),
				iABC(CONCAT, 0, 0, 2),
			},
		}
		vm := NewVM()
		value, programCounter, err := vm.eval(fnproto, nil)
		assert.NoError(t, err)
		assert.Nil(t, value)
		assert.Equal(t, int64(len(fnproto.ByteCodes)), programCounter)
		assert.Equal(t, &String{val: "100200Don't touch me"}, vm.Stack[1])
	})

	t.Run("JMP", func(t *testing.T) {
		fnproto := &FuncProto{
			ByteCodes: []Bytecode{iAsBx(JMP, 0, 20)},
		}
		vm := NewVM()
		value, programCounter, err := vm.eval(fnproto, nil)
		assert.NoError(t, err)
		assert.Nil(t, value)
		assert.Equal(t, int64(21), programCounter)
	})

	t.Run("JMP close brokers", func(t *testing.T) {
		fnproto := &FuncProto{
			FnTable: []*FuncProto{
				{
					UpIndexes: []UpIndex{{fromStack: true}, {fromStack: true}, {fromStack: true}},
				},
			},
			ByteCodes: []Bytecode{
				iAB(CLOSURE, 0, 0),
				iAsBx(JMP, 2, 20),
			},
		}
		vm := NewVM()
		value, programCounter, err := vm.eval(fnproto, nil)
		assert.NoError(t, err)
		assert.Nil(t, value)
		assert.Equal(t, int64(22), programCounter)
		closure := vm.GetStack(0).(*Closure)
		assert.Len(t, closure.upvalues, 3)
		assert.True(t, closure.upvalues[0].open)
		assert.False(t, closure.upvalues[1].open)
		assert.False(t, closure.upvalues[2].open)
	})

	t.Run("EQ", func(t *testing.T) {
		t.Run("is false expecting false should not increment pc", func(t *testing.T) {
			fnproto := &FuncProto{ByteCodes: []Bytecode{iABx(LOADI, 0, 2), iABx(LOADI, 1, 1), iABC(EQ, 0, 0, 1)}}
			_, programCounter, _ := NewVM().eval(fnproto, nil)
			assert.Equal(t, int64(3), programCounter)
		})
		t.Run("is true expecting false should increment pc", func(t *testing.T) {
			fnproto := &FuncProto{ByteCodes: []Bytecode{iABx(LOADI, 0, 1), iABx(LOADI, 1, 1), iABC(EQ, 0, 0, 1)}}
			_, programCounter, _ := NewVM().eval(fnproto, nil)
			assert.Equal(t, int64(4), programCounter)
		})
		t.Run("is true expecting true should not increment pc", func(t *testing.T) {
			fnproto := &FuncProto{ByteCodes: []Bytecode{iABx(LOADI, 0, 1), iABx(LOADI, 1, 1), iABC(EQ, 1, 0, 1)}}
			_, programCounter, _ := NewVM().eval(fnproto, nil)
			assert.Equal(t, int64(3), programCounter)
		})
		t.Run("is false expecting true should increment pc", func(t *testing.T) {
			fnproto := &FuncProto{ByteCodes: []Bytecode{iABx(LOADI, 0, 2), iABx(LOADI, 1, 1), iABC(EQ, 1, 0, 1)}}
			_, programCounter, _ := NewVM().eval(fnproto, nil)
			assert.Equal(t, int64(4), programCounter)
		})
	})

	t.Run("LT", func(t *testing.T) {
		t.Run("is false expecting false should not increment pc", func(t *testing.T) {
			fnproto := &FuncProto{ByteCodes: []Bytecode{iABx(LOADI, 0, 2), iABx(LOADI, 1, 1), iABC(LT, 0, 0, 1)}}
			_, programCounter, _ := NewVM().eval(fnproto, nil)
			assert.Equal(t, int64(3), programCounter)
		})
		t.Run("is true expecting false should increment pc", func(t *testing.T) {
			fnproto := &FuncProto{ByteCodes: []Bytecode{iABx(LOADI, 0, 0), iABx(LOADI, 1, 1), iABC(LT, 0, 0, 1)}}
			_, programCounter, _ := NewVM().eval(fnproto, nil)
			assert.Equal(t, int64(4), programCounter)
		})
		t.Run("is true expecting true should not increment pc", func(t *testing.T) {
			fnproto := &FuncProto{ByteCodes: []Bytecode{iABx(LOADI, 0, 0), iABx(LOADI, 1, 1), iABC(LT, 1, 0, 1)}}
			_, programCounter, _ := NewVM().eval(fnproto, nil)
			assert.Equal(t, int64(3), programCounter)
		})
		t.Run("is false expecting true should increment pc", func(t *testing.T) {
			fnproto := &FuncProto{ByteCodes: []Bytecode{iABx(LOADI, 0, 2), iABx(LOADI, 1, 1), iABC(LT, 1, 0, 1)}}
			_, programCounter, _ := NewVM().eval(fnproto, nil)
			assert.Equal(t, int64(4), programCounter)
		})
		t.Run("compare non-number should err", func(t *testing.T) {
			fnproto := &FuncProto{Constants: []any{"nope"}, ByteCodes: []Bytecode{iABx(LOADK, 0, 0), iABx(LOADI, 1, 1), iABC(LT, 1, 0, 1)}}
			_, _, err := NewVM().eval(fnproto, nil)
			assert.Error(t, err)
		})
	})

	t.Run("LE", func(t *testing.T) {
		t.Run("is false expecting false should not increment pc", func(t *testing.T) {
			fnproto := &FuncProto{ByteCodes: []Bytecode{iABx(LOADI, 0, 2), iABx(LOADI, 1, 1), iABC(LE, 0, 0, 1)}}
			_, programCounter, _ := NewVM().eval(fnproto, nil)
			assert.Equal(t, int64(3), programCounter)
		})
		t.Run("is true expecting false should increment pc", func(t *testing.T) {
			fnproto := &FuncProto{ByteCodes: []Bytecode{iABx(LOADI, 0, 0), iABx(LOADI, 1, 1), iABC(LE, 0, 0, 1)}}
			_, programCounter, _ := NewVM().eval(fnproto, nil)
			assert.Equal(t, int64(4), programCounter)
		})
		t.Run("is true expecting true should not increment pc", func(t *testing.T) {
			fnproto := &FuncProto{ByteCodes: []Bytecode{iABx(LOADI, 0, 0), iABx(LOADI, 1, 1), iABC(LE, 1, 0, 1)}}
			_, programCounter, _ := NewVM().eval(fnproto, nil)
			assert.Equal(t, int64(3), programCounter)
		})
		t.Run("is false expecting true should increment pc", func(t *testing.T) {
			fnproto := &FuncProto{ByteCodes: []Bytecode{iABx(LOADI, 0, 2), iABx(LOADI, 1, 1), iABC(LE, 1, 0, 1)}}
			_, programCounter, _ := NewVM().eval(fnproto, nil)
			assert.Equal(t, int64(4), programCounter)
		})
		t.Run("compare non-number should err", func(t *testing.T) {
			fnproto := &FuncProto{Constants: []any{"nope"}, ByteCodes: []Bytecode{iABx(LOADK, 0, 0), iABx(LOADI, 1, 1), iABC(LE, 1, 0, 1)}}
			_, _, err := NewVM().eval(fnproto, nil)
			assert.Error(t, err)
		})
	})

	t.Run("TEST", func(t *testing.T) {
		t.Run("is false expecting false should not increment pc", func(t *testing.T) {
			fnproto := &FuncProto{ByteCodes: []Bytecode{iABx(LOADBOOL, 0, 0), iAB(TEST, 0, 0)}}
			_, programCounter, _ := NewVM().eval(fnproto, nil)
			assert.Equal(t, int64(2), programCounter)
		})
		t.Run("is true expecting false should increment pc", func(t *testing.T) {
			fnproto := &FuncProto{ByteCodes: []Bytecode{iABx(LOADBOOL, 0, 1), iAB(TEST, 0, 0)}}
			_, programCounter, _ := NewVM().eval(fnproto, nil)
			assert.Equal(t, int64(3), programCounter)
		})
		t.Run("is true expecting true should not increment pc", func(t *testing.T) {
			fnproto := &FuncProto{ByteCodes: []Bytecode{iABx(LOADBOOL, 0, 1), iAB(TEST, 0, 1)}}
			_, programCounter, _ := NewVM().eval(fnproto, nil)
			assert.Equal(t, int64(2), programCounter)
		})
		t.Run("is false expecting true should increment pc", func(t *testing.T) {
			fnproto := &FuncProto{ByteCodes: []Bytecode{iABx(LOADBOOL, 0, 0), iAB(TEST, 0, 1)}}
			_, programCounter, _ := NewVM().eval(fnproto, nil)
			assert.Equal(t, int64(3), programCounter)
		})
	})

	t.Run("TESTSET", func(t *testing.T) {
		t.Run("is false expecting false should not increment pc", func(t *testing.T) {
			fnproto := &FuncProto{ByteCodes: []Bytecode{iABx(LOADI, 0, 0), iABC(TESTSET, 0, 0, 0)}}
			vm := NewVM()
			_, programCounter, _ := vm.eval(fnproto, nil)
			assert.Equal(t, int64(2), programCounter)
			assert.Equal(t, &Boolean{val: false}, vm.Stack[1])
		})
		t.Run("is true expecting false should increment pc", func(t *testing.T) {
			fnproto := &FuncProto{ByteCodes: []Bytecode{iABx(LOADI, 0, 1), iABC(TESTSET, 0, 0, 0)}}
			vm := NewVM()
			_, programCounter, _ := vm.eval(fnproto, nil)
			assert.Equal(t, int64(3), programCounter)
			assert.Equal(t, &Integer{val: 1}, vm.Stack[1])
		})
		t.Run("is true expecting true should not increment pc", func(t *testing.T) {
			fnproto := &FuncProto{ByteCodes: []Bytecode{iABx(LOADI, 0, 1), iABC(TESTSET, 0, 0, 1)}}
			vm := NewVM()
			_, programCounter, _ := vm.eval(fnproto, nil)
			assert.Equal(t, int64(2), programCounter)
			assert.Equal(t, &Boolean{val: true}, vm.Stack[1])
		})
		t.Run("is false expecting true should increment pc", func(t *testing.T) {
			fnproto := &FuncProto{ByteCodes: []Bytecode{iABx(LOADI, 0, 0), iABC(TESTSET, 0, 0, 1)}}
			vm := NewVM()
			_, programCounter, _ := vm.eval(fnproto, nil)
			assert.Equal(t, int64(3), programCounter)
			assert.Equal(t, &Integer{val: 0}, vm.Stack[1])
		})
	})

	t.Run("LEN", func(t *testing.T) {
		t.Run("String", func(t *testing.T) {
			fnproto := &FuncProto{
				Constants: []any{"test string"},
				ByteCodes: []Bytecode{iABCK(LEN, 0, 0, true, 0, false)},
			}
			vm := NewVM()
			_, _, err := vm.eval(fnproto, nil)
			require.NoError(t, err)
			assert.Equal(t, &Integer{val: int64(len("test string"))}, vm.Stack[1])
		})
		t.Run("Table", func(t *testing.T) {
			fnproto := &FuncProto{
				ByteCodes: []Bytecode{
					iABC(NEWTABLE, 0, 3, 0),
					iABx(LOADI, 1, 21),
					iABx(LOADI, 2, 22),
					iABx(LOADI, 3, 23),
					iABC(SETLIST, 0, 4, 1),
					iAB(LEN, 1, 0),
				},
			}
			vm := NewVM()
			_, _, err := vm.eval(fnproto, nil)
			require.NoError(t, err)
			assert.Equal(t, &Integer{val: 3}, vm.Stack[2])
		})
		t.Run("Others", func(t *testing.T) {
			fnproto := &FuncProto{
				Constants: []any{123.0},
				ByteCodes: []Bytecode{iABCK(LEN, 0, 0, true, 0, false)},
			}
			vm := NewVM()
			_, _, err := vm.eval(fnproto, nil)
			assert.Error(t, err)
		})
	})

	t.Run("SETTABLE", func(t *testing.T) {
		fnproto := &FuncProto{
			Constants: []any{"hello", "world"},
			ByteCodes: []Bytecode{
				iABC(NEWTABLE, 0, 0, 1),
				iABCK(SETTABLE, 0, 0, true, 1, true),
			},
		}
		vm := NewVM()
		_, _, err := vm.eval(fnproto, nil)
		require.NoError(t, err)
		expectedTable := &Table{
			val:       []Value{},
			hashtable: map[any]Value{"hello": &String{val: "world"}},
			keyCache:  []any{"hello"},
		}
		assert.Equal(t, expectedTable, vm.Stack[1])
	})

	t.Run("GETTABLE", func(t *testing.T) {
		fnproto := &FuncProto{
			Constants: []any{"hello", "world"},
			ByteCodes: []Bytecode{
				iABC(NEWTABLE, 0, 0, 1),
				iABCK(SETTABLE, 0, 0, true, 1, true),
				iABCK(GETTABLE, 1, 0, false, 0, true),
			},
		}
		vm := NewVM()
		_, _, err := vm.eval(fnproto, nil)
		require.NoError(t, err)
		expectedTable := &Table{
			val:       []Value{},
			hashtable: map[any]Value{"hello": &String{val: "world"}},
			keyCache:  []any{"hello"},
		}
		assert.Equal(t, expectedTable, vm.Stack[1])
		assert.Equal(t, &String{val: "world"}, vm.Stack[2])
	})

	t.Run("SETLIST", func(t *testing.T) {
		t.Run("with defined count at zero position", func(t *testing.T) {
			fnproto := &FuncProto{
				ByteCodes: []Bytecode{
					iABC(NEWTABLE, 0, 3, 0),
					iABx(LOADI, 1, 20),
					iABx(LOADI, 2, 20),
					iABx(LOADI, 3, 20),
					iABC(SETLIST, 0, 4, 1),
				},
			}
			vm := NewVM()
			_, _, err := vm.eval(fnproto, nil)
			require.NoError(t, err)
			expectedTable := &Table{
				val:       []Value{&Integer{val: 20}, &Integer{val: 20}, &Integer{val: 20}},
				hashtable: map[any]Value{},
			}
			assert.Equal(t, expectedTable, vm.Stack[1])
		})

		t.Run("with defined count at c position", func(t *testing.T) {
			fnproto := &FuncProto{
				ByteCodes: []Bytecode{
					iABC(NEWTABLE, 0, 3, 0),
					iABx(LOADI, 1, 20),
					iABx(LOADI, 2, 20),
					iABx(LOADI, 3, 20),
					iABC(SETLIST, 0, 4, 3),
				},
			}
			vm := NewVM()
			_, _, err := vm.eval(fnproto, nil)
			require.NoError(t, err)
			expectedTable := &Table{
				val:       []Value{nil, nil, &Integer{val: 20}, &Integer{val: 20}, &Integer{val: 20}},
				hashtable: map[any]Value{},
			}
			assert.Equal(t, expectedTable, vm.Stack[1])
		})
	})

	t.Run("GETUPVAL", func(t *testing.T) {
		t.Run("open upval", func(t *testing.T) {
			fnproto := &FuncProto{
				ByteCodes: []Bytecode{
					iAB(LOADI, 0, 42),
					iAB(GETUPVAL, 1, 0),
				},
			}
			vm := NewVM()
			_, _, err := vm.eval(fnproto, []*Broker{vm.newValueBroker("value", &Integer{val: 42}, 1)})
			require.NoError(t, err)
			assert.Equal(t, &Integer{val: 42}, vm.Stack[1])
			assert.Equal(t, &Integer{val: 42}, vm.Stack[2])
		})
		t.Run("closed upval", func(t *testing.T) {
			fnproto := &FuncProto{
				ByteCodes: []Bytecode{
					iAB(LOADI, 0, 42),
					iAB(GETUPVAL, 1, 0),
				},
			}
			vm := NewVM()
			upval := &Broker{name: "value", val: &Integer{val: 77}, open: false}
			_, _, err := vm.eval(fnproto, []*Broker{upval})
			require.NoError(t, err)
			assert.Equal(t, &Integer{val: 42}, vm.Stack[1])
			assert.Equal(t, &Integer{val: 77}, vm.Stack[2])
		})
	})

	t.Run("SETUPVAL", func(t *testing.T) {
		t.Run("open upval", func(t *testing.T) {
			fnproto := &FuncProto{
				ByteCodes: []Bytecode{
					iAB(LOADI, 0, 42),
					iAB(LOADI, 1, 77),
					iAB(SETUPVAL, 1, 0),
				},
			}
			vm := NewVM()
			_, _, err := vm.eval(fnproto, []*Broker{vm.newValueBroker("value", &Integer{val: 42}, 1)})
			require.NoError(t, err)
			assert.Equal(t, &Integer{val: 77}, vm.Stack[1])
		})
		t.Run("closed upval", func(t *testing.T) {
			fnproto := &FuncProto{
				ByteCodes: []Bytecode{
					iAB(LOADI, 0, 42),
					iAB(LOADI, 1, 77),
					iAB(SETUPVAL, 1, 0),
				},
			}
			vm := NewVM()
			upval := &Broker{name: "value", val: &Integer{val: 42}, open: false}
			_, _, err := vm.eval(fnproto, []*Broker{upval})
			require.NoError(t, err)
			assert.Equal(t, &Integer{val: 42}, vm.Stack[1])
			assert.Equal(t, &Integer{val: 77}, upval.val)
		})
	})

	t.Run("GETTABUP", func(t *testing.T) {
		t.Run("open upval", func(t *testing.T) {
			fnproto := &FuncProto{
				ByteCodes: []Bytecode{
					iABC(NEWTABLE, 0, 3, 0),
					iABx(LOADI, 1, 20),
					iABx(LOADI, 2, 22),
					iABx(LOADI, 3, 24),
					iABC(SETLIST, 0, 4, 1),
					iABx(LOADI, 1, 1),
					iABC(GETTABUP, 1, 0, 1),
				},
			}
			vm := NewVM()
			_, _, err := vm.eval(fnproto, []*Broker{vm.newValueBroker("value", nil, 1)})
			require.NoError(t, err)
			expectedTable := &Table{
				val:       []Value{&Integer{val: 20}, &Integer{val: 22}, &Integer{val: 24}},
				hashtable: map[any]Value{},
			}
			assert.Equal(t, expectedTable, vm.Stack[1])
			assert.Equal(t, &Integer{val: 20}, vm.Stack[2])
		})
		t.Run("with key", func(t *testing.T) {
			fnproto := &FuncProto{
				Constants: []any{"hello", "world"},
				ByteCodes: []Bytecode{
					iABC(NEWTABLE, 0, 0, 1),
					iABCK(SETTABLE, 0, 0, true, 1, true),
					iABCK(GETTABUP, 1, 0, false, 0, true),
				},
			}
			vm := NewVM()
			_, _, err := vm.eval(fnproto, []*Broker{vm.newValueBroker("value", nil, 1)})
			require.NoError(t, err)
			expectedTable := &Table{
				val:       []Value{},
				hashtable: map[any]Value{"hello": &String{val: "world"}},
				keyCache:  []any{"hello"},
			}
			assert.Equal(t, expectedTable, vm.Stack[1])
			assert.Equal(t, &String{val: "world"}, vm.Stack[2])
		})
		t.Run("closed upval", func(t *testing.T) {
			fnproto := &FuncProto{
				ByteCodes: []Bytecode{
					iABC(NEWTABLE, 0, 3, 0),
					iABx(LOADI, 1, 20),
					iABx(LOADI, 2, 22),
					iABx(LOADI, 3, 24),
					iABC(SETLIST, 0, 4, 1),
					iABx(LOADI, 1, 1),
					iABC(GETTABUP, 1, 0, 1),
				},
			}
			table := &Table{
				val:       []Value{&Integer{val: 20}, &Integer{val: 22}, &Integer{val: 24}},
				hashtable: map[any]Value{},
			}
			vm := NewVM()
			_, _, err := vm.eval(fnproto, []*Broker{{name: "value", val: table, open: false}})
			require.NoError(t, err)
			expectedTable := &Table{
				val:       []Value{&Integer{val: 20}, &Integer{val: 22}, &Integer{val: 24}},
				hashtable: map[any]Value{},
			}
			assert.Equal(t, expectedTable, vm.Stack[1])
			assert.Equal(t, &Integer{val: 20}, vm.Stack[2])
		})
	})

	t.Run("SETTABUP", func(t *testing.T) {
		t.Run("open upval", func(t *testing.T) {
			fnproto := &FuncProto{
				ByteCodes: []Bytecode{
					iABC(NEWTABLE, 0, 3, 0),
					iABx(LOADI, 1, 20),
					iABx(LOADI, 2, 22),
					iABx(LOADI, 3, 24),
					iABC(SETLIST, 0, 4, 1),
					iABx(LOADI, 1, 1),
					iABx(LOADI, 2, 55),
					iABC(SETTABUP, 0, 1, 2),
				},
			}
			vm := NewVM()
			_, _, err := vm.eval(fnproto, []*Broker{vm.newValueBroker("value", nil, 1)})
			require.NoError(t, err)
			expectedTable := &Table{
				val:       []Value{&Integer{val: 20}, &Integer{val: 55}, &Integer{val: 24}},
				hashtable: map[any]Value{},
			}
			assert.Equal(t, expectedTable, vm.Stack[1])
		})
		t.Run("with key", func(t *testing.T) {
			fnproto := &FuncProto{
				Constants: []any{"hello", "world", "tim"},
				ByteCodes: []Bytecode{
					iABC(NEWTABLE, 0, 0, 1),
					iABCK(SETTABLE, 0, 0, true, 1, true),
					iABCK(SETTABUP, 0, 0, true, 2, true),
				},
			}
			vm := NewVM()
			_, _, err := vm.eval(fnproto, []*Broker{vm.newValueBroker("value", nil, 1)})
			require.NoError(t, err)
			expectedTable := &Table{
				val:       []Value{},
				hashtable: map[any]Value{"hello": &String{val: "tim"}},
				keyCache:  []any{"hello"},
			}
			assert.Equal(t, expectedTable, vm.Stack[1])
		})
		t.Run("closed upval", func(t *testing.T) {
			fnproto := &FuncProto{
				ByteCodes: []Bytecode{
					iABC(NEWTABLE, 0, 3, 0),
					iABx(LOADI, 1, 20),
					iABx(LOADI, 2, 22),
					iABx(LOADI, 3, 24),
					iABC(SETLIST, 0, 4, 1),
					iABx(LOADI, 1, 1),
					iABx(LOADI, 2, 99),
					iABC(SETTABUP, 0, 1, 2),
				},
			}
			table := &Table{
				val:       []Value{&Integer{val: 20}, &Integer{val: 22}, &Integer{val: 24}},
				hashtable: map[any]Value{},
			}
			vm := NewVM()
			_, _, err := vm.eval(fnproto, []*Broker{{name: "value", val: table, open: false}})
			require.NoError(t, err)
			expectedTable := &Table{
				val:       []Value{&Integer{val: 20}, &Integer{val: 99}, &Integer{val: 24}},
				hashtable: map[any]Value{},
			}
			assert.Equal(t, expectedTable, table)
		})
	})

	t.Run("RETURN", func(t *testing.T) {
		t.Run("All return values", func(t *testing.T) {
			fnproto := &FuncProto{
				Constants: []any{"don't touch me", "hello", "world"},
				ByteCodes: []Bytecode{
					iABx(LOADK, 0, 0),
					iABx(LOADK, 1, 1),
					iABx(LOADK, 2, 2),
					iAB(RETURN, 1, 0),
				},
			}
			vm := NewVM()
			values, pc, err := vm.eval(fnproto, nil)
			assert.NoError(t, err)
			assert.Equal(t, int64(3), pc)
			assert.Equal(t, []Value{&String{"hello"}, &String{"world"}}, values)
		})

		t.Run("specified return vals", func(t *testing.T) {
			fnproto := &FuncProto{
				Constants: []any{"don't touch me", "hello", "world"},
				ByteCodes: []Bytecode{
					iABx(LOADK, 0, 0),
					iABx(LOADK, 1, 1),
					iABx(LOADK, 2, 2),
					iAB(RETURN, 1, 2),
				},
			}
			vm := NewVM()
			values, pc, err := vm.eval(fnproto, nil)
			assert.NoError(t, err)
			assert.Equal(t, int64(3), pc)
			assert.Equal(t, []Value{&String{"hello"}}, values)
		})

		t.Run("specified return vals more than provided", func(t *testing.T) {
			fnproto := &FuncProto{
				Constants: []any{"don't touch me", "hello", "world"},
				ByteCodes: []Bytecode{
					iABx(LOADK, 0, 0),
					iABx(LOADK, 1, 1),
					iABx(LOADK, 2, 2),
					iAB(RETURN, 1, 5),
				},
			}
			vm := NewVM()
			values, pc, err := vm.eval(fnproto, nil)
			assert.NoError(t, err)
			assert.Equal(t, int64(3), pc)
			assert.Equal(t, []Value{&String{"hello"}, &String{"world"}, &Nil{}, &Nil{}}, values)
		})
	})

	t.Run("VARARG", func(t *testing.T) {
		t.Run("All xargs", func(t *testing.T) {
			fnproto := &FuncProto{
				Constants: []any{"don't touch me", "hello", "world"},
				ByteCodes: []Bytecode{iAB(VARARG, 0, 0)},
			}
			vm := NewVM()
			vm.Stack = append(vm.Stack, &Integer{val: 11}, &Float{val: 42}, &String{val: "hello"})
			_, _, err := vm.eval(fnproto, nil)
			require.NoError(t, err)
			assert.Equal(t, &Integer{val: 11}, vm.Stack[1])
			assert.Equal(t, &Float{val: 42}, vm.Stack[2])
			assert.Equal(t, &String{val: "hello"}, vm.Stack[3])
		})
		t.Run("nargs", func(t *testing.T) {
			fnproto := &FuncProto{
				Constants: []any{"don't touch me", "hello", "world"},
				ByteCodes: []Bytecode{iAB(VARARG, 0, 2)},
			}
			vm := NewVM()
			vm.Stack = append(vm.Stack, &Integer{val: 11}, &Float{val: 42}, &String{val: "hello"})
			_, _, err := vm.eval(fnproto, nil)
			require.NoError(t, err)
			assert.Equal(t, &Integer{val: 11}, vm.Stack[1])
			assert.Len(t, vm.Stack, 2)
		})

		t.Run("nargs with offset", func(t *testing.T) {
			fnproto := &FuncProto{
				Constants: []any{"don't touch me", "hello", "world"},
				ByteCodes: []Bytecode{iAB(VARARG, 1, 2)},
			}
			vm := NewVM()
			vm.Stack = append(vm.Stack, &Integer{val: 11}, &Float{val: 42}, &String{val: "hello"})
			_, _, err := vm.eval(fnproto, nil)
			require.NoError(t, err)
			assert.Equal(t, &Integer{val: 11}, vm.Stack[1])
			assert.Len(t, vm.Stack, 2)
		})
	})

	t.Run("CALL", func(t *testing.T) {
		t.Skip("TODO")
	})

	t.Run("CLOSURE", func(t *testing.T) {
		t.Skip("TODO")
	})

	t.Run("SELF", func(t *testing.T) {
		t.Skip("TODO")
	})

	t.Run("TAILCALL", func(t *testing.T) {
		t.Skip("TODO")
	})

	t.Run("FORLOOP", func(t *testing.T) {
		t.Skip("TODO")
	})

	t.Run("FORPREP", func(t *testing.T) {
		t.Skip("TODO")
	})

	t.Run("TFORLOOP", func(t *testing.T) {
		t.Skip("TODO")
	})

	t.Run("TFORCALL", func(t *testing.T) {
		t.Skip("TODO")
	})
}
