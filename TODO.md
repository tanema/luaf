## Fixes
- [x] upvals still wonky. test totals are wrong
  - Problem was actually with SETTABLE on nested tables with compound expression keys
- [x] hex numbers (`0xFF03`) cause parsing issues directly after them.
  - was actually an overflow which was not being exposed. needed to allow lexing
    errors to be propagated even when peeking.
- [x] template library broken (from parsing not lib) try running skipped test.
  - problem was that if local assignment had no values, the locals were not added to the scope
- [ ] Call traces are ALL MESSED UP
- [ ] REPL is just trash, it just doesnt really work but worse, it looks like it does.
- [ ] vm.call is not working as expected. In sort it is currently overwriting the table value.
- [ ] many string issues with unicode escapes like \0 or \x00 that are not being parsed
      by go properly because we are parsing them ourselves.
      https://pkg.go.dev/unicode/utf8#DecodeRuneInString

## TODO
- [ ] Finish integrating the rest of the lua tests.

## Type system
- [ ] definitions
- [ ] checking
- [ ] casting
