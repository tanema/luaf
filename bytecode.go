package shine

import (
	"fmt"
)

// bytecode op u8
// bytecode register number u8
// bytecode format 32 bits
// | iABC  | C: 8 | B: 8 | A: 8 | Opcode: 8 |
// | iABx  |   Bx: 16    | A: 8 | Opcode: 8 |
// | iAsBx |  sBx: 16    | A: 8 | Opcode: 8 |

type (
	BytecodeOp   uint8
	Bytecode     uint32
	BytecodeType string
)

const (
	BytecodeTypeABC  BytecodeType = "iABC"
	BytecodeTypeABx  BytecodeType = "iABx"
	BytecodeTypeAsBx BytecodeType = "iAsBx"

	MOVE     BytecodeOp = iota //Copy a value between registers
	LOADK                      // Load a constant into a register
	LOADBOOL                   // Load a boolean into a register
	LOADNIL                    // Load nil values into a range of registers
	GETUPVAL                   // Read an upvalue into a register
	GETTABUP                   // Read a value from table in up-value into a register
	GETTABLE                   // Read a table element into a register
	SETTABUP                   // Write a register value into table in up-value
	SETUPVAL                   // Write a register value into an upvalue
	SETTABLE                   // Write a register value into a table element
	NEWTABLE                   // Create a new table
	SELF                       // Prepare an object method for calling
	ADD                        // Addition operator
	SUB                        // Subtraction operator
	MUL                        // Multiplication operator
	MOD                        // Modulus (remainder) operator
	POW                        // Exponentation operator
	DIV                        // Division operator
	IDIV                       // Integer division operator
	BAND                       // Bit-wise AND operator
	BOR                        // Bit-wise OR operator
	BXOR                       // Bit-wise Exclusive OR operator
	SHL                        // Shift bits left
	SHR                        // Shift bits right
	UNM                        // Unary minus
	BNOT                       // Bit-wise NOT operator
	NOT                        // Logical NOT operator
	LEN                        // Length operator
	CONCAT                     // Concatenate a range of registers
	JMP                        // Unconditional jump
	EQ                         // Equality test, with conditional jump
	LT                         // Less than test, with conditional jump
	LE                         // Less than or equal to test, with conditional jump
	TEST                       // Boolean test, with conditional jump
	TESTSET                    // Boolean test, with conditional jump and assignment
	CALL                       // Call a closure
	TAILCALL                   // Perform a tail call
	RETURN                     // Return from function call
	FORLOOP                    // Iterate a numeric for loop
	FORPREP                    // Initialization for a numeric for loop
	TFORLOOP                   // Iterate a generic for loop
	TFORCALL                   // Initialization for a generic for loop
	SETLIST                    // Set a range of array elements for a table
	CLOSURE                    // Create a closure of a function prototype
	VARARG                     // Assign vararg function arguments to registers
)

var codeToString = map[BytecodeOp]string{
	MOVE:     "MOVE",
	LOADK:    "LOADK",
	LOADBOOL: "LOADBOOL",
	LOADNIL:  "LOADNIL",
	GETUPVAL: "GETUPVAL",
	GETTABUP: "GETTABUP",
	GETTABLE: "GETTABLE",
	SETTABUP: "SETTABUP",
	SETUPVAL: "SETUPVAL",
	SETTABLE: "SETTABLE",
	NEWTABLE: "NEWTABLE",
	SELF:     "SELF",
	ADD:      "ADD",
	SUB:      "SUB",
	MUL:      "MUL",
	MOD:      "MOD",
	POW:      "POW",
	DIV:      "DIV",
	IDIV:     "IDIV",
	BAND:     "BAND",
	BOR:      "BOR",
	BXOR:     "BXOR",
	SHL:      "SHL",
	SHR:      "SHR",
	UNM:      "UNM",
	BNOT:     "BNOT",
	NOT:      "NOT",
	LEN:      "LEN",
	CONCAT:   "CONCAT",
	JMP:      "JMP",
	EQ:       "EQ",
	LT:       "LT",
	LE:       "LE",
	TEST:     "TEST",
	TESTSET:  "TESTSET",
	CALL:     "CALL",
	TAILCALL: "TAILCALL",
	RETURN:   "RETURN",
	FORLOOP:  "FORLOOP",
	FORPREP:  "FORPREP",
	TFORLOOP: "TFORLOOP",
	TFORCALL: "TFORCALL",
	SETLIST:  "SETLIST",
	CLOSURE:  "CLOSURE",
	VARARG:   "VARARG",
}

func IABC(op BytecodeOp, a, b, c uint8) Bytecode {
	return Bytecode(uint32(c)<<24 | uint32(b)<<16 | uint32(a)<<8 | uint32(op))
}

func IABx(op BytecodeOp, a uint8, b uint16) Bytecode {
	return Bytecode(uint32(b)<<16 | uint32(a)<<8 | uint32(op))
}

func IAsBx(op BytecodeOp, a uint8, b int16) Bytecode {
	return Bytecode(uint32(b)<<16 | uint32(a)<<8 | uint32(op))
}

func (bc Bytecode) Op() BytecodeOp {
	return BytecodeOp(uint32(bc) & 0xFF)
}

func (bc Bytecode) ABC() (uint8, uint8, uint8) {
	f := uint32(bc)
	return uint8(f >> 8 & 0xFF), uint8(f >> 16 & 0xFF), uint8(f >> 24 & 0xFF)
}

func (bc Bytecode) ABx() (uint8, uint16) {
	f := uint32(bc)
	return uint8(f >> 8 & 0xFF), uint16(f >> 16)
}

func (bc Bytecode) AsBx() (uint8, int16) {
	f := uint32(bc)
	return uint8(f >> 8 & 0xFF), int16(f >> 16)
}

func (bc *Bytecode) String() string {
	op, ok := codeToString[bc.Op()]
	if !ok {
		op = "UNDEFINED"
	}
	switch bc.Kind() {
	case BytecodeTypeABx:
		reg, val := bc.ABx()
		return fmt.Sprintf("%v  %v  %v;", op, reg, val)
	case BytecodeTypeAsBx:
		reg, val := bc.AsBx()
		return fmt.Sprintf("%v  %v  %v;", op, reg, val)
	default:
		a, b, c := bc.ABC()
		return fmt.Sprintf("%v  %v  %v  %v;", op, a, b, c)
	}
}

func (op Bytecode) Kind() BytecodeType {
	switch op.Op() {
	case LOADK, FORLOOP, FORPREP, CLOSURE:
		return BytecodeTypeABx
	case JMP, TEST, TESTSET:
		return BytecodeTypeAsBx
	default:
		return BytecodeTypeABC
	}
}
