local t = require("src.runtime.lib.test")
local literalsTests = {}

function literalsTests.testBasic()
	local fn, err = load("x \v\f = \t 'a\0a' \v\f\f")
	t.assertNotNil(fn, err)
	fn()
	t.assertEq("a\0a", x)
	t.assertLen(string.len(x), 3)

	t.assertEq(
		"\n\"'\\",
		[[

"'\]]
	)

	t.assertNotNil(string.find("\a\b\f\n\r\t\v", "^%c%c%c%c%c%c%c$"))
	t.assertEq("\09912", "c12")
	t.assertEq("\99ab", "cab")
	t.assertEq("\099", "\99")
	t.assertEq("\099\n", "c\10")
	t.assertEq("\0\0\0alo", "\0" .. "\0\0" .. "alo")
	t.assertEq(010 .. 020 .. -030, "1020-30")
	t.assertEq("\x00\x05\x10\x1f\x3C\xfF\xe8", "\0\5\16\31\60\255\232")
	t.assertEq("\u{0}\u{00000000}\x00\0", string.char(0, 0, 0, 0))
	t.assertEq("\u{0}\u{7F}", "\x00\x7F")
	t.assertEq("\u{80}\u{7FF}", "\xC2\x80\xDF\xBF")
	t.assertEq("\u{800}\u{FFFF}", "\xE0\xA0\x80\xEF\xBF\xBF")
	t.assertEq("\u{10000}\u{1FFFFF}", "\xF0\x90\x80\x80\xF7\xBF\xBF\xBF")
	t.assertEq("\u{200000}\u{3FFFFFF}", "\xF8\x88\x80\x80\x80\xFB\xBF\xBF\xBF\xBF")
	t.assertEq("\u{4000000}\u{7FFFFFFF}", "\xFC\x84\x80\x80\x80\x80\xFD\xBF\xBF\xBF\xBF\xBF")

	t.assertEq(
		"abc\z
        def\z
        ghi\z
       ",
		"abcdefghi"
	)
end

function literalsTests.testLexErrors()
	local function lexerror(s, err)
		local st, msg = load("return " .. s, "")
		if err ~= "<eof>" then
			err = err .. "'"
		end
		t.assertFalse(st)
		t.assertTrue(string.find(msg, "near .-" .. err))
	end

	lexerror([["abc\x"]], [[\x"]])
	lexerror([["abc\x]], [[\x]])
	lexerror([["\x]], [[\x]])
	lexerror([["\x5"]], [[\x5"]])
	lexerror([["\x5]], [[\x5]])
	lexerror([["\xr"]], [[\xr]])
	lexerror([["\xr]], [[\xr]])
	lexerror([["\x.]], [[\x.]])
	lexerror([["\x8%"]], [[\x8%%]])
	lexerror([["\xAG]], [[\xAG]])
	lexerror([["\g"]], [[\g]])
	lexerror([["\g]], [[\g]])
	lexerror([["\."]], [[\%.]])

	lexerror([["\999"]], [[\999"]])
	lexerror([["xyz\300"]], [[\300"]])
	lexerror([["   \256"]], [[\256"]])

	-- errors in UTF-8 sequences
	lexerror([["abc\u{100000000}"]], [[abc\u{100000000]]) -- too large
	lexerror([["abc\u11r"]], [[abc\u1]]) -- missing '{'
	lexerror([["abc\u"]], [[abc\u"]]) -- missing '{'
	lexerror([["abc\u{11r"]], [[abc\u{11r]]) -- missing '}'
	lexerror([["abc\u{11"]], [[abc\u{11"]]) -- missing '}'
	lexerror([["abc\u{11]], [[abc\u{11]]) -- missing '}'
	lexerror([["abc\u{r"]], [[abc\u{r]]) -- no digits

	-- unfinished strings
	lexerror("[=[alo]]", "<eof>")
	lexerror("[=[alo]=", "<eof>")
	lexerror("[=[alo]", "<eof>")
	lexerror("'alo", "<eof>")
	lexerror("'alo \\z  \n\n", "<eof>")
	lexerror("'alo \\z", "<eof>")
	lexerror([['alo \98]], "<eof>")
end

function literalsTests.variableNames()
	-- valid characters in variable names
	for i = 0, 255 do
		local s = string.char(i)
		assert(not string.find(s, "[a-zA-Z_]") == not load(s .. "=1", ""))
		assert(not string.find(s, "[a-zA-Z_0-9]") == not load("a" .. s .. "1 = 1", ""))
	end

	-- long variable names
	local var1 = string.rep("a", 15000) .. "1"
	local var2 = string.rep("a", 15000) .. "2"
	local prog = string.format(
		[[
  %s = 5
  %s = %s + 1
  return function () return %s - %s end
]],
		var1,
		var2,
		var1,
		var1,
		var2
	)
	local fn = load(prog)
	t.assertNotNil(fn)
	fn()
	t.assertEq(_G[var1], 5)
	t.assertEq(_G[var2], 6)
	t.assertEq(fn(), -1)
	_G[var1], _G[var2] = nil
end

function literalsTests.Escapes()
	t.assertEq(
		"\n\t",
		[[

	]]
	)
	t.assertEq(
		[[

 $debug]],
		"\n $debug"
	)
	t.assertNotEq([[ [ ]], [[ ] ]])
end

function literalsTests.Longstrings()
	local b =
		"001234567890123456789012345678901234567891234567890123456789012345678901234567890012345678901234567890123456789012345678912345678901234567890123456789012345678900123456789012345678901234567890123456789123456789012345678901234567890123456789001234567890123456789012345678901234567891234567890123456789012345678901234567890012345678901234567890123456789012345678912345678901234567890123456789012345678900123456789012345678901234567890123456789123456789012345678901234567890123456789001234567890123456789012345678901234567891234567890123456789012345678901234567890012345678901234567890123456789012345678912345678901234567890123456789012345678900123456789012345678901234567890123456789123456789012345678901234567890123456789001234567890123456789012345678901234567891234567890123456789012345678901234567890012345678901234567890123456789012345678912345678901234567890123456789012345678900123456789012345678901234567890123456789123456789012345678901234567890123456789"
	t.assertEq(string.len(b) == 960)
	local prog = [=[
print('+')

local a1 = [["this is a 'string' with several 'quotes'"]]
local a2 = "'quotes'"

assert(string.find(a1, a2) == 34)
print('+')

a1 = [==[temp = [[an arbitrary value]]; ]==]
assert(load(a1))()
assert(temp == 'an arbitrary value')
_G.temp = nil
-- long strings --
local b = "001234567890123456789012345678901234567891234567890123456789012345678901234567890012345678901234567890123456789012345678912345678901234567890123456789012345678900123456789012345678901234567890123456789123456789012345678901234567890123456789001234567890123456789012345678901234567891234567890123456789012345678901234567890012345678901234567890123456789012345678912345678901234567890123456789012345678900123456789012345678901234567890123456789123456789012345678901234567890123456789001234567890123456789012345678901234567891234567890123456789012345678901234567890012345678901234567890123456789012345678912345678901234567890123456789012345678900123456789012345678901234567890123456789123456789012345678901234567890123456789001234567890123456789012345678901234567891234567890123456789012345678901234567890012345678901234567890123456789012345678912345678901234567890123456789012345678900123456789012345678901234567890123456789123456789012345678901234567890123456789"
assert(string.len(b) == 960)
print('+')

local a = [[00123456789012345678901234567890123456789123456789012345678901234567890123456789
00123456789012345678901234567890123456789123456789012345678901234567890123456789
00123456789012345678901234567890123456789123456789012345678901234567890123456789
00123456789012345678901234567890123456789123456789012345678901234567890123456789
00123456789012345678901234567890123456789123456789012345678901234567890123456789
00123456789012345678901234567890123456789123456789012345678901234567890123456789
00123456789012345678901234567890123456789123456789012345678901234567890123456789
00123456789012345678901234567890123456789123456789012345678901234567890123456789
00123456789012345678901234567890123456789123456789012345678901234567890123456789
00123456789012345678901234567890123456789123456789012345678901234567890123456789
00123456789012345678901234567890123456789123456789012345678901234567890123456789
00123456789012345678901234567890123456789123456789012345678901234567890123456789
00123456789012345678901234567890123456789123456789012345678901234567890123456789
00123456789012345678901234567890123456789123456789012345678901234567890123456789
00123456789012345678901234567890123456789123456789012345678901234567890123456789
00123456789012345678901234567890123456789123456789012345678901234567890123456789
00123456789012345678901234567890123456789123456789012345678901234567890123456789
00123456789012345678901234567890123456789123456789012345678901234567890123456789
00123456789012345678901234567890123456789123456789012345678901234567890123456789
00123456789012345678901234567890123456789123456789012345678901234567890123456789
00123456789012345678901234567890123456789123456789012345678901234567890123456789
00123456789012345678901234567890123456789123456789012345678901234567890123456789
00123456789012345678901234567890123456789123456789012345678901234567890123456789
]]
assert(string.len(a) == 1863)
assert(string.sub(a, 1, 40) == string.sub(b, 1, 40))
x = 1
]=]

	_G.x = nil
	load(prog)()
	t.assertTrue(x)
	_G.x = nil

	do -- reuse of long strings
		-- get the address of a string
		local function getadd(s)
			return string.format("%p", s)
		end

		local s1 <const> = "01234567890123456789012345678901234567890123456789"
		local s2 <const> = "01234567890123456789012345678901234567890123456789"
		local s3 = "01234567890123456789012345678901234567890123456789"
		local function foo()
			return s1
		end
		local function foo1()
			return s3
		end
		local function foo2()
			return "01234567890123456789012345678901234567890123456789"
		end
		local a1 = getadd(s1)
		assert(a1 == getadd(s2))
		assert(a1 == getadd(foo()))
		assert(a1 == getadd(foo1()))
		assert(a1 == getadd(foo2()))

		local sd = "0123456789" .. "0123456789012345678901234567890123456789"
		assert(sd == s1 and getadd(sd) ~= a1)
	end
end

function literalsTests.testComments()
	t.assertEq([==[]=]==], "]=")
	t.assertEq([==[[===[[=[]]=][====[]]===]===]==], "[===[[=[]]=][====[]]===]===")
	t.assertEq([====[[===[[=[]]=][====[]]===]===]====], "[===[[=[]]=][====[]]===]===")
	t.assertEq([=[]]]]]]]]]=], "]]]]]]]]")

	local x = { "=", "[", "]", "\n" }
	local len = 4
	local function gen(c, n)
		if n == 0 then
			coroutine.yield(c)
		else
			for _, a in pairs(x) do
				gen(c .. a, n - 1)
			end
		end
	end

	for s in
		coroutine.wrap(function()
			gen("", len)
		end)
	do
		t.assertEq(s, load("return [====[\n" .. s .. "]====]", "")())
	end
end

function literalsTests.testDecimalPoint()
	t.assertEq(tonumber("  -.4  "), -0.4)
	t.assertEq(tonumber("  +0x.41  "), 0X0.41)
	t.assertNil(load("a = (3,4)"))
	t.assertEq(load("return 3.4")(), 3.4)
	t.assertEq(load("return .4,3")(), 0.4)
	t.assertEq(load("return 4.")(), 4.)
	t.assertEq(load("return 4.+.5")(), 4.5)
	t.assertEq(" 0x.1 " + " 0x,1" + "-0X.1\t", 0x0.1)
	t.assertNil(tonumber("inf"))
	t.assertNil(tonumber("NAN"))
	t.assertEq(load(string.format("return %q", 4.51))(), 4.51)
	local a, b = load("return 4.5.")
	t.assertTrue(string.find(b, "'4%.5%.'"))
end

function literalsTests.testLineEnds()
	local s = "a string with \r and \n and \r\n and \n\r"
	local c = string.format("return %q", s)
	t.assertEq(load(c)() == s)
end

function literalsTests.testErrors()
	t.assertNil(load("a = 'non-ending string"))
	t.assertNil(load("a = 'non-ending string\n'"))
	t.assertNil(load("a = '\\345'"))
	t.assertNil(load("a = [=x]"))
end

function literalsTests.testMalformedNumber()
	local function malformednum(n, exp)
		local s, msg = load("return " .. n)
		t.assertNil(s)
		t.assertTrue(string.find(msg, exp))
	end

	malformednum("0xe-", "near <eof>")
	malformednum("0xep-p", "malformed number")
	malformednum("1print()", "malformed number")
end

return literalsTests
