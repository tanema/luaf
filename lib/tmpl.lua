local open_tag <const> = "<%"
local close_tag <const> = "%>"
local modifiers <const> = "^[=-]"
local html_escape_entities <const> = {
	["&"] = "&amp;",
	["<"] = "&lt;",
	[">"] = "&gt;",
	['"'] = "&quot;",
	["'"] = "&#039;",
}
local render_env <const> = {
	assert = assert,
	html_escape = function(str)
		return (str:gsub([=[["><'&]]=], html_escape_entities))
	end,
	tostring = tostring,
	tonumber = tonumber,
	type = type,
	next = next,
	pairs = pairs,
	pcall = pcall,
	xpcall = xpcall,
	rawget = rawget,
	rawset = rawset,
	rawequal = rawequal,
	rawlen = rawlen,
	select = select,
	string = string,
	table = table,
	math = math,
	utf8 = utf8,
}

local function error_for_pos(str, source_pos, err_msg)
	local source_line_no = 1
	for _ in str:sub(1, source_pos):gmatch("\n") do
		source_line_no = source_line_no + 1
	end
	local source_line
	for line in str:gmatch("([^\n]*)\n?") do
		if source_line_no == 1 then
			source_line = line
			break
		end
		source_line_no = source_line_no - 1
	end
	return tostring(err_msg) .. " [" .. tostring(source_line_no) .. "]: " .. tostring(source_line)
end

local function parseToLua(str)
	assert(type(str) == "string", "expecting string for parse")
	local pos = 1
	local buffer = [[
return function(_params)
  assert(type(_params) == 'table', 'params to template render should be a table')
  for __name, __val in pairs(_params) do
    _ENV[__name] = __val
  end
  local _tmpl_output = ''
]]
	while true do
		local start, stop = str:find(open_tag, pos, true)
		if not start then
			if pos < #str then
				buffer = buffer .. "  _tmpl_output = _tmpl_output .. " .. ("%q"):format(str:sub(pos, #str)) .. "\n"
			end
			break
		end

		if start ~= pos then
			buffer = buffer .. "  _tmpl_output = _tmpl_output .. " .. ("%q"):format(str:sub(pos, start - 1)) .. "\n"
		end
		pos = stop + 1

		local modifier
		if str:match(modifiers, pos) then
			modifier = str:sub(pos, pos)
			pos = pos + 1
		end

		local close_start, close_stop = str:find(close_tag, pos, true)
		if not close_start then
			error(error_for_pos(str, start, "failed to find closing tag"))
		end

		local chunk = str:sub(pos, close_start - 1)
		if modifier == "=" then
			buffer = buffer .. "  _tmpl_output = _tmpl_output .. html_escape(tostring(" .. chunk .. "))\n"
		elseif modifier == "-" then
			buffer = buffer .. "  _tmpl_output = _tmpl_output .. tostring(" .. chunk .. ")\n"
		else
			buffer = buffer .. chunk .. "\n"
		end

		pos = close_stop + 1
	end
	return buffer .. "  return _tmpl_output\nend"
end

local function parse(str)
	assert(type(str == "string"), "template should be a string")
	local fn, lerr = load(parseToLua(str), "elua", "t", render_env)
	if not fn then
		error(lerr)
	end
	return fn()
end

return {
	parse = parse,
	parseToLua = parseToLua,
}
