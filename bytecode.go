package shine

import (
	"fmt"
	"strconv"
	"strings"
)

// bytecode layout
// | iABC  | C: u8 | B: u8 | A: u8 | Opcode: u8 |
// | iABx  |    Bx: u16    | A: u8 | Opcode: u8 |
// | iAsBx |   sBx:  16    | A: u8 | Opcode: u8 |

type (
	BytecodeOp   uint8
	Bytecode     uint32
	BytecodeType string
)

const (
	BytecodeTypeABC  BytecodeType = "iABC"
	BytecodeTypeABx  BytecodeType = "iABx"
	BytecodeTypeAsBx BytecodeType = "iAsBx"

	MOVE     BytecodeOp = iota // Copy a value between registers
	LOADK                      // Load a constant into a register
	LOADBOOL                   // Load a boolean into a register
	LOADNIL                    // Load nil values into a range of registers
	LOADINT                    // Load a raw int
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
	TFORCALL: "TFORCALL",
	SETLIST:  "SETLIST",
	CLOSURE:  "CLOSURE",
	VARARG:   "VARARG",
}

var stringToOpcode = map[string]BytecodeOp{
	"MOVE":     MOVE,
	"LOADK":    LOADK,
	"LOADBOOL": LOADBOOL,
	"LOADNIL":  LOADNIL,
	"GETUPVAL": GETUPVAL,
	"GETTABUP": GETTABUP,
	"GETTABLE": GETTABLE,
	"SETTABUP": SETTABUP,
	"SETUPVAL": SETUPVAL,
	"SETTABLE": SETTABLE,
	"NEWTABLE": NEWTABLE,
	"SELF":     SELF,
	"ADD":      ADD,
	"SUB":      SUB,
	"MUL":      MUL,
	"MOD":      MOD,
	"POW":      POW,
	"DIV":      DIV,
	"IDIV":     IDIV,
	"BAND":     BAND,
	"BOR":      BOR,
	"BXOR":     BXOR,
	"SHL":      SHL,
	"SHR":      SHR,
	"UNM":      UNM,
	"BNOT":     BNOT,
	"NOT":      NOT,
	"LEN":      LEN,
	"CONCAT":   CONCAT,
	"JMP":      JMP,
	"EQ":       EQ,
	"LT":       LT,
	"LE":       LE,
	"TEST":     TEST,
	"TESTSET":  TESTSET,
	"CALL":     CALL,
	"TAILCALL": TAILCALL,
	"RETURN":   RETURN,
	"FORLOOP":  FORLOOP,
	"FORPREP":  FORPREP,
	"TFORLOOP": TFORLOOP,
	"TFORCALL": TFORCALL,
	"SETLIST":  SETLIST,
	"CLOSURE":  CLOSURE,
	"VARARG":   VARARG,
}

func parseOpcode(src string) Bytecode {
	op, err := ParseOpcode(src)
	if err != nil {
		panic(err)
	}
	return op
}

func ParseOpcode(src string) (Bytecode, error) {
	parts := strings.Split(src, " ")
	opcode, ok := stringToOpcode[strings.ToUpper(parts[0])]
	if !ok {
		return 0, fmt.Errorf("unknown opcode %v", parts[0])
	}
	bytecode := Bytecode(opcode)
	switch bytecode.Kind() {
	case BytecodeTypeABx:
		if len(parts) < 3 {
			return 0, fmt.Errorf("Not enough args  to ABx opcode")
		} else if a, err := strconv.ParseUint(parts[1], 10, 8); err != nil {
			return 0, err
		} else if b, err := strconv.ParseUint(parts[2], 10, 16); err != nil {
			return 0, err
		} else {
			return IABx(opcode, uint8(a), uint16(b)), nil
		}
	case BytecodeTypeAsBx:
		if len(parts) < 3 {
			return 0, fmt.Errorf("Not enough args  to AsBx opcode")
		} else if a, err := strconv.ParseUint(parts[1], 10, 8); err != nil {
			return 0, err
		} else if b, err := strconv.ParseInt(parts[2], 10, 16); err != nil {
			return 0, err
		} else {
			return IAsBx(opcode, uint8(a), int16(b)), nil
		}
	default:
		if len(parts) < 4 {
			return 0, fmt.Errorf("Not enough args  to ABx opcode")
		} else if a, err := strconv.ParseUint(parts[1], 10, 8); err != nil {
			return 0, err
		} else if b, err := strconv.ParseUint(parts[2], 10, 8); err != nil {
			return 0, err
		} else if c, err := strconv.ParseUint(parts[3], 10, 8); err != nil {
			return 0, err
		} else {
			return IABC(opcode, uint8(a), uint8(b), uint8(c)), nil
		}
	}
}

// IABC Creates a new Bytecode for an iABC format
func IABC(op BytecodeOp, a, b, c uint8) Bytecode {
	return Bytecode(uint32(c)<<24 | uint32(b)<<16 | uint32(a)<<8 | uint32(op))
}

// IABx Creates a new Bytecode for an iABx format
func IABx(op BytecodeOp, a uint8, b uint16) Bytecode {
	return Bytecode(uint32(b)<<16 | uint32(a)<<8 | uint32(op))
}

// IAsBx Creates a new Bytecode for an iAsBx format
func IAsBx(op BytecodeOp, a uint8, b int16) Bytecode {
	return Bytecode(uint32(b)<<16 | uint32(a)<<8 | uint32(op))
}

// Op calculates the op command from the bytecode
func (bc Bytecode) Op() BytecodeOp {
	return BytecodeOp(uint32(bc) & 0xFF)
}

// ABC returns the abc params for a iABC formatted bytecode
func (bc Bytecode) ABC() (int64, int64, int64) {
	f := uint32(bc)
	return int64(f >> 8 & 0xFF), int64(f >> 16 & 0xFF), int64(f >> 24 & 0xFF)
}

// ABx returns the ab params for a iABx formatted bytecode
func (bc Bytecode) ABx() (int64, int64) {
	f := uint32(bc)
	return int64(f >> 8 & 0xFF), int64(f >> 16 & 0xFFFF)
}

// AsBx returns the ab params for a iAsBx formatted bytecode
func (bc Bytecode) AsBx() (int64, int64) {
	f := uint32(bc)
	return int64(f >> 8 & 0xFF), int64(int16(f >> 16 & 0xFFFF))
}

// String will format the bytecode so that it is slightly more understandable
// and readable
func (bc *Bytecode) String() string {
	op, ok := opcodeToString[bc.Op()]
	if !ok {
		op = "UNDEFINED"
	}
	switch bc.Kind() {
	case BytecodeTypeABx:
		reg, val := bc.ABx()
		return fmt.Sprintf("%v %v %v", op, reg, val)
	case BytecodeTypeAsBx:
		reg, val := bc.AsBx()
		return fmt.Sprintf("%v %v %v", op, reg, val)
	default:
		a, b, c := bc.ABC()
		return fmt.Sprintf("%v %v %v %v", op, a, b, c)
	}
}

// Kind will return which type of bytecode it is, iABC, iABx, iAsBx
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
