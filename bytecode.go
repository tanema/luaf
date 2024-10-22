package luaf

import (
	"fmt"
)

// bytecode layout
// 32-bit opcode
// BK | CK = 0 or 1 indicate if the params B,C refer to a stack value or a constant value
// Opcode:u6 => 64 possible opcodes
// Since constants are loaded with u8 register index max local is 255
// However max constants would be 65,536 because LOADK is u16
// iAbx => Bx is u16 for referring to constants
// iAsbx => sBx is signed so it is good for jumps that can be positive or negative
// isBx => for exarg loading of bigger numbers
//
// Format:
// | iABC  | CK: 1 | C: u8 | BK: 1 | B: u8 | A: u8 | Opcode: u6 |
// | iABx  |            Bx: u16            | A: u8 | Opcode: u6 |
// | iAsBx |           sBx:  16            | A: u8 | Opcode: u6 |
// | isBx  |           sBx:  26                    | Opcode: u6 |

type (
	BytecodeOp   uint8
	Bytecode     uint32
	BytecodeType string
)

const (
	BytecodeTypeABC  BytecodeType = "iABC"
	BytecodeTypeABx  BytecodeType = "iABx"
	BytecodeTypeAsBx BytecodeType = "iAsBx"
	BytecodeTypesBx  BytecodeType = "isBx"

	MOVE     BytecodeOp = iota // Copy a value between registers
	LOADK                      // Load a constant into a register
	LOADBOOL                   // Load a boolean into a register
	LOADNIL                    // Load nil values into a range of registers
	LOADI                      // Load a raw int
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
	TFORPREP                   // Initialization for a generic for loop
	SETLIST                    // Set a range of array elements for a table
	CLOSURE                    // Create a closure of a function prototype
	VARARG                     // Assign vararg function arguments to registers
	EXARG                      // TODO: allow some commands to consume next arg
	// max possible is 6 bits or 64 codes
)

var opcodeToString = map[BytecodeOp]string{
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
	TFORPREP: "TFORPREP",
	SETLIST:  "SETLIST",
	CLOSURE:  "CLOSURE",
	VARARG:   "VARARG",
	EXARG:    "EXARG",
}

// Format values in the 32 bit opcode
const (
	aShift     = 6
	bShift     = aShift + 8
	bKShift    = bShift + 8
	cShift     = bKShift + 1
	cKShift    = cShift + 8
	mask6bits  = 0b00111111
	mask2Bytes = 0xFFFF
	maskByte   = 0xFF
	mask26bits = 0x3FFFFFFF
)

func iAB(op BytecodeOp, a uint8, b uint8) Bytecode {
	return iABC(op, a, b, 0)
}

func iABC(op BytecodeOp, a uint8, b uint8, c uint8) Bytecode {
	return iABCK(op, a, b, false, c, false)
}

// iABC format = | CK: 1 | C: u8 | BK: 1 | B: u8 | A: u8 | Opcode: u6 |
func iABCK(op BytecodeOp, a uint8, b uint8, bconst bool, c uint8, cconst bool) Bytecode {
	bbit, cbit := 0, 0
	if bconst {
		bbit = 1
	}
	if cconst {
		cbit = 1
	}
	return Bytecode(
		uint32(cbit)<<cKShift |
			uint32(c)<<cShift |
			uint32(bbit)<<bKShift |
			uint32(b)<<bShift |
			uint32(a)<<aShift |
			uint32(op))
}

// iABx format = | Bx: u16 | A: u8 | Opcode: u6 |
// TODO: we still have 2 bits we can stuff in here
func iABx(op BytecodeOp, a uint8, b uint16) Bytecode {
	return Bytecode(uint32(b)<<bShift | uint32(a)<<aShift | uint32(op))
}

// iAsBx format = | sBx:  16 | A: u8 | Opcode: u6 |
// TODO: we still have 2 bits we can stuff in here
func iAsBx(op BytecodeOp, a uint8, b int16) Bytecode {
	return Bytecode(uint32(b)<<bShift | uint32(a)<<aShift | uint32(op))
}

// isBx  | sBx:  24 | Opcode: u6 |
func exarg(b int32) Bytecode {
	return Bytecode(uint32(b)<<aShift | uint32(EXARG))
}

func (bc Bytecode) op() BytecodeOp  { return BytecodeOp(uint32(bc) & mask6bits) }
func (bc Bytecode) getA() int64     { return int64(uint32(bc) >> aShift & maskByte) }
func (bc Bytecode) getB() int64     { return int64(uint32(bc) >> bShift & maskByte) }
func (bc Bytecode) getC() int64     { return int64(uint32(bc) >> cShift & maskByte) }
func (bc Bytecode) getBx() int64    { return int64(uint32(bc) >> bShift & mask2Bytes) }
func (bc Bytecode) getsBx() int64   { return int64(int16(uint32(bc) >> bShift & mask2Bytes)) }
func (bc Bytecode) getExarg() int64 { return int64(int16(uint32(bc) >> aShift & mask26bits)) }

func (bc Bytecode) getBK() (int64, bool) {
	return int64(uint32(bc) >> bShift & maskByte), (uint32(bc) & (1 << bKShift)) > 0
}

func (bc Bytecode) getCK() (int64, bool) {
	return int64(uint32(bc) >> cShift & maskByte), (uint32(bc) & (1 << cKShift)) > 0
}

// String will format the bytecode so that it is slightly more understandable
// and readable
func (bc *Bytecode) String() string {
	op, ok := opcodeToString[bc.op()]
	if !ok {
		op = "UNDEFINED"
	}
	switch bc.Kind() {
	case BytecodeTypeABx:
		return fmt.Sprintf("%-10v %-5v %-5v", op, bc.getA(), bc.getBx())
	case BytecodeTypeAsBx:
		return fmt.Sprintf("%-10v %-5v %-5v", op, bc.getA(), bc.getsBx())
	default:
		b, bconst := bc.getBK()
		c, cconst := bc.getCK()
		bstr := fmt.Sprintf("%v", b)
		if bconst {
			bstr += "k"
		}
		cstr := fmt.Sprintf("%v", c)
		if cconst {
			cstr += "k"
		}
		return fmt.Sprintf("%-10v %-5v %-5v %-5v", op, bc.getA(), bstr, cstr)
	}
}

// Kind will return which type of bytecode it is, iABC, iABx, iAsBx
func (op Bytecode) Kind() BytecodeType {
	switch op.op() {
	case LOADK, CLOSURE:
		return BytecodeTypeABx
	case JMP, FORLOOP, FORPREP, TFORLOOP, TFORPREP:
		return BytecodeTypeAsBx
	case EXARG:
		return BytecodeTypesBx
	default:
		return BytecodeTypeABC
	}
}
