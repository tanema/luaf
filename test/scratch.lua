local tbl = { 93, 22, 78, 22 }
for key, val in ipairs(tbl) do
	assert(tbl[key] == val, "for in loop")
end
