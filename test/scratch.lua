local a = { 1, 2, 3, 4, 5, 6 }

local function makeNew(tbl)
	return { table.unpack(tbl) }
end

print(#makeNew(a))
