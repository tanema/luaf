-- ensure b can use a, and the final value is discarded
local function test( )
	return 1, 2, 3
end
local a, b, c = test()
print(a, b, c)
