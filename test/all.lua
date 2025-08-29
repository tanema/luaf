local t = require("test")
t.suite("test/_main")
t.suite("test/_metatables")
t.suite("test/_close")
t.suite("test/_testLib")
t.run({
	verbose = true,
	begin = function()
		local random_x, random_y = math.randomseed()
		print(string.format("random seeds: %d, %d", random_x, random_y))
	end,
})
