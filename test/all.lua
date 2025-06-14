print("running lua tests.")

do
	-- set random seed
	local random_x, random_y = math.randomseed()
	print(string.format("random seeds: %d, %d", random_x, random_y))
end

dofile("./test/_main.lua")
dofile("./test/_metatables.lua")
dofile("./test/_close.lua")
print("done lua tests.")
