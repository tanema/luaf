# special comment

-- Assignment tests
local a, b, c, d, e = 1, 2, 3, 4, 5
-- ensure b can use a, and the final value is discarded
local function varargReturn(x, y, ...)
	return ...
end
local x, y, z = varargReturn(a, b, c, d, e)
assert(x == 3)
assert(y == 4)
assert(z == 5)

-- Concat tests
assert((23 .. "value") == "23value", "Concat failed to output proper value")

-- GOTOs
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
assert(gotoSum == 2, "goto sum!")

-- if statement
if false then
	assert(false, "bad if statement")
elseif true then
	assert(true)
else
	assert(false, "bad if statement")
end
if false then
	assert(false, "bad if statement")
elseif false then
	assert(false, "bad if statement")
else
	assert(true)
end

-- repeat value
local repeatSum = 0
repeat
	repeatSum = repeatSum + 1
until repeatSum >= 10
assert(repeatSum == 10)

-- while loop
local whileSum = 0
while whileSum < 10 do
	whileSum = whileSum + 1
end
assert(whileSum == 10, "while loop")

-- for in loop
local forNumSum = 10
for i = 10, 1, -1 do
	forNumSum = forNumSum + i
end
assert(forNumSum == 65, "for num")

-- tables
local testTable = {1, 2, 3, foo = "bar", 22}
assert(testTable.foo == "bar", "table key index")
assert(testTable["foo"] == "bar", "table key index")
assert(testTable[1] == 1, "table index 1")
assert(testTable[2] == 2, "table index 2")
assert(testTable[3] == 3, "table index 3")
assert(testTable[4] == 22, "table index 4")

-- Function call and upvalues
local function testNoReturn()
	return
end

local function testOneReturn()
	return 22
end

local function test2Return()
	return 33, 44
end

assert(testNoReturn() == nil, "function call, no return")
assert(testOneReturn() == 22, "function call one return" )
local ret1, ret2 = test2Return()
assert(ret1 == 33, "function call 2 return")
assert(ret2 == 44, "function call 2 return")

local msg = "hello inside a function"
local function testUpval()
	return msg
end
assert(testUpval() == msg, "upvalue return!")
