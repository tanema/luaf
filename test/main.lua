# special comment

-- Assignment tests
print("ASSIGNMENT STATEMENT TESTS")
local a, b, c, d, e = 1, 2, 3, 4, 5
-- ensure b can use a, and the final value is discarded
local function varargReturn(x, y, ...)
	return ...
end
local x, y, z = varargReturn(a, b, c, d, e)
assert(x == 3, "x equals")
assert(y == 4, "y equals")
assert(z == 5, "z equals")

-- Concat tests
print("CONCAT STATEMENT TESTS")
assert((23 .. "value") == "23value", "Concat failed to output proper value")

-- GOTOs
print("GOTO STATEMENT TESTS")
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
print("IF STATEMENT TESTS")
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
	assert(true, "assert if statment")
end

-- tables
print("TABLE STATEMENT TESTS")
local testTable = {1, 2, 3, foo = "bar", 22}
assert(testTable.foo == "bar", "table key index")
assert(testTable["foo"] == "bar", "table key index")
assert(testTable[1] == 1, "table index 1")
assert(testTable[2] == 2, "table index 2")
assert(testTable[3] == 3, "table index 3")
assert(testTable[4] == 22, "table index 4")

-- Function call and upvalues
print("RETURN STATEMENT TESTS")
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

-- repeat value
print("REPEAT STATEMENT TESTS")
local repeatSum = 0
repeat
	repeatSum = repeatSum + 1
until repeatSum >= 10
assert(repeatSum == 10, "repeat stat")

-- while loop
print("WHILE STATEMENT TESTS")
local whileSum = 0
while whileSum < 10 do
	whileSum = whileSum + 1
end
assert(whileSum == 10, "while loop")

-- for num loop
print("LOOP STATEMENT TESTS")
local forNumSum = 0
for i = 10, 1, -1 do
	forNumSum = forNumSum + i
end
assert(forNumSum == 65, "for num")

local tbl = {93, 22, 78, 22}
for i, val in ipairs(tbl) do
	assert(tbl[i] == val, "for in loop")
end

local tbl2 = {a = 12, b = 54, c = 99}
for key, val in pairs(tbl2) do
	print(key, val, tbl2[key])
end
print("done.")
