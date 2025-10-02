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
