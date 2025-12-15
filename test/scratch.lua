local tbl = { 1, 2, 3, 4 }

for i, val in ipairs(tbl) do
	print(i, val)
	if i == 3 then
		error("too far")
	end
end

print("done.")
