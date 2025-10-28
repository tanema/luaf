local function dumpLuaExt(obj, depth, circular)
	local valType = type(obj)
	if valType == "table" then
		circular[tostring(obj)] = true
		local parts = {}
		for key, val in pairs(obj) do
			if val ~= nil and not circular[tostring(key)] and not circular[tostring(val)] then
				local keyDump = dumpLuaExt(key, depth + 1, circular)
				local valDump = dumpLuaExt(val, depth + 1, circular)
				if type(key) == "number" then
					table.insert(parts, valDump)
				else
					table.insert(parts, string.format("[%s]=%s", keyDump, valDump))
				end
			end
		end
		circular[tostring(obj)] = false
		if #parts < 4 then
			return string.format("{%s}", table.concat(parts, ","))
		else
			local indent = string.rep("  ", depth)
			return string.format("{\n%s%s}", indent, table.concat(parts, string.format(",\n%s", indent)))
		end
	elseif valType == "function" then
		return string.format("%q", string.dump(obj))
	elseif valType == "number" or valType == "boolean" or valType == "nil" then
		return tostring(obj)
	elseif valType == "string" then
		return string.format("%q", tostring(obj))
	else
		error(string.format("unexpected data type %s while dumping", valType))
	end
end

local function marshalLua(val)
	return string.format("return %s", dumpLuaExt(val, 1, {}))
end

local function unmarshalLua(str)
	assert(type(str) == "string", "bad argument #1 to lua.unmarshal, expected string")
	return load(str)()
end

local function dumpJSON(obj, circular)
	local valType = type(obj)
	if valType == "table" then
		circular[tostring(obj)] = true
		local mapParts = {}
		local arrayParts = {}
		for key, val in pairs(obj) do
			if val ~= nil and not circular[tostring(key)] and not circular[tostring(val)] then
				local keyDump = dumpJSON(key, circular)
				local valDump = dumpJSON(val, circular)
				if type(key) == "number" then
					assert(#mapParts == 0, "mixed keyed and indexed table. cannot marshal")
					table.insert(arrayParts, valDump)
				else
					assert(#arrayParts == 0, "mixed keyed and indexed table. cannot marshal")
					table.insert(mapParts, string.format("%s:%s", keyDump, valDump))
				end
			end
		end
		circular[tostring(obj)] = false
		if #arrayParts > 0 then
			return string.format("[%s]", table.concat(arrayParts, ","))
		else
			return string.format("{%s}", table.concat(mapParts, ","))
		end
	elseif valType == "function" then
		return string.format("%q", string.dump(obj))
	elseif valType == "number" or valType == "boolean" then
		return tostring(obj)
	elseif valType == "nil" then
		return "null"
	elseif valType == "string" then
		return string.format("%q", tostring(obj))
	else
		error(string.format("unexpected data type %s while dumping", valType))
	end
end

local function marshalJSON(val)
	return dumpJSON(val, {})
end

local JSONDecoder = {}

local function NewJSONDecoder(str)
	return setmetatable({
		buffer = str,
		index = 1,
	}, { __index = JSONDecoder })
end

function JSONDecoder:decode()
	return self:decodeItem()
end

function JSONDecoder:decodeItem()
	self:skipWhitespace()
	local ch = string.sub(self.buffer, self.index, self.index)
	if ch == '"' then
		return self:parseString()
	elseif ch == "[" then
		return self:parseArray()
	elseif ch == "{" then
		return self:parseObject()
	else
		return self:parseLiteral()
	end
end

function JSONDecoder:skipWhitespace()
	while string.sub(self.buffer, self.index, self.index) == " " do
		self.index = self.index + 1
	end
end

function JSONDecoder:parseString()
	self.index = self.index + 1
	local start = self.index
	while string.sub(self.buffer, self.index, self.index) ~= '"' do
		self.index = self.index + 1
	end
	local str = string.sub(self.buffer, start, self.index - 1)
	self.index = self.index + 1
	return str
end

function JSONDecoder:parseLiteral()
	local ch = string.sub(self.buffer, self.index, self.index)
	local start = self.index
	while ch ~= "," and ch ~= "}" and ch ~= "]" and ch ~= "" do
		self.index = self.index + 1
		ch = string.sub(self.buffer, self.index, self.index)
	end
	local literal = string.sub(self.buffer, start, self.index - 1)
	if literal == "true" then
		return true
	elseif literal == "false" then
		return false
	elseif literal == "null" then
		return nil
	else
		return tonumber(literal)
	end
end

function JSONDecoder:parseArray()
	local result = {}
	self.index = self.index + 1 -- skip "["
	while string.sub(self.buffer, self.index, self.index) ~= "]" do
		table.insert(result, self:decodeItem())
		if string.sub(self.buffer, self.index, self.index) == "," then
			self.index = self.index + 1
		end
	end
	self.index = self.index + 1 -- skip "]"
	return result
end

function JSONDecoder:parseObject()
	local result = {}
	self.index = self.index + 1 -- skip "{"
	while string.sub(self.buffer, self.index, self.index) ~= "}" do
		local ch = string.sub(self.buffer, self.index, self.index)
		assert(ch == '"', string.format('expected " to start object key but found %s at index %d', ch, self.index))
		local key = self:parseString()

		ch = string.sub(self.buffer, self.index, self.index)
		assert(ch == ":", string.format("expected : but found %s at index %d", ch, self.index))
		self.index = self.index + 1 -- skip ":"

		result[key] = self:decodeItem()
		if string.sub(self.buffer, self.index, self.index) == "," then
			self.index = self.index + 1
		end
	end
	self.index = self.index + 1 -- skip "}"
	return result
end

local function unmarshalJSON(str)
	assert(type(str) == "string", "bad argument #1 to json.unmarshal, expected string")
	return NewJSONDecoder(str):decode()
end

return {
	lua = { marshal = marshalLua, unmarshal = unmarshalLua },
	json = { marshal = marshalJSON, unmarshal = unmarshalJSON },
}
