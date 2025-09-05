#! lua
local t = require("test")

local mainTests = {}

function mainTests.testAssignment()
	local a, b, c, d, e = 1, 2, 3, 4, 5
	-- ensure b can use a, and the final value is discarded
	local function varargReturn(x, y, ...)
		return ...
	end

	local x, y, z = varargReturn(a, b, c, d, e)
	t.assert(x == 3, "x equals")
	t.assert(y == 4, "y equals")
	t.assert(z == 5, "z equals")
end

function mainTests.testConcat()
	t.assert((23 .. "value") == "23value", "Concat failed to output proper value")
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
	t.assert(gotoSum == 2, "goto sum!")
end

function mainTests.testIfStmt()
	if false then
		t.fail("bad if statement")
	elseif true then
		t.assert(true)
	else
		t.fail("bad if statement")
	end

	if false then
		t.fail("bad if statement")
	elseif false then
		t.fail("bad if statement")
	else
		t.assert(true, "assert if statment")
	end
end

function mainTests.testTables()
	local testTable = { 1, 2, 3, foo = "bar", 22 }
	t.assert(testTable.foo == "bar", "table key index")
	t.assert(testTable["foo"] == "bar", "table key index")
	t.assert(testTable[1] == 1, "table index 1 ")
	t.assert(testTable[2] == 2, "table index 2")
	t.assert(testTable[3] == 3, "table index 3")
	t.assert(testTable[4] == 22, "table index 4")
end

function mainTests.testFnCalls()
	local function testNoReturn() end

	local function testOneReturn()
		return 22
	end

	local function test2Return()
		return 33, 44
	end

	t.assert(testNoReturn() == nil, "function call, no return")
	t.assert(testOneReturn() == 22, "function call one return")
	local ret1, ret2 = test2Return()
	t.assert(ret1 == 33, "function call 2 return")
	t.assert(ret1 == 33, "function call 2 return")
	t.assert(ret2 == 44, "function call 2 return")

	local msg = "hello inside a function"
	local function testUpval()
		return msg
	end
	t.assert(testUpval() == msg, "upvalue return!")
end

function mainTests.testRepeatLoop()
	local repeatSum = 0
	repeat
		repeatSum = repeatSum + 1
	until repeatSum >= 10
	t.assert(repeatSum == 10, "repeat stat")
end

function mainTests.testWhileLoop()
	local whileSum = 0
	while whileSum < 10 do
		whileSum = whileSum + 1
	end
	t.assert(whileSum == 10, "while loop")
end

function mainTests.testForNumLoop()
	local forNumSum = 0
	for i = 10, 1, -1 do
		forNumSum = forNumSum + i
	end
	t.assert(forNumSum == 55, "for num" .. forNumSum)
end

function mainTests.testIPairs()
	local tbl = { 93, 22, 78, 22 }
	for key, val in ipairs(tbl) do
		t.assert(tbl[key] == val, "for in loop")
	end
end

function mainTests.testPairs()
	local tbl2 = { a = 12, b = 54, c = 99 }
	local valSums = 0
	for key, val in pairs(tbl2) do
		t.assert(key == "a" or key == "b" or key == "c")
		valSums = valSums + val
	end
	t.assert(valSums == 165, "forlist val" .. valSums)
end

function mainTests.testSelect()
	local function test()
		return 1, 2, 3, 4
	end

	local s1, s2, s3 = select(-3, test())
	t.assert(s1 == 2, "select1")
	t.assert(s2 == 3, "select3")
	t.assert(s3 == 4, "select4")
end

function mainTests.changeFnArgs()
	local obj = { count = 0 }
	local function add(arg)
		arg.count = arg.count + 1
	end
	add(obj)
	t.assert(obj.count == 1, "function call should have changed param")
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

return mainTests
