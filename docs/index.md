---
layout: default
---

# Luaf

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
      <a href="{{ post.url }}">{{ post.title }}</a> <small><em>{{ post.tags | join: "</em> - <em>" }}</em></small>
    </li>
  {% endfor %}
</ul>
