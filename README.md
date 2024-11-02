# luaf
luaf is an attempt at an implementation of lua 5.4 mostly for my own learning purposes and luafs 🤠

## Getting Started
- install `go install ./cmd/luaf`
- test `go test ./...`

## TODOs Main Parser
- [x] do block
- [x] if/else
- [x] while loop
- [x] goto
- [x] single return
- [x] repeat stat
- [x] multiple return
- [x] vararg
- [ ] for number loop
- [ ] for generic loop
- [ ] break
- [ ] tail call
- [ ] JMP close brokers
- [ ] meta methods
- [ ] string metatable
- [ ] local const
- [ ] local close calls `__close` metamethod when goes out of scope
- [ ] stdfns
    - [ ] \_G
    - [x] \_VERSION
    - [x] print()
    - [x] assert()
    - [x] tostring (v)
    - [x] type (v)
    - [x] tonumber (e [, base])
    - [ ] dofile()
    - [ ] collectgarbage()
    - [ ] error()
    - [ ] ipairs()
    - [ ] load()
    - [ ] loadfile()
    - [ ] next()
    - [ ] pairs()
    - [ ] pcall()
    - [ ] xpcall (f, msgh [, arg1, ···])
    - [ ] rawequal (v1, v2)
    - [ ] rawget (table, index)
    - [ ] rawlen (v)
    - [ ] rawset (table, index, value)
    - [ ] select (index, ···)
    - [ ] getmetatable()
    - [ ] setmetatable (table, metatable)
    - [ ] require()
- [ ] stdlib
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

## Ideas for built in functionality
- Magic comments at the start of a file to enable optional functionality like ruby
- [Roblox Typesafe lua](https://luau.org/)
- [lua server pages](https://github.com/clark15b/luasp)
- templating
- database interactions
- http handlers
- json library
- WASM
- JIT [go assembler](https://github.com/twitchyliquid64/golang-asm)
