Lua Bytecode notes
============
  
- A function's environment is stored in an upvalue, named _ENV
- VM doesnt need to know about variable names! Just keep track of index where that value
is used during parsing and use index
- Immediate parsing execution is slow because you have an AST to traverse every
time you want to call a function.


```lua
local a = "hello world"
```
- push constant `"hello world"` into the constants data store
- parse notes that the variable `a` starts with the constant value at position 0 (where hello world was pushed)
                                       
```lua
print(a)
```
- `print` fn value is loaded into register 1 `GETTABUP 1 0 1; _ENV "print"` A: dst register, B: Ksource for ENV, C: Ksource for print
- a value is moved into a register for calling. In this case moved from register 0 register 2 `MOVE 2 0`
- fn in register 1 is called. `CALL 1 2 1; 1 in 0 out` Parameters to the 
    function are placed in the registers following R(A). If B is 1, the function 
    has no parameters. If B is 2 or more, there are (B-1) parameters. If B >= 2, 
    then upon entry to the called function, R(A+1) will become the base. In this 
    case you can see that we are calling with 2-1 or 1 param.
    If C is 1, no return results are saved. If C is 2 or more, (C-1) return values 
    are saved. If C == 0, then ‘top’ is set to last_result+1, so that the next 
    open instruction (OP_CALL, OP_RETURN, OP_SETLIST) can use ‘top’. So in this
    case we are returning nothing
- Finish program: `RETURN 1 1 1; 0 out`


- By default, Lua has a maximum stack frame size of 250. This is encoded as 
  MAXSTACK in llimits.h.
- The maximum stack frame size in turn limits the maximum number of locals per 
  function, which is set at 200, encoded as LUAI_MAXVARS in luaconf.h
- Other limits found in the same file include the maximum number of upvalues 
  per function (60), encoded as LUAI_MAXUPVALUES, call depths, the minimum C 
  stack size, etc. Also, with an sBx field of 18 bits, jumps and control 
  structures cannot exceed a jump distance of about 131071


MOVE      Copy a value between registers
LOADK     Load a constant into a register
LOADKX    Load a constant into a register
LOADBOOL  Load a boolean into a register
    LOADBOOL A B C    R(A) := (Bool)B; if (C) pc++
    Loads a boolean value (true or false) into register R(A). true is usually encoded as an integer 1, false is always 0. If C is non-zero, then the next instruction is skipped (this is used when you have an assignment statement where the expression uses relational operators, e.g. M = K>5.) You can use any non-zero value for the boolean true in field B, but since you cannot use booleans as numbers in Lua, it’s best to stick to 1 for true.
    LOADBOOL is used for loading a boolean value into a register. It’s also used where a boolean result is supposed to be generated, because relational test instructions, for example, do not generate boolean results – they perform conditional jumps instead. The operand C is used to optionally skip the next instruction (by incrementing PC by 1) in order to support such code. For simple assignments of boolean values, C is always 0.

LOADNIL   Load nil values into a range of registers
    LOADNIL A B     R(A), R(A+1), ..., R(A+B) := nil
    Sets a range of registers from R(A) to R(B) to nil. If a single register is to be assigned to, then R(A) = R(B). When two or more consecutive locals need to be assigned nil values, only a single LOADNIL is needed.

GETUPVAL  Read an upvalue into a register
GETTABUP  Read a value from table in up-value into a register
GETTABLE  Read a table element into a register
SETTABUP  Write a register value into table in up-value
SETUPVAL  Write a register value into an upvalue
SETTABLE  Write a register value into a table element
NEWTABLE  Create a new table
SELF      Prepare an object method for calling
ADD       Addition operator
SUB       Subtraction operator
MUL       Multiplication operator
MOD       Modulus (remainder) operator
POW       Exponentation operator
DIV       Division operator
IDIV      Integer division operator
BAND      Bit-wise AND operator
BOR       Bit-wise OR operator
BXOR      Bit-wise Exclusive OR operator
SHL       Shift bits left
SHR       Shift bits right
UNM       Unary minus
BNOT      Bit-wise NOT operator
NOT       Logical NOT operator
LEN       Length operator
CONCAT    Concatenate a range of registers
JMP       Unconditional jump
EQ        Equality test, with conditional jump
LT        Less than test, with conditional jump
LE        Less than or equal to test, with conditional jump
TEST      Boolean test, with conditional jump
TESTSET   Boolean test, with conditional jump and assignment
CALL      Call a closure
TAILCALL  Perform a tail call
RETURN    Return from function call
FORLOOP   Iterate a numeric for loop
FORPREP   Initialization for a numeric for loop
TFORLOOP  Iterate a generic for loop
TFORCALL  Initialization for a generic for loop
SETLIST   Set a range of array elements for a table
CLOSURE   Create a closure of a function prototype
VARARG    Assign vararg function arguments to registers
