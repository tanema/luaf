// Package bytecode handles formatting uint32 values which have meaning for the vm.
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

// Format values in the 32 bit opcode.
const (
	sizeC  = 8
	sizevC = 10
	sizeB  = 8
	sizevB = 6
	sizeBx = (sizeC + sizeB + 1)
	sizeA  = 8
	sizeAx = (sizeBx + sizeA)
	sizesJ = (sizeBx + sizeA)
	sizeOP = 7

	posOP = 0
	posA  = (posOP + sizeOP)
	posk  = (posA + sizeA)
	posB  = (posk + 1)
	posvB = (posk + 1)
	posC  = (posB + sizeB)
	posvC = (posvB + sizevB)
	posBx = posk
	posAx = posA
	possJ = posA

	mask7bits  = 0x7F
	mask6bits  = 0x3F
	mask10bits = 0x3FF
	mask17bits = 0x1FFFF
	mask25bits = 0x1FFFFFF
	mask2Bytes = 0xFFFF
	maskByte   = 0xFF
)

// ToString will convert the op number to a human readable string representation.
func (op *Op) ToString() string {
	return opcodeToString[*op]
}

// ToString will generate a human readable string representation of the bytecode type.
func (t *Type) ToString() string {
	return string(*t)
}

func assertCodeOpType(code uint32, expected Type) {
	assertOpType(Op(code&mask7bits), expected)
}

func assertOpType(op Op, expected Type) {
	if actual := opKind(op); actual != expected {
		panic(fmt.Sprintf("[%v] is of kind %s but %v was incorrectly used.", opcodeToString[op], expected, actual))
	}
}

// IABC will generate a iABC bytecode. See virtual machine documentation for deeper
// definition of how the bytecode is formatted.
func IABC(op Op, a, b, c uint8, hasConst bool) uint32 {
	assertOpType(op, TypeABC)

	kbit := 0
	if hasConst {
		kbit = 1
	}

	return uint32(c)<<posC |
		uint32(b)<<posB |
		uint32(kbit)<<posk |
		uint32(a)<<posA |
		uint32(op)
}

// IvABC will generate a ivABC bytecode which is the extended version of iABC.
// See virtual machine documentation for deeper definition of how the bytecode is formatted.
func IvABC(op Op, a, b uint8, c uint16, hasConst bool) uint32 {
	assertOpType(op, TypevABC)

	kbit := 0
	if hasConst {
		kbit = 1
	}

	return uint32(c)<<posvC |
		uint32(b)<<posvB |
		uint32(kbit)<<posk |
		uint32(a)<<posA |
		uint32(op)
}

// IAB is a helper to create an IABC instruction without constants or a c param.
func IAB(op Op, a uint8, b uint8) uint32 { return IABC(op, a, b, 0, false) }

// IABx creates an instruction with a register and a uint16 value usually load constant.
func IABx(op Op, a uint8, b uint16) uint32 {
	assertOpType(op, TypeABx)
	return uint32(b)<<posB | uint32(a)<<posA | uint32(op)
}

// IAsBx creates an instruction with a register and a signed int16 value often used for jumps.
func IAsBx(op Op, a uint8, b int16) uint32 {
	assertOpType(op, TypeAsBx)
	return uint32(b)<<posBx | uint32(a)<<posA | uint32(op)
}

// ExArg creates a new EXARG instruction.
func ExArg(a uint32) uint32 { return a<<posAx | uint32(EXARG) }

// Jump creates a new JMP instruction.
func Jump(j int32) uint32 { return uint32(j)<<possJ | uint32(JMP) }

// True creates an instruction loads a true value.
func True(dst uint8) uint32 { return IAB(LOADTRUE, dst, 0) }

// False creates an instruction loads a false value.
func False(dst uint8) uint32 { return IAB(LOADFALSE, dst, 0) }

// Bool creates an instruction that loads true or false depending on the value.
func Bool(val bool, dst uint8) uint32 {
	if val {
		return True(dst)
	}
	return False(dst)
}

// Return is an easy way to generate a RETURN, RETURN0 or RETURN1 code without
// having to copy the behaviour all over the codebase.
func Return(sp0 uint8, nret int8) uint32 {
	if nret < 0 {
		return IAB(RETURN, sp0, 0)
	} else if nret == 0 {
		return IAB(RETURN0, sp0, 0)
	} else if nret == 1 {
		return IAB(RETURN1, sp0, 0)
	}
	return IAB(RETURN, sp0, uint8(nret+1))
}

// GetOp gets what type of instruction it is. Used for the switch in the vm.
func GetOp(bc uint32) Op { return Op(bc & mask7bits) }

// GetA gets the a param in all of the instructions.
func GetA(bc uint32) int64 { return int64(bc >> posA & maskByte) }

// GetAx will return the Ax value in an iAx instruction.
func GetAx(bc uint32) uint64 {
	assertCodeOpType(bc, TypeAx)
	return uint64(bc >> posAx & mask25bits)
}

// GetJump will return the jump value.
func GetJump(bc uint32) int64 {
	assertCodeOpType(bc, TypesJ)
	return int64(int16(bc >> possJ & mask25bits))
}

// GetB gets the b param in IABC instructions.
func GetB(bc uint32) int64 {
	assertCodeOpType(bc, TypeABC)
	return int64(bc >> posB & maskByte)
}

// GetvB gets the b param in IvABC instructions.
func GetvB(bc uint32) int64 {
	assertCodeOpType(bc, TypevABC)
	return int64(bc >> posvB & mask6bits)
}

// GetBx gets the b param in IABx instructions.
func GetBx(bc uint32) int64 {
	assertCodeOpType(bc, TypeABx)
	return int64(bc >> posB & mask2Bytes)
}

// GetsBx gets the b param in IAsBx instructions.
func GetsBx(bc uint32) int64 {
	assertCodeOpType(bc, TypeAsBx)
	return int64(int16(bc >> posBx & mask17bits))
}

// GetC gets the c param in IABC instructions.
func GetC(bc uint32) int64 {
	assertCodeOpType(bc, TypeABC)
	return int64(bc >> posC & maskByte)
}

// GetvC gets the c param in IABC instructions.
func GetvC(bc uint32) int64 {
	assertCodeOpType(bc, TypevABC)
	return int64(bc >> posvC & mask10bits)
}

// GetK gets the K param in IABC instructions with an indicator if there is a const
// value in the instruction or not. This has a different meaning per instruction.
func GetK(bc uint32) bool { return (bc & (1 << posk)) > 0 }

// ToString will format an instruction to be understandable.
func ToString(bc uint32) string {
	switch op := Kind(bc); op {
	case TypeABx:
		return fmt.Sprintf(
			"%-10v %-5v %-5v %-5v ",
			opcodeToString[GetOp(bc)],
			GetA(bc),
			GetBx(bc),
			"",
		)
	case TypeAsBx:
		return fmt.Sprintf(
			"%-10v %-5v %-5v %-5v ",
			opcodeToString[GetOp(bc)],
			GetA(bc),
			GetsBx(bc),
			"",
		)
	case TypeABC, TypevABC:
		var b, c string
		switch op {
		case TypeABC:
			b, c = strconv.FormatInt(GetB(bc), 10), strconv.FormatInt(GetC(bc), 10)
		case TypevABC:
			b, c = strconv.FormatInt(GetvB(bc), 10), strconv.FormatInt(GetvC(bc), 10)
		}
		kInd := " "
		if GetK(bc) {
			kInd = "k"
		}
		return fmt.Sprintf("%-10v %-5v %-5v %-5v%v", opcodeToString[GetOp(bc)], GetA(bc), b, c, kInd)
	case TypeAx:
		return fmt.Sprintf("%-10v %-5v", "EXARG", GetAx(bc))
	case TypesJ:
		return fmt.Sprintf("%-10v %-5v", "JMP", GetJump(bc))
	default:
		panic("this should not be able to happen")
	}
}

// Kind will return which type of bytecode it is, iABC, iABx, iAsBx.
func Kind(bc uint32) Type {
	return opKind(Op(bc & mask7bits))
}

// IsReturn checks if an opcode is a RETURN, RETURN1 or RETURN0 opcode.
func IsReturn(bc uint32) bool {
	op := GetOp(bc)
	return op == RETURN || op == RETURN1 || op == RETURN0
}

func opKind(op Op) Type {
	switch op {
	case MOVE, LOADTRUE, LOADFALSE, LFALSESKIP, GETUPVAL, GETTABUP,
		GETTABLE, GETI, GETFIELD, SETTABUP, SETUPVAL, SETTABLE, SETI, SETFIELD,
		SELF, ADD, ADDI, SUB, MUL, MOD, POW, DIV, IDIV, BAND, BOR, BXOR, SHL, SHR,
		SHLI, SHRI, UNM, BNOT, NOT, LEN, CONCAT, TBC, CLOSE, EQ, EQK, EQI, LT, LTI,
		LE, LEI, TEST, TESTSET, CALL, TAILCALL, RETURN, RETURN0, RETURN1, VARARG, SUBK,
		MULK, MODK, POWK, DIVK, IDIVK, BANDK, BORK, BXORK, ADDK, MMBIN, MMBINI, MMBINK,
		GETVARG:
		return TypeABC
	case NEWTABLE, SETLIST:
		return TypevABC
	case LOADI, LOADF, TFORCALL:
		return TypeAsBx
	case LOADK, LOADKX, FORLOOP, FORPREP, TFORLOOP, CLOSURE, LOADNIL, ERRNNIL:
		return TypeABx
	case JMP:
		return TypesJ
	case EXARG:
		return TypeAx
	default:
		return "UNKNOWN"
	}
}
