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
- [ ] multiple return
    - fncall as last parameter in a fncall should spread
    - fncall as last return value should be tailcall and spread
    - fncall as last item in assignment should spread
    - fncall as last item in table construction
- [ ] vararg
    - vararg as last param in fncall should spread
    - vararg as last return value should spread
    - vararg as last item in assignment should spread
    - vararg as last item in table construction should spread
- [ ] for number loop
- [ ] for generic loop
- [ ] break
- [ ] tail call
- [ ] meta methods
- [ ] string metatable
- [ ] local const
- [ ] local close calls `__close` metamethod when goes out of scope
- [ ] stdfns
    - [x] print()
    - [ ] assert()
    - [ ] dofile()
    - [ ] collectgarbage()
    - [ ] error()
    - [ ] \_G
    - [ ] getmetatable()
    - [ ] ipairs()
    - [ ] load()
    - [ ] loadfile()
    - [ ] next()
    - [ ] pairs()
    - [ ] pcall()
    - [ ] rawequal (v1, v2)
    - [ ] rawget (table, index)
    - [ ] rawlen (v)
    - [ ] rawset (table, index, value)
    - [ ] select (index, ···)
    - [ ] setmetatable (table, metatable)
    - [ ] tonumber (e [, base])
    - [ ] tostring (v)
    - [ ] type (v)
    - [ ] \_VERSION
    - [ ] xpcall (f, msgh [, arg1, ···])
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
- [ ] JIT

## TODOs Optimizations
- [ ] boolean shortcircuit. Right now only short circuits per binary and it could
    be patched to jump the rest of the boolean condition
- [ ] const folding
- [ ] LOADI
- [ ] EXARG we can use loadi more often for numbers with exarg because that gives us 24 bits
- [ ] const upvalues should just be locals since they don't get mutated
- [ ] Refer to what roblox did https://luau.org/performance

## Ideas for built in functionality
- lua server pages
- templating
- database interactions
- http handlers
- json library
- wasm
