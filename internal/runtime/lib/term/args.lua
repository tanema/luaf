local class = require('class')

local Parser = class()
local Flag = class()

function Parser:init(cli_name, usage, doc)
	self.index = 1
	self.parsed = false
	self.name = cli_name
	self.usage = usage
	self.doc = doc
	self.flags = {}
	self.all_flags = {}
	self.args = {}
	self.max_name_width = 0

	self:boolFlag("help", "h", "show help")
end

function Parser:stringFlag(long, short, desc, default)
	return self:flag({
		long = long,
		short = short,
		desc = desc,
		default = default,
		type = "string"
	})
end

function Parser:boolFlag(long, short, desc)
	return self:flag({
		long = long,
		short = short,
		desc = desc,
		default = false,
		type = "boolean"
	})
end

function Parser:numberFlag(long, short, desc, default)
	return self:flag({
		long = long,
		short = short,
		desc = desc,
		default = default,
		type = "number"
	})
end

function Parser:flag(args)
	local flag = Flag(args)
	self.flags[args.long or args.short] = flag

	if args.long ~= nil and #args.long > 1 then
		self.all_flags[args.long] = flag
	end

	if args.short ~= nil and #args.short > 0 then
		self.all_flags[args.short] = flag
	end

	self.max_name_width = math.max(self.max_name_width, #flag.name)
	return flag
end

function Parser:parse()
	if self.parsed then
		return
	end

	while true do
		local input = arg[self.index]
		if input == nil then
			break
		end

		local key, val = string.match(input, "^%-%-?(%w+)=(.+)$")
		local name = string.match(input, "^%-%-?(%w+)$")

		if key ~= nil then
			local definedFlag = self.all_flags[key]
			if definedFlag ~= nil then
				definedFlag:setValue(val)
			end
		elseif name then
			local definedFlag = self.all_flags[name]
			if definedFlag ~= nil and definedFlag.type == "boolean" then
				definedFlag.value = not definedFlag.value
			elseif definedFlag ~= nil then
				self.index = self.index + 1
				if #arg >= self.index then
					error(string.format("expected value for flag %q but none provided.", definedFlag.name))
				end
				definedFlag:setValue(arg[self.index])
			end
		else
			table.insert(self.args, input)
		end

		self.index = self.index + 1
	end

	if self:get("help").value then
		self:printUsage()
		os.exit(0)
	end
end

function Parser:get(name)
	return self.all_flags[name]
end

function Parser:printUsage()
	print(string.format("Usage: %s", self.usage))

	if self.doc ~= nil and #self.doc > 0 then
		print("\n" .. self.doc .. "\n")
	end

	print("Flags:")
	for _, flag in pairs(self.flags) do
		print(string.format("%-" .. self.max_name_width .. "s %s", flag.name, flag.desc))
	end
end

function Flag:init(args)
	self.default = args.default
	self.type = args.type
	self.value = self.default

	local labels = {}
	if args.long ~= nil and #args.long > 0 then
		table.insert(labels, "--" .. args.long)
	end
	if args.short ~= nil and #args.short > 0 then
		table.insert(labels, "-" .. args.short)
	end

	if args.type == "string" then
		for i in ipairs(labels) do
			labels[i] = labels[i] .. "=<string>"
		end
	elseif args.type == "number" then
		for i in ipairs(labels) do
			labels[i] = labels[i] .. "=<number>"
		end
	end
	self.name = table.concat(labels, ", ")

	local default = ""
	if self.default ~= nil and self.type ~= "boolean" then
		default = string.format(" (default: %s)", tostring(self.default))
	end
	self.desc = args.desc .. default
end

function Flag:setValue(value)
	if self.type == "string" then
		self.value = value
	elseif self.type == "number" then
		self.value = tonumber(value)
	elseif self.type == "boolean" then
		self.value = value == "1" or value == "yes" or value == "true"
	end
end

return {
	new = function(cli_name, usage, doc)
		return Parser(cli_name, usage, doc)
	end,
}
