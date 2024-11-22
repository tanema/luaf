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

local foo = setmetatable({val = 22}, metaTbl)
local bar = setmetatable({val = 77}, metaTbl)

assert((foo + bar) == 99, foo+bar)
assert(foo ~= bar)
assert(foo <= bar)
