package bytecode

// Format reference
//  iABCK (OP: u7, A: u8, B: u8, C: u8,  K: u1)
// ivABCK (OP: u7, A: u8, B: u6, C: u10, K: u1)
//   iABx (OP: u7, A: u8,  Bx: u17)
//  iAsBx (OP: u7, A: u8, sBx: i17)
//    iAx (OP: u7, Ax: u25)
//    isJ (OP: u7, sJ: i25)

// K in iABC & ivABC
// NewTable  k = has extra arg
// TAILCALL  k = close upvalues
// RETURN    k = close upvalues
// SETLIST   k = has extra arg
// SETTABUP  k = C is const
// SETTABLE  k = C is const
// SETI      k = C is const
// SETFIELD  k = "

const (
	// TypeABC is an instruction with an a b and c param all uint8.
	TypeABC Type = "iABC"
	// TypevABC is a variant (v) of iABC with a larger C and smaller B.
	TypevABC Type = "ivABC"
	// TypeABx is an instruction with an a uint8 and b uint16 param.
	TypeABx Type = "iABx"
	// TypeAsBx is an instruction with an a uint8 and b int16 param.
	TypeAsBx Type = "iAsBx"
	// TypeAx is a raw uint32 value allowing extra args.
	TypeAx Type = "iAx"
	// TypesJ is a raw uint32 jump value.
	TypesJ Type = "isJ"
	// TypeUNKNOWN is really just an error state. All instructions should be known.
	TypeUNKNOWN = "UNKNOWN"
)

const (
	// MOVE Copy a value between registers.
	// A B	R[A] := R[B]
	MOVE Op = iota
	// LOADI Load a raw int.
	// A sBx	R[A] := sBx
	LOADI
	// LOADF Load raw float.
	// A sBx	R[A] := (lua_Number)sBx
	LOADF
	// LOADK Load a constant into a register.
	// A Bx	R[A] := K[Bx]
	LOADK
	// LOADKX Load a constant into a register. The next instruction is always EXTRAARG.
	// A	R[A] := K[extra arg]
	LOADKX
	// LOADFALSE loads a false into a register.
	// A	R[A] := false
	LOADFALSE
	// LFALSESKIP is used to convert a condition to a boolean value, in a code
	// equivalent to (not cond ? false : true).  (It produces false and skips the
	// next instruction producing true.)
	// A	R[A] := false; pc++
	LFALSESKIP
	// LOADTRUE loads a true value into a register.
	// A	R[A] := true
	LOADTRUE
	// LOADNIL Load nil values into a range of registers.
	// A B	R[A], R[A+1], ..., R[A+B] := nil
	LOADNIL
	// GETUPVAL Read an upvalue into a register.
	// A B	R[A] := UpValue[B
	GETUPVAL
	// SETUPVAL Write a register value into an upvalue.
	// A B	UpValue[B] := R[A]
	SETUPVAL
	// GETTABUP Read a value from table in up-value into a register.
	// A B C	R[A] := UpValue[B][K[C]:shortstring]
	GETTABUP
	// GETTABLE Read a table element into a register.
	// A B C	R[A] := R[B][R[C]
	GETTABLE
	// GETI Reads a table element by index into a register.
	// A B C	R[A] := R[B][C]
	GETI
	// GETFIELD reads a table element by key into a register.
	// A B C	R[A] := R[B][K[C]:shortstring]
	GETFIELD
	// SETTABUP Write a register value into table in up-value.
	// A B C	UpValue[A][K[B]:shortstring] := RK(C)
	SETTABUP
	// SETTABLE Write a register value into a table element.
	// A B C	R[A][R[B]] := RK(C)
	SETTABLE
	// SETI Write a register value into a table element at an index.
	// A B C	R[A][B] := RK(C)
	SETI
	// SETFIELD Write a register value into a table element at a key index.
	// A B C	R[A][K[B]:shortstring] := RK(C)
	SETFIELD
	// NEWTABLE Create a new table. The next instruction is always OP_EXTRAARG.
	// A vB vC k	R[A] := {}
	// vB is log2 of the hash size (which is always a power of 2) plus 1, or zero for size zero.
	// If not k, the array size is vC. Otherwise, the array size is EXTRAARG _ vC.
	NEWTABLE
	// SELF Prepare an object method for calling.
	// A B C	R[A+1] := R[B]; R[A] := R[B][K[C]:shortstring
	SELF
	// ADDI Add specifically a positive int number.
	// A B sC	R[A] := R[B] + sC
	ADDI
	// ADDK Add a specific const.
	// A B C	R[A] := R[B] + K[C]:number
	ADDK
	// SUBK subtracts a constant K instead of loading K.
	// A B C	R[A] := R[B] - K[C]:number
	SUBK
	// MULK multiplies a constant K instead of loading K.
	// A B C	R[A] := R[B] * K[C]:numbe
	MULK
	// MODK modulus a constant K instead of loading K.
	// A B C	R[A] := R[B] % K[C]:number
	MODK
	// POWK exponent of a constant K instead of loading K.
	// A B C	R[A] := R[B] ^ K[C]:number
	POWK
	// DIVK divides a constant K instead of loading K.
	// A B C	R[A] := R[B] / K[C]:number
	DIVK
	// IDIVK int divides a constant K instead of loading K.
	// A B C	R[A] := R[B] // K[C]:number
	IDIVK
	// BANDK boolean and a constant K instead of loading K.
	// A B C	R[A] := R[B] & K[C]:integer
	BANDK
	// BORK boolean or a constant K instead of loading K.
	// A B C	R[A] := R[B] | K[C]:integer
	BORK
	// BXORK boolean xor a constant K instead of loading K.
	// A B C	R[A] := R[B] ~ K[C]:integer
	BXORK
	// SHLI shift left a constant K instead of loading K.
	// A B sC	R[A] := sC << R[B]
	SHLI
	// SHRI shift right a constant K instead of loading K.
	// A B sC	R[A] := R[B] >> sC
	SHRI
	// ADD Addition operator.
	// A B C	R[A] := R[B] + R[C]
	ADD
	// SUB Subtraction operator.
	// A B C	R[A] := R[B] - R[C]
	SUB
	// MUL Multiplication operator.
	// A B C	R[A] := R[B] * R[C]
	MUL
	// MOD Modulus (remainder) operator.
	// A B C	R[A] := R[B] % R[C]
	MOD
	// POW Exponentation operator.
	// A B C	R[A] := R[B] ^ R[C]
	POW
	// DIV Division operator.
	// A B C	R[A] := R[B] / R[C]
	DIV
	// IDIV Integer division operator.
	// A B C	R[A] := R[B] // R[C]
	IDIV
	// BAND Bit-wise AND operator.
	// A B C	R[A] := R[B] & R[C]
	BAND
	// BOR Bit-wise OR operator.
	// A B C	R[A] := R[B] | R[C]
	BOR
	// BXOR Bit-wise Exclusive OR operator.
	// A B C	R[A] := R[B] ~ R[C]
	BXOR
	// SHL Shift bits left.
	// A B C	R[A] := R[B] << R[C]
	SHL
	// SHR Shift bits right.
	// A B C	R[A] := R[B] >> R[C]
	SHR
	// MMBIN call metamethod.
	// A B C	call C metamethod over R[A] and R[B]
	// MMBIN and variants follow each arithmetic and bitwise opcode. If the operation
	// succeeds, it skips this next opcode. Otherwise, this opcode calls the corresponding metamethod.
	MMBIN
	// MMBINI call metamethod. k means the arguments were flipped (the constant is the first operand).
	// A sB C k	call C metamethod over R[A] and sB
	MMBINI
	// MMBINK call metamethod. k means the arguments were flipped (the constant is the first operand).
	// A B C k		call C metamethod over R[A] and K[B]
	MMBINK
	// UNM Unary minus.
	// A B	R[A] := -R[B]
	UNM
	// BNOT Bit-wise NOT operator.
	// A B	R[A] := ~R[B]
	BNOT
	// NOT Logical NOT operator.
	// A B	R[A] := not R[B]
	NOT
	// LEN Length operator.
	// A B	R[A] := #R[B] (length operator)
	LEN
	// CONCAT Concatenate a range of registers.
	// A B	R[A] := R[A].. ... ..R[A + B - 1]
	CONCAT
	// CLOSE close upvalues.
	// A	close all upvalues >= R[A]
	CLOSE
	// TBC To be closed marke local as needing to be closed.
	// A	mark variable A "to be closed"
	TBC
	// JMP Unconditional jump.
	// sJ	pc += sJ
	JMP

	//
	// All Comparisons
	// - k specifies what condition the test should accept (true or false).
	// - comparison and test instructions assume that the instruction being skipped (pc++) is a jump.
	// - In comparisons with an immediate operand, C signals whether the original
	//   operand was a float. (It must be corrected in case of metamethods.)
	//

	// EQ Equality test, with conditional jump.
	// A B k	if ((R[A] == R[B]) ~= k) then pc++
	EQ
	// LT Less than test, with conditional jump.
	// A B k	if ((R[A] <  R[B]) ~= k) then pc+
	LT
	// LE Less than or equal to test, with conditional jump.
	// A B k	if ((R[A] <= R[B]) ~= k) then pc++
	LE
	// EQK compare a value with a constant.
	// A B k	if ((R[A] == K[B]) ~= k) then pc++
	EQK
	// EQI compare a value with an int.
	// A sB k	if ((R[A] == sB) ~= k) then pc++
	EQI
	// LTI less than compare size with int.
	// A sB k	if ((R[A] < sB) ~= k) then pc++
	LTI
	// LEI less than equal with int.
	// A sB k	if ((R[A] <= sB) ~= k) then pc++
	LEI
	// TEST Boolean test, with conditional jump.
	// A k	if (not R[A] == k) then pc++
	TEST
	// TESTSET test against a value but then save it into a register. Used in short-circuit expressions
	// that need both to jump and to produce a value, such as (a = b or c).
	// A B k	if (not R[B] == k) then pc++ else R[A] := R[B]
	TESTSET
	// CALL Call a closure. if (B == 0) then B = top - A. If (C == 0), then 'top' is set
	// to last_result+1, so next open instruction (CALL, RETURN*, SETLIST) may use 'top'.
	// A B C	R[A], ... ,R[A+C-2] := R[A](R[A+1], ... ,R[A+B-1])
	CALL
	// TAILCALL Perform a tail call.
	// 'k' specifies that the function builds upvalues, which may need to be closed. C > 0 means
	// the function has hidden vararg arguments, so that its 'func' must be corrected
	// before returning; in this case, (C - 1) is its number of fixed parameters.
	// A B C k	return R[A](R[A+1], ... ,R[A+B-1])
	TAILCALL
	// RETURN Return from function call. if (B == 0) then return up to 'top'.
	// 'k' specifies that the function builds upvalues, which may need to be closed. C > 0 means
	// the function has hidden vararg arguments, so that its 'func' must be corrected
	// before returning; in this case, (C - 1) is its number of fixed parameters.
	// A B C k	return R[A], ... ,R[A+B-2]
	RETURN
	// RETURN0 quick return no values.
	RETURN0
	// RETURN1 short form to just return a single value.
	// A	return R[A]
	RETURN1
	// FORLOOP Iterate a numeric for loop.
	// A Bx	update counters; if loop continues then pc-=Bx;
	FORLOOP
	// FORPREP Initialization for a numeric for loop.
	// A Bx	<check values and prepare counters>; if not to run then pc+=Bx+1;
	FORPREP
	// TFORCALL Initialization for a generic for loop.
	// A C	R[A+4], ... ,R[A+3+C] := R[A](R[A+1], R[A+2]);
	TFORCALL
	// TFORLOOP Iterate a generic for loop.
	// A Bx	if R[A+2] ~= nil then { R[A]=R[A+2]; pc -= Bx }
	TFORLOOP
	// SETLIST Set a range of array elements for a table. if (B == 0) then real B = 'top';
	// if k, then real C = EXTRAARG _ C (the bits of EXTRAARG concatenated with the bits of C).
	// A vB vC k	R[A][vC+i] := R[A+i], 1 <= i <= vB
	SETLIST
	// CLOSURE Create a closure of a function prototype.
	// A Bx	R[A] := closure(KPROTO[Bx])
	CLOSURE
	// VARARG Assign vararg function arguments to registers. if (C == 0) then use actual number
	// of varargs and set top (like in OP_CALL with C == 0). 'k' means function has a
	// vararg table, which is in R[B].
	// A B C k	R[A], ..., R[A+C-2] = varargs
	VARARG
	// GETVARG Get a value in varargs at index
	// A B C	R[A] := R[B][R[C]], R[B] is vararg parameter
	GETVARG
	// ERRNNIL Raise an error if value is nil
	// A Bx	raise error if R[A] ~= nil (K[Bx - 1] is global name)
	// (Bx == 0) means index of global name doesn't fit in Bx. (So, that name is not available for the error message.)
	ERRNNIL
	// EXARG is an extra arg to other methods such as NEWTABLE.
	// Ax	extra (larger) argument for previous opcode	*
	EXARG
	// MAXCODES is an opcode to indicate max possible is 6 bits or 64 codes.
	MAXCODES
)

var opcodeToString = map[Op]string{
	MOVE:       "MOVE",
	LOADK:      "LOADK",
	LOADKX:     "LOADKX",
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
	EXARG:      "EXARG",
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
	GETVARG:    "GETVARG",
	ERRNNIL:    "ERRNNIL",
}
