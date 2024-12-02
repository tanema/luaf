local class = {name = "blah blah "}

function class:new(name)
	return {new = self.name .. name}
end

print(class:new("tim"))
