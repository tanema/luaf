<div align="center">
  <img src="https://github.com/tanema/luaf/raw/main/doc/luaf.svg?sanitize=true" width=300/>
  <h1><code>Luaf</code></h1>
  <p>
    <strong>Lua for learning and laufs </strong>
  </p>
  <p>
    <a href="https://github.com/tanema/luaf/actions">
      <img src="https://github.com/tanema/luaf/actions/workflows/go.yml/badge.svg?sanitize=true" alt="build status" />
    </a>
    <a href="https://opensource.org/licenses/MIT">
      <img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="MIT License" />
    </a>
    <a href="https://pkg.go.dev/github.com/tanema/luaf">
      <img src="https://pkg.go.dev/badge/github.com/tanema/luaf.svg" alt="Go Reference">
    </a>
  </p>
</div>

luaf is an implementation of lua 5.4 for learning purposes and luafs 🤠. It aims
to be fully feature compatible with lua 5.4 as well as additions to the standard
library to make it more of an everyday use language instead of just as an embedded
language.

> [!WARNING]
> `luaf` is still very WIP and really shouldn't be used by anyone except me and
> maybe people who are interested in lua implementations.

## Getting Started
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
- [ ] `__gc` is not called on table items
- [ ] support for big numbers big.Int and big.Float right now `10000000000000000000000` overflows
- [ ] reduce allocations for optimization
  - Value as unified struct rather than an interface
  - When doing binops, just change left operand to the result and return

## Ideas for built in functionality
- http handlers
- database interactions
- Output compilation
  - WASM
  - JIT [go assembler](https://github.com/twitchyliquid64/golang-asm)

## Documentation
Some notes that I have written myself in efforts to keep track of ideas

- [Parsing](./doc/parser.md)
- [Runtime, Bytecode & Virtual Machine](./doc/virtualmachine.md)
- [Lua Metamethods](./doc/metamethods.md)
- [Upvalues](./doc/upvalues.md)
- [JIT learnings](./docs/jit.md)

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
- [Go Allocation optimization](https://gist.github.com/CAFxX/e96e8a5c3841d152f16d266a1fe7f8bd)
- [WASM Intro](https://webassembly.github.io/spec/core/intro/introduction.html)
- [Understanding Every Byte in a WASM Module](https://danielmangum.com/posts/every-byte-wasm-module/)
- [avo](https://github.com/mmcloughlin/avo)
- [generating asm](https://github.com/akyoto/asm)
- [Go Performance Patterns](https://goperf.dev)
