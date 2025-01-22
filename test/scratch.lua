local open_tag <const> = "<%"
local close_tag <const> = "%>"
local modifiers <const> = "^[=-]"

local function parseToLua(str)
	assert(type(str) == "string", "expecting string for parse")
	local pos = 1
	local buffer = "local _tmpl_output = ''\n"
	while true do
		local start, stop = str:find(open_tag, pos, true)
		if not start then
			if pos < #str then
				buffer = buffer .. "_tmpl_output = _tmpl_output .. " .. ("%q"):format(str:sub(pos, #str)) .. "\n"
			end
			break
		end

		if start ~= pos then
			buffer = buffer .. "_tmpl_output = _tmpl_output .. " .. ("%q"):format(str:sub(pos, start - 1)) .. "\n"
		end
		pos = stop + 1

		local modifier
		if str:match(modifiers, pos) then
			modifier = str:sub(pos, pos)
			pos = pos + 1
		end

		local close_start, close_stop = str:find(close_tag, pos, true)
		if not close_start then
			error("failed to find closing tag")
		end

		local chunk = str:sub(pos, close_start - 1)
		if modifier == "=" then
			buffer = buffer .. "_tmpl_output = _tmpl_output .. html_escape(tostring(" .. chunk .. "))\n"
		elseif modifier == "-" then
			buffer = buffer .. "_tmpl_output = _tmpl_output .. tostring(" .. chunk .. ")\n"
		else
			buffer = buffer .. chunk .. "\n"
		end

		pos = close_stop + 1
	end
	return buffer .. "return _tmpl_output"
end

print(parseToLua("<%= name %>"))
