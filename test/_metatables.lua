local t = require("src.runtime.lib.test")
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
	t.assertEq(99, (foo + bar), foo + bar)
	t.assertNotEq(foo, bar)
	t.assertTrue(foo <= bar)
	t.assertEq(foo.nother, "goop")

	local cache = {}
	local indmt = {}
	indmt.__newindex = cache
	local baz = setmetatable({}, indmt)
	baz.foo = true
	t.assertTrue(cache.foo)

	local didCall = false
	local callmt = {
		__call = function()
			didCall = true
		end,
	}
	local callme = setmetatable({}, callmt)
	callme()
	t.assertTrue(didCall, "didCall")
end

function metaTableTests.testFormatToString()
	local m = setmetatable({}, {
		__tostring = function()
			return "hello"
		end,
		__name = "hi",
	})
	t.assertEq(string.format("%s %.10s", m, m), "hello hello")
	getmetatable(m).__tostring = nil -- will use '__name' from now on
	t.assertEq(string.format("%.4s", m), "hi")

	getmetatable(m).__tostring = function()
		return {}
	end
	t.assertError(function()
		print(tostring(m))
	end)
end

return metaTableTests
