local t = require("test")
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
	t.assert((foo + bar) == 99, foo + bar)
	t.assert(foo ~= bar)
	t.assert(foo <= bar)
	t.assert(foo.nother == "goop")

	local cache = {}
	local indmt = {}
	indmt.__newindex = cache
	local baz = setmetatable({}, indmt)
	baz.foo = true
	t.assert(cache.foo)

	local didCall = false
	local callmt = {
		__call = function()
			didCall = true
		end,
	}
	local callme = setmetatable({}, callmt)
	callme()
	t.assert(didCall, "didCall")
end

return metaTableTests
