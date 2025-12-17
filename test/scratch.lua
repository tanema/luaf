local function test()
	local a = 42
	return function()
		a = 32
		return a
	end
end

local a = test()()

return a
