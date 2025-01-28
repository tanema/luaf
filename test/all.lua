print("running lua tests.")

do
	-- set random seed
	local random_x, random_y = math.randomseed()
	print(string.format("random seeds: %d, %d", random_x, random_y))
end

dofile("./test/_main.lua")
dofile("./test/metatables.lua")
dofile("./test/close.lua")
print("done lua tests.")
