local a = {}
local lim = 2000
for i = 1, lim do
	a[i] = i
end

print(#{ table.unpack(a, lim - 2) })
