---
layout: post
title: Reverse Engineering A Language
tags: learnings
---

I have always been interested in language development. I have written many of lisps,
and always found it fun however I have never really wanted to write lisp so I
generally didn't go far with these implementations. So at some point I felt frustrated
that I could not create a fully featured language. There are so many things that
are part of language development that I could not even learn because my language
never got far enough to need them. Things like garbage collection, closures, fully
fledged vm, and type systems. So I wanted to go over how and why I started writing
a lua implementation for learning.

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

## Runtime
bytecode
vm implementation

## Cont.
const folding
std lib

