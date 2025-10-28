-- quick simple ansi library for color output in terminals
local escapeString = "\x1b[%dm"
local api = {}
local keys = {
	reset = 0,

	bright = 1,
	dim = 2,
	underline = 4,
	blink = 5,
	reverse = 7,
	hidden = 8,

	black = 30,
	red = 31,
	green = 32,
	yellow = 33,
	blue = 34,
	magenta = 35,
	cyan = 36,
	white = 37,

	blackbg = 40,
	redbg = 41,
	greenbg = 42,
	yellowbg = 43,
	bluebg = 44,
	magentabg = 45,
	cyanbg = 46,
	whitebg = 47,
}

-- Generate a method call for each code
for name, code in pairs(keys) do
	api[name] = function(str)
		return "\x1b[" .. tostring(code) .. "m" .. tostring(str) .. "\x1b[0m"
	end
end

return api
