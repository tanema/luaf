---
layout: post
title: Things I Realized While Implementing a Language.
tags: learnings
comments: true
---

I just wanted to list a couple things that may not be apparent if you went into
building a programming language. These are things that made me 🤔.

### The VM does't care about variable names.
Humans need variable names, machines don't. The VM and especially assembly code
doesn't need variable names, they are just positional values operated on at specific
locations in the stack or specific registers. A lot of naive implementations of
languages will start off keeping track of variable names and this will overcomplicate
their design. Going further, the names are kept around but only for humans to debug.

### Garbage collection cares about data that escapes it's declaration scope
We so often talk about garbage collection because it can be a point of slow down
however that talk usually lacks a nuance about what is collected and when. It
wasn't until writing my own GC that I realized, if a value is allocated on the stack
but never leaves it's scope, it is simply removed from the stack at the end of the
scope. Not a big deal. GC only cares if that value was allocated and then is
captured in another scope like a closure. Now it has to keep track of that value
and how much it is used so that it can still be available when needed and discarded
when no longer needed.

### You're really writing two languages
When you write a new higher level language you also have to write your intermediate
representation bytecode which is another simpler language. This is often referred to
as the language's frontend and backend. They can largely be unit tested separately,
because of this. The frontend is just a parser that converts high language to a
low language API commands. The backend responds to each command and updates its
stack, gc, etc based on the action.

### Const folding
Const folding is a really cool feature of languages. There are many times where
you will define config varaibles like this:

```lua
local speed = 1000
local accel = speed * 2
```

But by the time the variable `accel` makes it to the backend, it has already been
evaluated to `2000` as it doesn't make sense to run extra code if it can be consistently
reasoned that the value starts as such. This is called const folding. It looks for
const values and will try to optimize any evaluation that we can ensure wont change
during evaluation.

### To be continued
I will probably add to this in the future but that is it for now.
