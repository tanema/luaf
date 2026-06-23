## Fixes
- [ ] Re-evaluate closing locals in scopes after refactor If block close upvals broken because of simpler jump
  - while
  - for
  - repeat
- [ ] EXARG in NEWTABLE
- [ ] SETTABLE allow RCK (constant in c param)
- [ ] String lib
  - [x] string.find
  - [ ] string patterns
  - [ ] string.pack
- [ ] Parsing huge numbers (There are numbers that just overflow int64 but lua can parse them somehow)
- [ ] Finish integrating the rest of the lua tests.
- [ ] Config to disable libs like io to disable file access

## Optimizations
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
- [ ] Pigeonhole optimizations on bytecode
- [ ] constant Upvalue replacement so just value is passed and upvalue does not need to remain opened.
- [x] LOADTRUE
- [x] LFALSESKIP
- [x] LOADFALSE
- [x] const folding
- [x] const folding in parsing should just fail quietly. For instance if there is divide by 0 it should not fail until runtime. This is because maybe that branch of logic is never executed.

# Features
- [ ] Error message localization
- [ ] Enable better supportive error messages.

## Type system
- [ ] definitions
- [ ] checking
- [ ] casting
