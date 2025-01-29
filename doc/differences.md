List of features that are unsupported
======================================
This is a place that I will try to keep track of a list of functionalities that
I have intentionally decided to not support.

- Byte Order Marks
- library flag `-l`
- `-` flag, just use a pipe?
- debug library
  - I find the lua debug library quite bad so I am trying to provide more `binding.pry`
    type behaviour here and not have to understand the interpreter to find local values
