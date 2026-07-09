local object = {}
object.__index = object
object.class = object

function object:init(...) end
function object:isa(cls)
	local currentClass = self
	while currentClass ~= nil do
		if currentClass == cls then
			return true
		else
			currentClass = currentClass.super
		end
	end
	return false
end

return function(parent)
	local super = parent or object
	local child = {
		super = super,
		__gc = super.__gc,
		__newindex = super.__newindex,
		__mode = super.__mode,
		__tostring = super.__tostring,
		__len = super.__len,
		__unm = super.__unm,
		__add = super.__add,
		__sub = super.__sub,
		__mul = super.__mul,
		__div = super.__div,
		__mod = super.__mod,
		__pow = super.__pow,
		__concat = super.__concat,
		__eq = super.__eq,
		__lt = super.__lt,
		__le = super.__le,
	}
	child.__index = child
	child.class = child
	return setmetatable(child, {
		__index = child.super,
		__call = function(_, ...)
			local instance = setmetatable({}, child)
			instance.super = child
			instance:init(...)
			return instance
		end,
	})
end
