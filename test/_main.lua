#! lua
local t = require("src.runtime.lib.test")

local mainTests = {}

function mainTests.testAssignment()
	local a, b, c, d, e = 1, 2, 3, 4, 5
	-- ensure b can use a, and the final value is discarded
	local function varargReturn(x, y, ...)
		return ...
	end

	local x, y, z = varargReturn(a, b, c, d, e)
	t.assertEq(3, x, "x equals")
	t.assertEq(4, y, "y equals")
	t.assertEq(5, z, "z equals")
end

function mainTests.testConcat()
	t.assertEq("23value", (23 .. "value"), "Concat failed to output proper value")
end

function mainTests.testGOTO()
	local gotoSum = 0
	goto test1
	::test2::
	gotoSum = gotoSum + 1
	goto test3
	::test1::
	gotoSum = gotoSum + 1
	goto test2
	gotoSum = gotoSum + 44
	::test3::
	t.assertEq(2, gotoSum, "goto sum!")
end

function mainTests.testIfStmt()
	if false then
		t.fail("bad if statement")
	elseif true then
		t.assertTrue(true)
	else
		t.fail("bad if statement")
	end

	if false then
		t.fail("bad if statement")
	elseif false then
		t.fail("bad if statement")
	else
		t.assertTrue(true, "assert if statment")
	end
end

function mainTests.testTables()
	local testTable = { 1, 2, 3, foo = "bar", 22 }
	t.assertEq("bar", testTable.foo, "table key index")
	t.assertEq("bar", testTable["foo"], "table key index")
	t.assertEq(1, testTable[1], "table index 1 ")
	t.assertEq(2, testTable[2], "table index 2")
	t.assertEq(3, testTable[3], "table index 3")
	t.assertEq(22, testTable[4], "table index 4")
end

function mainTests.testFnCalls()
	local function testNoReturn() end

	local function testOneReturn()
		return 22
	end

	local function test2Return()
		return 33, 44
	end

	t.assertNil(testNoReturn(), "function call, no return")
	t.assertEq(22, testOneReturn(), "function call one return")
	local ret1, ret2 = test2Return()
	t.assertEq(33, ret1, "function call 2 return")
	t.assertEq(44, ret2, "function call 2 return")

	local msg = "hello inside a function"
	local function testUpval()
		return msg
	end
	t.assertEq(msg, testUpval(), "upvalue return!")
end

function mainTests.testRepeatLoop()
	local repeatSum = 0
	repeat
		repeatSum = repeatSum + 1
	until repeatSum >= 10
	t.assertEq(10, repeatSum, "repeat stat")
end

function mainTests.testWhileLoop()
	local whileSum = 0
	while whileSum < 10 do
		whileSum = whileSum + 1
	end
	t.assertEq(10, whileSum, "while loop")
end

function mainTests.testForNumLoop()
	local forNumSum = 0
	for i = 10, 1, -1 do
		forNumSum = forNumSum + i
	end
	t.assertEq(55, forNumSum, "for num" .. forNumSum)
end

function mainTests.testIPairs()
	local tbl = { 93, 22, 78, 22 }
	for key, val in ipairs(tbl) do
		t.assertEq(val, tbl[key], "for in loop")
	end
end

function mainTests.testPairs()
	local tbl2 = { a = 12, b = 54, c = 99 }
	local valSums = 0
	for key, val in pairs(tbl2) do
		t.assertTrue(key == "a" or key == "b" or key == "c")
		valSums = valSums + val
	end
	t.assertEq(165, valSums, "forlist val" .. valSums)
end

function mainTests.testSelect()
	local function test()
		return 1, 2, 3, 4
	end

	local s1, s2, s3 = select(-3, test())
	t.assertEq(2, s1, "select1")
	t.assertEq(3, s2, "select3")
	t.assertEq(4, s3, "select4")
end

function mainTests.testChangeFnArgs()
	local obj = { count = 0 }
	local function add(arg)
		arg.count = arg.count + 1
	end
	add(obj)
	t.assertEq(1, obj.count, "function call should have changed param")
end

function mainTests.testTailCall()
	local function fib(n, a, b)
		if n == 0 then
			return a
		elseif n == 1 then
			return b
		end
		return fib(n - 1, b, a + b)
	end
	fib(35, 0, 1)
end

function mainTests.testLogic()
	t.assertEq(2, 10 and 2)
	t.assertEq(10, 10 or 2)
	t.assertEq(10, 10 or false)
	t.assertTrue(not (nil and nil))
	t.assertEq("alo", nil or "alo")
	t.assertNil(nil and 10)
	t.assertFalse(false and 10)
	t.assertTrue(true or 10)
	t.assertEq(10, false or 10)
	t.assertNotEq(false, nil)
	t.assertNotEq(nil, false)
	t.assertTrue(not nil)
	t.assertFalse(not not nil)
	t.assertTrue(not not 1)
	t.assertTrue(not not (6 or nil))
	t.assertFalse(not not (nil and 56))
	t.assertFalse(not not (nil and true))
	t.assertFalse(not 10)
	t.assertFalse(not {})
	t.assertFalse(not 0.5)
	t.assertFalse(not "x")
	t.assertTrue({} ~= {})
end

function mainTests.testMultiplReturnValues()
	t.skip("BROKEN")

	local a = {}
	local b
	local function f()
		return 10, 11, 12
	end
	a.x, b, a[1] = 1, 2, f()
	t.assertEq(1, a.x)
	t.assertEq(2, b)
	t.assertEq(10, a[1])

	a[f()], b, a[f() + 3] = f(), a, "x"
	t.assertEq(10, a[10])
	t.assertEq(b, a)
	t.assertEq("x", a[13])

	local fillTable = function(n)
		local x = {}
		for i = 1, n do
			x[i] = i
		end
		return table.unpack(x)
	end

	local a, b, c
	a, b = 0, fillTable(1)
	t.assertEq(0, a)
	t.assertEq(1, b)

	a, b, c = 0, 5, fillTable(4)
	t.assertEq(0, a)
	t.assertEq(5, b)
	t.assertEq(1, c)

	a, b, c = 0, 5, fillTable(0)
	t.assertEq(0, a)
	t.assertEq(5, b)
	t.assertNil(c)
end

function mainTests.testTableBoolVals()
	local a = {}
	a[true] = 20
	a[false] = 10
	t.assertEq(20, a[1 < 2])
	t.assertEq(10, a[1 > 2])
end

function mainTests.testConflictWithMultAssign()
	t.skip("TODO")
	local a, i, j, b
	a = { "a", "b" }
	i = 1
	j = 2
	b = a
	i, a[i], a, j, a[j], a[i + j] = j, i, i, b, j, i
	t.assertEq(2, i)
	t.assertEq(i == 2 and b[1] == 1 and a == 1 and j == b and b[2] == 2 and b[3] == 1)
	t.assertEq(1, b[1])
	t.assertEq(1, a)
	t.assertEq(j, b)
	t.assertEq(2, b[2])
	t.assertEq(1, b[3])
	a = {}
	local function foo() -- assigining to upvalues
		b, a.x, a = a, 10, 20
	end
	foo()
	t.assertEq(20, a)
	t.assertEq(10, b.x)
end

function mainTests.testConflictsWithUpval()
	t.skip("TODO")
	local a, i, j, b
	a = { "a", "b" }
	i = 1
	j = 2
	b = a
	local function foo()
		i, a[i], a, j, a[j], a[i + j] = j, i, i, b, j, i
	end
	foo()
	t.assertEq(2, i)
	t.assertEq(1, b[1])
	t.assertEq(1, a)
	t.assertEq(j, b)
	t.assertEq(b[2], 2)
	t.assertEq(1, b[3])
	local t = {}
	(function(a)
		t[a], a = 10, 20
	end)(1)
	t.assertEq(10, t[1])
end

function mainTests.testUpvalCalls()
	local function foo()
		local a
		return function()
			local b
			a, b = 3, 14 -- local and upvalue have same index
			return a, b
		end
	end

	local a, b = foo()()
	t.assertEq(3, a)
	t.assertEq(14, b)
end

function mainTests.testLongNameFn()
	local a = {}
	a.aVeryLongName012345678901234567890123456789012345678901234567890123456789 = 10
	local function foo()
		return a.aVeryLongName012345678901234567890123456789012345678901234567890123456789
	end
	t.assertEq(10, foo())
	t.assertEq(10, a.aVeryLongName012345678901234567890123456789012345678901234567890123456789)
end

function mainTests.testType()
	t.assertEq("boolean", type(1 < 2))
	t.assertEq("boolean", type(true))
	t.assertEq("boolean", type(false))
	t.assertEq("nil", type(nil))
	t.assertEq("number", type(-3))
	t.assertEq("string", type("x"))
	t.assertEq("table", type({}))
	t.assertEq("function", type(type))
	t.assertEq(type(assert), type(print))
	local function f(x) end
	t.assertEq(type(f), "function")
end

return mainTests
