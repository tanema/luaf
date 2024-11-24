# luaf
luaf is an attempt at an implementation of lua 5.4 mostly for my own learning
purposes and luafs 🤠

## Getting Started
- `make check` ensure you have the tools installed for the project
- `make install` install luaf
- `make test` run tests
- `make help` for more commands to develop with

## TODOs Main Parser
[Lua 5.4 Reference Manual](https://www.lua.org/manual/5.4/)
- [x] do block
- [x] if/else
- [x] while loop
- [x] goto
- [x] single return
- [x] repeat stat
- [x] multiple return
- [x] vararg
- [x] for number loop
- [x] for generic loop
- [x] break
- [x] tail call
- [x] local const
- [ ] meta methods
    - [x] string metatable
    - [x] getmetatable()
    - [x] setmetatable (table, metatable)
    - [x] `__add`
    - [x] `__sub`
    - [x] `__mul`
    - [x] `__div`
    - [x] `__mod`
    - [x] `__pow`
    - [x] `__unm`
    - [x] `__idiv`
    - [x] `__band`
    - [x] `__bor`
    - [x] `__bxor`
    - [x] `__bnot`
    - [x] `__shl`
    - [x] `__shr`
    - [x] `__eq`
    - [x] `__lt`
    - [x] `__le`
    - [x] `__len`
    - [x] `__concat`
    - [x] `__index`
    - [x] `__newindex`
    - [ ] `__call`
- [ ] local close calls `__close` metamethod when goes out of scope
    - [ ] `__close` requires TBC opcode
- [ ] error trace
- [ ] stdfns
    - [x] \_ENV
    - [x] \_G
    - [x] \_VERSION
    - [x] print()
    - [x] assert()
    - [x] tostring (v)
    - [x] type (v)
    - [x] tonumber (e [, base])
    - [x] pairs()
    - [x] next()
    - [x] ipairs()
    - [x] dofile()
    - [x] pcall()
    - [x] xpcall (f, msgh [, arg1, ···])
    - [x] rawequal (v1, v2)
    - [x] rawget (table, index)
    - [x] rawset (table, index, value)
    - [x] rawlen (v)
    - [ ] load()
    - [ ] loadfile()
    - [ ] error()
    - [ ] select (index, ···)
    - [ ] collectgarbage()
- [ ] stdlib
    - [ ] package
        - [ ] require()
    - [ ] table
    - [ ] string
    - [ ] utf8
    - [ ] io
    - [ ] os
    - [ ] coroutine
    - [ ] math
    - [ ] debug

## TODOs Optimizations
- [ ] boolean shortcircuit. Right now only short circuits per binary and it could
    be patched to jump the rest of the boolean condition
- [ ] const folding. if we can precompute constants like 1+1 then we dont need an op
- [ ] LOADI LOADF ect. Opcodes that allow faster minimal operations
- [ ] EXARG we can use loadi & setlist better
- [ ] const upvalues should just be locals since they don't get mutated
- [ ] Refer to what roblox did https://luau.org/performance
- [ ] settables should use loading constants more.

## Ideas for built in functionality
- Magic comments at the start of a file to enable optional functionality like ruby
    - disable auto string coersion to numbers
    - enable type checking strict
- Doc comments
- [Roblox Typesafe lua](https://luau.org/)
- [lua server pages](https://github.com/clark15b/luasp)
- templating
- database interactions
- http handlers
- json library
- WASM
- JIT [go assembler](https://github.com/twitchyliquid64/golang-asm)
