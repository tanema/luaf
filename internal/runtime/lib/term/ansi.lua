-- quick simple ansi library for color output in terminals
local escapeString = "\x1b[%dm"
local reset <const> = "\x1b[0m"
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

for name, code in pairs(keys) do
  api[name] = function(str) return string.format(escapeString, code) .. tostring(str) .. reset end
end

local keywords = {
	["and"] = true,
	["break"] = true,
	["do"] = true,
	["else"] = true,
	["elseif"] = true,
	["end"] = true,
	["false"] = true,
	["for"] = true,
	["function"] = true,
	["if"] = true,
	["in"] = true,
	["local"] = true,
	["nil"] = true,
	["not"] = true,
	["or"] = true,
	["repeat"] = true,
	["return"] = true,
	["then"] = true,
	["true"] = true,
	["until"] = true,
	["while"] = true,
}

local colormap = {
	["nil"] = api.blue,
	string = api.yellow,
	punctuation = function(...) return api.green(api.bright(...)) end,
	ident = api.red,
	boolean = api.green,
	number = api.cyan,
	path = api.white,
	misc = api.magenta,
}

local function isinteger(n)
	return type(n) == "number" and math.floor(n) == n
end

local function isident(s)
	return type(s) == "string" and not keywords[s] and s:match("^[a-zA-Z_][a-zA-Z0-9_]*$")
end

local type_order = {
	number = 0,
	string = 1,
	userdata = 2,
	table = 3,
	thread = 4,
	boolean = 5,
	["function"] = 6,
	cdata = 7,
}

local function cross_type_order(a, b)
	local pos_a = type_order[type(a)]
	local pos_b = type_order[type(b)]

	if pos_a == pos_b then
		return a < b
	else
		return pos_a < pos_b
	end
end

local function sortedpairs(t)
	local keys = {}
	local seen_non_string

	for k in pairs(t) do
		keys[#keys + 1] = k

		if not seen_non_string and type(k) ~= "string" then
			seen_non_string = true
		end
	end

	local sort_func = seen_non_string and cross_type_order or nil
	table.sort(keys, sort_func)

	local index = 1
	local next_fn = function()
		if keys[index] == nil then
			return nil
		else
			local key = keys[index]
			local value = t[key]
			index = index + 1

			return key, value
		end
	end

	return next_fn, keys
end

local function find_longstring_nest_level(s)
	local level = 0
	while s:find("]" .. string.rep("=", level) .. "]", 1, true) do
		level = level + 1
	end
	return level
end

local function dump_ext(params)
	local pieces = params.pieces
	local seen = params.seen or {}
	local path = params.path or "<topvalue>"
	local v = params.value
	local indent = params.indent or 1

	local t = type(v)

	if t == "nil" or t == "boolean" or t == "number" then
		pieces[#pieces + 1] = colormap[t](tostring(v))
	elseif t == "string" then
		if v:match("\n") then
			local level = find_longstring_nest_level(v)
			pieces[#pieces + 1] =
				colormap.string("[" .. string.rep("=", level) .. "[" .. v .. "]" .. string.rep("=", level) .. "]")
		else
			pieces[#pieces + 1] = colormap.string(string.format("%q", v))
		end
	elseif t == "table" then
		if seen[v] then
			pieces[#pieces + 1] = colormap.path(seen[v])
			return
		end

		seen[v] = path

		local lastintkey = 0

		pieces[#pieces + 1] = colormap.punctuation("{\n")
		for i, v in ipairs(v) do
			for j = 1, indent do
				pieces[#pieces + 1] = "  "
			end
			dump_ext({
				pieces = pieces,
				seen = seen,
				path = path .. "[" .. tostring(i) .. "]",
				value = v,
				indent = indent + 1,
			})
			pieces[#pieces + 1] = colormap.punctuation(",\n")
			lastintkey = i
		end

		for k, v in sortedpairs(v) do
			if not (isinteger(k) and k <= lastintkey and k > 0) then
				for j = 1, indent do
					pieces[#pieces + 1] = "  "
				end

				if isident(k) then
					pieces[#pieces + 1] = colormap.ident(k)
				else
					pieces[#pieces + 1] = colormap.punctuation("[")
					dump_ext({
						pieces = pieces,
						seen = seen,
						path = path .. "." .. tostring(k),
						value = k,
						indent = indent + 1,
					})
					pieces[#pieces + 1] = colormap.punctuation("]")
				end
				pieces[#pieces + 1] = colormap.punctuation(" = ")
				dump_ext({
					pieces = pieces,
					seen = seen,
					path = path .. "." .. tostring(k),
					value = v,
					indent = indent + 1,
				})
				pieces[#pieces + 1] = colormap.punctuation(",\n")
			end
		end

		for j = 1, indent - 1 do
			pieces[#pieces + 1] = "  "
		end

		pieces[#pieces + 1] = colormap.punctuation("}")
	else
		pieces[#pieces + 1] = colormap.misc(tostring(v))
	end
end

function api.dump(results)
	local pieces = {}
	for i = 1, results.n do
		dump_ext({ pieces = pieces, value = results[i] })
		pieces[#pieces + 1] = "\n"
	end
	io.stderr:write(table.concat(pieces, ""))
end


return api
