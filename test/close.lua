local closed = false

local function test()
	local a <close> = setmetatable({}, {
		__close = function()
			closed = true
		end,
	})
end

print("calling test")
test()
print("done test")
assert(closed)
