## Fixes
- [x] upvals still wonky. test totals are wrong
  - Problem was actually with SETTABLE on nested tables with compound expression keys
- [x] hex numbers (`0xFF03`) cause parsing issues directly after them.
  - was actually an overflow which was not being exposed. needed to allow lexing
    errors to be propagated even when peeking.
- [x] template library broken (from parsing not lib) try running skipped test.
  - problem was that if local assignment had no values, the locals were not added to the scope

## TODO
- [ ] Finish integrating the rest of the lua tests.

## Type system
- [ ] definitions
- [ ] checking
- [ ] casting
