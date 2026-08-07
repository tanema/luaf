---@disable stringarith,globals
---@enable requireOnly, readonlyenv, strict
---A simple module to test parsing docs
---@author Tim Anema
---@license MIT
---@copyright 2026

--- It says hello
--- @type func(name:string):string
--- @param name string person who you are saying hi to
--- @return string greeting
--- @raise runtime error if name is not a string
local function hello(name) return "Hello " .. name end
