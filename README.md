# luaf
luaf is an attempt at an implementation of lua 5.4 mostly for my own learning purposes and luafs 🤠

## Getting Started
- install `go install ./cmd/luaf`
- test `go test ./...`

## Reference
- [lua 5.3 bytecode](https://the-ravi-programming-language.readthedocs.io/en/latest/lua_bytecode_reference.html)
- [Lua Bytecode Explorer](http://luac.nl/)
- [Roblox Typesafe lua](https://luau.org/)
- [go build info](https://pkg.go.dev/runtime/debug@go1.23.2#BuildInfo)
- [go assembler](https://github.com/twitchyliquid64/golang-asm)
- [lua server pages](https://github.com/clark15b/luasp)
- [simple template example](https://github.com/jeremyevans/erubi)

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
- [ ] stdlib
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

## Ideas for built in functionality
- lua server pages
- templating
- database interactions
- http handlers
- json library
- output wasm
- JIT
