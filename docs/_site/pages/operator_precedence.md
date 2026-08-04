## Operator Precedence Table

These are the precedences set by the luajit proposal, marked here to check along
with my own implemented precedence table.

| Operator / Construct                 | Arity   | Associativity |
|--------------------------------------|---------|---------------|
| `( )`                                | Unary   | -             |
| `->`                                 | Binary  | Right         |
| `.` `:` `[]` `f()` `?.`              | Binary  | Left          |
| `^`                                  | Binary  | Right         |
| `not` `!` `-` `~` `#`                | Unary   | -             |
| `*` `/` `//` `%`                     | Binary  | Left          |
| `+` `-`                              | Binary  | Left          |
| `..`                                 | Binary  | Right         |
| `<<` `>>` `~>>`                      | Binary  | Left          |
| `&`                                  | Binary  | Left          |
| `~`                                  | Binary  | Left          |
| `|`                                  | Binary  | Left          |
| `==` `~=` `!=` `<` `>` `<=` `>=`     | Binary  | Left          |
| `and` `&&`                           | Binary  | Left          |
| `or` `||` `??`                       | Binary  | Left          |
| `?:`                                 | Ternary | Right         |
| `=` `compound=`                      | Binary  | -             |

## Current Implemented Precedence Table

| Operator / Construct          | Arity   | Associativity |
|-------------------------------|---------|---------------|
| `^`                           | Binary  | Right         |
| `*` `/` `//` `%`              | Binary  | Left          |
| `+` `-`                       | Binary  | Left          |
| `..`                          | Binary  | Right         |
| `<<` `>>`                     | Binary  | Left          |
| `&`                           | Binary  | Left          |
| `~`                           | Binary  | Left          |
| `|`                           | Binary  | Left          |
| `==` `~=` `<` `>` `<=` `>=`   | Binary  | Left          |
| `and`                         | Binary  | Left          |
| `or`                          | Binary  | Left          |
