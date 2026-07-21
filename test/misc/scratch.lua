local cli = require('term/args')

local parser = cli.new("scratch", "scratch [opts]", "Scratch is a func little test for the cli args library")
local nameFlag = parser:stringFlag("name", "n", "name for person", "tim")
local portFlag = parser:numberFlag("port", "p", "port for server")
local enabledFlag = parser:boolFlag("enabled", "e", "Enable functionality")

parser:parse()

print(nameFlag.value)
print(enabledFlag.value)
for _i, val in ipairs(parser.args) do
	print(val)
end
