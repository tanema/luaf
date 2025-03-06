function testNoReturn() end

print("start")

assert(testNoReturn() == nil, "no ret")
print("after")

local a, b, c, d, e = 1, 2, 3, 4, 5
-- ensure b can use a, and the final value is discarded
function varargReturn(x, y, ...)
	return ...
end
local x, y, z = varargReturn(a, b, c, d, e)
assert(x == 3, "x equals")
assert(y == 4, "y equals")
assert(z == 5, "z equals")

local function test()
	return 1, 2, 3, 4
end

local s1, s2, s3 = select(-3, test())
print(s1, s2, s3)
assert(s1 == 2, "select1")
assert(s2 == 3, "select3")
assert(s3 == 4, "select4")
print("done")
