local tbl = { 6, 5, 4, 3, 2, 1 }

table.sort(tbl, function(a, b)
	local aeven = a % 2 == 0
	local beven = b % 2 == 0
	if aeven and beven then
		return 0
	elseif aeven and not beven then
		return -1
	elseif not aeven and beven then
		return 1
	end
	return 0
end)

for _, v in ipairs(tbl) do
	print(v)
end
