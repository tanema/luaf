---
layout: doc
title: Upvalue Life-Cycle
---

Upvalues are an encapsulation concept that allows for values to be enclosed in
a closure function. This allows for the value in parent scoped to be changed by
a function that refers to it even outside of it's original scope.

## Parsing
Parsing has the job of identifying which values link to the above scopes.

- Parser finds an identifier that needs to be resolved
- If the identifier is identified as not being a local
  - The parser will search the parent scope for a local and upwards until it is found.
  - If the upvalue is found in a parent scope.
    - The local is marked as having an upvalue reference for closing when it falls
      out of scope
    - An upindex is created in the first child scope, but marked as from stack,
      as it can be assumed that it is still located in the stack at the time of
      the closure creation. This is because the value is still a local at the time
      of closure creation
    - Upindexes are then created from the first child scope down to the destination
      child scope referencing it. These upindexes are then sourced from the upindex
      of their direct parents unindex values.
- If the identifier is not found it will be set as a global index on `_ENV`
  - `_Env` will then be added as an upindex as it is from the parent scope

## Virtual Machine
In the virtual machine, upvalues are wrapped in a broker. This allows the value
to come from the stack, if it is still in the stack. If the upvalue broker is closed
however, the value is on the heap and the value is stored in the broker.

- CLOSURE instruction called
  - Upindexes are checked:
    - If the upindex is declared as from the stack, it means it is local to the current fn
      - We check if we already have a broker for this value and if not we make a new one.
      - The broker for the value is passed to the closure
    - If the upindex is declared as not from stack, it is indexes from the fn's upindexes
      - already a broker so it is passed to the closure
- JMP instruction called:
  - If instruction A param is > 0, locals from A-1 should be cleaned up
  - If the local is in a broker, that broker should be closed and removed from the broker list
  - the locals are then truncated from the stack
- RETURN instruction called:
  - All open brokers are closed
  - All to be closed values have `__close` called on them
- TAILCALL instruction called:
  - All open brokers are closed
  - All to be closed values have `__close` called on them
  - open broker values and tbc values are reset to empty arrays
- CLOSE instruction called:
  - If instruction A param is > 0, locals from A-1 should be cleaned up
  - If the local is in a broker, that broker should be closed and removed from the broker list
  - the locals are then truncated from the stack
