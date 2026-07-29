local t = require("internal.runtime.lib.test")
local errorTests = {}

function errorTests.testErrorInfo()
	t.assert.Error(function()
		error("hi", 0)
	end, "hi")
	t.assert.Error(function()
		error("hi", 0)
	end, "")
end

function errorTests.testCommonLuaErrors()
	t.assert.Error(function()
		return math.sin()
	end, "bad argument #1")
	t.assert.Error(function()
		return math.sin(io.input())
	end, "bad argument #1")
	t.assert.NoError(function()
		return tostring(1)
	end)
	t.assert.Error(function()
		return tostring()
	end, "bad argument #1")
	t.assert.Error(function()
		return tonumber()
	end, "bad argument #1")
	t.assert.Error(function()
		return assert(false)
	end, "assertion failed")
	t.assert.Error(function()
		return assert(nil)
	end, "assertion failed")
	t.assert.Error(function()
		local Var
		local function main()
			NoSuchName(function()
				Var = 0
			end)
		end
		main()
	end, "global 'NoSuchName'")
end

function errorTests.testCommonSyntaxErrors()
	t.assert.SyntaxError("table.unpack({}, 1, n=2 ^ 30)", "expected")
	t.assert.SyntaxError("return;;", "unexpected symbol near ';'")
	t.assert.SyntaxError("function a (... , ...) end", "expected")
	t.assert.SyntaxError("function a (, ...) end", "expected")
	t.assert.SyntaxError("local a = {4\n\n", "'}' expected (to close '{' at line 1)")
end

function errorTests.testGotoBreakErrors()
	t.assert.SyntaxError(
		[[
    ::A:: a = 1
    ::A::
  ]],
		"label 'A' already defined"
	)
	t.assert.SyntaxError(
		[[
    a = 1
    goto A
    do ::A:: end
  ]],
		"no visible label 'A'"
	)
end

function errorTests.testTableErrorMessages()
	t.assert.Error(function()
		local a = {} + 1
	end, "attempt to perform arithmetic on a table value")
	t.assert.Error(function()
		local a = {} | 1
	end, "attempt to perform bitwise operation on a table value")
	t.assert.Error(function()
		local a = {} < 1
	end, "attempt to compare table with number")
	t.assert.Error(function()
		local a = {} <= 1
	end, "attempt to compare table with number")
end

function errorTests.testBetterErrorMessages()
	t.assert.Error(function()
		local bbbb = 2
		local aaa = math.sin(3) + bbbb(3)
	end, "attempt to call a number value (local 'bbbb')")
	t.assert.Error(function()
		local aaa = {}
		do
			local aaa = 1
		end
		aaa:bbbb(3)
	end, "attempt to call a nil value (method 'bbbb')")
	t.assert.Error(function()
		local a = {}
		a.bbbb(3)
	end, "attempt to call a nil value (field 'bbbb')")
	t.assert.Error(function()
		local aaa = { 13 }
		aaa[1](3)
	end, "attempt to call a number value (field 'integer index')")
	t.assert.Error(function()
		local a = 1 .. {}
	end, "attempt to concatenate a table value")
	t.assert.Error(function()
		local a = { _ENV = {} }
		local b = a._ENV.x + 1
	end, "attempt to perform arithmetic on a nil value (field 'x')")
end

function errorTests.testCallErrors()
	t.assert.Error(function()
		local a
		a(13)
	end, "attempt to call a nil value (local 'a')")
	t.assert.Error(function()
		local a = setmetatable({}, { __add = 34 })
		a = a + 1
	end, "attempt to call a number value (metamethod 'add')")
	t.assert.Error(function()
		local a = setmetatable({}, { __lt = {} })
		a = a > a
	end, "attempt to call a table value (metamethod 'lt')")
	t.assert.Error(function()
		local a = setmetatable({}, { __index = 10 }).x
	end, "attempt to index a number value")
	t.assert.Error(function()
		local a = {}
		return a.bbbb(3)
	end, "attempt to call a nil value (field 'bbbb')")
	t.assert.Error(function()
		local aaa = {}
		do
			local aaa = 1
		end
		return aaa:bbbb(3)
	end, "attempt to call a nil value (method 'bbbb')")
	t.assert.Error(function()
		local aaa = #print
	end, "length of a function value")
	t.assert.Error(function()
		local aaa = #3
	end, "length of a number value")
	t.assert.Error(function()
		aaa.bbb:ddd(9)
	end, "attempt to index a nil value (global 'aaa')")
	t.assert.Error(function()
		local aaa = { bbb = 1 }
		aaa.bbb:ddd(9)
	end, "attempt to index a number value (field 'bbb')")
	t.assert.Error(function()
		local aaa = { bbb = {} }
		aaa.bbb:ddd(9)
	end, "attempt to call a nil value (method 'ddd')")
	t.assert.Error(function()
		local a, b, c
		(function()
			a = b + 1.1
		end)()
	end, "attempt to perform arithmetic on a nil value (upvalue 'b')")
	t.assert.NoError(function()
		local aaa = { bbb = { ddd = next } }
		return aaa.bbb:ddd(nil)
	end)
end

function errorTests.testUpvalues()
	t.assert.Error(function()
		local a, b, cc
		(function()
			a = cc[1]
		end)()
	end, "upvalue 'cc'")
	t.assert.Error(function()
		local a, b, cc
		(function()
			a.x = 1
		end)()
	end, "upvalue 'a'")
	t.assert.Error(function()
		local a, b, cc
		(function()
			a.x = 1
		end)()
	end, "upvalue 'a'")
	t.assert.Error(function()
		local _ENV = { x = {} }
		a = a + 1
	end, "global 'a'")
	t.assert.Error(function()
		BB = 1
		local aaa = {}
		x = aaa + BB
	end, "local 'aaa'")
	t.assert.Error(function()
		aaa = {}
		x = 3.3 / aaa
	end, "global 'aaa'")
	t.assert.Error(function()
		aaa = 2
		BB = nil
		x = aaa * BB
	end, "global 'BB'")
	t.assert.Error(function()
		aaa = {}
		x = -aaa
	end, "global 'aaa'")
end

function errorTests.testShortCircuit()
	t.assert.Error(function()
		aaa = 1
		local aaa, bbbb = 2, 3
		aaa = math.sin(1) and bbbb(3)
	end, "local 'bbbb'")
	t.assert.Error(function()
		aaa = 1
		local aaa, bbbb = 2, 3
		aaa = bbbb(1) or aaa(3)
	end, "local 'bbbb'")
	t.assert.Error(function()
		local a, b, c, f = 1, 1, 1
		f((a and b) or c)
	end, "local 'f'")
	t.assert.Error(function()
		local a, b, c = 1, 1, 1
		((a and b) or c)()
	end, "call a number value")
	t.assert.Error(function()
		print(print < 10)
	end, "function with number")
	t.assert.Error(function()
		print(print < print)
	end, "two function values")
	t.assert.Error(function()
		print("10" < 10)
	end, "string with number")
	t.assert.Error(function()
		print(10 < "10")
	end, "number with string")
end

function errorTests.testFloatIntConversion()
	t.assert.Error(function()
		local a = 2.0 ^ 100
		return a << 2
	end, "local 'a'")
	t.assert.Error(function()
		return 1 >> 2.0 ^ 100
	end, "has no integer representation")
	t.assert.Error(function()
		return 10.1 << 2.0 ^ 100
	end, "has no integer representation")
	t.assert.Error(function()
		return 2.0 ^ 100 & 1
	end, "has no integer representation")
	t.assert.Error(function()
		return 2.0 ^ 100 & 1e100
	end, "has no integer representation")
	t.assert.Error(function()
		return 2.0 | 1e40
	end, "has no integer representation")
	t.assert.Error(function()
		return 2e100 ~ 1
	end, "has no integer representation")
	t.assert.Error(function()
		return string.sub("a", 2.0 ^ 100)
	end, "has no integer representation")
	t.assert.Error(function()
		return string.rep("a", 3.3)
	end, "has no integer representation")
	t.assert.Error(function()
		return 6e40 & 7
	end, "has no integer representation")
	t.assert.Error(function()
		return 34 << 7e30
	end, "has no integer representation")
	t.assert.Error(function()
		return ~-3e40
	end, "has no integer representation")
	t.assert.Error(function()
		return ~-3.009
	end, "has no integer representation")
	t.assert.Error(function()
		return 3.009 & 1
	end, "has no integer representation")
	t.assert.Error(function()
		return 34 >> {}
	end, "table value")
	t.assert.Error(function()
		return 24 // 0
	end, "divide by zero")
	t.assert.Error(function()
		return 1 % 0
	end, "attempt to perform 'n%0'")
end

function errorTests.testNumericForLoops()
	t.assert.Error(function()
		for i = {}, 10 do
		end
	end, "table")
	t.assert.Error(function()
		for i = io.stdin, 10 do
		end
	end, "FILE")
	t.assert.Error(function()
		for i = 1, "x", 10 do
		end
	end, "string")
	t.assert.Error(function()
		for i = 1, {}, 10 do
		end
	end, "limit")
	t.assert.Error(function()
		for i = 1, 10, print do
		end
	end, "step")
	t.assert.Error(function()
		for i = 1, 10, print do
		end
	end, "step")
end

function errorTests.testNamedObjects()
	local a = setmetatable({}, { __name = "My Type" })
	t.assert.Eq(tostring(a), "My Type")
	t.assert.Error(function()
		io.input(a)
	end, "(FILE* expected, got My Type)")
	t.assert.Error(function()
		return a + 1
	end, "on a My Type value")
	t.assert.Error(function()
		return ~io.stdin
	end, "on a FILE* value")
	t.assert.Error(function()
		return a < a
	end, "two My Type values")
	t.assert.Error(function()
		return {} < a
	end, "table with My Type")
	t.assert.Error(function()
		return a < io.stdin
	end, "My Type with FILE*")
end

function errorTests.testErrorsWithoutDebugInfo()
	local f = function(a)
		return a + 1
	end
	f = load(string.dump(f, true))
	t.assert.Eq(f(3), 4)
	t.assert.Error(function()
		return f({})
	end, "table value")
	f = function()
		local a
		a = {}
		return a + 2
	end
	f = assert(load(string.dump(f, true)))
	t.assert.Error(function()
		return f({})
	end, "table value")
end

function errorTests.testFieldAccessConstLimit()
	local lines = {}
	for i = 1, 1000 do
		lines[i] = "aaa = x" .. i
	end
	local s = table.concat(lines, "; ")
	lines = nil

	local check = function(src, msg)
		local fn, err = load(src)
		t.assert.Nil(err)
		t.assert.Error(fn, msg)
	end

	check(s .. "; aaa = bbb + 1", "global 'bbb'")
	check("local _ENV=_ENV;" .. s .. "; aaa = bbb + 1", "global 'bbb'")
	check(s .. "; local t = {}; aaa = t.bbb + 1", "field 'bbb'")
	check(s .. "; local t = {}; t:bbb()", "field 'bbb'")
end

function errorTests.testConcatErrors()
	t.assert.Error(function()
		return print .. "a"
	end, "concatenate")
	t.assert.Error(function()
		return "a" .. false
	end, "concatenate")
	t.assert.Error(function()
		return {} .. 2
	end, "concatenate")
end

function errorTests.testGCMetaMethod()
	t.skip("not implemented")
	-- checkmessage("getmetatable(io.stdin).__gc()", "no value")
end

function errorTests.testIndexCalls()
	t.assert.Error(function()
		local aaa = {}
		setmetatable(aaa, { __index = string })
		aaa:sub()
	end, "bad argument #1 to 'string.sub' (string expected, got table)")
	t.assert.Error(function()
		local aaa = {}
		setmetatable(aaa, { __index = string })
		return string.sub("a", {})
	end, "#2")
	t.assert.Error(function()
		local aaa = {}
		setmetatable(aaa, { __index = string })
		return ("a"):sub({})
	end, "#2")
	t.assert.Error(function()
		table.sort({ 1, 2, 3 }, table.sort)
	end, "'table.sort'")
	t.assert.Error(function()
		string.gsub("s", "s", setmetatable)
	end, "'setmetatable'")
end

function errorTests.testCoroutineErrors()
	t.skip("Hangs")
	local function f()
		local c = coroutine.create(f)
		local _, b = coroutine.resume(c)
		return b
	end
	local res = f()
	t.assert.Contains(res, "C stack overflow")
	t.assert.Error(function()
		coroutine.yield()
	end, "outsite a coroutine")
	f = coroutine.wrap(function()
		table.sort({ 1, 2, 3 }, coroutine.yield)
	end)
	t.assert.Error(f, "yield across")
end

function errorTests.testErrorLine()
	local function lineerror(s, l)
		local _, msg = pcall(load(s))
		t.assert.Eq(l, tonumber(string.match(msg, ":(%d+):")))
	end
	lineerror("local a\n for i=1,'a' do \n print(i) \n end", 2)
	lineerror("\n local a \n for k,v in 3 \n do \n print(k) \n end", 3)
	lineerror("\n\n for k,v in \n 3 \n do \n print(k) \n end", 4)
	lineerror("function a.x.y ()\na=a+1\nend", 1)
	lineerror("a = \na\n+\n{}", 3)
	lineerror("a = \n3\n+\n(\n4\n/\nprint)", 6)
	lineerror("a = \nprint\n+\n(\n4\n/\n7)", 3)
	lineerror("a\n=\n-\n\nprint\n;", 3)
	lineerror(
		[[
a
(     -- <<
23)
]],
		2
	)
	lineerror(
		[[
local a = {x = 13}
a
.
x
(     -- <<
23
)
]],
		5
	)
	lineerror(
		[[
local a = {x = 13}
a
.
x
(
23 + a
)
]],
		6
	)
	local p = [[
  function g() f() end
  function f(x) error('a', XX) end
g()
]]
	XX = 3
	lineerror(p, 3)
	XX = 0
	lineerror(p, nil)
	XX = 1
	lineerror(p, 2)
	XX = 2
	lineerror(p, 1)

	lineerror(
		[[
local b = false
if not b then
  error 'test'
end]],
		3
	)

	lineerror(
		[[
local b = false
if not b then
  if not b then
    if not b then
      error 'test'
    end
  end
end]],
		5
	)

	-- bug in 5.4.0
	lineerror(
		[[
  local a = 0
  local b = 1
  local c = b % a
]],
		3
	)

	do
		local s = string.format("%s return __A.x", string.rep("\n", 300))
		lineerror(s, 301)
	end
end

function errorTests.testNonStringErrors()
	local tbl = {}
	t.assert.Error(function()
		error(tbl)
	end, tbl)
	t.assert.Error(function()
		error(nil)
	end, nil)
	t.assert.Error(function()
		assert(false, "X", tbl)
	end, "X")
	t.assert.Error(function()
		assert(false)
	end, "assertion failed!")
	t.assert.Error(function()
		assert(false, tbl)
	end, tbl)
	t.assert.Error(function()
		assert(false, nil)
	end, nil)
	t.assert.Error(function()
		assert()
	end, "value expected")
end

function errorTests.testXpcallArgs()
	local function f()
		error({ msg = "x" })
	end
	local res, msg = xpcall(f, function(r)
		return { msg = r.msg .. "y" }
	end)
	t.assert.False(res)
	t.assert.Eq(msg.msg, "xy")
	local a, b, c = xpcall(string.find, error, "alo", "al")
	t.assert.True(a)
	t.assert.Eq(b, 1)
	t.assert.Eq(c, 2)
	a, b, c = xpcall(string.find, function(x)
		return {}
	end, true, "al")
	t.assert.False(a)
	t.assert.IsTable(b)
	t.assert.Nil(c)
end

function errorTests.testSyntaxLimits()
	local function testrep(init, rep, close, repc, finalresult)
		local s = init .. string.rep(rep, 100) .. close .. string.rep(repc, 100)
		local res, _ = load(s)
		t.assert.NotNil(res)
		if finalresult then
			t.assert.Eq(res(), finalresult)
		end
		s = init .. string.rep(rep, 500)
		local res, msg = load(s)
		t.assert.Nil(res)
		t.assert.True(string.find(msg, "too many") ~= nil or string.find(msg, "overflow") ~= nil, msg)
	end

	testrep("local a; a", ",a", "= 1", ",1")
	testrep("local a; a=", "{", "0", "}")
	testrep("return ", "(", "2", ")", 2)
	testrep("local function a (x) return x end; return ", "a(", "2.2", ")", 2.2)
	testrep("", "do ", "", " end")
	testrep("", "while a do ", "", " end")
	testrep("local a; ", "if a then else ", "", " end")
	testrep("", "function foo () ", "", " end")
	testrep("local a = ''; return ", "a..", "'a'", "", "a")
	testrep("local a = 1; return ", "a^", "a", "", 1)

	local fn, err = load("a = f(x" .. string.rep(",x", 260) .. ")")
	t.assert.Nil(fn)
	t.assert.Contains(err, "too many registers")
end

function errorTests.testUpvalueLimits()
	local lim = 127
	local s = "local function fooA ()\n  local "
	for j = 1, lim do
		s = s .. "a" .. j .. ", "
	end
	s = s .. "b,c\n"
	s = s .. "local function fooB ()\n  local "
	for j = 1, lim do
		s = s .. "b" .. j .. ", "
	end
	s = s .. "b\n"
	s = s .. "function fooC () return b+c"
	local c = 1 + 2
	for j = 1, lim do
		s = s .. "+a" .. j .. "+b" .. j
		c = c + 2
	end
	s = s .. "\nend  end end"
	local _, b = load(s)
	t.assert.Greater(c, 255)
	t.assert.Contains(b, "too many upvalues")
end

function errorTests.testLocalVariablesLimits()
	local s = "\nfunction foo ()\n  local "
	for j = 1, 300 do
		s = s .. "a" .. j .. ", "
	end
	local _, b = load(s .. "b\n")
	t.assert.Contains(b, "too many local variables")
end

return errorTests
