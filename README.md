# luaf
luaf is an attempt at an implementation of lua 5.4 mostly for my own learning
purposes and luafs 🤠

## Getting Started
- `make check` ensure you have the tools installed for the project
- `make install` install luaf
- `make test` run tests
- `make help` for more commands to develop with

## Compatibility
`luaf` should be fully compatible with the lua APIs that are default in lua,
however it will not provide the same API as the C API. It will also be able to
precompile and run precompiled code however that precompiled code is not compatible
with `lua`. `luac` will not be able to run code from `luaf` and visa versa.

Since the point of this implementation is more for using lua than it's use in Go
there is less of an emphasis on a go API though a simple APi exists.

## TODOs
- [ ] boolean shortcircuit. Right now only short circuits per binary and it could
    be patched to jump the rest of the boolean condition
- [ ] const folding. if we can precompute constants like 1+1 then we dont need an op
  - [ ] type hinting will help deeping const folding
- [ ] Upvalue optimizations. Since upvalues need to be closed if we can minimize upvalues then we can speed things up.
  - [ ] const upvalues should just be locals since they don't get mutated.
  - [ ] unmutated upvalues can also be treated like locals.
- [ ] Bytecode param checking so that we do not overflow uints
- [ ] Peephole optimization on bytecode
- [ ] should use binary encoding for string.dump
- [ ] yield
- [ ] debug
  - [ ] debug
  - [ ] traceback

## Bugs
- Patterns still aren't perfect and will need more tweaking
- repl needs multi line and show results
- `__gc` is not called on table items

## Ideas for built in functionality
- Magic comments at the start of a file to enable optional functionality like ruby
    - disable auto string coersion to numbers
    - env readonly
    - disable new globals, only locals.
    - enable type checking levels
    - require only, do not use stdlib like `io` without a require like `local io = require("io")`
- Doc comments
- Type declarations
  - Type hints help deeper const folding
- Templating
- database interactions
- http handlers
- json library
- WASM
- JIT [go assembler](https://github.com/twitchyliquid64/golang-asm)

## Documentation
- [Parsing](./doc/parser.md)
- [Runtime, Bytecode & Virtual Machine](./doc/virtual_machine.md)
- [Lua Metamethods](./doc/metamethods.md)

## References
This repo is not an island. I learned a lot about implementing this from the following
sources and possibly used parts of them as well:

- [Lua](https://lua.org/)
- [Lua Sourcecode](https://github.com/lua/lua)
- [Lua 5.4 Reference Manual](https://www.lua.org/manual/5.4/)
- [LuaJIT](https://github.com/LuaJIT/LuaJIT)
- [Incomplete Lua bytecode reference](https://the-ravi-programming-language.readthedocs.io/en/latest/lua_bytecode_reference.html)
- [Lua bytecode explorer](https://www.luac.nl/)
- [Great write up on how lua works and tutorial](https://wubingzheng.github.io/build-lua-in-rust/en/)
- [glua a good reference how someone else did it in Go](https://github.com/yuin/gopher-lua)
- [Roblox Typesafe lua](https://luau.org/)
- [reference for better fs api from node](https://nodejs.org/docs/latest-v12.x/api/fs.html)
  - [File path](https://github.com/moteus/lua-path)
- [reference for net api](https://nodejs.org/docs/latest-v12.x/api/net.htmlnet.url)
  - [lua url parse](https://github.com/golgote/neturl)
  - [Net.http](https://nodejs.org/docs/latest-v12.x/api/http.html)
- [lua argparse](https://github.com/lunarmodules/lua_cliargs)
- [Lua templates](https://github.com/leafo/etlua)
