# Lua Stack and Registers

Lua employs two stacks. The Callinfo stack tracks activation frames. There is the
secondary stack L->stack that is an array of TValue objects. The Callinfo objects
index into this array. Registers are basically slots in the L->stack array.

When a function is called - the stack is setup as follows:

```
stack
|            function reference
|            var arg 1
|            ...
|            var arg n
| base->     fixed arg 1
|            ...
|            fixed arg n
|            local 1
|            ...
|            local n
|            temporaries
|            ...
|  top->
|
V
```

So top is just past the registers needed by the function. The number of registers
is determined based on parameters, locals and temporaries. For each Lua function,
the base of the stack is set to the first fixed parameter or local. All register
addressing is done as offset from base - so R(0) is at base+0 on the stack.

## Drawing of Lua Stack
The figure above shows how the stack is related to other Lua objects. When the
function returns the return values are copied to location starting at the function
reference.

Instruction Notation
- **R(A)**        Register A (specified in instruction field A)
- **R(B)**        Register B (specified in instruction field B)
- **R(C)**        Register C (specified in instruction field C)
- **PC**          Program Counter
- **Kst(n)**      Element n in the constant list
- **Upvalue[n]**  Name of upvalue with index n
- **Gbl[sym]**    Global variable indexed by symbol sym
- **RK(B)**       Register B or a constant index
- **RK(C)**       Register C or a constant index
- **sBx**         Signed displacement (in field sBx) for all kinds of jumps

## Instruction Summary
Lua bytecode instructions are 32-bits in size. All instructions have an opcode
in the first 6 bits. Instructions can have the following fields:

- **A**  8 bits
- **B**  9 bits
- **C**  9 bits
- **Ax** 26 bits ('A', 'B', and 'C' together)
- **Bx** 18 bits ('B' and 'C' together)
- **sBx** signed Bx

A signed argument is represented in excess K; that is, the number value is the
unsigned value minus K. K is exactly the maximum value for that argument (so
that -max is represented by 0, and +max is represented by 2*max), which is half
the maximum for the corresponding unsigned argument. Note that B and C operands
need to have an extra bit compared to A. This is because B and C can reference
registers or constants, and the extra bit is used to decide which one. But A
always references registers so it doesn’t need the extra bit.

| Opcode   | Description                                         |
|----------|-----------------------------------------------------|
| `MOVE`     | Copy a value between registers                      |
| `LOADK`    | Load a constant into a register                     |
| `LOADKX`   | Load a constant into a register                     |
| `LOADBOOL` | Load a boolean into a register                      |
| `LOADNIL`  | Load nil values into a range of registers           |
| `GETUPVAL` | Read an upvalue into a register                     |
| `GETTABUP` | Read a value from table in up-value into a register |
| `GETTABLE` | Read a table element into a register                |
| `SETTABUP` | Write a register value into table in up-value       |
| `SETUPVAL` | Write a register value into an upvalue              |
| `SETTABLE` | Write a register value into a table element         |
| `NEWTABLE` | Create a new table                                  |
| `SELF`     | Prepare an object method for calling                |
| `ADD`      | Addition operator                                   |
| `SUB`      | Subtraction operator                                |
| `MUL`      | Multiplication operator                             |
| `MOD`      | Modulus (remainder) operator                        |
| `POW`      | Exponentation operator                              |
| `DIV`      | Division operator                                   |
| `IDIV`     | Integer division operator                           |
| `BAND`     | Bit-wise AND operator                               |
| `BOR`      | Bit-wise OR operator                                |
| `BXOR`     | Bit-wise Exclusive OR operator                      |
| `SHL`      | Shift bits left                                     |
| `SHR`      | Shift bits right                                    |
| `UNM`      | Unary minus                                         |
| `BNOT`     | Bit-wise NOT operator                               |
| `NOT`      | Logical NOT operator                                |
| `LEN`      | Length operator                                     |
| `CONCAT`   | Concatenate a range of registers                    |
| `JMP`      | Unconditional jump                                  |
| `EQ`       | Equality test, with conditional jump                |
| `LT`       | Less than test, with conditional jump               |
| `LE`       | Less than or equal to test, with conditional jump   |
| `TEST`     | Boolean test, with conditional jump                 |
| `TESTSET`  | Boolean test, with conditional jump and assignment  |
| `CALL`     | Call a closure                                      |
| `TAILCALL` | Perform a tail call                                 |
| `RETURN`   | Return from function call                           |
| `FORLOOP`  | Iterate a numeric for loop                          |
| `FORPREP`  | Initialization for a numeric for loop               |
| `TFORLOOP` | Iterate a generic for loop                          |
| `TFORCALL` | Initialization for a generic for loop               |
| `SETLIST`  | Set a range of array elements for a table           |
| `CLOSURE`  | Create a closure of a function prototype            |
| `VARARG`   | Assign vararg function arguments to registers       |

# `CALL`
```
CALL A B C    R(A), ... ,R(A+C-2) := R(A)(R(A+1), ... ,R(A+B-1))
```
Performs a function call, with register R(A) holding the reference to the function
object to be called. Parameters to the function are placed in the registers following
R(A). If B is 1, the function has no parameters. If B is 2 or more, there are
(B-1) parameters. If B >= 2, then upon entry to the called function, R(A+1) will
become the base.

If B is 0, then B = ‘top’, i.e., the function parameters range from R(A+1) to
the top of the stack. This form is used when the number of parameters to pass is
set by the previous VM instruction, which has to be one of `CALL` or `VARARG`.

If C is 1, no return results are saved. If C is 2 or more, (C-1) return values
are saved. If C == 0, then ‘top’ is set to `last_result+1`, so that the next open
instruction (`CALL`, `RETURN`, `SETLIST`) can use ‘top’. Results returned
by the function call are placed in a range of registers starting from CI->func.
If C is 1, no return results are saved. If C is 2 or more, (C-1) return values
are saved. If C is 0, then multiple return results are saved.

**When a function call is the last parameter to another function call, the former
can pass multiple return values, while the latter can accept multiple parameters.**

# `TAILCALL`
```
TAILCALL  A B C return R(A)(R(A+1), ... ,R(A+B-1))
```
Performs a tail call, which happens when a return statement has a single function
call as the expression, e.g. return foo(bar). A tail call results in the function
being interpreted within the same call frame as the caller - the stack is replaced
and then a ‘goto’ executed to start at the entry point in the VM. Only Lua
functions can be tailcalled. Tailcalls allow infinite recursion without growing
the stack. Like `CALL`, register `R(A)` holds the reference to the function
object to be called. B encodes the number of parameters in the same manner as a
`CALL` instruction. C isn’t used by TAILCALL, since all return results are
significant.

# `RETURN`
```
RETURN  A B return R(A), ... ,R(A+B-2)
```
Returns to the calling function, with optional return values. First `RETURN`
closes any open upvalues.

If B is 1, there are no return values. If B is 2 or more, there are (B-1) return
values, located in consecutive registers from R(A) onwards. If B is 0, the set
of values range from R(A) to the top of the stack.
If B is 0 then the previous instruction (which must be either `CALL` or
`VARARG` ) would have set L->top to indicate how many values to return. The
number of values to be returned in this case is R(A) to L->top. If B > 0 then
the number of values to be returned is simply B-1.

# `JMP`
```
JMP A sBx   pc+=sBx; if (A) close all upvalues >= R(A - 1)
```
Performs an unconditional jump, with sBx as a signed displacement. sBx is added
to the program counter (PC), which points to the next instruction to be executed.
If sBx is 0, the VM will proceed to the next instruction. If R(A) is not 0 then
all upvalues >= R(A-1) will be closed. `JMP` is used in loops, conditional
statements, and in expressions when a boolean true/false need to be generated.

# `VARARG`
```
VARARG  A B R(A), R(A+1), ..., R(A+B-1) = vararg
```
`VARARG` implements the vararg operator ... in expressions. `VARARG` copies B-1
parameters into a number of registers starting from R(A), padding with nils if
there aren’t enough values. If B is 0, `VARARG` copies as many values as it can
based on the number of parameters passed. If a fixed number of values is required,
B is a value greater than 1. If any number of values is required, B is 0.

# `LOADBOOL`
```
LOADBOOL A B C    R(A) := (Bool)B; if (C) pc++
```
Loads a boolean value (true or false) into register R(A). true is usually encoded
as an integer 1, false is always 0. If C is non-zero, then the next instruction
is skipped (this is used when you have an assignment statement where the expression
uses relational operators, e.g. M = K>5.) You can use any non-zero value for the
boolean true in field B, but since you cannot use booleans as numbers in Lua,
it’s best to stick to 1 for true. `LOADBOOL` is used for loading a boolean value
into a register. It’s also used where a boolean result is supposed to be generated,
because relational test instructions, for example, do not generate boolean results,
they perform conditional jumps instead. The operand C is used to optionally skip
the next instruction (by incrementing PC by 1) in order to support such code.
For simple assignments of boolean values, C is always 0.

# `EQ`, `LT` and `LE`
```
EQ  A B C if ((RK(B) == RK(C)) ~= A) then PC++
LT  A B C if ((RK(B) <  RK(C)) ~= A) then PC++
LE  A B C if ((RK(B) <= RK(C)) ~= A) then PC++
```
Relational and logic instructions are used in conjunction with other instructions
to implement control structures or expressions. Instead of generating boolean
results, these instructions conditionally perform a jump over the next instruction;
the emphasis is on implementing control blocks. Instructions are arranged so
that there are two paths to follow based on the relational test. Compares RK(B)
and RK(C), which may be registers or constants. If the boolean result is not A,
then skip the next instruction. Conversely, if the boolean result equals A,
continue with the next instruction.
`EQ` is for equality. `LT` is for “less than” comparison. `LE` is for
“less than or equal to” comparison. The boolean A field allows the full set of
relational comparison operations to be synthesized from these three instructions.
The Lua code generator produces either 0 or 1 for the boolean A.

For the fall-through case, a `JMP` instruction is always expected, in order
to optimize execution in the virtual machine. In effect, EQ, LT and LE must
always be paired with a following JMP instruction.

# `TEST` and `TESTSET`
```
TEST        A C     if (boolean(R(A)) != C) then PC++
TESTSET     A B C   if (boolean(R(B)) != C) then PC++ else R(A) := R(B)
```
These two instructions used for performing boolean tests and implementing Lua’s
logical operators. Used to implement and and or logical operators, or for testing
a single register in a conditional statement. For `TESTSET`, register R(B) is
coerced into a boolean (i.e. false and nil evaluate to 0 and any other value to
1) and compared to the boolean field C (0 or 1). If boolean(R(B)) does not match
C, the next instruction is skipped, otherwise R(B) is assigned to R(A) and the
VM continues with the next instruction. The and operator uses a C of 0 (false)
while or uses a C value of 1 (true). `TEST` is a more primitive version of
`TESTSET`. `TEST` is used when the assignment operation is not needed, otherwise
it is the same as `TESTSET` except that the operand slots are different.
For the fall-through case, a `JMP` is always expected, in order to optimize
execution in the virtual machine. In effect, `TEST` and `TESTSET` must always be
paired with a following `JMP` instruction.

# `FORPREP` and `FORLOOP`
```
FORPREP    A sBx   R(A)-=R(A+2); pc+=sBx
FORLOOP    A sBx   R(A)+=R(A+2);
                   if R(A) <?= R(A+1) then { pc+=sBx; R(A+3)=R(A) }
```
Lua has dedicated instructions to implement the two types of for loops, while
the other two types of loops uses traditional test-and-jump. `FORPREP` initializes
a numeric for loop, while `FORLOOP` performs an iteration of a numeric for loop.
A numeric for loop requires 4 registers on the stack, and each register must be
a number. R(A) holds the initial value and doubles as the internal loop variable
(the internal index); R(A+1) is the limit; R(A+2) is the stepping value; R(A+3)
is the actual loop variable (the external index) that is local to the for block.
`FORPREP` sets up a for loop. Since `FORLOOP` is used for initial testing of the
loop condition as well as conditional testing during the loop itself, `FORPREP`
performs a negative step and jumps unconditionally to `FORLOOP` so that `FORLOOP`
is able to correctly make the initial loop test. After this initial test,
`FORLOOP` performs a loop step as usual, restoring the initial value of the loop
index so that the first iteration can start. In `FORLOOP`, a jump is made back
to the start of the loop body if the limit has not been reached or exceeded.
The sense of the comparison depends on whether the stepping is negative or positive,
hence the “<?=” operator. Jumps for both instructions are encoded as signed
displacements in the sBx field. An empty loop has a `FORLOOP` `sBx` value of -1.

`FORLOOP` also sets `R(A+3)`, the external loop index that is local to the loop
block. This is significant if the loop index is used as an upvalue (see below.)
R(A), R(A+1) and R(A+2) are not visible to the programmer. The loop variable ends
with the last value before the limit is reached (unlike C) because it is not
updated unless the jump is made. However, since loop variables are local to the
loop itself, you should not be able to use it unless you cook up an
implementation-specific hack.

# `TFORCALL` and `TFORLOOP`
```
TFORCALL    A C        R(A+3), ... ,R(A+2+C) := R(A)(R(A+1), R(A+2))
TFORLOOP    A sBx      if R(A+1) ~= nil then { R(A)=R(A+1); pc += sBx }
```
Apart from a numeric for loop (implemented by `FORPREP` and `FORLOOP`), Lua has a
generic for loop, implemented by `TFORCALL` and `TFORLOOP`. The generic for loop
keeps 3 items in consecutive register locations to keep track of things.
- `R(A)` is the iterator function, which is called once per loop.
- `R(A+1)` is the state
- `R(A+2)` is the control variable.
At the start, R(A+2) has an initial value. R(A), R(A+1) and R(A+2) are internal
to the loop and cannot be accessed by the programmer. In addition to these internal
loop variables, the programmer specifies one or more loop variables that are
external and visible to the programmer. These loop variables reside at locations
R(A+3) onwards, and their count is specified in operand C. Operand C must be at
least 1. They are also local to the loop body, like the external loop index in
a numerical for loop.
Each time `TFORCALL` executes, the iterator function referenced by R(A) is
called with two arguments: the state and the control variable (R(A+1) and R(A+2)).
The results are returned in the local loop variables, from R(A+3) onwards, up
to R(A+2+C). Next, the `TFORLOOP` instruction tests the first return value,
R(A+3). If it is nil, the iterator loop is at an end, and the for loop block
ends by simply moving to the next instruction. If R(A+3) is not nil, there is
another iteration, and R(A+3) is assigned as the new value of the control
variable, R(A+2). Then the `TFORLOOP` instruction sends execution back to the
beginning of the loop (the sBx operand specifies how many instructions to move
to get to the start of the loop body).

# `CLOSURE`
```
CLOSURE A Bx    R(A) := closure(KPROTO[Bx])
```
Creates an instance (or closure) of a function prototype. The Bx parameter
identifies the entry in the parent function’s table of closure prototypes (the
field p in the struct Proto). The indices start from 0, i.e., a parameter of
Bx = 0 references the first closure prototype in the table. The `CLOSURE`
instruction also sets up the upvalues for the closure being defined. This is an
involved process that is worthy of detailed discussion, and will be described
through examples.

# `GETUPVAL` and `SETUPVAL`
```
GETUPVAL  A B     R(A) := UpValue[B]
SETUPVAL  A B     UpValue[B] := R(A)
```
`GETUPVAL` copies the value in upvalue number B into register R(A). Each Lua
function may have its own upvalue list. This upvalue list is internal to the
virtual machine; the list of upvalue name strings in a prototype is not mandatory.
`SETUPVAL` copies the value from register R(A) into the upvalue number B in the
upvalue list for that function.

# `NEWTABLE`
```
NEWTABLE A B C   R(A) := {} (size = B,C)
```
Creates a new empty table at register R(A). B and C are the encoded size
information for the array part and the hash part of the table, respectively.
Appropriate values for B and C are set in order to avoid rehashing when initially
populating the table with array values or hash key-value pairs. If an empty table
is created, both sizes are zero. If a table is created with a number of objects,
the code generator counts the number of array elements and the number of hash
elements.

# `SETLIST`
```
SETLIST A B C   R(A)[(C-1)*FPF+i] := R(A+i), 1 <= i <= B
```
Sets the values for a range of array elements in a table referenced by R(A).
Field B is the number of elements to set. Field C encodes the block number of
the table to be initialized. The values used to initialize the table are located
in registers R(A+1), R(A+2), and so on.
If B is 0, the table is set with a variable number of array elements, from
register R(A+1) up to the top of the stack. This happens when the last element
in the table constructor is a function call or a vararg operator.
If C is 0, the next instruction is cast as an integer, and used as the C value.
This happens only when operand C is unable to encode the block number, i.e. when
C > 511, equivalent to an array index greater than 25550.

# `GETTABLE` and `SETTABLE`
```
GETTABLE A B C   R(A) := R(B)[RK(C)]
SETTABLE A B C   R(A)[RK(B)] := RK(C)
```
`GETTABLE` copies the value from a table element into register R(A). The
table is referenced by register R(B), while the index to the table is given by
RK(C), which may be the value of register R(C) or a constant number.
`SETTABLE` copies the value from register R(C) or a constant into a table
element. The table is referenced by register R(A), while the index to the table
is given by RK(B), which may be the value of register R(B) or a constant number.

# `SELF`
```
SELF  A B C   R(A+1) := R(B); R(A) := R(B)[RK(C)]
```
For object-oriented programming using tables. Retrieves a function reference from
a table element and places it in register R(A), then a reference to the table
itself is placed in the next register, R(A+1). This instruction saves some messy
manipulation when setting up a method call. R(B) is the register holding the
reference to the table with the method. The method function itself is found
using the table index RK(C), which may be the value of register R(C) or a
constant number.

# `GETTABUP` and `SETTABUP`
```
GETTABUP A B C   R(A) := UpValue[B][RK(C)]
SETTABUP A B C   UpValue[A][RK(B)] := RK(C)
```
`GETTABUP` and `SETTABUP` instructions are similar to the `GETTABLE`
and `SETTABLE` instructions except that the table is referenced as an upvalue.
These instructions are used to access global variables, which since Lua 5.2 are
accessed via the upvalue named _ENV.

# `CONCAT`
```
CONCAT A B C   R(A) := R(B).. ... ..R(C)
```
Performs concatenation of two or more strings. In a Lua source, this is
equivalent to one or more concatenation operators (‘..’) between two or more
expressions. The source registers must be consecutive, and C must always be
greater than B. The result is placed in R(A).

# `LEN`
```
LEN A B     R(A) := length of R(B)
```
Returns the length of the object in R(B). For strings, the string length is
returned, while for tables, the table size (as defined in Lua) is returned. For
other objects, the metamethod is called. The result, which is a number, is
placed in R(A).

# `MOVE`
```
MOVE A B     R(A) := R(B)
```
Copies the value of register R(B) into register R(A). If R(B) holds a table,
function or userdata, then the reference to that object is copied. MOVE is often
used for moving values into place for the next operation.

# `LOADNIL`
```
LOADNIL A B     R(A), R(A+1), ..., R(A+B) := nil
```
Sets a range of registers from R(A) to R(B) to nil. If a single register is to
be assigned to, then R(A) = R(B). When two or more consecutive locals need to be
assigned nil values, only a single LOADNIL is needed.

# `LOADK`
```
LOADK A Bx    R(A) := Kst(Bx)
```
Loads constant number Bx into register R(A). Constants are usually numbers or
strings. Each function prototype has its own constant list, or pool.

# Binary operators
```
ADD   A B C   R(A) := RK(B) + RK(C)
SUB   A B C   R(A) := RK(B) - RK(C)
MUL   A B C   R(A) := RK(B) * RK(C)
MOD   A B C   R(A) := RK(B) % RK(C)
POW   A B C   R(A) := RK(B) ^ RK(C)
DIV   A B C   R(A) := RK(B) / RK(C)
IDIV  A B C   R(A) := RK(B) // RK(C)
BAND  A B C   R(A) := RK(B) & RK(C)
BOR   A B C   R(A) := RK(B) | RK(C)
BXOR  A B C   R(A) := RK(B) ~ RK(C)
SHL   A B C   R(A) := RK(B) << RK(C)
SHR   A B C   R(A) := RK(B) >> RK(C)
```
Lua 5.3 implements a bunch of binary operators for arithmetic and bitwise
manipulation of variables. These insructions have a common form. Binary operators
(arithmetic operators and bitwise operators with two inputs.) The result of the
operation between RK(B) and RK(C) is placed into R(A). These instructions are in
the classic 3-register style. The source operands, RK(B) and RK(C), may be
constants. If a constant is out of range of field B or field C, then the constant
will be loaded into a temporary register in advance.

# Unary operators
```
UNM   A B     R(A) := -R(B)
BNOT  A B     R(A) := ~R(B)
NOT   A B     R(A) := not R(B)
```
