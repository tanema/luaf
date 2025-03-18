print("testing stand-alone interpreter")

assert(os.execute())

local arg = arg
local prog = "./tmp/prog"
local otherprog = "./tmp/other"
local out = "./tmp/out"

local progname
do
	local i = 0
	while arg[i] do
		i = i - 1
	end
	progname = arg[i + 1]
end
print("progname: " .. progname)

local prepfile = function(s, mod, p)
	mod = mod and "wb" or "w" -- mod true means binary files
	p = p or prog -- file to write the program
	local f = io.open(p, mod)
	f:write(s)
	assert(f:close())
end

local function getoutput()
	local f = io.open(out)
	local t = f:read("a")
	f:close()
	assert(os.remove(out))
	return t
end

local function checkprogout(s)
	-- expected result must end with new line
	assert(string.sub(s, -1) == "\n")
	local t = getoutput()
	for line in string.gmatch(s, ".-\n") do
		assert(string.find(t, line, 1, true))
	end
end

local function checkout(s)
	local t = getoutput()
	if s ~= t then
		print(string.format("'%s' - '%s'\n", s, t))
	end
	assert(s == t)
	return t
end

local function RUN(p, ...)
	p = string.gsub(p, "lua", '"' .. progname .. '"', 1)
	local s = string.format(p, ...)
	print("RUNNING => ", s)
	assert(os.execute(s))
end

local function NoRun(msg, p, ...)
	p = string.gsub(p, "lua", '"' .. progname .. '"', 1)
	local s = string.format(p, ...)
	s = string.format("%s >%s 2>&1", s, out) -- send output and error to 'out'
	assert(not os.execute(s))
	local output = getoutput()
	print(output)
	assert(string.find(output, msg, 1, true)) -- check error message
end

RUN("lua -v")

print(string.format("(temporary program file used in these tests: %s)", prog))

-- running stdin as a file
prepfile("")
RUN("cat %s | lua > %s", prog, out)
checkout("")

prepfile("print(\n1, a\n)")
RUN("lua - < %s > %s", prog, out)
checkout("1\tnil\n")

RUN('echo "print(10)\nprint(2)\n" | lua > %s', out)
checkout("10\n2\n")

-- test environment variables used by Lua
prepfile("print(package.path)")

-- test 'arg' table
local a = [[
  assert(#arg == 3 and arg[1] == 'a' and arg[2] == 'b' and arg[3] == 'c')
  assert(arg[0] == '--' and arg[-1] == "%s" and arg[-2] == '%s')
  assert(arg[4] == undef and arg[-4] == undef)
  local a, b, c = ...
  assert(... == 'a' and a == 'a' and b == 'b' and c == 'c')
]]
a = string.format(a, prog, progname)
prepfile(a)
RUN("lua %s -- a b c", prog) -- "-e " runs an empty command

print("testing warnings")
-- no warnings by default
RUN('echo "io.stderr:write(1); warn[[XXX]]" | lua 2> %s', out)
checkout("1")

prepfile([[
warn("@allow")               -- unknown control, ignored
warn("@off", "XXX", "@off")  -- these are not control messages
warn("@off")                 -- this one is
warn("@on", "YYY", "@on")    -- not control, but warn is off
warn("@off")                 -- keep it off
warn("@on")                  -- restart warnings
warn("", "@on")              -- again, no control, real warning
warn("@on")                  -- keep it "started"
warn("Z", "Z", "Z")          -- common warning
]])
RUN("lua -W %s 2> %s", prog, out)
checkout([[
Lua warning: @offXXX@off
Lua warning: @on
Lua warning: ZZZ
]])

-- TODO
prepfile([[
warn("@allow")
-- create two objects to be finalized when closing state
-- the errors in the finalizers must generate warnings
local u1 = setmetatable({}, {__gc = function () error("XYZ") end})
local u2 = setmetatable({}, {__gc = function () error("ZYX") end})
]])
RUN("lua -W %s 2> %s", prog, out)
checkprogout("ZYX)\nXYZ)\n")

-- bug since 5.2: finalizer called when closing a state could
-- subvert finalization order
-- prepfile([[
-- -- should be called last
-- print("creating 1")
-- setmetatable({}, {__gc = function () print(1) end})

-- print("creating 2")
-- setmetatable({}, {__gc = function ()
--   print("2")
--   print("creating 3")
--   -- this finalizer should not be called, as object will be
--   -- created after 'lua_close' has been called
--   setmetatable({}, {__gc = function () print(3) end})
--   print(collectgarbage() or false)    -- cannot call collector here
--   os.exit(0, true)
-- end})
-- ]])
-- RUN("lua -W %s > %s", prog, out)
-- checkout([[
-- creating 1
-- creating 2
-- 2
-- creating 3
-- false
-- 1
-- ]])

-- test many arguments
prepfile([[print(({...})[30])]])
RUN("lua %s -- %s > %s", prog, string.rep(" a", 31), out)
checkout("a\n")

-- test for error objects
prepfile("error{}")
NoRun("error object is a table value", [[lua %s]], prog)

prepfile([[#comment in 1st line without \n at the end]])
RUN("lua %s", prog)

-- first-line comment with binary file
-- prepfile("#comment\n" .. string.dump(load("print(3)")), true)
-- RUN("lua %s > %s", prog, out)
-- checkout("3\n")

-- close Lua with an open file
prepfile(string.format([[io.output(%q); io.write('alo')]], out))
RUN("lua %s", prog)
checkout("alo")

-- bug in 5.2 beta (extra \0 after version line)
RUN([[lua -v -e="print'hello'" > %s]], out)
assert(string.find(getoutput(), "hello", 1, true))

-- testing os.exit
prepfile("os.exit(nil, true)")
RUN("lua %s", prog)
prepfile("os.exit(0, true)")
RUN("lua %s", prog)
prepfile("os.exit(true, true)")
RUN("lua %s", prog)
prepfile("os.exit(1, true)")
NoRun("", "lua %s", prog) -- no message
prepfile("os.exit(false, true)")
NoRun("", "lua %s", prog) -- no message

-- to-be-closed variables in main chunk
prepfile([[
  local x <close> = setmetatable({},
        {__close = function (self, err)
                     assert(err == nil)
                     print("Ok")
                   end})
  local e1 <close> = setmetatable({}, {__close = function () print(120) end})
  os.exit(true, true)
]])
RUN("lua %s > %s", prog, out)
checkprogout("120\nOk\n")

-- remove temporary files
assert(os.remove(prog))
assert(os.remove(otherprog))
assert(not os.remove(out))

do
	-- 'warn' must get at least one argument
	local st, msg = pcall(warn)
	assert(string.find(msg, "string expected"))

	-- 'warn' does not leave unfinished warning in case of errors
	-- (message would appear in next warning)
	st, msg = pcall(warn, "SHOULD NOT APPEAR", {})
	assert(string.find(msg, "string expected"))
end

print("+")

print("testing Ctrl C")
do
	-- interrupt a script
	local function kill(pid)
		return os.execute(string.format("kill -INT %s 2> /dev/null", pid))
	end

	-- function to run a script in background, returning its output file
	-- descriptor and its pid
	local function runback(luaprg)
		-- shell script to run 'luaprg' in background and echo its pid
		local shellprg = string.format('%s -e "%s" & echo $!', progname, luaprg)
		local f = io.popen(shellprg, "r") -- run shell script
		local pid = f:read() -- get pid for Lua script
		print("(if test fails now, it may leave a Lua script running in \z
            background, pid " .. pid .. ")")
		return f, pid
	end

	-- Lua script that runs protected infinite loop and then prints '42'
	local f, pid = runback([[
    pcall(function () print(12); while true do end end); print(42)]])
	-- wait until script is inside 'pcall'
	assert(f:read() == "12")
	kill(pid) -- send INT signal to Lua script
	-- check that 'pcall' captured the exception and script continued running
	assert(f:read() == "42") -- expected output
	assert(f:close())
	print("done")

	-- Lua script in a long unbreakable search
	local f, pid = runback([[
    print(15); string.find(string.rep('a', 100000), '.*b')]])
	-- wait (so script can reach the loop)
	assert(f:read() == "15")
	assert(os.execute("sleep 1"))
	-- must send at least two INT signals to stop this Lua script
	local n = 100
	for i = 0, 100 do -- keep sending signals
		if not kill(pid) then -- until it fails
			n = i -- number of non-failed kills
			break
		end
	end
	assert(f:close())
	assert(n >= 2)
	print(string.format("done (with %d kills)", n))
end

print("OK")
