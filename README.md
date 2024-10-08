# Shine
Shine is an attempt at an implementation of lua 5.4 mostly for my own learning purposes

## Reference
- [lua 5.3 bytecode](https://the-ravi-programming-language.readthedocs.io/en/latest/lua_bytecode_reference.html)
- [build a lua](https://wubingzheng.github.io/build-lua-in-rust/en)
- [Lua Bytecode Explorer](http://luac.nl/)
- [Roblox Typesafe lua](https://luau.org/)

## TODOs Main Parser
- [x] local const
- [ ] do block
- [ ] return
- [ ] tail call
- [ ] for loop
- [ ] loop
- [ ] if/else
- [ ] goto
- [ ] break
- [ ] local close
- [ ] string as table
- [ ] stdfns
    - [x] print()
    - [ ] require()
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
    - [ ] table
    - [ ] string
- [ ] JIT

## TODOs Optimizations
Refer to what roblox did https://luau.org/performance
- [ ] const folding
- [ ] LOADI
- [ ] LOADF
- [ ] const upvalues should just be plain values, unbrokered
