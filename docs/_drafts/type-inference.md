---
layout: post
title: Adding a Type System to Lua
tags: design
---

- Why add typing.
- How type systems could speed up execution in vm.
- How tables with metamethods complicate all type inference.
- Duck Typing, Should a map[string]any, if it has the correct field, still match a struct?
- type inference or left side of assignment
- How const folding helps in inference.
- Type checking should be
  any -> nil, string, bool, int, float, number, struct
  nil -> nil
  string -> string
  bool -> bool
  int -> int
  float -> float
  number -> int, float
  struct -> struct, map[string]any, freeform
