# Contributing to LUAF

**Prerequisites**
- Language Development
  [Go >= 1.23](https://go.dev/doc/install)
- Docs Website
  - [Ruby >= 3.0](https://www.ruby-lang.org/en/documentation/installation/)
- Linting:
  - [Rust/Cargo](https://rust-lang.org/tools/install/)
    - stylua `cargo install stylua --features lua54`
  - [golangci-lint](https://golangci-lint.run/docs/welcome/install/#local-installation)
- GNU Make or equivalent

### Building and Running

Any action needed should be supported by the make file. Please run `make help`
to list any actions you can use.

You can run `make repl` to build and run the repl, or `make install` to install it
to your system.

When making changes make sure to run `make test` to ensure code quality.


### Project Layout

- `/cmd/luaf`: Main CLI entrypoint.
- `/docs`: The github pages docs website.
- `/test`: Lua tests, copied from the original lua source but updated to keep clean
- `/src`: Internal functionality of luaf
  - `/src/bytecode`: implements the bytecode interface
  - `/src/conf`: constants that are used across the application
  - `/src/lerror`: shared error format used across the application
  - `/src/lfile`: file library to support lua functionality
  - `/src/lstring`: string library to support lua functionality
  - `/src/parse`: lua parsing and lexing
  - `/src/runtime`: VM runtime for the bytecode produced by parse
  - `/src/types`: early experiments on typechecking

