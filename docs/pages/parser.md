---
layout: doc
title: Lua Parsing
---
## AST vs Immediate Code Generation

The parser in this project uses a mixture or immediate bytecode generation and AST
parsing. The parser will generate an AST for `<expr>` in statements but then the
actual root statements will directly generate the bytecode. The original luac
interpreter would generate code immediately from every statement however, I found
that an AST helps cost folding a lot easier. The tree can be built up, then reduced,
then discharged to the VM. So the parser in this repo has a combination of methods
on how code is generated.

## Lua EBNF
```ebnf
chunk            ::= block
block            ::= {stat} [retstat]
stat             ::=  ‘;’ | 
                      varlist ‘=’ explist | 
                      functioncall | 
                      label | 
                      break | 
                      goto Name | 
                      do block end | 
                      while exp do block end | 
                      repeat block until exp | 
                      if exp then block {elseif exp then block} [else block] end | 
                      for Name ‘=’ exp ‘,’ exp [‘,’ exp] do block end | 
                      for namelist in explist do block end | 
                      function funcname funcbody | 
                      local function Name funcbody | 
                      global function Name funcbody | 
                      local attnamelist [‘=’ explist] | 
                      global attnamelist | 
                      global [attrib] ‘*’ 
attnamelist      ::= [attrib] Name [attrib] {‘,’ Name [attrib]}
attrib           ::= ‘<’ Name ‘>’
retstat          ::= return [explist] [‘;’]
label            ::= ‘::’ Name ‘::’
funcname         ::= Name {‘.’ Name} [‘:’ Name]
varlist          ::= var {‘,’ var}
var              ::= Name | prefixexp ‘[’ exp ‘]’ | prefixexp ‘.’ Name 
namelist         ::= Name {‘,’ Name}
explist          ::= exp {‘,’ exp}
exp              ::= nil | false | true | Numeral | LiteralString | ‘...’ | functiondef | 
                     prefixexp | tableconstructor | exp binop exp | unop exp 
prefixexp        ::= var | functioncall | ‘(’ exp ‘)’
functioncall     ::= prefixexp args | prefixexp ‘:’ Name args 
args             ::= ‘(’ [explist] ‘)’ | tableconstructor | LiteralString 
functiondef      ::= function funcbody
funcbody         ::= ‘(’ [parlist] ‘)’ block end
parlist          ::= namelist [‘,’ varargparam] | varargparam
varargparam      ::= ‘...’ [Name]
tableconstructor ::= ‘{’ [fieldlist] ‘}’
fieldlist        ::= field {fieldsep field} [fieldsep]
field            ::= ‘[’ exp ‘]’ ‘=’ exp | Name ‘=’ exp | exp
fieldsep         ::= ‘,’ | ‘;’
binop            ::= ‘+’ | ‘-’ | ‘*’ | ‘/’ | ‘//’ | ‘^’ | ‘%’ | 
                     ‘&’ | ‘~’ | ‘|’ | ‘>>’ | ‘<<’ | ‘..’ | 
                     ‘<’ | ‘<=’ | ‘>’ | ‘>=’ | ‘==’ | ‘~=’ | 
                     and | or
unop             ::= ‘-’ | not | ‘#’ | ‘~’
```
