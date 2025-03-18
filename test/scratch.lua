warn("@allow")
-- create two objects to be finalized when closing state
-- the errors in the finalizers must generate warnings
local u1 = setmetatable({}, {
	__gc = function()
		error("XYZ")
	end,
})
local u2 = setmetatable({}, {
	__gc = function()
		error("ZYX")
	end,
})
