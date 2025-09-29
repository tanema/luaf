-- $Id: testes/attrib.lua $
-- See Copyright Notice in file all.lua
print("testing assignments, logical operators, and constructors")

local res, res2 = 27

local a, b = 1, 2 + 3
assert(a == 1 and b == 5)
a = {}
local function f()
	return 10, 11, 12
end
a.x, b, a[1] = 1, 2, f()
assert(a.x == 1 and b == 2 and a[1] == 10)
a[f()], b, a[f() + 3] = f(), a, "x"
assert(a[10] == 10 and b == a and a[13] == "x")

do
	local f = function(n)
		local x = {}
		for i = 1, n do
			x[i] = i
		end
		return table.unpack(x)
	end
	local a, b, c
	a, b = 0, f(1)
	assert(a == 0 and b == 1)
	a, b = 0, f(1)
	assert(a == 0 and b == 1)
	a, b, c = 0, 5, f(4)
	assert(a == 0 and b == 5 and c == 1)
	a, b, c = 0, 5, f(0)
	assert(a == 0 and b == 5 and c == nil)
end

local a, b, c, d = 1 and nil, 1 or nil, (1 and (nil or 1)), 6
assert(not a and b and c and d == 6)

d = 20
a, b, c, d = f()
assert(a == 10 and b == 11 and c == 12 and d == nil)
a, b = f(), 1, 2, 3, f()
assert(a == 10 and b == 1)

assert(a < b == false and a > b == true)
assert((10 and 2) == 2)
assert((10 or 2) == 10)
assert((10 or assert(nil)) == 10)
assert(not (nil and assert(nil)))
assert((nil or "alo") == "alo")
assert((nil and 10) == nil)
assert((false and 10) == false)
assert((true or 10) == true)
assert((false or 10) == 10)
assert(false ~= nil)
assert(nil ~= false)
assert(not nil == true)
assert(not not nil == false)
assert(not not 1 == true)
assert(not not a == true)
assert(not not (6 or nil) == true)
assert(not not (nil and 56) == false)
assert(not not (nil and true) == false)
assert(not 10 == false)
assert(not {} == false)
assert(not 0.5 == false)
assert(not "x" == false)

assert({} ~= {})
print("+")

a = {}
a[true] = 20
a[false] = 10
assert(a[1 < 2] == 20 and a[1 > 2] == 10)

function f(a)
	return a
end

local a = {}
for i = 3000, -3000, -1 do
	a[i + 0.0] = i
end
a[10e30] = "alo"
a[true] = 10
a[false] = 20
assert(a[10e30] == "alo" and a[not 1] == 20 and a[10 < 20] == 10)
for i = 3000, -3000, -1 do
	assert(a[i] == i)
end
a[print] = assert
a[f] = print
a[a] = a
assert(a[a][a][a][a][print] == assert)
a[print](a[a[f]] == a[print])
assert(not pcall(function()
	local a = {}
	a[nil] = 10
end))
assert(not pcall(function()
	local a = { [nil] = 10 }
end))
assert(a[nil] == undef)
a = nil

local a, b, c
a = { 10, 9, 8, 7, 6, 5, 4, 3, 2, [-3] = "a", [f] = print, a = "a", b = "ab" }
a, a.x, a.y = a, a[-3]
assert(a[1] == 10 and a[-3] == a.a and a[f] == print and a.x == "a" and not a.y)
a[1], f(a)[2], b, c = { ["alo"] = assert }, 10, a[1], a[f], 6, 10, 23, f(a), 2
a[1].alo(a[2] == 10 and b == 10 and c == print)

a.aVeryLongName012345678901234567890123456789012345678901234567890123456789 = 10
local function foo()
	return a.aVeryLongName012345678901234567890123456789012345678901234567890123456789
end
assert(foo() == 10 and a.aVeryLongName012345678901234567890123456789012345678901234567890123456789 == 10)

do
	-- _ENV constant
	local function foo()
		local _ENV <const> = 11
		X = "hi"
	end
	local st, msg = pcall(foo)
	assert(not st and string.find(msg, "number"))
end

-- test of large float/integer indices

-- compute maximum integer where all bits fit in a float
local maxint = math.maxinteger

-- trim (if needed) to fit in a float
while maxint ~= (maxint + 0.0) or (maxint - 1) ~= (maxint - 1.0) do
	maxint = maxint // 2
end

local maxintF = maxint + 0.0 -- float version

assert(maxintF == maxint and math.type(maxintF) == "float" and maxintF >= 2.0 ^ 14)

-- floats and integers must index the same places
a[maxintF] = 10
a[maxintF - 1.0] = 11
a[-maxintF] = 12
a[-maxintF + 1.0] = 13

assert(a[maxint] == 10 and a[maxint - 1] == 11 and a[-maxint] == 12 and a[-maxint + 1] == 13)

a[maxint] = 20
a[-maxint] = 22

assert(a[maxintF] == 20 and a[maxintF - 1.0] == 11 and a[-maxintF] == 22 and a[-maxintF + 1.0] == 13)

a = nil

-- test conflicts in multiple assignment
do
	local a, i, j, b
	a = { "a", "b" }
	i = 1
	j = 2
	b = a
	i, a[i], a, j, a[j], a[i + j] = j, i, i, b, j, i
	assert(i == 2 and b[1] == 1 and a == 1 and j == b and b[2] == 2 and b[3] == 1)
	a = {}
	local function foo() -- assigining to upvalues
		b, a.x, a = a, 10, 20
	end
	foo()
	assert(a == 20 and b.x == 10)
end

-- repeat test with upvalues
do
	local a, i, j, b
	a = { "a", "b" }
	i = 1
	j = 2
	b = a
	local function foo()
		i, a[i], a, j, a[j], a[i + j] = j, i, i, b, j, i
	end
	foo()
	assert(i == 2 and b[1] == 1 and a == 1 and j == b and b[2] == 2 and b[3] == 1)
	local t = {}
	(function(a)
		t[a], a = 10, 20
	end)(1)
	assert(t[1] == 10)
end

-- bug in 5.2 beta
local function foo()
	local a
	return function()
		local b
		a, b = 3, 14 -- local and upvalue have same index
		return a, b
	end
end

local a, b = foo()()
assert(a == 3 and b == 14)

print("OK")

return res
