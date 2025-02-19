local data = { 1, 2, 3 }

local iterItem = setmetatable(data, {
	__call = ipairs,
})

for k, v in iterItem() do
	print(k, v)
end
