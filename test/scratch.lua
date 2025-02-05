local x <close> = setmetatable({}, {
	__close = function(self, err)
		assert(err == nil)
		print("Ok")
	end,
})
local e1 <close> = setmetatable({}, {
	__close = function()
		print(120)
	end,
})
os.exit(true, true)
