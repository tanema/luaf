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
- [ ] metamethods
  - [ ] `__gc`: finalizer good for closing connections or files
  - [ ] `__tostring`: allow custom to string behaviour
  - [ ] `__pairs`: allow custom pairs behaviour
  - [ ] `__name`: fallback if __string is not defined
  - [ ] `__mode`: might not use, used for weak reference gc which we don't do
  - [ ] `__metatable` // allow custom getmetatable
- [ ] stdlib
  - [x] package
  - [ ] table
    - [ ] concat
    - [ ] insert
    - [ ] move
    - [ ] pack
    - [ ] remove
    - [ ] sort
    - [ ] unpack
  - [ ] string
    - [ ] byte
    - [ ] char
    - [ ] dump
    - [ ] find
    - [ ] format
    - [ ] gmatch
    - [ ] gsub
    - [ ] len
    - [ ] lower
    - [ ] match
    - [ ] pack
    - [ ] packsize
    - [ ] rep
    - [ ] reverse
    - [ ] sub
    - [ ] unpack
    - [ ] upper
  - [ ] utf8
    - [ ] char
    - [ ] charpattern
    - [ ] codepoint
    - [ ] codes
    - [ ] len
    - [ ] offset
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
  - [ ] os
    - [ ] clock
    - [ ] date
    - [ ] difftime
    - [ ] execute
    - [ ] exit
    - [ ] getenv
    - [ ] remove
    - [ ] rename
    - [ ] setlocale
    - [ ] time
    - [ ] tmpname
  - [ ] coroutine
    - [ ] close
    - [ ] create
    - [ ] isyieldable
    - [ ] resume
    - [ ] running
    - [ ] status
    - [ ] wrap
    - [ ] yield
  - [ ] math
    - [ ] abs
    - [ ] acos
    - [ ] asin
    - [ ] atan
    - [ ] ceil
    - [ ] cos
    - [ ] deg
    - [ ] exp
    - [ ] floor
    - [ ] fmod
    - [ ] huge
    - [ ] log
    - [ ] max
    - [ ] maxinteger
    - [ ] min
    - [ ] mininteger
    - [ ] modf
    - [ ] pi
    - [ ] rad
    - [ ] random
    - [ ] randomseed
    - [ ] sin
    - [ ] sqrt
    - [ ] tan
    - [ ] tointeger
    - [ ] type
    - [ ] ult
  - [ ] debug
    - [ ] debug
    - [ ] traceback

## TODOs Optimizations
- [ ] boolean shortcircuit. Right now only short circuits per binary and it could
    be patched to jump the rest of the boolean condition
- [ ] const folding. if we can precompute constants like 1+1 then we dont need an op
- [ ] LOADI LOADF ect. Opcodes that allow faster minimal operations
- [ ] EXARG we can use loadi & setlist better
- [ ] const upvalues should just be locals since they don't get mutated
- [ ] Refer to what roblox did https://luau.org/performance
- [ ] GC shrink stack
- [ ] Bytecode param checking so that we do not overflow uints

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
