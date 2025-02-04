warn("@allow")
u1 = setmetatable({}, {
	__gc = function()
		error("XYZ")
	end,
})
u2 = setmetatable({}, {
	__gc = function()
		error("ZYX")
	end,
})
