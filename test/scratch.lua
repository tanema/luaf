-- local a = { 1, 2, 3, 4, 5, 6 }

-- local function makeNew(tbl)
--	return { table.unpack(tbl) }
-- end

-- print(#makeNew(a))

local function test()
	local a = 42

	return function()
		return a
	end
end

local a = test()()
return a
