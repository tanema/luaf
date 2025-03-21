# Lua Parsing

## Language EBNF Definition
An almost correct EBNF definition of lua. I don't think it would actually compile
but it is good for reference.

```ebnf
<block>        ::= <statlist>?
<statlist>     ::= <stat> ";"? <statlist>*
<stat>         ::= ";" | <ifstat> | <whilestat> | <dostat> | <forstat> | <repeatstat> | <funcstat> | <localstat> | <label> | <retstat> | "break" | "goto" <name> | <fncallstat> | <assignment>
<ifstat>       ::= "if" <expr> "then" <block> <elseifstat>* <elsestat>? "end"
<elseifstat>   ::= "elseif" <expr> "then" <block>
<elsestat>     ::= "else" <block>
<whilestat>    ::= "while" <expr> "do" <block> "end"
<forstat>      ::= "for" (<fornum> | <forlist>) "end"
<fornum>       ::= <name> "=" <expr> "," <expr> ("," <expr>)? "do"
<forlist>      ::= <name> ( "," <name> )? "in" <explist> "do"
<repeatstat>   ::= "repeat" <block> "until" <expr>
<funcstat>     ::= "function" <funcname> <funcbody>
<funcname>     ::= <name> ("." <name>)* (":" <name>)?
<funcbody>     ::= <parlist> <block> "end"
<parlist>      ::= "(" (<namelist> "," "..." | <namelist> | "...") ")"
<namelist>     ::= E | <name> | <name> "," <namelist>
<localstat>    ::= "local" (<localfunc> | <localassign>)
<localfunc>    ::= "function" <name> <funcbody>
<localassign>  ::= <name> <attrib>? ("," <name> <attrib>? )* ("=" <explist>)?
<attrib>       ::= "<" ("const" | "close") ">"
<label>        ::= "::" <name> "::"
<retstat>      ::= "return" <explist>? ";"?
<dostat>       ::= "do" <block> "end"
<fncallstat>   ::= <suffixedexp> <funcargs>
<funcargs>     ::= "(" <explist>? ")" | <constructor> | <string>
<assignment>   ::= <suffixedexp> ("," <suffixedexp> )* "=" <explist>
<explist>      ::= <expr> | <expr> "," <explist>
<expr>         ::= (<simpleexp> | <unop> <expr>) (<binop> <expr>)*
<simpleexp>    ::= <number> | <string> | "nil" | "true" | "false" | "..." | <constructor> | "function" <funcbody> | <suffixedexp>
<sep>          ::= "," | ";"
<constructor>  ::= "{" <fieldlist>? "}"
<fieldlist>    ::= <field> <sep>? | <field> <sep> <fieldlist>
<field>        ::= <name> "=" <expr> | "[" <expr> "]" "=" <expr> | <expr>
<suffixedexp>  ::= <primaryexp> ( "." <name> | "[" <expr> "]" | ":" <name> <funcargs> | <funcargs> )?
<primaryexp>   ::= <name> | "(" <expr> ")"
<unop>         ::= "-" | "~" | "#" | "not"
<binop>        ::= "or" | "and" | "==" | "<" | "<=" | ">" | ">=" | "~=" | "||" | "~" | "&" | "<<" | ">>" | ".." | "+" | "-" | "*" | "%" | "/" | "//" | "^"
<name>         ::= ([a-Z] | "_") ( [a-Z] | [0-9] | "_" )*
<string>       ::= <quote> <chars>* <quote> | "[[" <chars>* "]]"
<quote>        ::= "'" | "\""
<number>       ::= "-"? [0-9]+ ("." [0-9]+ ("e" "-"? [0-9]+))? | "-"? "0x" ([0-9] | [A-F] | [a-f])+
<chars>        ::= [a-Z] | [0-9] | " " | "\n"  /* ....... more obviously */
```

## AST vs Immediate Code Generation
The parser in this project uses a mixture or immediate bytecode generation and AST
parsing. The parser will generate an AST for `<expr>` in statements but then the
actual root statements will directly generate the bytecode. The original luac
interpreter would generate code immediately from every statement however, I found
that an AST helps cost folding a lot easier. The tree can be built up, then reduced,
then discharged to the VM. So the parser in this repo has a combination of methods
on how code is generated.
