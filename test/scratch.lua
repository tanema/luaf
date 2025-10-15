local a = {}
local lim = 2000
for i = 1, lim do
	a[i] = i
end

print(#a)

local x = { table.unpack(a) }
print(#x)
