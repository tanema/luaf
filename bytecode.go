package shine

import (
	"fmt"
	"strings"
)

type (
	BytecodeOp uint16
	Bytecode   struct {
		Op  BytecodeOp
		Arg []uint16
	}
)

const (
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

func AsBytecode(op BytecodeOp, args ...uint16) Bytecode {
	return Bytecode{
		Op:  op,
		Arg: args,
	}
}

func (bc *Bytecode) String() string {
	return fmt.Sprintf("%v\t%v\t;", bc.Op, join(bc.Arg, " "))
}

func (op BytecodeOp) String() string {
	switch op {
	case MOVE:
		return "MOVE"
	case LOADK:
		return "LOADK"
	case LOADBOOL:
		return "LOADBOOL"
	case LOADNIL:
		return "LOADNIL"
	case GETUPVAL:
		return "GETUPVAL"
	case GETTABUP:
		return "GETTABUP"
	case GETTABLE:
		return "GETTABLE"
	case SETTABUP:
		return "SETTABUP"
	case SETUPVAL:
		return "SETUPVAL"
	case SETTABLE:
		return "SETTABLE"
	case NEWTABLE:
		return "NEWTABLE"
	case SELF:
		return "SELF"
	case ADD:
		return "ADD"
	case SUB:
		return "SUB"
	case MUL:
		return "MUL"
	case MOD:
		return "MOD"
	case POW:
		return "POW"
	case DIV:
		return "DIV"
	case IDIV:
		return "IDIV"
	case BAND:
		return "BAND"
	case BOR:
		return "BOR"
	case BXOR:
		return "BXOR"
	case SHL:
		return "SHL"
	case SHR:
		return "SHR"
	case UNM:
		return "UNM"
	case BNOT:
		return "BNOT"
	case NOT:
		return "NOT"
	case LEN:
		return "LEN"
	case CONCAT:
		return "CONCAT"
	case JMP:
		return "JMP"
	case EQ:
		return "EQ"
	case LT:
		return "LT"
	case LE:
		return "LE"
	case TEST:
		return "TEST"
	case TESTSET:
		return "TESTSET"
	case CALL:
		return "CALL"
	case TAILCALL:
		return "TAILCALL"
	case RETURN:
		return "RETURN"
	case FORLOOP:
		return "FORLOOP"
	case FORPREP:
		return "FORPREP"
	case TFORLOOP:
		return "TFORLOOP"
	case TFORCALL:
		return "TFORCALL"
	case SETLIST:
		return "SETLIST"
	case CLOSURE:
		return "CLOSURE"
	case VARARG:
		return "VARARG"
	default:
		return "UNDEFINED"
	}
}

func join[T any](arr []T, sep string) string {
	out := []string{}
	for _, val := range arr {
		out = append(out, fmt.Sprint(val))
	}
	return strings.Join(out, sep)
}
