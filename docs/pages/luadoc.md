# LuaDoc
I want to try documentation parsing and output so I am using emmyLua as reference for 
a community format so that I can validate my tooling easy.

[reference](https://github.com/EmmyLuaLs/emmylua-analyzer-rust/tree/main/docs/emmylua_doc/annotations_EN)

### Best Practices
- Types First: Define types before using them
- Progressive Enhancement: Start with basic annotations, gradually add more complex ones
- Consistency: Maintain consistent annotation style throughout the project
- Documentation: Provide detailed descriptions for complex types and functions
- Test Validation: Use type checking tools to validate annotation correctness

### Design
Documentation comments start with 3 `-` instead of 2.

### Module Wide Tags
| Tag         | Scope  | Format                                                         | Description                                                                                                    |
|-------------|--------|----------------------------------------------------------------|----------------------------------------------------------------------------------------------------------------|
| author      | Module | `---@author <text>`                                            | An author of the module or file.                                                                               |
| copyright   | Module | `---@copyright <text`                                          | The copyright notice of the module or file. LuaDoc adds a © sign between the label (Copyright).                |
| license     | Module | `---@license <text>`                                           | The Software Copyright license for the module or file.                                                         |
| meta        | Module | `---@meta`                                                     | Marks a file as "meta", meaning it is used for definitions and not for its functional Lua code.                |
| release     | Module | `---@release <text>`                                           | Free format string to describe the module or file release.                                                     |
| module      | Module | `---@module <name>`                                            | A way to split a file into multiple modules.                                                                   |
| alias       | Module | `---@alias <alias> <type_expression>`                          | Use it to name a type definition `@alias Handler fun(type: string, data: any):void`                            |
|             |        | `---@alias <alias><<generic_parameter_list>> <type_expression>`|                                                                                                                |
|             |        | `---@alias IOEventEnum "'onClosed'" \| "'onData'"`             |                                                                                                                |
|             |        | `---@alias <name>\\n---\| '<value>' [# description]`           |                                                                                                                |
|             |        | `---@alias Dictionary<K, V> table<K, V>`                       |                                                                                                                |
| enum        | Module | `---@enum [(key)] <name>`                                      | Mark a Lua table values (or keys) as an enum to use similar to alias.                                          |
| generic     | Module | `---@generic <name>[:<constraint>][, <name>[:<constraint>]...]`| Defines a generic type name used in other docs.                                                                |
| type        | Module | `---@type [name]`                                              | Use annotation to specify the type of the target variable                                                      |
|             |        | `---@type <TYPE>[]`                                            | to specify an array type                                                                                       |
|             |        | `---@type table<KEY_TYPE, VALUE_TYPE>`                         | to specify that a variable’s type is a table (a.k.a. dictionary, map) type                                     |
|             |        | `---@type fun(param:MY_TYPE):RETURN_TYPE`                      | to specify that a variable’s type is a function type                                                           |
| deprecated  | All    | `---@deprecated [explanation]`                                 | declare a module, function, variable or any other item usage is deprecated                                     |
| diagnostic  | System | `---@diagnostic <action>: <diag_name>[, <diag_name>...]`       | Controls diagnostics for errors, warnings, information and hints. Actions are disable-next-line,disable,enable |
| todo        | System | `---@todo [reason]`                                            |                                                                                                                |
| fixme       | System | `---@fixme [reason]`                                           |                                                                                                                |
| warning     | System | `---warning [reason]`                                          |                                                                                                                |
| class       | Var    | `---@class <name>[: <parent>[, <parent>...]]`                  | If LuaDoc cannot infer the type of documentation, the programmer can specify it explicitly.                    |
|             |        | `---@class (exact) <name>[: <parent>...]`                      | Exact class definition (prohibits dynamic field addition)                                                      |
|             |        | `---@class (partial) <name>`                                   | Partial class definition (allows extending existing classes)                                                   |
|             |        | `---@class <GEN_NAME><T1, T2, ...>[: <parent>...]`             | Class definition with generics.                                                                                |
|             |        | `---@class Container<T>`                                       |                                                                                                                |
| name        | All    | `---@name <word>`                                              | The name of the function or table definition. This is usually inferred.                                        |
| description | All    | `---@description <text>`                                       | The description of the function or table. This is usually inferred automatically.                              |
| field       | Table  | `---@field [<scope>] <name>[?] <type> [description]`           | Describe a table field definition. Scope is for class fields                                                   |
|             |        | `---@field [<scope>] [<key_type>] <value_type> [description]`  | scope = public, private, protected, package                                                                    |
| operator    | Table  | `---@operator <operation>[(input_type)]:<resulting_type>`      | Provides type declarations for an operator metamethod.                                                         |
|             |        | `---@operator add(string): string`                             | For metamethod overload __add                                                                                  |
| see         | All    | `---@see <text>`                                               | Refers to other descriptions of functions or tables.                                                           |
| source      | All    | `---@source <path>`                                            | Provide a reference to some source code which lives in another file.                                           |
| usage       | Fn,Var | `---@usage <text>`                                             | Describe the usage of the function or variable.                                                                |
| package     | Fn     | `---@package`                                                  | Mark a function as private to the file it is defined in.                                                       |
| private     | Fn     | `---@private`                                                  | Mark a function as private to a class                                                                          |
| protected   | Fn     | `---@protected`                                                | Mark a function as protected within a class                                                                    |
| language    | Var    | `---@language <name>`                                          | Use to inject syntax highlight to a piece of text `@language json`                                             |
| overload    | Fn     | `---@overload fun(<parameters>): <return_types>`               | Specifies multiple signatures for the same function because of default values.                                 |
| param       | Fn     | `---@param <parameter_name>[?] <type_expression> [description]`| Describe function parameters. It requires the name of the parameter and its description.                       |
| return      | Fn     | `---@return <type> [<name> [comment]]`                         | Describe a returning value of the function.                                                                    |
| nodiscard   | Fn     | `---@nodiscard [reason]`                                       | Mark a function as having return values that cannot be ignored/discarded.                                      |
| version     | Fn     | `---@version <version>`                                        | Marks if a function or class is exclusive to specific Lua versions: 5.1, 5.2, 5.3, 5.4, JIT                    |
| async       | Fn     | `---@async`                                                    | Mark a function as being asynchronous                                                                          |
