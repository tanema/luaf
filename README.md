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
there is no userdata behaviour implemented.

## TODOs Main
[Lua 5.4 Reference Manual](https://www.lua.org/manual/5.4/)
- [ ] stdlib
  - [x] package
  - [x] table
  - [x] math
  - [x] utf8
  - [x] os
  - [ ] string
    - [x] byte
    - [x] char
    - [x] dump
    - [x] find
    - [x] sub
    - [x] rep
    - [x] len
    - [x] lower
    - [x] reverse
    - [x] upper
    - [x] match
    - [x] format
    - [ ] gmatch
    - [ ] gsub
    - [ ] pack
    - [ ] packsize
    - [ ] unpack
  - [ ] io
    - [ ] close
    - [ ] flush
    - [ ] input
    - [ ] lines
    - [ ] open
    - [ ] output
    - [ ] popen
    - [ ] read
    - [ ] stderr
    - [ ] stdin
    - [ ] stdout
    - [ ] tmpfile
    - [ ] type
    - [ ] write
    - [ ] file:close
    - [ ] file:flush
    - [ ] file:lines
    - [ ] file:read
    - [ ] file:seek
    - [ ] file:setvbuf
    - [ ] file:write
  - [ ] coroutine
    - [ ] close
    - [ ] create
    - [ ] isyieldable
    - [ ] resume
    - [ ] running
    - [ ] status
    - [ ] wrap
    - [ ] yield
  - [ ] debug
    - [ ] debug
    - [ ] traceback

## TODOs Optimizations
- [ ] boolean shortcircuit. Right now only short circuits per binary and it could
    be patched to jump the rest of the boolean condition
- [ ] const folding. if we can precompute constants like 1+1 then we dont need an op
  - [ ] type hinting will help deeping const folding
- [ ] LOADI LOADF ect. Opcodes that allow faster minimal operations
- [ ] EXARG we can use loadi & setlist better
- [ ] Upvalue optimizations. Since upvalues need to be closed if we can minimize upvalues then we can speed things up.
  - [ ] const upvalues should just be locals since they don't get mutated.
  - [ ] unmutated upvalues can also be treated like locals.
- [ ] Bytecode param checking so that we do not overflow uints
- [ ] Peephole optimization on bytecode

Refer to what [roblox](https://luau.org/performance) did because they did some
brilliant stuff.

## Ideas for built in functionality
- Magic comments at the start of a file to enable optional functionality like ruby
    - disable auto string coersion to numbers
    - env readonly
    - disable new globals, only locals.
    - enable type checking levels
- Doc comments
- [Roblox Typesafe lua](https://luau.org/)
- [lua server pages](https://github.com/clark15b/luasp)
- templating
- database interactions
- http handlers
- json library
- WASM
- JIT [go assembler](https://github.com/twitchyliquid64/golang-asm)
