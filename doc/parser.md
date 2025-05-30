# Lua Parsing

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
<funcbody>     ::= "(" <parlist> ")" <block> "end"
<parlist>      ::= <namelist> "," "..." | <namelist> | "..."
<namelist>     ::= <name> ("," <namelist>)*
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
<explist>      ::= <expr> ("," <expr>)*
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

## Luau-like EBNF
While trying to write a luau parser, I found issues with their grammar that make
parsing harder than it needs to be. Like for instance

```
<type>         ::= <simpletype> "?"? ("|" <simpletype> "?"?)* | <simpletype> ("&" <simpletype>)*
<simpletype>   ::= /* other types */ | <fntype> | "(" <type> ")"
<fntype>       ::= ("<" <gtypelist> ">") "(" <boundtypelist> ")" "->" <rettype>
```

This leaves parsing the following unclear:
```
type a = (x: number) -> number
type b = (number | string)
```
Once you reach a `(` you cannot determine which item you are parsing, you can then
peek an identifier, but still you have not identified what you're parsing. It requires
2 look-aheads to identify what the statement is. To make parsing easier I have added
a `function` prefix to function types, similar to go.

```ebnf
/* an extended version of lua where types are optional */
<block>        ::= <statlist>?
<statlist>     ::= <stat> ";"? <statlist>*
<stat>         ::= ";" | <ifstat> | <whilestat> | <dostat> | <forstat> | <repeatstat> | <funcstat> | <localstat> | <label> | <retstat> | "break" | "goto" <name> | <fncallstat> | <assignment> | <typedef>
<ifstat>       ::= "if" <expr> "then" <block> <elseifstat>* <elsestat>? "end"
<elseifstat>   ::= "elseif" <expr> "then" <block>
<elsestat>     ::= "else" <block>
<whilestat>    ::= "while" <expr> "do" <block> "end"
<forstat>      ::= "for" (<fornum> | <forlist>) "end"
<fornum>       ::= <name> (":" <type>)? "=" <expr> "," <expr> ("," <expr>)? "do"
<forlist>      ::= <bindinglist> "in" <explist> "do"
<repeatstat>   ::= "repeat" <block> "until" <expr>
<funcstat>     ::= "function" <funcname> <funcbody>
<funcname>     ::= <name> ("." <name>)* (":" <name>)?
<funcbody>     ::= ("<" <gtypelist> ">")? "(" <parlist> ")" (":" <rettype>) <block> "end"
<parlist>      ::= <bindinglist> | <bindinglist> "," "..." (":" (<name> "..." | <type>)?)  | "..." (":" (<name> "..." | <type>)?
<bindinglist>  ::= <name> (":" <type>)? ("," <name> (":" <type>)?)*
<localstat>    ::= "local" (<localfunc> | <localassign>)
<localfunc>    ::= "function" <name> <funcbody>
<localassign>  ::= <name> <attrib>? ("," <name> <attrib>? )* ("=" <explist>)?
<attrib>       ::= ("<" ("const" | "close") ">") | (":" <type>)
<label>        ::= "::" <name> "::"
<retstat>      ::= "return" <explist>? ";"?
<dostat>       ::= "do" <block> "end"
<fncallstat>   ::= <suffixedexp> <funcargs>
<funcargs>     ::= "(" <explist>? ")" | <constructor> | <string>
<cmpndassign>  ::= <suffixedexp> <compoundop> <expr>
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
<compoundop>   ::= '+=' | '-=' | '*=' | '/=' | '//=' | '%=' | '^=' | '..='
<unop>         ::= "-" | "~" | "#" | "not"
<binop>        ::= "or" | "and" | "==" | "<" | "<=" | ">" | ">=" | "~=" | "||" | "~" | "&" | "<<" | ">>" | ".." | "+" | "-" | "*" | "%" | "/" | "//" | "^"
<name>         ::= ([a-Z] | "_") ( [a-Z] | [0-9] | "_" )*
<string>       ::= <quote> <chars>* <quote> | "[[" <chars>* "]]"
<quote>        ::= "'" | "\""
<number>       ::= "-"? [0-9]+ ("." [0-9]+ ("e" "-"? [0-9]+))? | "-"? "0x" ([0-9] | [A-F] | [a-f])+
<chars>        ::= [a-Z] | [0-9] | " " | "\n"

/* type defs */
<typedef>      ::= "export"? "type" <name> ("<" <gtypelistwithdefaults> ">")? "=" <type> | "export"? "type" "function" <name> <funcbody>
<type>         ::= <simpletype> "?"? ("|" <simpletype> "?"?)* | <simpletype> ("&" <simpletype>)*
<simpletype>   ::= "nil" | <string> | "true" | "false" | <name> ("." <name>)* ("<" <typeparamlist> ">") | "typeof" "(" <expr> ")" | <tbltype> | <fntype> | "(" <type> ")"
<tbltype>      ::= "{" (<type> | <proplist>)* "}"
<prop>         ::= ("read" | "write")? (<name> ":" <type> | "[" <type> "]" ":" <type>)
<proplist>     ::= <prop> (<sep> <proplist>)*
<fntype>       ::= "function" ("<" <gtypelist> ">") "(" <boundtypelist> ")" "->" <rettype>
<rettype>      ::= <type> | <typepack> | <name> "..." | "..." <type>
<boundtype>    ::= <type> | <name> ":" <type> | <name> "..." | "..." <type>
<boundtypelist> ::= <boundtype> ("," <boundtype>)*
<typeparam>    ::= <type> | <typepack> | "..." <type> | <name> "..."
<typeparamlist> ::= <typeparam> | <typeparam> "," <typeparamlist>
<typepack>     ::= "(" <typelist>* ")"
<typelist>     ::= <type> | <type> "," <typelist> | "..." <type>
<gtypelist>    ::= <name> | "..." | <name> "," <gtypelist>
<gtypelistwithdefaults> ::= <name> ("=" <type>)? ("," <gtypelistwithdefaults>)* | <gtypepackparameterwithdefault> ("," <gtypepackparameterwithdefault>)*
<gtypepackparameterwithdefault> ::= <name> "..." "=" (<typepack> | "..." <type> | <name> "...")
```
