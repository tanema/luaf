local t = require("test")
t.suite("test/_main")
t.suite("test/_metatables")
t.suite("test/_close")
if os.getenv("TESTLIB") ~= nil then
	t.suite("test/_testLib")
end
t.run({
	verbose = os.getenv("VERBOSE") ~= nil,
	begin = function()
		local random_x, random_y = math.randomseed()
		print(string.format("random seeds: %d, %d", random_x, random_y))
	end,
})
