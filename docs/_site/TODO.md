## Fixes
- [x] upvals still wonky. test totals are wrong
  - Problem was actually with SETTABLE on nested tables with compound expression keys
- [x] hex numbers (`0xFF03`) cause parsing issues directly after them.
  - was actually an overflow which was not being exposed. needed to allow lexing
    errors to be propagated even when peeking.
- [x] template library broken (from parsing not lib) try running skipped test.
  - problem was that if local assignment had no values, the locals were not added to the scope
- [x] many string issues with unicode escapes like \0 or \x00 that are not being parsed
      by go properly because we are parsing them ourselves.
- [x] table.sort broken
- [x] table.unpack broken
  - [x] unpack as a last argument is not being expanded in table constructor
- [x] vm.call is not working as expected. In sort it is currently overwriting the table value.
- [x] `CLOSE` called after a return so it functionally does not work
- [x] Call traces are ALL MESSED UP
  - [x] line numbers pointed to call sight not method def
  - [x] Stack consistent around vm\.call
  - [x] pcall and xpcall are not cleaned up properly still, seen in failing tests.
- [x] table len not quite right when expanding last arg which means a bad top pointer.
- [x] redeclare locals is buggy? The value doesn't change?
- [ ] REPL is just trash, it just doesnt really work but worse, it looks like it does.
  - [x] REPL main now works better
  - [ ] debug.debug() does not work well right now
- [ ] Parsing huge numbers
- [ ] String lib
  - [ ] string patterns
  - [x] string.find

## TODO
- [ ] Optimizations
  - [x] LOADTRUE
  - [x] LFALSESKIP
  - [x] LOADFALSE
  - [ ] GETI
  - [ ] GETFIELD
  - [ ] SETI
  - [ ] SETFIELD
  - [ ] ADDI
  - [ ] ADDK
  - [ ] SUBK
  - [ ] MULK
  - [ ] MODK
  - [ ] POWK
  - [ ] DIVK
  - [ ] IDIVK
  - [ ] BANDK
  - [ ] BORK
  - [ ] BXORK
  - [ ] SHLI
  - [ ] SHRI
  - [ ] MMBIN    A B C      call C metamethod over R[A] and R[B]
  - [ ] MMBINI   A sB C k   call C metamethod over R[A] and sB
  - [ ] MMBINK   A B C k    call C metamethod over R[A] and K[B]
  - [ ] EQK
  - [ ] EQI
  - [ ] LTI
  - [ ] LEI
  - [ ] GTI
  - [ ] GEI
  - [ ] RETURN0
  - [ ] RETURN1
  - [ ] TESTSET
  - [x] If statement dead branch elimination.
    - [ ] Dead branch eliminations still pollute upindexes
  - [ ] Loop unrolling.
  - [x] const folding
  - [x] const folding in parsing should just fail quietly. For instance if there is
        divide by 0 it should not fail until runtime. This is because maybe that
        branch of logic is never executed.
  - [ ] Pigeonhole optimizations on bytecode
  - [ ] constant Upvalue replacement so just value is passed and upvalue does not need to remain opened.
- [ ] Finish integrating the rest of the lua tests.
- [ ] Config to disable libs like io to disable file access

## Type system
- [ ] definitions
- [ ] checking
- [ ] casting
