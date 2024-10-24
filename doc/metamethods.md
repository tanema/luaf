 # Metamethods
Every value in Lua can have a metatable. This metatable is an ordinary Lua table
that defines the behavior of the original value under certain events. You can
change several aspects of the behavior of a value by setting specific fields
in its metatable. For instance, when a non-numeric value is the operand of an
addition, Lua checks for a function in the field `__add` of the value's metatable.
If it finds one, Lua calls this function to perform the addition.

Tables have individual metatables, although multiple tables can share their metatables.
Values of all other types share one single metatable per type; that is, there is
one single metatable for all numbers, one for all strings, etc. By default, a
value has no metatable, but the string library sets a metatable for the string type

### API
- `getmetatable` : query the metatable of a value.
- `setmetatable` : replace the metatable of a table. You cannot change the metatable of other types.

### Binary Operations
If either operand for an operation is not the datatype needed, Lua will try to
call a metamethod.  The operation is as follows
- Checking the first operand even if it is the expected datatype
- If that operand does not define a metamethod for the operation, then Lua will check the second operand.
- With a found metamethod, it is called with the two operands as arguments,
  and the result of the call is the result of the operation.
- If no metamethod is found, Lua raises an error.

| Metamethod | Description                                                     |
|------------|-----------------------------------------------------------------|
| `__add`    | the addition (+) operation.
| `__sub`    | the subtraction (-) operation.
| `__mul`    | the multiplication (\*) operation.
| `__div`    | the division (/) operation.
| `__mod`    | the modulo (%) operation.
| `__pow`    | the exponentiation (^) operation.
| `__unm`    | the negation (unary -) operation.
| `__idiv`   | the floor division (//) operation.
| `__band`   | the bitwise AND (&) operation.
| `__bor`    | the bitwise OR (|) operation.
| `__bxor`   | the bitwise exclusive OR (binary ~) operation.
| `__bnot`   | the bitwise NOT (unary ~) operation.
| `__shl`    | the bitwise left shift (<<) operation.
| `__shr`    | the bitwise right shift (>>) operation.
| `__concat` | the concatenation (..) operation. Invoked if any operand is neither a string nor a number (which is always coercible to a string).
| `__len`    | the length (#) operation. If the object is not a string, Lua will try its metamethod. If there is a metamethod, Lua calls it with the object as argument, and the result of the call is the result of the operation. If there is no metamethod but the object is a table, then Lua uses the table length operation
| `__eq`     | the equal (==) operation. Invoked only when the values being compared are both tables and they are not primitively equal.
| `__lt`     | the less than (<) operation. Invoked only when the values are neither both numbers nor both strings
| `__le`     | the less equal (<=) operation.
| `__index`  | The indexing access operation table[key]. This event happens when table is not a table or when key is not present in table. The metavalue is looked up in the metatable of table. The metavalue for this event can be either a function, a table, or any value with an `__index` metavalue. If it is a function, it is called with table and key as arguments, and the result of the call (adjusted to one value) is the result of the operation. Otherwise, the final result is the result of indexing this metavalue with key. This indexing is regular, not raw, and therefore can trigger another `__index` metavalue.
|`__newindex`| The indexing assignment table[key] = value. Like the index event, this event happens when table is not a table or when key is not present in table. The metavalue is looked up in the metatable of table. Like with indexing, the metavalue for this event can be either a function, a table, or any value with an `__newindex` metavalue. If it is a function, it is called with table, key, and value as arguments. Otherwise, Lua repeats the indexing assignment over this metavalue with the same key and value. This assignment is regular, not raw, and therefore can trigger another `__newindex` metavalue. Whenever a `__newindex` metavalue is invoked, Lua does not perform the primitive assignment. If needed, the metamethod itself can call rawset to do the assignment.
| `__call`   | The call operation func(args). This event happens when Lua tries to call a non-function value (that is, func is not a function). The metamethod is looked up in func. If present, the metamethod is called with func as its first argument, followed by the arguments of the original call (args). All results of the call are the results of the operation. This is the only metamethod that allows multiple results.
