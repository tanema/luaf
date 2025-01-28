local child = {}
child.__index = child
setmetatable(child, {
	__call = function(self, name)
		print("called", self, "name", name)
	end,
})("Tim")
