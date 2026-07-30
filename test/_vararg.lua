local t = require("internal.runtime.lib.test")
local varargTests = {}

function varargTests.testVarArgCountAndSpread()
	local function none(a)
		return nil
	end
	local function vararg(...)
		return { n = select("#", ...), ... }
	end
	local call = function(f, args)
		return f(table.unpack(args))
	end
	local function countArgs(a, ...)
		local x = { n = select("#", ...), ... }
		for i = 1, x.n do
			t.assert.Eq(a[i], x[i])
		end
		return x.n
	end
	local function c12(...)
		local x = { ... }
		x.n = #x
		local res = (x.n == 2 and x[1] == 1 and x[2] == 2)
		if res then
			res = 55
		end
		return res, 2
	end
	t.assert.Eq(countArgs(), 0)
	t.assert.Eq(countArgs({ 1, 2, 3 }, 1, 2, 3), 3)
	t.assert.Eq(countArgs({ "alo", nil, 45, countArgs, nil }, "alo", nil, 45, countArgs, nil), 5)
	t.assert.Eq(vararg().n, 0)
	t.assert.Eq(vararg(nil, nil).n, 2)
	local a = vararg(call(next, { _G, nil, n = 2 }))
	local b, c = next(_G)
	t.assert.Eq(a[1], b)
	t.assert.Eq(a[2], c)
	t.assert.Eq(a.n, 2)
	a = vararg(call(call, { c12, { 1, 2 } }))
	t.assert.Eq(a[1], 55)
	t.assert.Eq(a[2], 2)
	t.assert.Eq(a.n, 2)
	t.assert.Eq(c12(1, 2), 55)
	a, b = call(c12, { 1, 2 })
	t.assert.Eq(a, 55)
	t.assert.Eq(b, 2)
	a, b = call(c12, { 1, 2, n = 2 })
	t.assert.Eq(a, 55)
	t.assert.Eq(b, 2)
	t.assert.False(c12(1, 2, 3))
	a = call(none, { "+" })
	t.assert.Nil(a, nil)
end

function varargTests.testMethodCallVarArg()
	local tbl = { 1, 10 }
	function tbl:f(...)
		local arg = { ... }
		return self[...] + #arg
	end

	t.assert.Eq(tbl:f(1, 4), 3)
	t.assert.Eq(tbl:f(2), 11)
end

function varargTests.testTableSpread()
	local call = function(f, args)
		return f(table.unpack(args))
	end
	local i, a, lim = 1, {}, 20
	while i <= lim do
		a[i] = i + 0.3
		i = i + 1
	end

	function f(a, b, c, d, ...)
		local more = { ... }
		t.assert.Eq(1.3, a)
		t.assert.Eq(5.3, more[1])
		t.assert.Eq(lim + 0.3, more[lim - 4])
		t.assert.False(more[lim - 3])
	end

	local function g(a, b, c)
		t.assert.Eq(a, 1.3)
		t.assert.Eq(b, 2.3)
		t.assert.Eq(c, 3.3)
	end

	call(f, a)
	call(g, a)
end

function varargTests.testBuiltingFns()
	local call = function(f, args)
		return f(table.unpack(args))
	end
	local a, lim = {}, 20
	for i = 1, lim do
		a[i] = i
	end
	t.assert.Eq(lim, call(math.max, a))
end

function varargTests.testNewStyleVarargs()
	local function oneless(a, ...)
		return ...
	end

	local function f(n, a, ...)
		local b
		t.assert.Eq(arg, _G.arg) -- no local 'arg'
		if n == 0 then
			local b, c, d = ...
			return a, b, c, d, oneless(oneless(oneless(...)))
		else
			n, b, a = n - 1, ..., a
			t.assert.Eq(b, ...)
			return f(n, a, ...)
		end
	end

	local a, b, c, d, e = f(10, 5, 4, 3, 2, 1)
	t.assert.Eq(a, 5)
	t.assert.Eq(b, 4)
	t.assert.Eq(c, 3)
	t.assert.Eq(d, 2)
	t.assert.Eq(e, 1)

	a, b, c, d, e = f(4)
	t.assert.Nil(a)
	t.assert.Nil(b)
	t.assert.Nil(c)
	t.assert.Nil(d)
	t.assert.Nil(e)
end

function varargTests.testMainChunkVararg()
	local x = load([[ return {...} ]])(2, 3)
	t.assert.Eq(x[1], 2)
	t.assert.Eq(x[2], 3)
	t.assert.Nil(x[3])
end

function varargTests.testLoadVararg()
	local f = load([[
  local x = {...}
  for i=1,select('#', ...) do assert(x[i] == select(i, ...)) end
  assert(x[select('#', ...)+1] == undef)
  return true
]])

	t.assert.True(f("a", "b", nil, {}, assert))
	t.assert.True(f())

	local a = { select(3, table.unpack({ 10, 20, 30, 40 })) }
	t.assert.Len(a, 2)
	t.assert.Eq(a[1], 30)
	t.assert.Eq(a[2], 40)

	a = { select(1) }
	t.assert.Nil(next(a))

	a = { select(-1, 3, 5, 7) }
	t.assert.Eq(a[1], 7)
	t.assert.Nil(a[2])

	a = { select(-2, 3, 5, 7) }
	t.assert.Eq(a[1], 5)
	t.assert.Eq(a[2], 7)
	t.assert.Nil(a[3])
end

function varargTests.testTooManyParams()
	local function f(
		p1,
		p2,
		p3,
		p4,
		p5,
		p6,
		p7,
		p8,
		p9,
		p10,
		p11,
		p12,
		p13,
		p14,
		p15,
		p16,
		p17,
		p18,
		p19,
		p20,
		p21,
		p22,
		p23,
		p24,
		p25,
		p26,
		p27,
		p28,
		p29,
		p30,
		p31,
		p32,
		p33,
		p34,
		p35,
		p36,
		p37,
		p38,
		p39,
		p40,
		p41,
		p42,
		p43,
		p44,
		p45,
		p46,
		p48,
		p49,
		p50,
		...
	)
		local a1, a2, a3, a4, a5, a6, a7
		local a8, a9, a10, a11, a12, a13, a14
	end
	f()
end

function varargTests.testMissingArgumentsInTailCall()
	local function f(a, b, c)
		return c, b
	end
	local function g()
		return f(1, 2)
	end
	local a, b = g()
	assert(a == nil and b == 2)
end

return varargTests
