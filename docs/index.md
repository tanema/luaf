---
layout: default
---

# Luaf

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

luaf is an experimental implementation of lua 5.4 for learning purposes and luafs 🤠.
It aims to be fully feature compatible with lua 5.4 as well as additions that also
aims to make lua a more feature complete language.

## Documentation
I have some living documents that I update on what I learn about implementing lua
and other programming languages.

- [References](/references.md)
- [Parsing](/parser.md)
- [Runtime, Bytecode & Virtual Machine](/virtualmachine.md)
- [Lua Metamethods](/metamethods.md)
- [Upvalues](/upvalues.md)
- [JIT learnings](/jit.md)

## Dev Log

<ul>
  {% for post in site.posts %}
    <li>
      <a href="{{ post.url }}">{{ post.title }}</a>
    </li>
  {% endfor %}
</ul>
