---
layout: post
title: Reverse Engineering Lua
tags: learnings
---

I have always been interested in language development. I have written many of lisps,
and always found it fun however I have never really wanted to write lisp so I
generally didn't go far with these implementations. So at some point I felt frustrated
that I could not create a fully featured language. There are so many things that
are part of language development that I could not even learn because my language
never got far enough to need them. Things like gc, closures, fully fledged vm, and
type systems. So I wanted to go over how and why I started writing a lua implementation
for learning.

### How to get started.
To start, I chose lua because not only have I written a lot of lua but it also has
a very small language definition. I also wanted to learn established ideas and
patterns rather than invent them so that is why I chose not to create my own language.
So I have a language developed and designed by the smart people at Pontifical Catholic
University in Rio, with a small surface area that should be pretty quick to at least
parse quickly. Also I want to mention that I decided to write this in Go simply
because it is my strongest language. I am aware of the silliness of developing a
programming language in a language with full GC but I have no intention of making
a language that everyone uses so if it is slower but I can iterate faster and learn
faster, then that is an alright trade off.

So before I start writing I wanted to see some prior art that already exists and
found:

- [Lua Sourcecode](https://github.com/lua/lua)
- [LuaJIT](https://github.com/LuaJIT/LuaJIT)
- [yuin/gopher-lua](https://github.com/yuin/gopher-lua)
- [Shopify/go-lua](https://github.com/Shopify/go-lua)

- How I figure out intricacies.
  - [Lua bytecode explorer](https://www.luac.nl/)
  - Using Lua and LuaJIT REPL to discover behaviour
  - The manual can sometimes lie [Lua 5.4 Reference Manual](https://www.lua.org/manual/5.4/)

