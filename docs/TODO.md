## Bugs found in codebase review
- [ ] `SETLIST` with the extended-arg (`EXARG`) path reads the array index via `GetAx` on the SETLIST instruction itself instead of the fetched EXARG instruction, panicking the whole process. Any table constructor with 600+ positional elements crashes. (internal/runtime/vm.go:493)
- [ ] `explistWant` computes `uint8(want - len(exprs))`, which underflows to a huge value whenever more expressions are given than wanted, emitting a `LOADNIL` for hundreds of registers past the function's real window. (internal/parse/parser.go:1407)
- [ ] `goto`-based loops (`::label:: ... goto label`) never emit `CLOSE`, so a closure created inside the loop body shares one upvalue across all iterations instead of getting a fresh one each time — the same bug class already fixed for `for` loops, just missed here. (internal/parse/parser.go, gotostat)
- [ ] `TAILCALL` explicitly calls `vm.closeUpvalues(f)` and then calls `vm.cleanup(f, ...)`, which closes the same frame's upvalues and `<close>` values again, running a `__close` metamethod twice on a tail call. (internal/runtime/vm.go:551-553)
- [ ] The CLI's `arg` table and `...` are built from raw, unparsed `os.Args` instead of the actual script path and its trailing arguments, so `luaf script.lua a b` gives scripts the wrong/missing `arg[0..N]`. (cmd/luaf/main.go, lib_table.go argsToTableValues)
- [ ] `luaf -- script.lua args...` always drops into the REPL instead of running the script, because Go's `flag` package already strips `--` before `main()`'s own `--` handling runs, which then clears the args list unconditionally. (cmd/luaf/main.go)
- [ ] `coroutine.running()` requires a mandatory thread argument, breaking the standard zero-argument form used to introspect the current coroutine. (internal/runtime/lib_coroutine.go)

## Fixes
- [ ] global keyword https://www.lua.org/manual/5.5/manual.html#2.2
- [ ] named varargs `...name`
- [ ] Parsing huge numbers. There are numbers that just overflow int64 but lua can parse them somehow this may require a huge rewrite in how I pass around values and I am not excited about it.
- [ ] Finish integrating the rest of the lua tests.
    - [ ] events
    - [ ] coroutines
    - [ ] math
    - [ ] files
    - [ ] locals
    - [ ] nextvar

## Optimizations
- [ ] Table Bytecode
    - [ ] GETI
    - [ ] GETFIELD
    - [ ] SETI
    - [ ] SETFIELD
- [ ] Arithmetic
    - [x] ADDI
    - [x] ADDK
    - [ ] SHLI
    - [ ] SHRI
    - [ ] SUBK
    - [ ] MULK
    - [ ] MODK
    - [ ] POWK
    - [ ] DIVK
    - [ ] IDIVK
    - [ ] BANDK
    - [ ] BORK
    - [ ] BXORK
- [ ] Boolean logic
    - [ ] EQK
    - [ ] EQI
    - [ ] LTI
    - [ ] LEI
    - [ ] TESTSET
- [ ] Metamethods
    - [ ] MMBIN    A B C      call C metamethod over R[A] and R[B]
    - [ ] MMBINI   A sB C k   call C metamethod over R[A] and sB
    - [ ] MMBINK   A B C k    call C metamethod over R[A] and K[B]
- [ ] Loop unrolling.
- [ ] Pigeonhole optimizations on bytecode
- [ ] constant Upvalue replacement so just value is passed and upvalue does not need to remain opened.

# Features
- [ ] Parser Config 
    - [x] Comment parsing to change config
    - [ ] StringCoers to allow strings to be coorced in arith
    - [ ] RequireOnly to required the file to require std libs like `os` instead of just assume they are available.
    - [ ] EnvReadonly to disallow to define new globals or changes existing globals
    - [ ] LocalOnly to disallow to define new globals, only locals 
    - [ ] Locale set locale across file and project without having to call setlocale
    - [ ] Disable libs like `os` to disable file access
- [ ] Error message localization depending on locale
- [ ] Enable better supportive error messages.

## Type system
- [ ] definitions
- [ ] checking
- [ ] casting
