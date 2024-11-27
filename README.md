# luaf
luaf is an attempt at an implementation of lua 5.4 mostly for my own learning
purposes and luafs 🤠

## Getting Started
- `make check` ensure you have the tools installed for the project
- `make install` install luaf
- `make test` run tests
- `make help` for more commands to develop with

## TODOs Main
[Lua 5.4 Reference Manual](https://www.lua.org/manual/5.4/)
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
- [x] top indicator to reduce amount of slice resizing.
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
