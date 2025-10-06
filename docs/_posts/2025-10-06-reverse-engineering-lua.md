---
layout: post
title: Reverse Engineering A Language
tags: learnings
comments: true
---

I have always been interested in language development. I have written many of lisps,
and always found it fun however I have never really wanted to write lisp so I
generally didn't go far with these implementations. So at some point I felt frustrated
that I could not create a fully featured language. There are so many things that
are part of language development that I could not even learn because my language
never got far enough to need them. Things like garbage collection, closures, fully
fledged vm, and type systems. So I wanted to go over how and why I started writing
a lua implementation for learning.

### Disclaimer
I am not a language design/development expert. This is very much a hobby for me
and I am trying to help others start the hobby. Sometime when approaching topics
like this in software development, a lot of material starts teaching small aspects
before ever getting to anything functional and I want to share how I jump into
a language feet first.

### Why Lua
To start, I chose lua because not only have I written a lot of lua but it also has
a very small language definition. I also wanted to learn established ideas and
patterns rather than invent them. Lastly I wanted to mention that I decided to
write this in Go simply because it is my strongest language. I am aware of the
silliness of developing a programming language in a language with full GC but I
have no intention of making a language that everyone uses so if it is slower but
I can iterate faster and learn faster, then that is an alright trade off.

### How to get started.
So before I start writing I wanted to see some prior art that already exists and
found. These would be good to see how other implemented but I would stay away from
copying as that would not help me learn:

- [Lua Sourcecode](https://github.com/lua/lua)
- [LuaJIT](https://github.com/LuaJIT/LuaJIT)
- [yuin/gopher-lua](https://github.com/yuin/gopher-lua)
- [Shopify/go-lua](https://github.com/Shopify/go-lua)

After I had some examples, I needed tools to dig deeper into the standard lua
implementation:
  - [Lua bytecode explorer](https://www.luac.nl/)
  - [Incomplete Lua bytecode reference](https://the-ravi-programming-language.readthedocs.io/en/latest/lua_bytecode_reference.html)
  - Using Lua and LuaJIT REPL to discover behaviour
  - [Lua 5.4 Reference Manual](https://www.lua.org/manual/5.4/) however comparing the manual with the repl, the manual is not always correct.

### Start Writing
Okay after I had these references and ways to compare my implementation to the
standard implementation, I had to start implementing. I should address that there
are many tools to generate a parser from a language definition. I however did not
want to use one of these because I simply wanted to write it myself, but also
because it adds an extra step. The generated parser would parse an abstract syntax
tree and then I would have to walk that to generate my bytecode, whereas if I wrote
the parser myself I could generate bytecode directly from the source. So first I
need tokens to use. This is pretty straight forward and can be implemented pretty
easy. In pseudo code this can look like this:

```golang
func NextToken(reader io.Reader) {
  for ch := reader.NextChr(); ch != EOS {
    switch ch {
    case '(':
      return tokenOpenParen
    case ')':
      return tokenCloseParen
    case '=':
      if reader.Peek() == '=' {
        reader.Next()
        return tokenCompareEq
      } else {
        return tokenAssign
      }
    // ..... many more cases
    default:
      identifier := string([]byte{ch})
      for ch := reader.NextChr(); isTextCharacter(ch) {
        identifier = append(identifier, ch)
      }
      return tokenIdentifier{value: identifier}
    }
  }
}
```

See the [lexer in luaf for full example](https://github.com/tanema/luaf/blob/064aec3bc33a2d7066d454356b13e82f92c04bbe/src/parse/lexer.go)

#### Parsing

I have found that having a string BNF definition of the language has be benificial
to follow while implementing and we can go over this as well. So we have a snippet
of a lua EBNF definition below. (To see the full definition I work with see
[Parsing](/parser.html#lua-ebnf))

```ebnf
<block>        ::= <statlist>?
<statlist>     ::= <stat> ";"? <statlist>*
<stat>         ::= ";" | <ifstat> | <whilestat> | <dostat> | <forstat> | <repeatstat> | <funcstat> | <localstat> | <label> | <retstat> | "break" | "goto" <name> | <fncallstat> | <assignment>
<localstat>    ::= "local" (<localfunc> | <localassign>)
<attrib>       ::= "<" ("const" | "close") ">"
<explist>      ::= <expr> ("," <expr>)*
<expr>         ::= (<simpleexp> | <unop> <expr>) (<binop> <expr>)*
<simpleexp>    ::= <number> | <string> | "nil" | "true" | "false" | "..." | <constructor> | "function" <funcbody> | <suffixedexp>
// Much more definition
```

We can then start to implement this directly like the following:

```golang
func block() {
  for {
    stat()
  }
}

func stat() {
  tk := parser.NextToken()
  swich tk {
  case tokenLocal:
    return localstat()
  //...
  default:
    return errors.New("unexpected token")
  }
}

func localstat() {
  tk := parser.NextToken()
  if tk == tokenFunction {
    return functionStat()
  }

  name := tk
  if parser.NextToken() != tokenAssign {
    return errors.New("unexpected token")
  }
  value := expression()
}
```

Add some print statements and you can start out parsing lua code and do absolutely
nothing with it, but it will still be impressive if you can chomp through lua code
with all expectations met! You could already build a code linter with what you have
made now! However I wanted to implement a fully functional language so we need a
runtime.

## Intermediate representation
What the parser outputs is often an [intermediate representation](https://en.wikipedia.org/wiki/Intermediate_representation)
or IR. There are some parsers that output an [Abstract Syntax Tree](https://en.wikipedia.org/wiki/Abstract_syntax_tree),
but then usually a post-processor converts that into IR after optimizing and analyzing.

This is largely because of performance. Instead of constantly navigating tree like
data structures which would be very computationally heavy, we can simply run instruction
after instruction. Branching statements such as if statments just increment the
program counter (pc) to skip instructions if the if statment resolves to false.

What lua does and others like Java, the IR is a bytecode. This means that a single
instruction would be a large number. Lua bytecode instructions are 32-bits in size.
All instructions have an opcode in the first 6 bits. Instructions can have the following formats:
```
| Name  | Other Params                  | Param A | Opcode ID  |
|-------|-------------------------------|---------|------------|
| iABC  | CK: 1 | C: u8 | BK: 1 | B: u8 | 8 bits  | 6 bits     |
| iABx  |            B: unsigned int 16 | 8 bits  | 6 bits     |
| iAsBx |            B: signed int 16   | 8 bits  | 6 bits     |
```
BK | CK = 0 or 1 indicate if the params B,C refer to a stack value or a constant
value. Opcode:u6 means there are 64 possible opcodes.

To see more about this your can see the [Virtual Machine Docs](/pages/virtualmachine.html)
and the [Bytecode Package](https://github.com/tanema/luaf/blob/main/src/bytecode/bytecode.go)

## Runtime
Now that we have an IR bytecode, evaluation can be as simple and iterating through
the bytecode and executing each statement. With this you can see the power of the
IR.

```go
func eval(instructions []int32) error {
    pc := 0
    for {
        if int64(len(instructions)) <= pc {
            return nil, nil
        }

        instruction := f.fn.ByteCodes[pc]

        switch GetOp(instruction) {
        case MOVE:
          // ...
        case LOADK:
          // ...
        case LOADBOOL:
          // ...
        case LOADI:
          // ...
        case LOADF:
          // ...
        // CONTINUED
      }
      pc++
    }
    return nil
}
```

To see lua do this you can see it [implemented here with a jump table](https://github.com/lua/lua/blob/master/lvm.c#L1185)
and you can see [what I have done here](https://github.com/tanema/luaf/blob/064aec3bc33a2d7066d454356b13e82f92c04bbe/src/runtime/vm.go)


Of course this is nowhere close to an entire outline of how to implement a language.
We have not addressed garbage collection, closures, or a myriad of other topics
involved in building a language. But I wanted to give a deep dive into how I got
started digging into these things. Not by just starting from scratch but rather
copying existing languages and behaviour.
