---@module test_module
---@deprecated
---@author Tim Anema
---@license MIT
---@copyright Tim Anema 2026
---@version 23.2.1
---@release 1.0.1
---A simple module to test parsing docs
---@return Person
---@return string

---@class Person a real human being
---@field name string nominative denomination
---@field alive bool have they died
---@field birthdate number|nil when they were born
local Person = {}

---@enum states captures a user account state
---@field public enabled number the status is enabled
---@field public disabled number the status is disabled
---@field public suspended number the status is suspended
---@field private destroyed number|string the status is destroyed
local states = {
  enabled = 1,
  disabled = 2,
  suspended = 3,
  destroyed = 4,
}

---@class Config
---@description configuration management class
config = {}

---@description reads the config from file
---@param filename string
---@return table config
---@raise FileErr
function config:read(filename) return { filename = filename } end

---@todo add more to this
---@description It says hello
---To your name specifically.
---@usage hello(name())
---@usage hello("bobby")
---@param name string person who you are saying hi to
---@return string greeting a hello
local function hello(name) return "Hello " .. name end

---@description It returns the name of the participant
---@return string name bobby's name
---@raise IOERROR
local function name() return "bobby" end

---@description Start the webserver
---@param port string|number The port that the webserver should start on
---@return boolean started if starting the server was successful
---@raise PortErr
function startServer(port) end
