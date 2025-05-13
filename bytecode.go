package luaf

import (
	"fmt"
	"strconv"
)

type (
	// Bytecode is a single instruction that runs in the vm.
	Bytecode uint32
	// BytecodeOp describes which kind of instruction each instruction is.
	BytecodeOp   uint8
	bytecodeType string
)

const (
	bytecodeTypeABC  bytecodeType = "iABC"
	bytecodeTypeABx  bytecodeType = "iABx"
	bytecodeTypeAsBx bytecodeType = "iAsBx"
	bytecodeTypesBx  bytecodeType = "isBx"
	bytecodeTypesEx  bytecodeType = "EXARG"

	// MOVE Copy a value between registers.
	MOVE BytecodeOp = iota
	// LOADK Load a constant into a register.
	LOADK
	// LOADBOOL Load a boolean into a register.
	LOADBOOL
	// LOADNIL Load nil values into a range of registers.
	LOADNIL
	// LOADI Load a raw int.
	LOADI
	// LOADF Load raw float.
	LOADF
	// GETUPVAL Read an upvalue into a register.
	GETUPVAL
	// GETTABUP Read a value from table in up-value into a register.
	GETTABUP
	// GETTABLE Read a table element into a register.
	GETTABLE
	// SETTABUP Write a register value into table in up-value.
	SETTABUP
	// SETUPVAL Write a register value into an upvalue.
	SETUPVAL
	// SETTABLE Write a register value into a table element.
	SETTABLE
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
	// max possible is 6 bits or 64 codes.
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
	TBC:      "TBC",
	JMP:      "JMP",
	CLOSE:    "CLOSE",
	EQ:       "EQ",
	LT:       "LT",
	LE:       "LE",
	TEST:     "TEST",
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
	LOADI:    "LOADI",
	LOADF:    "LOADF",
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

func iAB(op BytecodeOp, a uint8, b uint8) Bytecode {
	return iABC(op, a, b, 0)
}

func iABC(op BytecodeOp, a uint8, b uint8, c uint8) Bytecode {
	return iABCK(op, a, b, false, c, false)
}

// iABC format = | CK: 1 | C: u8 | BK: 1 | B: u8 | A: u8 | Opcode: u6 |.
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

// TODO: we still have 2 bits we can stuff in here.
func iABx(op BytecodeOp, a uint8, b uint16) Bytecode {
	return Bytecode(uint32(b)<<bShift | uint32(a)<<aShift | uint32(op))
}

// TODO: we still have 2 bits we can stuff in here.
func iAsBx(op BytecodeOp, a uint8, b int16) Bytecode {
	return Bytecode(uint32(b)<<bShift | uint32(a)<<aShift | uint32(op))
}

func (bc Bytecode) op() BytecodeOp { return BytecodeOp(uint32(bc) & mask6bits) }
func (bc Bytecode) getA() int64    { return int64(uint32(bc) >> aShift & maskByte) }
func (bc Bytecode) getB() int64    { return int64(uint32(bc) >> bShift & maskByte) }
func (bc Bytecode) getC() int64    { return int64(uint32(bc) >> cShift & maskByte) }
func (bc Bytecode) getBx() int64   { return int64(uint32(bc) >> bShift & mask2Bytes) }
func (bc Bytecode) getsBx() int64  { return int64(int16(uint32(bc) >> bShift & mask2Bytes)) }

func (bc Bytecode) getBK() (int64, bool) {
	return int64(uint32(bc) >> bShift & maskByte), (uint32(bc) & (1 << bKShift)) > 0
}

func (bc Bytecode) getCK() (int64, bool) {
	return int64(uint32(bc) >> cShift & maskByte), (uint32(bc) & (1 << cKShift)) > 0
}

func (bc *Bytecode) String() string {
	op, ok := opcodeToString[bc.op()]
	if !ok {
		op = "UNDEFINED"
	}
	switch bc.kind() {
	case bytecodeTypeABx:
		return fmt.Sprintf("%-10v %-5v %-5v %-5v", op, bc.getA(), bc.getBx(), "")
	case bytecodeTypeAsBx:
		return fmt.Sprintf("%-10v %-5v %-5v %-5v", op, bc.getA(), bc.getsBx(), "")
	case bytecodeTypeABC:
		b, bconst := bc.getBK()
		c, cconst := bc.getCK()
		bstr := strconv.FormatInt(b, 10)
		if bconst {
			bstr += "k"
		}
		cstr := strconv.FormatInt(c, 10)
		if cconst {
			cstr += "k"
		}
		return fmt.Sprintf("%-10v %-5v %-5v %-5v", op, bc.getA(), bstr, cstr)
	case bytecodeTypesEx:
		return fmt.Sprintf("%-10v %-5v", "EXARG", uint32(*bc))
	default:
		return "UNKNOWN OPCODE"
	}
}

// Kind will return which type of bytecode it is, iABC, iABx, iAsBx.
func (bc Bytecode) kind() bytecodeType {
	return opKind(bc.op())
}

func opKind(op BytecodeOp) bytecodeType {
	switch op {
	case LOADK, CLOSURE:
		return bytecodeTypeABx
	case JMP, FORLOOP, FORPREP, TFORLOOP, TFORCALL, LOADI, LOADF:
		return bytecodeTypeAsBx
	case MOVE, LOADBOOL, LOADNIL, GETUPVAL, GETTABUP, GETTABLE, SETTABUP, SETUPVAL,
		SETTABLE, NEWTABLE, SELF, ADD, SUB, MUL, MOD, POW, DIV, IDIV, BAND, BOR, BXOR,
		SHL, SHR, UNM, BNOT, NOT, LEN, CONCAT, TBC, CLOSE, EQ, LT, LE, TEST, CALL,
		TAILCALL, RETURN, SETLIST, VARARG:
		return bytecodeTypeABC
	default:
		return bytecodeTypesEx
	}
}
