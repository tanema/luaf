-- ensure b can use a, and the final value is discarded
local function test(...)
	return ...
end
local a, b, c = test(1, 2, 3)
print(a)
print(b)
print(c)
