<div align="center">
  <img src="https://github.com/tanema/luaf/raw/main/docs/luaf.svg?sanitize=true" width=300/>
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
> `luaf` should be fully compatible with the lua APIs that are default in lua,
> however it will not provide the same API as the C API. It will also be able to
> precompile and run precompiled code however that precompiled code is not compatible
> with `lua`. `luac` will not be able to run code from `luaf` and vise versa.
> Since the point of this implementation is more for using lua than it's use in Go
> there is less of an emphasis on a go API though a simple API exists.

## Getting Started
- `make install` install luaf
- `make test` run tests
- `make help` for more commands to develop with
