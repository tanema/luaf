<div align="center">
  <h1>
    <code>Luaf</code>
  </h1>
  <p>
    <strong>Lua for learning and laufs </strong>
  </p>
  <p>
    <a href="https://github.com/tanema/luaf/actions">
      <img src="https://github.com/tanema/luaf/actions/workflows/go.yml/badge.svg" alt="build status" />
    </a>
    <a href="https://opensource.org/licenses/MIT"><img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="MIT License" /></a>
  </p>
</div>

luaf is an attempt at an implementation of lua 5.4 mostly for my own learning
purposes and luafs 🤠

![](./docs/luaf.svg | width=100)

## Getting Started
- `make check` ensure you have the tools installed for the project
- `make install` install luaf
- `make test` run tests
- `make help` for more commands to develop with

> [!IMPORTANT]
> `luaf` should be fully compatible with the lua APIs that are default in lua,
> however it will not provide the same API as the C API. It will also be able to
> precompile and run precompiled code however that precompiled code is not compatible
> with `lua`. `luac` will not be able to run code from `luaf` and vise versa.
> Since the point of this implementation is more for using lua than it's use in Go
> there is less of an emphasis on a go API though a simple API exists.

## TODOs
- [ ] `__call` is not called correctly with self defined.
- [ ] function calls are still messy, leftover stack data can end up in params
- [ ] `__gc` is not called on table items
- [ ] should use binary encoding for string.dump
- [ ] yield
- [ ] debug
  - [ ] debug
  - [ ] traceback

## Ideas for built in functionality
- http handlers
- database interactions
- WASM
- JIT [go assembler](https://github.com/twitchyliquid64/golang-asm)

## Documentation
Some notes that I have written myself in efforts to keep track of ideas

- [Parsing](./doc/parser.md)
- [Runtime, Bytecode & Virtual Machine](./doc/virtualmachine.md)
- [Lua Metamethods](./doc/metamethods.md)
- [Upvalues](./doc/upvalues.md)

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
- [lua argparse](https://github.com/mpeterv/argparse)
- [Lua templates](https://github.com/leafo/etlua)
