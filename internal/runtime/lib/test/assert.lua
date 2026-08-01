local util = require("test.util")
local assert = {}

local function fmtVal(v)
	return type(v) == "string" and string.format("%q", v) or tostring(v)
end

local function annotateAssertion(fn)
	return function(...)
		_G["__LUA_TEST_ASSERTION_TOTAL"] = (_G["__LUA_TEST_ASSERTION_TOTAL"] or 0) + 1
		fn(...)
	end
end

-- customAssert brings in the assertion value, a user messsage, and the message for
-- the assertion and will hydrate the message with line info
local function customAssert(got, msg, assertMsg, ...)
	if got then
		return
	end

	local linfo = debug.getinfo(4)
	local location = linfo and string.format("%s:%d: ", linfo.short_src, linfo.currentline) or ""

	local msgVals = {}
	for i, val in ipairs({ ... }) do
		msgVals[i] = fmtVal(val)
	end

	local base = location .. string.format(assertMsg, table.unpack(msgVals))
	if msg ~= nil and msg ~= "" then
		base = base .. " (" .. tostring(msg) .. ")"
	end

	util.fail(base)
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

function assert.Eq(expected, actual, msg)
	if deepEq(expected, actual) then
		return
	end
	if type(expected) == "table" and type(actual) == "table" then
		customAssert(
			false,
			msg,
			"expected table to equal, but found differences:\n    "
				.. table.concat(diffTables(expected, actual), "\n    ")
		)
	end
	customAssert(false, msg, "expected %s, got %s", expected, actual)
end

function assert.NotEq(expected, actual, msg)
	customAssert(not deepEq(expected, actual), msg, "expected %s to not equal %s", expected, actual)
end

function assert.Less(actual, compare, msg)
	customAssert(actual < compare, msg, "expected %s < %s", actual, compare)
end

function assert.LessEq(actual, compare, msg)
	customAssert(actual <= compare, msg, "expected %s <= %s", actual, compare)
end

function assert.Greater(actual, compare, msg)
	customAssert(actual > compare, msg, "expected %s > %s", actual, compare)
end

function assert.GreaterEq(actual, compare, msg)
	customAssert(actual >= compare, msg, "expected %s >= %s", actual, compare)
end

function assert.Contains(bucket, val, msg)
	customAssert(
		type(bucket) == "string" or type(bucket) == "table",
		nil,
		"bad argument #1 to Contains, should be table or string but received %s",
		type(bucket)
	)
	if type(bucket) == "string" then
		customAssert(string.find(bucket, val, 1, true), msg, "Contains: expected %s to contain %s", bucket, val)
	else
		for _, bval in ipairs(bucket) do
			if bval == val then
				return
			end
		end
		customAssert(false, msg, "Contains: expected %s to contain %s", bucket, val)
	end
end

function assert.IsType(val, typeName, msg)
	customAssert(type(val) == typeName, msg, "expected %s, got %s", typeName, val)
end

function assert.True(got, msg)
	customAssert(got, msg, "expected a truthy value, got %s", got)
end

function assert.False(got, msg)
	customAssert(not got, msg, "expected false, got %s", got)
end

function assert.Nil(actual, msg)
	customAssert(actual == nil, msg, "expected nil, got %s", actual)
end

function assert.IsTable(val, msg)
	customAssert(type(val) == "table", msg, "expected table, got %s", type(val))
end

function assert.IsNumber(val, msg)
	customAssert(type(val) == "number", msg, "expected number, got %s", type(val))
end

function assert.IsString(val, msg)
	customAssert(type(val) == "string", msg, "expected string, got %s", type(val))
end

function assert.NotNil(actual, msg)
	customAssert(actual ~= nil, msg, "expected not nil value, got nil")
end

function assert.Len(actual, expectedLen, msg)
	customAssert(
		type(actual) == "string" or type(actual) == "table",
		msg,
		"Len: assertion failed! value is %s",
		type(actual)
	)
	customAssert(#actual == expectedLen, msg, "expected length %d, got %d", expectedLen, #actual)
end

function assert.Empty(actual, msg)
	if actual ~= nil then
		customAssert(
			type(actual) == "string" or type(actual) == "table",
			msg,
			"Empty: assertion failed! value is %s",
			type(actual)
		)
		customAssert(#actual == 0, msg, "expected empty got %d", #actual)
	end
end

function assert.Error(fn, errMatch, msg)
	customAssert(
		type(fn) == "function" or type(fn) == "table",
		nil,
		"bad argument #1 to Error, should be function but received %s",
		type(fn)
	)
	local ok, err = pcall(fn)
	if ok == true then
		customAssert(false, msg, "expected function to raise an error, got %s", err)
	end
	if type(err) == "string" and type(errMatch) == "string" then
		customAssert(string.find(err, errMatch), msg, "Error: expected error %s to contain %s", err, errMatch)
	else
		customAssert(deepEq(errMatch, err), msg, "Error: expected error %s to eq %s", err, errMatch)
	end
end

function assert.NoError(fn, msg)
	customAssert(
		type(fn) == "function",
		nil,
		"bad argument #1 to NoError, should be function but received %s",
		type(fn)
	)
	local ok, result = pcall(fn)
	if not ok then
		customAssert(false, msg, "expected function not to raise an error, but got an error: %s", result)
	end
	return result
end

function assert.Load(src, msg)
	local fn, err = load(src)
	customAssert(err == nil, msg, "SyntaxError: assertion failed, fn failed to load with err: %s", err)
	return fn
end

function assert.SyntaxError(src, errMatch, msg)
	customAssert(type(src) == "string", nil, "bad argument #1 to SyntaxError (string expected, got %s)", type(src))
	customAssert(type(errMatch) == "string", nil, "bad argument #2 to SyntaxError (string expected, got %s)", type(src))
	local fn, err = load(src)
	customAssert(fn == nil, msg, "SyntaxError: expected fn to not load successfully but got function")
	customAssert(err ~= nil, msg, "SyntaxError: expected err to not be nil but got nil.")
	customAssert(string.find(err, errMatch, 1, true), msg, "SyntaxError: expected %s to contain %s", err, errMatch)
end

for name, assertion in pairs(assert) do
	assert[name] = annotateAssertion(assertion)
end

return assert
