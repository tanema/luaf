-- ensure b can use a, and the final value is discarded
local function test(a, b, ...)
	local x, y = a, b
	print(x, y, ...)
	return ...
end
local a, b, c = test(1, 2, 3, 4, 5)
print(a, b, c)
