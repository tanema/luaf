local class = {name = "blah blah "}

local mt = {__gc = function()
	print("gc called")
end}

setmetatable(class, mt)

print("done.")

