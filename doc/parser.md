# Lua Parsing

## Language BNF Definition
```bnf
<block>        ::= <statlist>
<statlist>     ::= { <stat> [';'] }
<stat>         ::= ';' | <ifstat> | <whilestat> | <dostat> | <forstat> | <repeatstat> | <funcstat> | <localstat> | <label> | <retstat> | 'break' | 'goto' NAME | <fncallstat> | <assignment>
<ifstat>       ::= 'if' <expr> 'then' <block> {'elseif' <expr> 'then' <block>} ['else' <block>] 'end'
<whilestat>    ::= 'while' <expr> 'do' <block> 'end'
<forstat>      ::= 'for' (fornum | forlist) 'end'
<fornum>       ::= NAME = <expr>, <expr>[, <expr>] 'do'
<forlist>      ::= NAME {,NAME} 'in' <explist> 'do'
<repeatstat>   ::= 'repeat' <block> 'until' <expr>
<funcstat>     ::= 'function' <funcname> <funcbody>
<funcname>     ::= NAME {<fieldsel>} [':' NAME]
<fieldsel>     ::= ['.' | ':'] NAME
<funcbody>     ::= <parlist> <block> 'end'
<parlist>      ::= '(' [ {NAME ','} (NAME | '...') ] ')'
<localstat>    ::= 'local' [<localfunc> | <localassign>]
<localfunc>    ::= 'function' NAME <funcbody>
<localassign>  ::= NAME <attrib> { ',' NAME <attrib> } ['=' <explist>]
<attrib>       ::= ['<' ('const' | 'close') '>']
<label>        ::= '::' NAME '::'
<retstat>      ::= 'return' [<explist>] [';']
<dostat>       ::= 'do' <block> 'end'
<fncallstat>   ::= <suffixedexp> <funcargs>
<funcargs>     ::= '(' [ <explist> ] ')' | <constructor> | STRING
<assignment>   ::= <suffixedexp> { ',' <suffixedexp> } '=' <explist>
<explist>      ::= <expr> { ',' <expr> }
<expr>         ::= (<simpleexp> | <unop> <expr>) { <binop> <expr> }
<simpleexp>    ::= FLOAT | INTEGER | STRING | 'nil' | 'true' | 'false' | '...' | <constructor> | 'function' <funcbody> | <suffixedexp>
<sep>          ::= ',' | ';'
<constructor>  ::= '{' [ <field> { <sep> <field> } [<sep>] ] '}'
<field>        ::= NAME = <expr> | '[' <expr> ']' = <expr> | <expr>
<suffixedexp>  ::= <primaryexp> { '.' NAME | '[' <expr> ']' | ':' NAME <funcargs> | <funcargs> }
<primaryexp>   ::= NAME | '(' <expr> ')'
<unop>         ::= '-' | '~' | '#' | 'not'
<binop>        ::= 'or' | 'and' | '==' | '<' | '<=' | '>' | '>=' | '~=' | '||' | '~' | '&' | '<<' | '>>' | '..' | '+' | '-' | '*' | '%' | '/' | '//' | '^'
```
