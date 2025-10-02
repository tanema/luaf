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

local function unmarshalJSON(str)
	error("not implemented")
end

return {
	lua = { marshal = marshalLua, unmarshal = unmarshalLua },
	json = { marshal = marshalJSON, unmarshal = unmarshalJSON },
}
