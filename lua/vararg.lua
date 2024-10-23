
local function rep(...)
	return {..., ..., ...}
end

print(table.concat(rep(1, 2, 3), ", "))
