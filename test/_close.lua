local t = require("src.runtime.lib.test")
local closeTests = {}

function closeTests.testClose()
	local closed = false
	local function test()
		local a <close> = setmetatable({}, {
			__close = function()
				closed = true
			end,
		})
	end
	test()
	t.assert(closed, "close not called")
end

return closeTests
