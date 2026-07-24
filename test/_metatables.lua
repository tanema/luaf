local t = require("internal.runtime.lib.test")
local metaTableTests = {}
local metaTbl = {}

metaTbl.__add = function(lval, rval)
	return lval.val + rval.val
end

metaTbl.__eq = function(lval, rval)
	return lval.val == rval.val
end

metaTbl.__le = function(lval, rval)
	return lval.val < rval.val
end

metaTbl.__index = function()
	return "goop"
end

function metaTableTests.testMetaMethods()
	local foo = setmetatable({ val = 22 }, metaTbl)
	local bar = setmetatable({ val = 77 }, metaTbl)
	t.assert.Eq(99, (foo + bar), foo + bar)
	t.assert.NotEq(foo, bar)
	t.assert.True(foo <= bar)
	t.assert.Eq(foo.nother, "goop")

	local cache = {}
	local indmt = {}
	indmt.__newindex = cache
	local baz = setmetatable({}, indmt)
	baz.foo = true
	t.assert.True(cache.foo)

	local didCall = false
	local callmt = {
		__call = function()
			didCall = true
		end,
	}
	local callme = setmetatable({}, callmt)
	callme()
	t.assert.True(didCall, "didCall")
end

function metaTableTests.testFormatToString()
	local m = setmetatable({}, {
		__tostring = function()
			return "hello"
		end,
		__name = "hi",
	})
	t.assert.Eq(string.format("%s %.10s", m, m), "hello hello")
	getmetatable(m).__tostring = nil -- will use '__name' from now on
	t.assert.Eq(string.format("%.4s", m), "hi")

	getmetatable(m).__tostring = function()
		return {}
	end
	t.assert.Error(function()
		print(tostring(m))
	end)
end

return metaTableTests
