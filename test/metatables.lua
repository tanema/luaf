local metaTbl = {}

metaTbl.__add = function(lval, rval)
	return lval.val + rval.val
end

metaTbl.__eq = function(lval, rval)
	return lval.val == rval.val
end

local foo = setmetatable({val = 22}, metaTbl)
local bar = setmetatable({val = 77}, metaTbl)

print(foo + bar)
print(foo == bar)
