local args = require("term.args")
local cmd = args.new("del", "del file", "delete a file")
local forceFlag = cmd:boolFlag("force", "f", "remove it now!")

cmd:parse()

print("force:", forceFlag.value)
