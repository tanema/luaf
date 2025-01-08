local open_tag <const> = "<%"
local close_tag <const> = "%>"
local modifiers <const> = "^[=-]"
local html_escape_entities <const> = {
  ['&'] = '&amp;',
  ['<'] = '&lt;',
  ['>'] = '&gt;',
  ['"'] = '&quot;',
  ["'"] = '&#039;'
}
local render_env <const> = {
	tostring = tostring,
	tonumber = tonumber,
	html_escape = function(str) return (str:gsub([=[["><'&]]=], html_escape_entities)) end,
}

local function renderTmpl(tmplSrc, render_args)
	assert(type(render_args == "table"), "render args should be a table")
	local load_env = {}
	for k, v in pairs(render_env) do
		load_env[k] = v
	end
	for k, v in pairs(render_args) do
		load_env[k] = v
	end
	local fn, lerr = load(tmplSrc, "elua", "t", load_env)
	if not fn then
		return nil, lerr
	end
	return fn()
end

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

local function parse(str)
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
			error(error_for_pos(str, start, "failed to find closing tag"))
		end

		local trim_newline
		if "-" == str:sub(close_start - 1, close_start - 1) then
			close_start = close_start - 1
			trim_newline = true
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
		if trim_newline then
			local match = str:match("^\n", pos)
			if match then
				pos = pos + #match
			end
		end
	end
 	buffer = buffer .. "return _tmpl_output"

	return function(render_args)
 		return renderTmpl(buffer, render_args)
	end
end

return {
  parse = parse,
  render = function(str, args)
		return parse(str)(args)
	end,
}
