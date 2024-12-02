
local class = {name = "blah blah"}

function class:new()
	return {new = self.name}
end

print(class:new())
