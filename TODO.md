## Fixes
- [x] upvals still wonky. test totals are wrong
  - Problem was actually with SETTABLE on nested tables with compound expression keys
- [ ] template library broken (from parsing not lib) try running skipped test.
- [ ] hex numbers (`0xFF03`) cause parsing issues directly after them.

## TODO
- [ ] Finish integrating the rest of the lua tests.

## Type system
- [ ] definitions
- [ ] checking
- [ ] casting
