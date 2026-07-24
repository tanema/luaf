local t = require("internal.runtime.lib.test")
local callTests = {}

function callTests.testLocalFuncRecursion()
	local res = 1
	local function fact(n)
		if n == 0 then
			return res
		else
			return n * fact(n - 1)
		end
	end
	t.assert.Eq(120, fact(5))
end

function callTests.testTableFnCallsSelf()
	local a = { i = 10 }
	local self = 20
	function a:x(x)
		return x + self.i
	end

	function a.y(x)
		return x + self
	end

	t.assert.Eq(a:x(1) + 10, a.y(1))

	a.t = { i = -100 }
	a["t"].x = function(self, a, b)
		return self.i + a + b
	end
	t.assert.Eq(-95, a.t:x(2, 3))
end

function callTests.testTableFnCallsMutateSelf()
	local a = { x = 0 }
	function a:add(x)
		self.x, a.y = self.x + x, 20
		return self
	end

	a:add(10):add(20):add(30)
	t.assert.Eq(60, a.x)
	t.assert.Eq(20, a.y)
end

function callTests.testTableFnCalls()
	local a = { b = { c = {} } }
	function a.b.c.f1(x)
		return x + 1
	end

	function a.b.c:f2(x, y)
		self[x] = y
	end

	t.assert.Eq(5, a.b.c.f1(4))
	a.b.c:f2("k", 12)
	t.assert.Eq(12, a.b.c.k)
end

function callTests.testFnParamMatch()
	local tbl = {}
	local function f(a, b, c)
		local d = "a"
		tbl = { a, b, c, d }
	end
	f(1, 2)
	t.assert.Eq(tbl[1], 1)
	t.assert.Eq(tbl[2], 2)
	t.assert.Eq(tbl[3], nil)
	t.assert.Eq(tbl[4], "a")
	f(1, 2, 3, 4)
	t.assert.Eq(tbl[1], 1)
	t.assert.Eq(tbl[2], 2)
	t.assert.Eq(tbl[3], 3)
	t.assert.Eq(tbl[4], "a")
end

function callTests.testLoadFnCall()
	function fat(x)
		if x <= 1 then
			return 1
		else
			return x * load("return fat(" .. x - 1 .. ")", "")()
		end
	end

	local a, b = load("return fat(5), 3")()
	t.assert.Eq(a, 120)
	t.assert.Eq(b, 3)
	fat = nil
end

function callTests.testFnDeclarationsScope()
	local function err_on_n(n)
		if n == 0 then
			error()
			os.exit(1)
		else
			err_on_n(n - 1)
			os.exit(1)
		end
	end

	do
		local function dummy(n)
			if n > 0 then
				t.assert.False(pcall(err_on_n, n))
				dummy(n - 1)
			end
		end

		dummy(10)
	end
end

function callTests.testTailCalls()
	local function deep(n)
		if n > 0 then
			deep(n - 1)
		end
	end

	deep(10)
	deep(180)

	local function deep2(n)
		if n > 0 then
			return deep2(n - 1)
		else
			return 101
		end
	end
	t.assert.Eq(deep2(30000), 101)

	local a = {}
	function a:deep(n)
		if n > 0 then
			return self:deep(n - 1)
		else
			return 101
		end
	end

	t.assert.Eq(a:deep(30000), 101)
end

function callTests.testTailCallsVarargs()
	local function foo(x, ...)
		local a = { ... }
		return x, a[1], a[2]
	end

	local function foo1(x)
		return foo(10, x, x + 1)
	end

	local a, b, c = foo1(-2)
	t.assert.Eq(a, 10)
	t.assert.Eq(b, -2)
	t.assert.Eq(c, -1)
end

function callTests.testTailCallsMetamethods()
	local function foo(x, ...)
		local a = { ... }
		return x, a[1], a[2]
	end

	local tbl = setmetatable({}, { __call = foo })
	local function bar(x)
		return tbl(10, x)
	end
	local a, b, c = bar(100)
	t.assert.Eq(a, tbl)
	t.assert.Eq(b, 10)
	t.assert.Eq(c, 100)

	a, b = (function()
		return foo()
	end)()
	t.assert.Eq(a, nil)
	t.assert.Eq(b, nil)

	local X, Y, A
	local function foo1(x, y, ...)
		X = x
		Y = y
		A = { ... }
	end
	local function foo2(...)
		return foo1(...)
	end

	foo2()
	t.assert.Nil(X)
	t.assert.Nil(Y)
	t.assert.Len(A, 0)

	foo2(10)
	t.assert.Eq(X, 10)
	t.assert.Nil(Y)
	t.assert.Len(A, 0)

	foo2(10, 20)
	t.assert.Eq(X, 10)
	t.assert.Eq(Y, 20)
	t.assert.Len(A, 0)

	foo2(10, 20, 30)
	t.assert.Eq(X, 10)
	t.assert.Eq(Y, 20)
	t.assert.Len(A, 1)
	t.assert.Eq(A[1], 30)
end

function callTests.testPcallStackOverflow()
	local function loop()
		pcall(loop)
	end

	local err, msg = xpcall(loop, loop)
	t.assert.True(err)
	t.assert.Nil(msg)
end

function callTests.testTailCallChain_Call()
	local n = 10000
	local function foo()
		if n == 0 then
			return 1023
		else
			n = n - 1
			return foo()
		end
	end

	for i = 1, 15 do
		foo = setmetatable({}, { __call = foo })
	end

	t.assert.Eq(
		coroutine.wrap(function()
			return foo()
		end)(),
		1023
	)
end

function callTests.testCallChain_Call()
	local N = 15
	local u = table.pack
	for i = 1, N do
		u = setmetatable({ i }, { __call = u })
	end

	local res = u("a", "b", "c")
	t.assert.Eq(res.n, N + 3)
	for i = 1, N do
		t.assert.Eq(res[i][1], i)
	end
	t.assert.Eq(res[N + 1], "a")
	t.assert.Eq(res[N + 2], "b")
	t.assert.Eq(res[N + 3], "c")
end

function callTests.testChainsTooLong()
	local a = {}
	for i = 1, 16 do -- one too many
		a = setmetatable({}, { __call = a })
	end
	local status, msg = pcall(a)
	t.assert.False(status)
	t.assert.True(string.find(msg, "too long"))

	setmetatable(a, { __call = a }) -- infinite chain
	status, msg = pcall(a)
	t.assert.False(status)
	t.assert.True(string.find(msg, "too long"))

	status, msg = pcall(function()
		return a()
	end)
	t.assert.False(status)
	t.assert.True(string.find(msg, "too long"))
end

function callTests.testClosures()
	local a = nil
	(function(x)
		a = x
	end)(23)
	t.assert.Eq(a, 23)
	t.assert.Eq(
		(function(x)
			return x * 2
		end)(20),
		40
	)
end

function callTests.testClosureParams()
	local Z = function(le) -- fixed-point operator
		local function a(f)
			return le(function(x)
				return f(f)(x)
			end)
		end
		return a(a)
	end
	local F = function(f) -- non-recursive factorial
		return function(n)
			if n == 0 then
				return 1
			else
				return n * f(n - 1)
			end
		end
	end
	local fat = Z(F)

	t.assert.Eq(fat(0), 1)
	t.assert.Eq(fat(4), 24)
	t.assert.Eq(Z(F)(5), 5 * Z(F)(4))

	local function g(z)
		local function f(a, b, c, d)
			return function(x, y)
				return a + b + c + d + a + x + y + z
			end
		end
		return f(z, z + 1, z + 2, z + 3)
	end

	t.assert.Eq(g(10)(9, 16), 10 + 11 + 12 + 13 + 10 + 9 + 16 + 10)
end

function callTests.testMultipleReturns()
	local function unlpack(j, i)
		i = i or 1
		if i <= #j then
			return j[i], unlpack(j, i + 1)
		end
	end

	local function equaltab(t1, t2)
		t.assert.Eq(#t1, #t2)
		for i = 1, #t1 do
			t.assert.Eq(t1[i], t2[i])
		end
	end

	local pack = function(...)
		return (table.pack(...))
	end
	local f = function()
		return 1, 2, 30, 4
	end
	local ret2 = function(a, b)
		return a, b
	end
	local a, b, c, d = unlpack({ 1, 2, 3 })

	t.assert.Eq(a, 1)
	t.assert.Eq(b, 2)
	t.assert.Eq(c, 3)
	t.assert.Nil(d)

	a = { 1, 2, 3, 4, false, 10, "alo", false, assert }
	equaltab(pack(unlpack(a)), a)
	equaltab(pack(unlpack(a), -1), { 1, -1 })

	a, b, c, d = ret2(f()), ret2(f())
	t.assert.Eq(a, 1)
	t.assert.Eq(b, 1)
	t.assert.Eq(c, 2)
	t.assert.Nil(d)

	a, b, c, d = unlpack(pack(ret2(f()), ret2(f())))
	t.assert.Eq(a, 1)
	t.assert.Eq(b, 1)
	t.assert.Nil(c)
	t.assert.Nil(d)

	a, b, c, d = unlpack(pack(ret2(f()), (ret2(f()))))
	t.assert.Eq(a, 1)
	t.assert.Eq(b, 1)
	t.assert.Nil(c)
	t.assert.Nil(d)

	a = ret2({ unlpack({ 1, 2, 3 }), unlpack({ 3, 2, 1 }), unlpack({ "a", "b" }) })
	t.assert.Eq(a[1], 1)
	t.assert.Eq(a[2], 3)
	t.assert.Eq(a[3], "a")
	t.assert.Eq(a[4], "b")
end

function callTests.testIncorrectArguments()
	rawget({}, "x", 1)
	rawset({}, "x", 1, 2)
	t.assert.Eq(math.sin(1, 2), math.sin(1))
	table.sort({ 10, 9, 8, 4, 19, 23, 0, 0 }, function(a, b)
		return a < b
	end, "extra arg")
end

function callTests.testGenericLoad()
	local x = "-- a comment\0\0\0\n  x = 10 + \n23; \
     local a = function () x = 'hi' end; \
     return '\0'"
	local function read1(input)
		local i = 0
		return function()
			i = i + 1
			return string.sub(input, i, i)
		end
	end

	local function cannotload(msg, a, b)
		t.assert.Nil(a)
		t.assert.True(string.find(b, msg))
	end

	local a = load(read1(x), "modname", "t", _G)
	t.assert.Eq(a(), "\0")
	t.assert.Eq(_G.x, 33)
	-- cannot read text in binary mode
	cannotload("attempt to load a text chunk", load(read1(x), "modname", "b", {}))
	cannotload("attempt to load a text chunk", load(x, "modname", "b"))

	a = load(function()
		return nil
	end)
	a() -- empty chunk

	t.assert.False(load(function()
		return true
	end))

	-- small bug
	local chunks = { nil, "return ", "3" }
	local f = load(function()
		return table.remove(chunks, 1)
	end)
	t.assert.Eq(f(), nil) -- should read the empty chunk

	-- bug in 5.2.1
	f = load(
		string.dump(function()
			return 1
		end),
		nil,
		"b",
		{}
	)
	t.assert.Eq(type(f), "function")
	t.assert.Eq(f(), 1)

	x = string.dump(load("x = 1; return x"))
	a = assert(load(read1(x), nil, "b"))
	t.assert.Eq(a(), 1)
	t.assert.Eq(_G.x, 1)
	cannotload("attempt to load a binary chunk", load(read1(x), nil, "t"))
	cannotload("attempt to load a binary chunk", load(x, nil, "t"))
	_G.x = nil

	t.assert.False(pcall(string.dump, print)) -- no dump of C functions

	cannotload("unexpected symbol", load(read1("*a = 123")))
	cannotload("unexpected symbol", load("*a = 123"))
	cannotload(
		"hhi",
		load(function()
			error("hhi")
		end)
	)

	-- any value is valid for _ENV
	t.assert.Eq(load("return _ENV", nil, nil, 123)(), 123)

	-- load when _ENV is not first upvalue
	local x
	XX = 123
	local function h()
		local y = x -- use 'x', so that it becomes 1st upvalue
		return XX -- global name
	end
	local d = string.dump(h)
	t.assert.Eq(load(d, "", "b")(), 123)
	t.assert.Eq(load("return XX + ...", nil, nil, { XX = 13 })(4), 17)
	XX = nil

	x = [[
  return function (x)
    return function (y)
     return function (z)
       return x+y+z
     end
   end
  end
]]
	a = load(read1(x), "read", "t")
	t.assert.Eq(a()(2)(3)(10), 15)

	-- repeat the test loading a binary chunk
	x = string.dump(a)
	a = load(read1(x), "read", "b")
	t.assert.Eq(a()(2)(3)(10), 15)
end

function callTests.testDumpUndumpWithValues()
	local a, b = 20, 30
	local x = load(
		string.dump(function(x)
			if x == "set" then
				a = 10 + b
				b = b + 1
			else
				return a
			end
		end),
		"",
		"b",
		nil
	)
	-- a and b are not preserved across the dump/load round trip - they start
	-- as fresh, independent nils.
	t.assert.Nil(x())
	t.assert.Error(function()
		x("set")
	end)
end

function callTests.testDumpUndumpWithManyValues()
	local nup = 200 -- maximum number of local variables

	local prog = { "local a1" }
	for i = 2, nup do
		prog[#prog + 1] = ", a" .. i
	end
	prog[#prog + 1] = " = 1"
	for i = 2, nup do
		prog[#prog + 1] = ", " .. i
	end
	local sum = 1
	prog[#prog + 1] = "; return function () return a1"
	for i = 2, nup do
		prog[#prog + 1] = " + a" .. i
		sum = sum + i
	end
	prog[#prog + 1] = " end"

	local f = load(table.concat(prog))()
	t.assert.Eq(f(), sum)

	f = load(string.dump(f)) -- main chunk now has many upvalues
	t.assert.Eq(type(f), "function")
	t.assert.Error(f) -- upvalues are not preserved across a dump/load round trip
end

function callTests.testFnLongNames()
	local tbl = { x = 1 }
	function tbl:_012345678901234567890123456789012345678901234567890123456789()
		return self.x
	end

	t.assert.Eq(tbl:_012345678901234567890123456789012345678901234567890123456789(), 1)
end

function callTests.testFnParameterAdjustment()
	t.assert.Nil((function()
		return nil
	end)(4))
	t.assert.Nil((function()
		local a
		return a
	end)(4))
	t.assert.Nil((function(a)
		return a
	end)())
end

function callTests.testBinaryChunks()
	local c = string.dump(function()
		local a = 1
		local b = 3
		local f = function()
			return a + b + _ENV.c
		end -- upvalues
		local s1 = "a constant"
		local s2 = "another constant"
		return a + b * 3
	end)

	t.assert.Eq(load(c)(), 10)

	-- corrupting the first byte breaks the dump's own signature check
	local corrupted = string.char(string.byte(c, 1) + 1) .. string.sub(c, 2, -1)
	t.assert.Eq(#corrupted, #c)
	t.assert.False(load(corrupted))

	-- loading truncated binary chunks should always fail, whatever the
	-- specific error (an incomplete signature reads as text and hits a
	-- lexer error; anything longer hits an undump EOF error).
	for i = 1, #c - 1 do
		local st = load(string.sub(c, 1, i))
		t.assert.False(st)
	end
end

function callTests.testReuseStringsInDumps()
	local str = "|" .. string.rep("X", 50) .. "|"
	local foo = load(string.format(
		[[
    local str <const> = "%s"
    return {
      function () return str end,
      function () return str end,
      function () return str end
    }
  ]],
		str
	))
	-- count occurrences of 'str' inside the dump
	local dump = string.dump(foo)
	local _, count = string.gsub(dump, str, {})
	-- the string constant is stored once and shared by all 3 closures - this
	-- dump format has no separate embedded source/debug text, so it doesn't
	-- duplicate it the way a debug-info-inclusive dump would.
	t.assert.Eq(count, 1)
end

function callTests.testLimitofMultiplReturns254()
	local code = "return 10" .. string.rep(",10", 253)
	local res = { assert(load(code))() }
	t.assert.Eq(#res, 254)
	t.assert.Eq(res[254], 10)

	code = code .. ",10"
	local status, msg = load(code)
	t.assert.Nil(status)
	t.assert.True(string.find(msg, "too many returns"))
end

return callTests
