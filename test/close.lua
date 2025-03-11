local closed = false

local function test()
	local a <close> = setmetatable({}, {
		__close = function()
			closed = true
		end,
	})
end

test()
assert(closed)
