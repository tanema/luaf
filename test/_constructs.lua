local t = require("internal.runtime.lib.test")
local constructTests = {}

function constructTests.testIfStatElimination()
	local a = 0
	if false then
		a = 3 // 0
		a = 0 % 0
	end
	t.assert.Eq(a, 0)
end

function constructTests.testMathPriorities()
	t.assert.Eq(2 ^ 3 ^ 2, 2 ^ (3 ^ 2))
	t.assert.Eq(2 ^ 3 * 4, (2 ^ 3) * 4)
	t.assert.Eq(2.0 ^ -2, 1 / 4)
	t.assert.Eq(-2 ^ -(-2), -(-(-4)))
	t.assert.True(not nil and 2 and not (2 > 3 or 3 < 2))
	t.assert.Eq(-3 - 1 - 5, 0 + 0 - 9)
	t.assert.Eq(-2 ^ 2, -4)
	t.assert.Eq((-2) ^ 2, 4)
	t.assert.Eq(2 * 2 - 3 - 1, 0)
	t.assert.Eq(-3 % 5, 2)
	t.assert.Eq(-3 + 5, 2)
	t.assert.Eq(2 * 1 + 3 / 3, 3)
	t.assert.Eq(1 + 2 .. 3 * 1, "33")
	t.assert.True(not (2 + 1 > 3 * 1))
	t.assert.Greater("a" .. "b", "a")
	t.assert.Eq(0xF0 | 0xCC ~ 0xAA & 0xFD, 0xF4)
	t.assert.Eq(0xFD & 0xAA ~ 0xCC | 0xF0, 0xF4)
	t.assert.Eq(0xF0 & 0x0F + 1, 0x10)
	t.assert.Eq(3 ^ 4 // 2 ^ 3 // 5, 2)
	t.assert.Eq(-3 + 4 * 5 // 2 ^ 3 ^ 2 // 9 + 4 % 10 / 3, -3 + (((4 * 5) // (2 ^ (3 ^ 2))) // 9) + ((4 % 10) / 3))
	t.assert.True(not ((true or false) and nil))
	t.assert.True(true or false and nil)
	t.assert.True((((1 or false) and true) or false) == true)
	t.assert.True((((nil and true) or false) and true) == false)
	t.assert.Eq(-(1 or 2), -1)
	t.assert.Eq((1 and 2) + (-1.25 or -4), 0.75)
	t.assert.Eq((nil or 1) + 1, 2)
	t.assert.Eq((10 or 1) + 1, 11)
	t.assert.True((2 < 3) or 1)
	t.assert.Eq(2 < 3 and 4, 4)
	t.assert.True((1 > 2) and 1 or 2 == 2)
	t.assert.True((2 > 1) and 2 or 1 == 2)
	t.assert.Eq(1234567890, tonumber("1234567890"))
	t.assert.Eq(1234567890 + 1, 1234567891)
end

function constructTests.testOperators()
	local operand = { 3, 100, 5.0, -10, -5.0, 10000, -10000 }
	local operator = { "+", "-", "*", "/", "//", "%", "^", "&", "|", "^", "<<", ">>", "==", "~=", "<", ">", "<=", ">=" }
	for _, op in ipairs(operator) do
		local f = load(string.format("return function (x,y) return x %s y end", op))()
		for _, o1 in ipairs(operand) do
			for _, o2 in ipairs(operand) do
				local gab = f(o1, o2)

				_ENV.XX = o1
				t.assert.Eq(load(string.format("return XX %s %s", op, o2))(), gab)

				_ENV.XX = o2
				t.assert.Eq(load(string.format("return (%s) %s XX", o1, op))(), gab)

				t.assert.Eq(load(string.format("return (%s) %s %s", o1, op, o2))(), gab)
			end
		end
	end
	_ENV.XX = nil
end

function constructTests.testSillyLoops()
	-- should not hang or do anything silly
	repeat
	until 1
	repeat
	until true
	while false do
	end
	while nil do
	end
	for i = 1, 1000 do
		break
	end
end

function constructTests.testFirstNameNotAnUpValue()
	local a
	local function f(x)
		x = { a = 1 }
		x = { x = 1 }
		x = { G = 1 }
	end
end

function constructTests.testTableWithMoreThan256Const()
	local code = { "local x = {" }
	for i = 1, 257 do
		code[#code + 1] = i .. ".1,"
	end
	code[#code + 1] = "};"
	local tblSrc = table.concat(code)
	t.assert.Eq(load(tblSrc .. "return (1 ~ (2 or 3))")(), 1 ~ 2)
	t.assert.Eq(load(tblSrc .. "return (1 | (2 or 3))")(), 1|2)
	t.assert.Eq(load(tblSrc .. "return (1 + (2 or 3))")(), 1 + 2)
	t.assert.Eq(load(tblSrc .. "return (1 << (2 or 3))")(), 1 << 2)
end

function constructTests.testReturnValuesInTables()
	local function f(i)
		if type(i) ~= "number" then
			return i, "jojo"
		end
		if i > 0 then
			return i, f(i - 1)
		end
	end

	local x = { f(3), f(5), f(10) }
	t.assert.Eq(x[1], 3)
	t.assert.Eq(x[2], 5)
	t.assert.Eq(x[3], 10)
	t.assert.Eq(x[4], 9)
	t.assert.Eq(x[12], 1)
	t.assert.Nil(x[nil])

	x = { f("alo"), f("xixi"), nil }
	t.assert.Eq(x[1], "alo")
	t.assert.Eq(x[2], "xixi")
	t.assert.Nil(x[3])

	x = { f("alo") .. "xixi" }
	t.assert.Eq(x[1], "aloxixi")

	x = { f({}) }
	t.assert.Eq(x[2], "jojo")
	t.assert.IsTable(x[1])
end

function constructTests.testFnCall()
	local f = function(i)
		if i < 10 then
			return "a"
		elseif i < 20 then
			return "b"
		elseif i < 30 then
			return "c"
		end
	end

	t.assert.Eq(f(3), "a")
	t.assert.Eq(f(12), "b")
	t.assert.Eq(f(26), "c")
	t.assert.Nil(f(100))
end

function constructTests.testGettingFunkyWithLoopVariables()
	local n, i, tbl, a = 100, 3, {}, nil
	while not a do
		a = 0
		for i = 1, n do
			for i = i, 1, -1 do
				a = a + 1
				tbl[i] = 1
			end
		end
	end
	t.assert.Eq(a, n * (n + 1) / 2)
	t.assert.Eq(i, 3)
	t.assert.NotNil(tbl[1])
	t.assert.NotNil(tbl[n])
	t.assert.Nil(tbl[0])
	t.assert.Nil(tbl[n + 1])
end

function constructTests.testRepeatLoops()
	local function fn(b)
		local x = 1
		repeat
			local a
			if b == 1 then
				local b = 1
				x = 10
				break
			elseif b == 2 then
				x = 20
				break
			elseif b == 3 then
				x = 30
			else
				local a, b, c, d = math.sin(1)
				x = x + 1
			end
		until x >= 12
		return x
	end
	t.assert.Eq(fn(1), 10)
	t.assert.Eq(fn(2), 20)
	t.assert.Eq(fn(3), 30)
	t.assert.Eq(fn(4), 12)
end

function constructTests.testIfStat()
	local fn = function(i)
		if i < 10 then
			return "a"
		elseif i < 20 then
			return "b"
		elseif i < 30 then
			return "c"
		else
			return 8
		end
	end
	t.assert.Eq(fn(3), "a")
	t.assert.Eq(fn(12), "b")
	t.assert.Eq(fn(26), "c")
	t.assert.Eq(fn(100), 8)
	t.assert.Eq(fn(100) * 2 + 3 or nil, 19)
end

function constructTests.testWhileLoops()
	local function f(i)
		while 1 do
			if i > 0 then
				i = i - 1
			else
				return
			end
		end
	end

	local function g(i)
		while 1 do
			if i > 0 then
				i = i - 1
			else
				return
			end
		end
	end

	f(10)
	g(10)
end

function constructTests.testReturnValues()
	function f()
		return 1, 2, 3
	end

	local a, b, c = f()
	t.assert.Eq(a, 1)
	t.assert.Eq(b, 2)
	t.assert.Eq(c, 3)

	a, b, c = (f())
	t.assert.Eq(a, 1)
	t.assert.Nil(b)
	t.assert.Nil(c)

	a, b = 3 and f()
	t.assert.Eq(a, 1)
	t.assert.Nil(b)

	local function g()
		f()
		return
	end

	t.assert.Nil(g())
	local function fg()
		return nil or f()
	end

	a, b = fg()
	t.assert.Eq(a, 1)
	t.assert.Nil(b)
end

function constructTests.testConstants()
	local prog <const> = [[local x <XXX> = 10]]
	t.assert.SyntaxError(prog, "unknown attribute 'XXX'")
	t.assert.SyntaxError([[local xxx <const> = 20; xxx = 10]], "attempt to assign to const variable 'xxx'")
	t.assert.SyntaxError([[
    local xx;
    local xxx <const> = 20;
    local yyy;
    local function foo ()
      local abc = xx + yyy + xxx;
      return function () return function () xxx = yyy end end
    end
  ]],
		"attempt to assign to const variable 'xxx'"
	)
	t.assert.SyntaxError(
		[[
    local x <close> = nil
    x = io.open()
  ]],
		"attempt to assign to const variable 'x'"
	)
end

function constructTests.testShortCircuit()
	_ENV.GLOB1 = math.random(0, 1)
	local basiccases = {
		{ "nil",             nil },
		{ "false",           false },
		{ "true",            true },
		{ "10",              10 },
		{ "(0==_ENV.GLOB1)", 0 == _ENV.GLOB1 },
	}

	local prog
	if _ENV.GLOB1 == 0 then
		basiccases[2][1] = "F" -- constant false

		prog = [[
    local F <const> = false
    if %s then IX = true end
    return %s
]]
	else
		basiccases[4][1] = "k10" -- constant 10

		prog = [[
    local k10 <const> = 10
    if %s then IX = true end
    return %s
  ]]
	end

	local binops <const> = {
		{
			" and ",
			function(a, b)
				if not a then
					return a
				else
					return b
				end
			end,
		},
		{
			" or ",
			function(a, b)
				if a then
					return a
				else
					return b
				end
			end,
		},
	}

	local cases <const> = {}
	local function createcases(n)
		local res = {}
		for i = 1, n - 1 do
			for _, v1 in ipairs(cases[i]) do
				for _, v2 in ipairs(cases[n - i]) do
					for _, op in ipairs(binops) do
						local t = {
							"(" .. v1[1] .. op[1] .. v2[1] .. ")",
							op[2](v1[2], v2[2]),
						}
						res[#res + 1] = t
						res[#res + 1] = { "not" .. t[1], not t[2] }
					end
				end
			end
		end
		return res
	end

	cases[1] = basiccases
	for i = 2, 3 do
		cases[i] = createcases(i)
	end

	local i = 0
	for n = 1, 3 do
		for _, v in pairs(cases[n]) do
			local s = v[1]
			local p = load(string.format(prog, s, s), "")
			IX = false
			t.assert.Eq(p(), v[2])
			t.assert.Eq(IX, not not v[2])
			i = i + 1
		end
	end

	IX = nil
	_ENV.GLOB1 = nil
end

function constructTests.testSyntaxErrors()
	t.assert.SyntaxError("for x do", "malformed for statement")
	t.assert.SyntaxError("x:call", "expected")
end

return constructTests
