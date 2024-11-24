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

metaTbl.__index = function(tbl, key)
	return "goop"
end

local foo = setmetatable({val = 22}, metaTbl)
local bar = setmetatable({val = 77}, metaTbl)

assert((foo + bar) == 99, foo+bar)
assert(foo ~= bar)
assert(foo <= bar)
assert(foo.nother == "goop")

local cache = {}
local indmt = {}
indmt.__newindex = cache
local baz = setmetatable({}, indmt)
baz.foo = true
assert(cache.foo)
