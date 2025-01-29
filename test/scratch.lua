local prog = os.tmpname()

local prepfile = function(s, mod, p)
	mod = mod and "wb" or "w" -- mod true means binary files
	p = p or prog -- file to write the program
	local f = io.open(p, mod)
	print("opened", f)
	f:write(s)
	print("writed")
	assert(f:close())
	print("closed")
	print("prepped")
end

prepfile("")
