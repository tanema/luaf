// Package bytecode handles formatting uint32 values which have meaning for the
// vm.
package bytecode

import (
	"fmt"
	"strconv"
)

type (
	// Op is the descriptor of which kind of instruction each bytecode is.
	Op uint8
	// Type is a descriptor of what format an instruction has.
	Type string
)

const (
	// TypeABC is an instruction with an a b and c param all uint8.
	TypeABC Type = "iABC"
	// TypeABx is an instruction with an a uint8 and b uint16 param.
	TypeABx Type = "iABx"
	// TypeAsBx is an instruction with an a uint8 and b int16 param.
	TypeAsBx Type = "iAsBx"
	// TypeEx is a raw uint32 value.
	TypeEx Type = "EXARG"
)

const (
	// MOVE Copy a value between registers.
	MOVE Op = iota
	// LOADI Load a raw int.
	LOADI
	// LOADF Load raw float.
	LOADF
	// LOADK Load a constant into a register.
	LOADK
	// LOADFALSE loads a false into a register.
	LOADFALSE
	// LFALSESKIP is used to convert a condition to a boolean value, in a code
	// equivalent to (not cond ? false : true).  (It produces false and skips the
	// next instruction producing true.)
	LFALSESKIP
	// LOADTRUE loads a true value into a register.
	LOADTRUE
	// LOADNIL Load nil values into a range of registers.
	LOADNIL
	// GETUPVAL Read an upvalue into a register.
	GETUPVAL
	// SETUPVAL Write a register value into an upvalue.
	SETUPVAL
	// GETTABUP Read a value from table in up-value into a register.
	GETTABUP
	// GETTABLE Read a table element into a register.
	GETTABLE
	// GETI Reads a table element by index into a register.
	GETI
	// GETFIELD reads a table element by key into a register.
	GETFIELD
	// SETTABUP Write a register value into table in up-value.
	SETTABUP
	// SETTABLE Write a register value into a table element.
	SETTABLE
	// SETI Write a register value into a table element at an index.
	SETI
	// SETFIELD Write a register value into a table element at a key index.
	SETFIELD
	// NEWTABLE Create a new table.
	NEWTABLE
	// SELF Prepare an object method for calling.
	SELF
	// ADD Addition operator.
	ADD
	// SUB Subtraction operator.
	SUB
	// MUL Multiplication operator.
	MUL
	// MOD Modulus (remainder) operator.
	MOD
	// POW Exponentation operator.
	POW
	// DIV Division operator.
	DIV
	// IDIV Integer division operator.
	IDIV
	// BAND Bit-wise AND operator.
	BAND
	// BOR Bit-wise OR operator.
	BOR
	// BXOR Bit-wise Exclusive OR operator.
	BXOR
	// SHL Shift bits left.
	SHL
	// SHR Shift bits right.
	SHR
	// UNM Unary minus.
	UNM
	// BNOT Bit-wise NOT operator.
	BNOT
	// NOT Logical NOT operator.
	NOT
	// LEN Length operator.
	LEN
	// CONCAT Concatenate a range of registers.
	CONCAT
	// TBC To be closed marke local as needing to be closed.
	TBC
	// JMP Unconditional jump.
	JMP
	// CLOSE close upvalues.
	CLOSE
	// EQ Equality test, with conditional jump.
	EQ
	// LT Less than test, with conditional jump.
	LT
	// LE Less than or equal to test, with conditional jump.
	LE
	// TEST Boolean test, with conditional jump.
	TEST
	// CALL Call a closure.
	CALL
	// TAILCALL Perform a tail call.
	TAILCALL
	// RETURN Return from function call.
	RETURN
	// FORLOOP Iterate a numeric for loop.
	FORLOOP
	// FORPREP Initialization for a numeric for loop.
	FORPREP
	// TFORLOOP Iterate a generic for loop.
	TFORLOOP
	// TFORCALL Initialization for a generic for loop.
	TFORCALL
	// SETLIST Set a range of array elements for a table.
	SETLIST
	// CLOSURE Create a closure of a function prototype.
	CLOSURE
	// VARARG Assign vararg function arguments to registers.
	VARARG

	/*
		EXTRAS WONT WORK UNTIL BYTECODES FIXED
	*/

	// ADDI Add specifically a positive int number.
	ADDI
	// ADDK Add a specific const.
	ADDK
	// SUBK subtracts a constant K instead of loading K.
	SUBK
	// MULK multiplies a constant K instead of loading K.
	MULK
	// MODK modulus a constant K instead of loading K.
	MODK
	// POWK exponent of a constant K instead of loading K.
	POWK
	// DIVK divides a constant K instead of loading K.
	DIVK
	// IDIVK int divides a constant K instead of loading K.
	IDIVK
	// BANDK boolean and a constant K instead of loading K.
	BANDK
	// BORK boolean or a constant K instead of loading K.
	BORK
	// BXORK boolean xor a constant K instead of loading K.
	BXORK
	// SHLI shift left a constant K instead of loading K.
	SHLI
	// SHRI shift right a constant K instead of loading K.
	SHRI
	// MMBIN call metamethod.
	MMBIN
	// MMBINI call metamethod.
	MMBINI
	// MMBINK call metamethod.
	MMBINK
	// EQK compare a value with a constant.
	EQK
	// EQI compare a value with an int.
	EQI
	// LTI less than compare size with int.
	LTI
	// LEI less than equal with int.
	LEI
	// RETURN0 quick return no values.
	RETURN0
	// RETURN1 short form to just return a single value.
	RETURN1
	// TESTSET test against a value but then save it into a register.
	TESTSET
	// MAXCODES is an opcode to indicate max possible is 6 bits or 64 codes.
	MAXCODES
)

var opcodeToString = map[Op]string{
	MOVE:       "MOVE",
	LOADK:      "LOADK",
	LOADFALSE:  "LOADFALSE",
	LFALSESKIP: "LFALSESKIP",
	LOADTRUE:   "LOADTRUE",
	LOADNIL:    "LOADNIL",
	GETUPVAL:   "GETUPVAL",
	GETTABUP:   "GETTABUP",
	GETTABLE:   "GETTABLE",
	GETI:       "GETI",
	GETFIELD:   "GETFIELD",
	SETTABUP:   "SETTABUP",
	SETUPVAL:   "SETUPVAL",
	SETTABLE:   "SETTABLE",
	SETI:       "SETI",
	SETFIELD:   "SETFIELD",
	NEWTABLE:   "NEWTABLE",
	SELF:       "SELF",
	ADD:        "ADD",
	SUB:        "SUB",
	MUL:        "MUL",
	MOD:        "MOD",
	POW:        "POW",
	DIV:        "DIV",
	IDIV:       "IDIV",
	BAND:       "BAND",
	BOR:        "BOR",
	BXOR:       "BXOR",
	SHL:        "SHL",
	SHR:        "SHR",
	UNM:        "UNM",
	BNOT:       "BNOT",
	NOT:        "NOT",
	LEN:        "LEN",
	CONCAT:     "CONCAT",
	TBC:        "TBC",
	JMP:        "JMP",
	CLOSE:      "CLOSE",
	EQ:         "EQ",
	LT:         "LT",
	LE:         "LE",
	TEST:       "TEST",
	CALL:       "CALL",
	TAILCALL:   "TAILCALL",
	RETURN:     "RETURN",
	FORLOOP:    "FORLOOP",
	FORPREP:    "FORPREP",
	TFORLOOP:   "TFORLOOP",
	TFORCALL:   "TFORCALL",
	SETLIST:    "SETLIST",
	CLOSURE:    "CLOSURE",
	VARARG:     "VARARG",
	LOADI:      "LOADI",
	LOADF:      "LOADF",
	ADDI:       "ADDI",
	ADDK:       "ADDK",
	SUBK:       "SUBK",
	MULK:       "MULK",
	MODK:       "MODK",
	POWK:       "POWK",
	DIVK:       "DIVK",
	IDIVK:      "IDIVK",
	BANDK:      "BANDK",
	BORK:       "BORK",
	BXORK:      "BXORK",
	SHLI:       "SHLI",
	SHRI:       "SHRI",
	MMBIN:      "MMBIN",
	MMBINI:     "MMBINI",
	MMBINK:     "MMBINK",
	EQK:        "EQK",
	EQI:        "EQI",
	LTI:        "LTI",
	LEI:        "LEI",
	RETURN0:    "RETURN0",
	RETURN1:    "RETURN1",
	TESTSET:    "TESTSET",
}

// Format values in the 32 bit opcode.
const (
	aShift     = 6
	bShift     = aShift + 8
	bKShift    = bShift + 8
	cShift     = bKShift + 1
	cKShift    = cShift + 8
	mask6bits  = 0x3F
	mask2Bytes = 0xFFFF
	maskByte   = 0xFF
)

// IABCK creates a new bytecode instruction with the format
// | CK: 1 | C: u8 | BK: 1 | B: u8 | A: u8 | Opcode: u6 |.
func IABCK(op Op, a uint8, b uint8, bconst bool, c uint8, cconst bool) uint32 {
	bbit, cbit := 0, 0
	if bconst {
		bbit = 1
	}
	if cconst {
		cbit = 1
	}
	return uint32(cbit)<<cKShift |
		uint32(c)<<cShift |
		uint32(bbit)<<bKShift |
		uint32(b)<<bShift |
		uint32(a)<<aShift |
		uint32(op)
}

// IAB is a helper to create an IABCK instruction without constants or a c param.
func IAB(op Op, a uint8, b uint8) uint32 { return IABC(op, a, b, 0) }

// IABC is a helper to create an IABCK without constants.
func IABC(op Op, a uint8, b uint8, c uint8) uint32 { return IABCK(op, a, b, false, c, false) }

// IABx creates an instruction with a register and a uint16 value usually load constant.
func IABx(op Op, a uint8, b uint16) uint32 { return uint32(b)<<bShift | uint32(a)<<aShift | uint32(op) }

// IAsBx creates an instruction with a register and a signed int16 value often used for jumps.
func IAsBx(op Op, a uint8, b int16) uint32 { return uint32(b)<<bShift | uint32(a)<<aShift | uint32(op) }

// GetOp gets what type of instruction it is. Used for the switch in the vm.
func GetOp(bc uint32) Op { return Op(bc & mask6bits) }

// GetA gets the a param in all of the instructions.
func GetA(bc uint32) int64 { return int64(bc >> aShift & maskByte) }

// GetB gets the p param in IABCK instructions.
func GetB(bc uint32) int64 { return int64(bc >> bShift & maskByte) }

// GetC gets the c param in IABCK instructions.
func GetC(bc uint32) int64 { return int64(bc >> cShift & maskByte) }

// GetBx gets the b param in IABx instructions.
func GetBx(bc uint32) int64 { return int64(bc >> bShift & mask2Bytes) }

// GetsBx gets the b param in IAsBx instructions.
func GetsBx(bc uint32) int64 { return int64(int16(bc >> bShift & mask2Bytes)) }

// GetBK gets the b param in IABCK instructions with an indicator if it is a const or not.
func GetBK(bc uint32) (int64, bool) { return int64(bc >> bShift & maskByte), (bc & (1 << bKShift)) > 0 }

// GetCK gets the c param in IABCK instructions with an indicator if it is a const or not.
func GetCK(bc uint32) (int64, bool) { return int64(bc >> cShift & maskByte), (bc & (1 << cKShift)) > 0 }

// ToString will format an instruction to be understandable.
func ToString(bc uint32) string {
	switch Kind(bc) {
	case TypeABx:
		return fmt.Sprintf("%-10v %-5v %-5v %-5v", opcodeToString[GetOp(bc)], GetA(bc), GetBx(bc), "")
	case TypeAsBx:
		return fmt.Sprintf("%-10v %-5v %-5v %-5v", opcodeToString[GetOp(bc)], GetA(bc), GetsBx(bc), "")
	case TypeABC:
		b, bconst := GetBK(bc)
		c, cconst := GetCK(bc)
		bstr := strconv.FormatInt(b, 10)
		if bconst {
			bstr += "k"
		}
		cstr := strconv.FormatInt(c, 10)
		if cconst {
			cstr += "k"
		}
		return fmt.Sprintf("%-10v %-5v %-5v %-5v", opcodeToString[GetOp(bc)], GetA(bc), bstr, cstr)
	case TypeEx:
		return fmt.Sprintf("%-10v %-5v", "EXARG", bc)
	default:
		panic("this should not be able to happen")
	}
}

// Kind will return which type of bytecode it is, iABC, iABx, iAsBx.
func Kind(bc uint32) Type {
	op := Op(bc & mask6bits)
	switch op {
	case LOADK, SUBK, MULK, MODK, POWK, DIVK, IDIVK, BANDK, BORK, BXORK, CLOSURE, ADDK:
		return TypeABx
	case ADDI, SHLI, SHRI, JMP, FORLOOP, FORPREP, TFORLOOP, TFORCALL, LOADI, LOADF:
		return TypeAsBx
	case MOVE, LOADTRUE, LOADFALSE, LFALSESKIP, LOADNIL, GETUPVAL, GETTABUP,
		GETTABLE, GETI, GETFIELD, SETTABUP, SETUPVAL, SETTABLE, SETI, SETFIELD,
		NEWTABLE, SELF, ADD, SUB, MUL, MOD, POW, DIV, IDIV, BAND, BOR, BXOR,
		SHL, SHR, UNM, BNOT, NOT, LEN, CONCAT, TBC, CLOSE, EQ, LT, LE, TEST, CALL,
		TAILCALL, RETURN, SETLIST, VARARG:
		return TypeABC
	default:
		return TypeEx
	}
}
