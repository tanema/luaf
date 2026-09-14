local repl = {
	_buffer = "",
	VERSION = 0.10,
}

local function gather_results(success, ...)
	return success, table.pack(...)
end

function repl:prompt(level)
	prefix = (level == 1 and ">" or ">>")
	io.stdout:write(prefix .. " ")
end

function repl:handleline(line)
	local chunk = self._buffer .. line
	local f, err = load("return " .. chunk, "REPL", _G)
	if not f then
		f, err = load(chunk, "REPL", _G)
	end

	if f then
		self._buffer = ""
		local success, results = gather_results(xpcall(f, debug.traceback))
		if success then
			self:displayresults(results)
			_G._ = results
			if results.n > 0 then
				print(table.unpack(results, 1, results.n))
			end
		else
			print(results[1])
		end
	else
		if string.match(err, "'<eof>'$") or string.match(err, "<eof>$") then
			self._buffer = chunk .. "\n"
			return 2
		else
			print(err)
			self._buffer = ""
		end
	end

	return 1
end

function repl:run()
	self:prompt(1)
	for line in io.stdin:lines() do
		local level = self:handleline(line)
		self:prompt(level)
	end
end

return repl
