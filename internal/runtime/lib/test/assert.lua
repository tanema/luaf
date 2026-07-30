local util = require('test.util')

local function fmtVal(v)
	if type(v) == "string" then
		return string.format("%q", v)
	end
	return tostring(v)
end

local function withMsg(base, msg)
	local info = debug.getinfo(3)
	local location = info and string.format("%s:%d: ", info.short_src, info.currentline) or ""
	if msg == nil or msg == "" then
		return location .. base
	end
	return location .. base .. " (" .. tostring(msg) .. ")"
end

local function addAssertion()
	_G["__LUA_TEST_ASSERTION_TOTAL"] = (_G["__LUA_TEST_ASSERTION_TOTAL"] or 0) + 1
end

local function customAssert(got, msg)
	if not got then util.fail(msg) end
end

local function deepEq(expected, actual)
	if expected == actual then
		return true
	elseif type(expected) == "table" and type(actual) == "table" then
		for key1, value1 in pairs(expected) do
			local value2 = actual[key1]
			if value2 == nil then
				return false
			elseif value1 ~= value2 then
				if type(value1) == "table" and type(value2) == "table" then
					if not deepEq(value1, value2) then
						return false
					end
				else
					return false
				end
			end
		end
		for key2, _ in pairs(actual) do
			if expected[key2] == nil then
				return false
			end
		end
		return true
	end
	return false
end

local function diffTables(expected, actual, path)
	path = path or ""
	local diffs = {}
	for k, v in pairs(expected) do
		local key = string.format("%s[%s]", path, fmtVal(k))
		local av = actual[k]
		if av == nil then
			table.insert(diffs, string.format("%s: missing (expected %s)", key, fmtVal(v)))
		elseif not deepEq(v, av) then
			if type(v) == "table" and type(av) == "table" then
				for _, d in ipairs(diffTables(v, av, key)) do
					table.insert(diffs, d)
				end
			else
				table.insert(diffs, string.format("%s: expected %s, got %s", key, fmtVal(v), fmtVal(av)))
			end
		end
	end
	for k, v in pairs(actual) do
		if expected[k] == nil then
			table.insert(diffs, string.format("%s[%s]: unexpected (got %s)", path, fmtVal(k), fmtVal(v)))
		end
	end
	return diffs
end

return {
	Eq = function(expected, actual, msg)
		addAssertion()
		if deepEq(expected, actual) then
			return
		end
		local detail
		if type(expected) == "table" and type(actual) == "table" then
			detail = "expected table to equal, but found differences:\n    "
					.. table.concat(diffTables(expected, actual), "\n    ")
		else
			detail = string.format("expected %s, got %s", fmtVal(expected), fmtVal(actual))
		end
		util.fail(withMsg(detail, msg))
	end,
	NotEq = function(expected, actual, msg)
		addAssertion()
		customAssert(
			not deepEq(expected, actual),
			withMsg(string.format("expected %s to not equal %s", fmtVal(expected), fmtVal(actual)), msg)
		)
	end,
	Less = function(actual, compare, msg)
		addAssertion()
		customAssert(actual < compare, withMsg(string.format("expected %s < %s", fmtVal(actual), fmtVal(compare)), msg))
	end,
	LessEq = function(actual, compare, msg)
		addAssertion()
		customAssert(
			actual <= compare,
			withMsg(string.format("expected %s <= %s", fmtVal(actual), fmtVal(compare)), msg)
		)
	end,
	Greater = function(actual, compare, msg)
		addAssertion()
		customAssert(actual > compare, withMsg(string.format("expected %s > %s", fmtVal(actual), fmtVal(compare)), msg))
	end,
	GreaterEq = function(actual, compare, msg)
		addAssertion()
		customAssert(
			actual >= compare,
			withMsg(string.format("expected %s >= %s", fmtVal(actual), fmtVal(compare)), msg)
		)
	end,
	Contains = function(bucket, val, msg)
		addAssertion()
		customAssert(
			type(bucket) == "string" or type(bucket) == "table",
			withMsg(
				string.format("bad argument #1 to Contains, should be table or string but received %s", type(bucket)),
				msg
			)
		)
		if type(bucket) == "string" then
			customAssert(
				string.find(bucket, val, 1, true),
				withMsg(string.format("Contains: expected %s to contain %s", fmtVal(bucket), fmtVal(val)), msg)
			)
		else
			for _, bval in ipairs(bucket) do
				if bval == val then
					return
				end
			end
			customAssert(
				false,
				withMsg(string.format("Contains: expected %s to contain %s", fmtVal(bucket), fmtVal(val)), msg)
			)
		end
	end,
	IsType = function(val, typeName, msg)
		addAssertion()
		customAssert(
			type(val) == typeName,
			withMsg(string.format("expected %s, got %s", fmtVal(typeName), fmtVal(val)), msg)
		)
	end,
	True = function(got, msg)
		addAssertion()
		customAssert(got, withMsg(string.format("expected a truthy value, got %s", fmtVal(got)), msg))
	end,
	False = function(got, msg)
		addAssertion()
		customAssert(not got, withMsg(string.format("expected false, got %s", fmtVal(got)), msg))
	end,
	Nil = function(actual, msg)
		addAssertion()
		customAssert(actual == nil, withMsg(string.format("expected nil, got %s", fmtVal(actual)), msg))
	end,
	IsTable = function(val, msg)
		addAssertion()
		customAssert(type(val) == "table", withMsg(string.format("expected table, got %s", fmtVal(type(val))), msg))
	end,
	IsNumber = function(val, msg)
		addAssertion()
		customAssert(type(val) == "number", withMsg(string.format("expected number, got %s", fmtVal(type(val))), msg))
	end,
	IsString = function(val, msg)
		addAssertion()
		customAssert(type(val) == "string", withMsg(string.format("expected string, got %s", fmtVal(type(val))), msg))
	end,
	NotNil = function(actual, msg)
		addAssertion()
		customAssert(actual ~= nil, withMsg("expected not nil value, got nil", msg))
	end,
	Len = function(actual, expectedLen, msg)
		addAssertion()
		customAssert(
			type(actual) == "string" or type(actual) == "table",
			withMsg(string.format("Len: assertion failed! value is %s", type(actual)), msg)
		)
		customAssert(
			#actual == expectedLen,
			withMsg(string.format("expected length %d, got %d", expectedLen, #actual), msg)
		)
	end,
	Empty = function(actual, msg)
		addAssertion()
		if actual ~= nil then
			customAssert(
				type(actual) == "string" or type(actual) == "table",
				withMsg(string.format("Empty: assertion failed! value is %s", type(actual)), msg)
			)
			customAssert(#actual == 0, withMsg(string.format("expected empty got %d", #actual), msg))
		end
	end,
	Error = function(fn, errMatch, msg)
		addAssertion()
		customAssert(
			type(fn) == "function" or type(fn) == "table",
			withMsg(string.format("bad argument #1 to Error, should be function but received %s", type(fn)), msg)
		)
		local ok, err = pcall(fn)
		if ok == true then
			util.fail(withMsg(string.format("expected function to raise an error, got %s", fmtVal(err)), msg))
		end
		if type(err) == "string" and type(errMatch) == "string" then
			customAssert(
				string.find(err, errMatch, 1, true),
				withMsg(string.format("Error: expected error %s to contain %s", fmtVal(err), fmtVal(errMatch)), msg)
			)
		else
			customAssert(
				deepEq(errMatch, err),
				withMsg(string.format("Error: expected error %s to eq %s", fmtVal(err), fmtVal(errMatch)), msg)
			)
		end
	end,
	NoError = function(fn, msg)
		addAssertion()
		customAssert(
			type(fn) == "function",
			withMsg(string.format("bad argument #1 to NoError, should be function but received %s", type(fn)), msg)
		)
		local ok, result = pcall(fn)
		if not ok then
			util.fail(
				withMsg(
					string.format("expected function not to raise an error, but got an error: %s", fmtVal(result)),
					msg
				)
			)
		end
		return result
	end,
	Load = function(src, msg)
		addAssertion()
		local fn, err = load(src)
		customAssert(
			err == nil,
			withMsg(string.format("SyntaxError: assertion failed, fn failed to load with err: %s", fmtVal(err)), msg)
		)
		return fn
	end,
	SyntaxError = function(src, errMatch, msg)
		addAssertion()
		customAssert(
			type(src) == "string",
			withMsg(string.format("bad argument #1 to SyntaxError (string expected, got %s)", type(src)), msg)
		)
		customAssert(
			type(errMatch) == "string",
			withMsg(string.format("bad argument #2 to SyntaxError (string expected, got %s)", type(src)), msg)
		)
		local fn, err = load(src)
		customAssert(fn == nil, withMsg("SyntaxError: expected fn to not load successfully but got function", msg))
		customAssert(err ~= nil, withMsg("SyntaxError: expected err to not be nil but got nil.", msg))
		customAssert(
			string.find(err, errMatch, 1, true),
			withMsg(string.format("SyntaxError: expected %s to contain %s", fmtVal(err), fmtVal(errMatch)), msg)
		)
	end,
}
