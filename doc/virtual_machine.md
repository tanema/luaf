# Lua Virtual Machine

## Stack and Registers
Lua employs two stacks. The Callinfo stack tracks activation frames. There is the
secondary stack that is an array of TValue objects. The Callinfo objects index into
this array. Registers are basically slots in the stack array. When a function is
called - the stack is setup as follows:

```
stack            _ENV
|                function reference
|                var arg 1
|                ...
|                var arg n
| framePointer-> fixed arg 1
|                ...
|                fixed arg n
|                local 1
|                ...
|                local n
|                temporaries
|                ...
|  top->
|
V
```

So top is just past the registers needed by the function. The number of registers
is determined based on parameters, locals and temporaries. For each Lua function,
the framePointer of the stack is set to the first fixed parameter or local. All
register addressing is done as offset from framePointer - so R(0) is at
framePointer+0 on the stack. When a function returns, the return values are
copied to location starting at the function reference.

## Function Prototypes.
Each function, including main is constructed as a function prototype. This prototype
contains the following 4 elements that are used during the VM runtime to allow
for execution of instructions. You may see these refernced in the guide

| Attribute | Description |
|-----------|-------------|
| Bytecodes | series of instructions for the VM to interpret
| Constants | list of strings or numbers to be loaded into the stack during runtime.
| FnTable   | definitions of function prototypes defined within this functions scope
| Upindexes | indexes of upvalues to be established when this function is constructed.

## Instruction Summary
Lua bytecode instructions are 32-bits in size. All instructions have an opcode
in the first 6 bits. Instructions can have the following formats:
```
| iABC  | CK: 1 | C: u8 | BK: 1 | B: u8 | A: u8 | Opcode: u6 |
| iABx  |            Bx: u16            | A: u8 | Opcode: u6 |
| iAsBx |           sBx:  16            | A: u8 | Opcode: u6 |
```
BK | CK = 0 or 1 indicate if the params B,C refer to a stack value or a constant
value. Opcode:u6 means there are 64 possible opcodes. Since constants are loaded
with u8 register index max local is 255, however max constants would be 65,536
because LOADK is u16

# Instructions
### Instruction Notation
| Notation   | Description |
|------------|-------------|
| R(N)       | Register N
| RK(N)      | Register N or a constant index X
| PC         | Program Counter
| Kst(n)     | Element n in the constant list
| Upvalue[n] | Name of upvalue with index n
| sBx        | Signed displacement (in field sBx) for all kinds of jumps

## `CALL A B C`
Performs a function call, with register R(A) holding the reference to the function
object to be called. Parameters to the function are placed in the registers following
R(A). **When a function call is the last parameter to another function call, the former
can pass multiple return values, while the latter can accept multiple parameters.**

```
R(A), ... ,R(A+C-2) := R(A)(R(A+1), ... ,R(A+B-1))
```

| Param | Value  | Description |
|-------|--------|-------------|
| A     |        | reference to the function in the stack
| B     | 0      | B = ‘top’, i.e., parameters range from R(A+1) to the top of the stack.
|       | >= 1   | (B-1) parameters. Upon entry to the called function, R(A+1) will become the framePointer.
| C     |  = 0   | ‘top’ is set to `last_result+1`, so that the next open instruction can use ‘top’.
|       | >= 1   | (C-1) return values are saved.

## `TAILCALL A B C`
Performs a tail call, which happens when a return statement has a single function
call as the expression, e.g. return foo(bar). A tail call results in the function
being interpreted within the same call frame as the caller - the stack is replaced
and then a ‘goto’ executed to start at the entry point in the VM.
Tailcalls allow infinite recursion without growing the stack.

```
return R(A)(R(A+1), ... ,R(A+B-1))
```

| Param | Value  | Description |
|-------|--------|-------------|
| A     |        | reference to the function in the stack
| B     | 0      | B = ‘top’, i.e., parameters range from R(A+1) to the top of the stack.
|       | >= 1   | (B-1) parameters. Upon entry to the called function, R(A+1) will become the framePointer.
| C     |        | not used by `TAILCALL`, since all return results are significant

## `RETURN A B`
Returns to the calling function, with optional return values. `RETURN` closes any
open upvalues.

```
return R(A), ... ,R(A+B-2)
```

| Param | Value  | Description |
|-------|--------|-------------|
| A     |        | start position of return values
| B     | 0      | the set of return values range from R(A) to the top of the stack.
|       | >= 1   | (B-1) return values located in consecutive registers from R(A) onwards

## `JMP A sBx`
Performs an unconditional jump, with sBx as a signed displacement. `JMP` is used
in loops, conditional statements, and in expressions when a boolean true/false
need to be generated.

```
pc+=sBx; if (A) close all upvalues >= R(A - 1)
```

| Param | Value  | Description |
|-------|--------|-------------|
| A     | 0      | don't touch upvalues
|       | >= 1   | all upvalues >= R(A-1) will be closed
| sBx   |        | added to the program counter, which points to the next instruction to be executed

## `VARARG A B`
`VARARG` implements the vararg operator `...` in expressions. `VARARG` copies
parameters into a number of registers starting from R(A), padding with nils if
there aren’t enough values.

```
R(A), R(A+1), ..., R(A+B-1) = vararg
```

| Param | Value  | Description |
|-------|--------|-------------|
| A     |        | start position of values.
| B     | 0      | copy all parameters passed.
|       | >= 1   | copy (B-1) parameters passed padded with nil if required.

## `LOADBOOL A B C`
Loads a boolean value (true or false) into register R(A). true is usually encoded
as an integer 1, false is always 0. Using C to skip the next instruction is often
used for conditionally loading a bool value.

```
R(A) := (Bool)B; if (C) pc++
```

| Param | Value  | Description |
|-------|--------|-------------|
| A     |        | destination of boolean value.
| B     | 0      | load true
|       | !0     | load false
| C     | !0     | pc++, skip next instruction.

## `EQ`, `LT` and `LE`
Relational and logic instructions are used in conjunction with other instructions
to implement control structures or expressions. Instead of generating boolean
results, these instructions conditionally perform a jump over the next instruction;
the emphasis is on implementing control blocks. Instructions are arranged so
that there are two paths to follow based on the relational test. Compares RK(B)
and RK(C), which may be registers or constants. If the boolean result is not A,
then skip the next instruction. Conversely, if the boolean result equals A,
continue with the next instruction. GT and GE can be done using LT and LE with
switched registers.

- **EQ**: `if ((RK(B) == RK(C)) ~= A) then PC++`
- **LT**: `if ((RK(B) <  RK(C)) ~= A) then PC++`
- **LE**: `if ((RK(B) <= RK(C)) ~= A) then PC++`

| Param | Value  | Description |
|-------|--------|-------------|
| A     | 1 || 0 | expected outcome of comparison, if not then PC++ (skip next)
| B     |        | left hand value for comparison, register location or constant
| C     |        | right hand value for comparison, register location or constant

## `TEST A B`
Used to implement and and or logical operators, or for testing a single register
in a conditional statement. `TEST` will check if a register equals an expected boolean
value and if not, skip the next instruction.

```
if (boolean(R(A)) != B) then PC++
```

| Param | Value  | Description |
|-------|--------|-------------|
| A     |        | register to be coerced into bool and checked
| B     | 1 || 0 | expected outcome of comparison, if not then PC++ (skip next)

## `TESTSET A B C`
Similar to `TEST`, `TESTSET` will check a register for boolean equality. However
if the value is as expected, it will assign that value to R(A). If not, it will
skip the next instruction (pc++)

```
if (boolean(R(B)) == C) then R(A) := R(B) else PC++
```

| Param | Value  | Description |
|-------|--------|-------------|
| A     |        | register to put the value into if matches C
| B     |        | register to be coerced into bool and checked
| C     | 1 || 0 | expected outcome of comparison, if true assign A, else PC++ (skip next)

## `FORPREP  A sBx`
`FORPREP` initializes a numeric for loop. A numeric for loop requires 4 registers
on the stack, and each register must be a number. The initial value, the limit,
the step and the local variable.

Since `FORLOOP` is used for initial testing of the loop condition as well as
conditional testing during the loop itself, `FORPREP` performs a negative step and
jumps unconditionally to `FORLOOP` so that `FORLOOP` is able to correctly make
the initial loop test. After this initial test, `FORLOOP` performs a loop step as
usual, restoring the initial value of the loop index so that the first iteration
can start.

```
R(A)-=R(A+2); pc+=sBx
```

| Param | Description |
|-------|-------------|
| A     | Initial value and internal loop variable (the internal index)
| A + 1 | Limit value
| A + 2 | Stepping value
| A + 3 | Actual loop variable (the external index) that is local to the for block.
| sBx   | `JMP` amount to the `FORLOOP` instruction

## `FORLOOP A sBx`
In `FORLOOP`, a jump is made back to the start of the loop body if the limit has
not been reached or exceeded. The sense of the comparison depends on whether the
stepping is negative or positive, hence the “<?=” operator. Jumps for both
instructions are encoded as signed displacements in the sBx field. An empty loop
has a `FORLOOP` `sBx` value of -1.

`FORLOOP` updates `R(A+3)`, the external loop index that is local to the loop
block. This is significant if the loop index is used as an upvalue (see below.)
R(A), R(A+1) and R(A+2) are not visible to the programmer. The loop variable ends
with the last value before the limit is reached (unlike C) because it is not
updated unless the jump is made.

```
R(A)+=R(A+2); if R(A) <?= R(A+1) then { pc+=sBx; R(A+3)=R(A) }
```

| Param | Description |
|-------|-------------|
| A     | Initial value and internal loop variable (the internal index)
| A + 1 | Limit value
| A + 2 | Stepping value
| A + 3 | Actual loop variable (the external index) that is local to the for block.
| sBx   | `JMP` amount to the beginning of the loop.

## `TFORCALL A B`
Lua has a generic for loop, implemented by `TFORCALL` and `TFORLOOP`. The generic
for loop keeps 3 items in consecutive register locations to keep track of things.
The iterator function, which is called once per loop, the state and the control variable.
At the start, R(A+2) has an initial value. R(A), R(A+1) and R(A+2) are internal
to the loop and cannot be accessed by the programmer. In addition to these internal
loop variables, the programmer specifies one or more loop variables that are
external and visible to the programmer. These loop variables reside at locations
R(A+3) onwards, and their count is specified in operand B. Operand B must be at
least 1. They are also local to the loop body, like the external loop index in
a numerical for loop. Each time `TFORCALL` executes, the iterator function
referenced by R(A) is called with two arguments: the state and the control
variable `R(A+1)` and `R(A+2)`. The results are returned in the local loop
variables, from `R(A+3)` onwards, up to `R(A+2+B)`.

```
R(A+3), ... ,R(A+2+B) := R(A)(R(A+1), R(A+2))
```

| Param  | Description |
|--------|-------------|
| A      | The iterator function, which is called once per loop.
| A + 1  | The State
| A + 2  | Control Variable
| A + 3  | Loop var
| A + 4  | optional loop var
| B >= 1 | Number of loop params

## `TFORLOOP A sBx`
The `TFORLOOP` instruction tests the first return value. If it is nil, the
iterator loop is at an end, and the for loop block ends by simply moving to
the next instruction. If the control is not nil, there is another iteration, and
the state is assigned as the new value of the control variable. Then the `TFORLOOP`
instruction sends execution back to the beginning of the loop at a sBx offset.

```
if R(A+1) ~= nil then { R(A)=R(A+1); pc += sBx }
```

| Param  | Description |
|--------|-------------|
| A      | The State
| A + 1  | Control Variable
| sBx    | Jump to beginning of the loop

## `CLOSURE A Bx`
Creates an instance (or closure) of a function prototype. The `CLOSURE` instruction
also sets up the upvalues for the closure being defined.

```
R(A) := closure(KPROTO[Bx])
```

| Param | Description |
|-------|-------------|
| A     | Destination of the closure value to be assigned
| Bx    | entry in the parent FnTable of closure prototypes

## `GETUPVAL A B`
`GETUPVAL` copies the value in upvalue number B into register R(A). Each Lua
function may have its own upvalue list. This upvalue list is internal to the
virtual machine

```
R(A) := UpValue[B]
```

| Param | Description |
|-------|-------------|
| A     | Destination of the upvalue into the stack for usage
| B     | Index of the upvalue in this function to be loaded

## `SETUPVAL A B`
`SETUPVAL` copies the value from register R(A) into the upvalue number B in the
upvalue list for that function.

```
UpValue[B] := R(A)
```

| Param | Description |
|-------|-------------|
| A     | Register of the new value to set in the upvalue
| B     | Index of the upvalue in this function to be updated

## `NEWTABLE A B C`
Creates a new empty table at register R(A). Appropriate size values are set in
order to avoid rehashing when initially populating the table with array values
or hash key-value pairs. If an empty table is created, both sizes are zero. If a
table is created with a number of objects, the code generator counts the number
of array elements and the number of hash elements.

```
R(A) := {} (size = B,C)
```

| Param | Description |
|-------|-------------|
| A     | Destination register of new table
| B     | Size of serial array values
| C     | Size of keyed values

## `SETLIST A B C`
Sets the values for a range of array elements in a table referenced by R(A).
Field B is the number of elements to set. The values used to initialize the table
are located in registers R(A+1), R(A+2), and so on.

```
R(A)[(C-1)*FPF+i] := R(A+i), 1 <= i <= B
```

| Param | Value  | Description |
|-------|--------|-------------|
| A     |        | Location of the table for value to be added to
| B     | 0      | the table is set with a variable number of array elements, from register R(A+1) up to the top of the stack. Used with a fncall or varargs
|       | >= 1   | (B-1) array elements to add to the array. R(A+1) .. R(A+1+B-1)
| C     |  = 0   | the next instruction is cast as an integer, and used as the C value. This happens only when operand C is unable to encode the block number, i.e. when C > 511, equivalent to an array index greater than 25550
|       | >= 1   | (C-1) index in the table to insert into the array.

## `GETTABLE A B C`
`GETTABLE` copies the value from a table element into register R(A).

```
R(A) := R(B)[RK(C)]
```

| Param | Description |
|-------|-------------|
| A     | Destination register of the value from the table
| B     | Location register of the table
| C     | Key value in the table, can be either a constant or register value

## `SETTABLE A B C`
`SETTABLE` copies the value from register R(C) or a constant into a table
element.

```
R(A)[RK(B)] := RK(C)
```

| Param | Description |
|-------|-------------|
| A     | Location register of the table
| B     | Key value in the table, can be either a constant or register value
| C     | Value to be added to the table, either register value or constant

## `SELF A B C`
For object-oriented-like programming using tables. Retrieves a function reference
from a table element and places it in register R(A), then a reference to the table
itself is placed in the next register, R(A+1). This instruction saves some messy
manipulation when setting up a method call. R(B) is the register holding the
reference to the table with the method. The method function itself is found
using the table index RK(C), which may be the value of register R(C) or a
constant number.

```
R(A+1) := R(B); R(A) := R(B)[RK(C)]
```

| Param | Description |
|-------|-------------|
| A     | Destination of function to be called, A+1 is destination of self table value
| B     | Table that contains the method to be called and the target of `self`
| C     | Key value in the table to index the function, this can be either a constant or register value

## `GETTABUP A B C`
`GETTABUP` is similar to the `GETTABLE` instruction except that the table is
referenced as an upvalue.

```
R(A) := UpValue[B][RK(C)]
```

| Param | Description |
|-------|-------------|
| A     | Destination register of the value from the table
| B     | index of upvalue for the table
| C     | Key value in the table, can be either a constant or register value

## `SETTABUP A B C`
`SETTABUP` is similar to the `SETTABLE` instruction except that the table is
referenced as an upvalue.

```
UpValue[A][RK(B)] := RK(C)
```

| Param | Description |
|-------|-------------|
| A     | index of the upvalue for the table
| B     | Key value in the table, can be either a constant or register value
| C     | Value to be added to the table, either register value or constant

## `CONCAT A B C`
Performs concatenation of two or more strings. In a Lua source, this is
equivalent to one or more concatenation operators (‘..’) between two or more
expressions. The source registers must be consecutive, and C must always be
greater than B. The result is placed in R(A).

```
R(A) := R(B).. ... ..R(C)
```

| Param | Description |
|-------|-------------|
| A     | Destination of the final string value.
| B     | Starting index of the values to be concatenated.
| C     | End index of the values to be concatenated.

## `LEN A B`
Returns the length of the object in R(B). For strings, and tables, the size is
returned. For other objects, the metamethod `__len` is called. The result, which
is a number, is placed in R(A).

```
R(A) := length of R(B)
```

| Param | Description |
|-------|-------------|
| A     | Destination of the counted value.
| B     | Register location of the value to be measured.

## `MOVE A B`
Copies the value of register R(B) into register R(A). If R(B) holds a table,
or function, then the reference to that object is copied. MOVE is often used for
moving values into place for the next operation.

```
R(A) := R(B)
```

| Param | Description |
|-------|-------------|
| A     | Destination register of the value.
| B     | Source register of the value.

## `LOADNIL A B`
Sets a range of registers from R(A) to R(A+B) to nil. When two or more consecutive
locals need to be assigned nil values, only a single LOADNIL is needed.

```
R(A), R(A+1), ..., R(A+B) := nil
```

| Param | Description |
|-------|-------------|
| A     | Destination register of the nil value.
| B     | Number of consecutive nils to load from A onwards

## `LOADK A Bx`
Loads constant number Bx into register R(A). Constants are usually numbers or
strings. **Each function prototype has its own constant list, or pool.**

```
R(A) := Kst(Bx)
```

| Param | Description |
|-------|-------------|
| A     | Destination register of the value.
| Bx    | Index of the constant to load into the stack.

## `LOADI A Bx`
Loads integer Bx into register R(A).

```
R(A) := Bx
```

| Param | Description |
|-------|-------------|
| A     | Destination register of the value.
| Bx    | Integer value to load into register

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

| Param | Description |
|-------|-------------|
| A     | destination of final computed value
| B     | left hand value, register location or constant
| C     | right hand value, register location or constant

# Unary operators
```
UNM   A B     R(A) := -R(B)
BNOT  A B     R(A) := ~R(B)
NOT   A B     R(A) := not R(B)
```

| Param | Description |
|-------|-------------|
| A     | destination of final computed value
| B     | right hand value, register location or constant
