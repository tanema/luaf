local t = require("src.runtime.lib.test")
local tblTests = {}

function tblTests.testTableConcat()
	t.assertEq("1:2:3", table.concat({ 1, 2, 3 }, ":"))
	t.assertEq("1:2:3", table.concat({ 1, 2, 3 }, ":", 1))
	t.assertEq("1:2:3", table.concat({ 1, 2, 3 }, ":", 1, 3))
	t.assertEq("2:3", table.concat({ 1, 2, 3 }, ":", 2))
	t.assertEq("1:2:3", table.concat({ name = "tim", 1, 2, 3 }, ":"))
	t.assertEq("", table.concat({}))
	t.assertEq("", table.concat({}, "x"))
	t.assertEq("\0.\0.\0\1.\0.\0\1\2", table.concat({ "\0", "\0\1", "\0\1\2" }, ".\0."))
	t.assertEq("", table.concat({ "a", "b", "c" }, ",", 1, 0))
	t.assertEq("a", table.concat({ "a", "b", "c" }, ",", 1, 1))
	t.assertEq("a,b", table.concat({ "a", "b", "c" }, ",", 1, 2))
	t.assertEq("b,c", table.concat({ "a", "b", "c" }, ",", 2))
	t.assertEq("c", table.concat({ "a", "b", "c" }, ",", 3))
	t.assertEq("", table.concat({ "a", "b", "c" }, ",", 4))
end

function tblTests.testTableKeys()
	t.assertEq(0, #table.keys({ 1, 2, 3 }))
	t.assertEq(1, #table.keys({ 1, 2, 3, name = "tim" }))
end

function tblTests.testTableInsert()
	local a = {}
	table.insert(a, 2)
	t.assertEq(2, a[1])
	t.assertLen(a, 1)
	table.insert(a, 1, 1)
	t.assertEq(1, a[1])
	t.assertEq(2, a[2])
	t.assertLen(a, 2)
	table.insert(a, 3, 3)
	t.assertEq(1, a[1])
	t.assertEq(2, a[2])
	t.assertEq(3, a[3])
	t.assertLen(a, 3)
	t.assertError(function()
		table.insert(a, 40, 1)
	end)
	t.assertError(function()
		table.insert(a, -1, 1)
	end)
	t.assertError(function()
		table.insert(a, "bad arg", 1)
	end)
	t.assertError(function()
		table.insert()
	end)
end

function tblTests.testTableMove()
	t.assertError(function()
		table.move(1, 2, 3, 4)
	end)

	local a = { 10, 20, 30, 40 }
	table.move(a, 1, 4, 2, a)
	t.assertEq(a, { 10, 10, 20, 30, 40 })

	t.assertEq(table.move({ 10, 20, 30 }, 1, 3, 2), { 10, 10, 20, 30 })
	t.assertEq(table.move({ 10, 20, 30 }, 1, 3, 3), { 10, 20, 10, 20, 30 })
	t.assertEq(table.move({ 10, 20, 30 }, 2, 3, 1), { 20, 30, 30 })
	t.assertEq(table.move({ 10, 20, 30 }, 1, 3, 1, {}), { 10, 20, 30 })
	t.assertEq(table.move({ 10, 20, 30 }, 1, 0, 3, {}), {})
	t.assertEq(table.move({ 10, 20, 30 }, 1, 10, 1), { 10, 20, 30 })
end

function tblTests.testTableRemove()
	local a = { 1, 2, 3, 4 }
	t.assertError(function()
		table.remove(a, 0)
	end)
	t.assertError(function()
		table.remove(a, 10)
	end)
	local b = table.remove(a)
	t.assertEq(a, { 1, 2, 3 })
	t.assertEq(b, 4)

	b = table.remove(a, 2)
	t.assertEq(a, { 1, 3 })
	t.assertEq(b, 2)

	table.remove(a)
	table.remove(a)
	t.assertEq(a, {})
end

function tblTests.testTableSortDefault()
	local a = {}
	table.sort(a)

	a = { 5, 4, 3, 2, 1 }
	table.sort(a)
	t.assertEq({ 1, 2, 3, 4, 5 }, a)

	a = { "e", "d", "c", "b", "a" }
	table.sort(a)
	t.assertEq({ "a", "b", "c", "d", "e" }, a)

	a = { 1.5, 1.4, 1.3, 1.2, 1.1 }
	table.sort(a)
	t.assertEq({ 1.1, 1.2, 1.3, 1.4, 1.5 }, a)

	t.assertError(function()
		table.sort(a, "nope")
	end)
end

function tblTests.testTableSortFn()
	t.skip("BROKEN")
	local tbl = { 6, 5, 4, 3, 2, 1 }
	table.sort(tbl, function(a, b)
		local aeven = a % 2 == 0
		local beven = b % 2 == 0
		if aeven and beven then
			return 0
		elseif aeven and not beven then
			return -1
		elseif not aeven and beven then
			return 1
		end
		return 0
	end)
	t.assertEq({ 1, 2, 3, 4, 5, 6 }, tbl)
end

function tblTests.testTablePack()
	local a = table.pack()
	t.assertNil(a[1])
	t.assertLen(a, 0)

	a = table.pack(table)
	t.assertEq(table, a[1])
	t.assertLen(a, 1)

	a = table.pack(nil, nil, nil, nil)
	t.assertNil(a[1])
	t.assertLen(a, 4)
end

function tblTests.testTableLen()
	local a = { 1, 2, 3 }
	t.assertLen(a, 3)

	a = {}
	t.assertLen(a, 0)

	a = { test = "test" }
	t.assertLen(a, 0)

	a = {}
	local lim = 2000
	for i = 1, lim do
		a[i] = i
	end
	t.assertLen(a, lim)
end

function tblTests.testTableUnpack()
	local a = {}
	local lim = 2000
	for i = 1, lim do
		a[i] = i
	end

	t.assertLen(a, lim)
	t.assertEq(lim, select(lim, table.unpack(a)), "select last")
	t.assertEq(lim, select("#", table.unpack(a)), "select count")
	t.assertTrue(1 == table.unpack(a))
	t.assertLen({ table.unpack(a) }, lim, "unpack inside")
	t.assertEq(lim, ({ table.unpack(a) })[lim])
	t.assertEq(1, ({ table.unpack(a) })[1])
	t.assertEq(3, #{ table.unpack(a, lim - 2) })
	t.assertEq(lim - 2, ({ table.unpack(a, lim - 2) })[1])
	t.assertEq(lim, ({ table.unpack(a, lim - 2) })[3])
	t.assertNil(next({ table.unpack(a, 10, 6) })) -- no elements
	t.assertNil(next({ table.unpack(a, 11, 10) })) -- no elements

	local x, y, z = table.unpack(a, 10, 11)
	t.assertEq(10, x)
	t.assertEq(11, y)
	t.assertNil(z)

	x, y = table.unpack(a, 10, 10)
	t.assertEq(10, x)
	t.assertNil(y)

	local d, e = table.unpack({ 1 })
	t.assertEq(1, d)
	t.assertNil(e)

	d, e = table.unpack({ 1, 2 }, 1, 1)
	t.assertEq(1, d)
	t.assertNil(e)
end

return tblTests
