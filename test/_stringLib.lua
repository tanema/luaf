local t = require("src.runtime.lib.test")
local stringsTest = {}

function stringsTest.testStringComparision()
	t.assertTrue("alo" < "alo1")
	t.assertTrue("" < "a")
	t.assertTrue("alo\0alo" < "alo\0b")
	t.assertTrue("alo\0alo\0\0" > "alo\0alo\0")
	t.assertTrue("alo" < "alo\0")
	t.assertTrue("alo\0" > "alo")
	t.assertTrue("\0" < "\1")
	t.assertTrue("\0\0" < "\0\1")
	t.assertTrue("\1\0a\0a" <= "\1\0a\0a")
	t.assertTrue(not ("\1\0a\0b" <= "\1\0a\0a"))
	t.assertTrue("\0\0\0" < "\0\0\0\0")
	t.assertTrue(not ("\0\0\0\0" < "\0\0\0"))
	t.assertTrue("\0\0\0" <= "\0\0\0\0")
	t.assertTrue(not ("\0\0\0\0" <= "\0\0\0"))
	t.assertTrue("\0\0\0" <= "\0\0\0")
	t.assertTrue("\0\0\0" >= "\0\0\0")
	t.assertTrue(not ("\0\0b" < "\0\0a\0"))
end

function stringsTest.testStringSub()
	t.assertEq("234", string.sub("123456789", 2, 4))
	t.assertEq("789", string.sub("123456789", 7))
	t.assertEq("", string.sub("123456789", 7, 6))
	t.assertEq("7", string.sub("123456789", 7, 7))
	t.assertEq("", string.sub("123456789", 0, 0))
	t.assertEq("123456789", string.sub("123456789", -10, 10))
	t.assertEq("123456789", string.sub("123456789", 1, 9))
	t.assertEq("", string.sub("123456789", -10, -20))
	t.assertEq("9", string.sub("123456789", -1))
	t.assertEq("6789", string.sub("123456789", -4))
	t.assertEq("456", string.sub("123456789", -6, -4))
	t.assertEq("234", string.sub("\000123456789", 3, 5))
	t.assertEq("789", ("\000123456789"):sub(8))
end

function stringsTest.testStringFind()
	t.skip("TODO")
	t.assertEq(3, string.find("123456789", "345"))
	local a, b = string.find("123456789", "345")
	t.assertEq("345", string.sub("123456789", a, b))
	t.assertEq(3, string.find("1234567890123456789", "345", 3))
	t.assertEq(13, string.find("1234567890123456789", "345", 4))
	t.assertNil(string.find("1234567890123456789", "346", 4))
	t.assertEq(13, string.find("1234567890123456789", ".45", -9))
	t.assertNil(string.find("abcdefg", "\0", 5, 1))
	t.assertEq(1, string.find("", ""))
	t.assertEq(1, string.find("", "", 1))
	t.assertNil(string.find("", "", 2))
	t.assertNil(string.find("", "aaa", 1))
	t.assertEq(4, ("alo(.)alo"):find("(.)", 1, 1))
end

function stringsTest.testStringLen()
	t.assertEq(0, string.len(""))
	t.assertEq(3, string.len("\0\0\0"))
	t.assertEq(10, string.len("1234567890"))
	t.assertEq(0, #"")
	t.assertEq(3, #"\0\0\0")
	t.assertEq(10, #"1234567890")
end

function stringsTest.testStringByte()
	t.assertEq(97, string.byte("a"))
	t.assertEq(92, string.byte("\x5c"))
	t.assertEq(255, string.byte("\255"))
	t.assertEq(255, string.byte(string.char(255)))
	t.assertEq(0, string.byte(string.char(0)))
	t.assertEq(0, string.byte("\0"))
	t.assertEq(120, string.byte("\0\0alo\0x", -1))
	t.assertEq(120, string.byte("x"))
	t.assertEq(string.byte("\0\0alo\0x", -1), string.byte("x"))
	t.assertEq(97, string.byte("ba", 2))
	t.assertEq(97, string.byte("ba", 2, -1))
	t.assertEq(97, string.byte("ba", 2, 2))
	t.assertNil(string.byte(""))
	t.assertNil(string.byte("hi", -3))
	t.assertNil(string.byte("hi", 3))
	t.assertNil(string.byte("hi", 9, 10))
	t.assertNil(string.byte("hi", 2, 1))
end

function stringsTest.testStringChar()
	t.assertEq("", string.char())
	t.assertEq("a", string.char(97))
	t.assertEq("\xff", string.char(255))
	t.assertEq("\0\xe4\0", string.char(0, string.byte("\xe4"), 0))
	t.assertEq("\xe4l\0óu", string.char(string.byte("\xe4l\0óu", 1, -1)))
	t.assertEq("", string.char(string.byte("\xe4l\0óu", 1, 0)))
	t.assertEq("\xe4l\0óu", string.char(string.byte("\xe4l\0óu", 1, 100)))
end

function stringsTest.testStringUpper()
	t.assertEq("AB\0C", string.upper("ab\0c"))
end

function stringsTest.testStringLower()
	t.assertEq("\0abcc%$", string.lower("\0ABCc%$"))
end

function stringsTest.testStringRep()
	t.assertEq("", string.rep("teste", 0))
	t.assertEq("tés\00têtés\00tê", string.rep("tés\00tê", 2))
	t.assertEq("", string.rep("", 10))
	t.assertEq("", string.rep("teste", 0, "xuxu"))
	t.assertEq("teste", string.rep("teste", 1, "xuxu"))
	t.assertEq("\1\0\1\0\0\1\0\1", string.rep("\1\0\1", 2, "\0\0"))
	t.assertEq(string.rep("", 10, "."), string.rep(".", 9))
end

function stringsTest.testStringReverse()
	t.assertEq("", string.reverse(""))
	t.assertEq("43210", string.reverse("01234"))
end

function stringsTest.testToString()
	t.assertEq("string", type(tostring(nil)))
	t.assertEq("string", type(tostring(12)))
	t.assertEq("table:", string.sub(tostring({}), 1, 6))
	t.assertEq("function:", string.sub(tostring(print), 1, 9))
	t.assertEq(1, #tostring("\0"))
	t.assertEq("true", tostring(true))
	t.assertEq("false", tostring(false))
	t.assertEq("-1203", tostring(-1203))
	t.assertEq("1203.125", tostring(1203.125))
	t.assertEq("-0.5", tostring(-0.5))
	t.assertEq("-32767", tostring(-32767))
	t.assertEq("0.1", tostring(0.1))
	t.assertEq("12", "" .. 12)
	t.assertEq("12.1", 12.1 .. "")
	t.assertEq("-1203.1", tostring(-1203 + -0.1))
end

function stringsTest.testStringFormat()
	t.skip("TODO")
	local function topointer(s)
		return string.format("%p", s)
	end

	do -- tests for '%p' format
		-- not much to test, as C does not specify what '%p' does.
		-- ("The value of the pointer is converted to a sequence of printing
		-- characters, in an implementation-defined manner.")
		local null = "(null)" -- nulls are formatted by Lua
		assert(string.format("%p", 4) == null)
		assert(string.format("%p", true) == null)
		assert(string.format("%p", nil) == null)
		assert(string.format("%p", {}) ~= null)
		assert(string.format("%p", print) ~= null)
		assert(string.format("%p", coroutine.running()) ~= null)
		assert(string.format("%p", io.stdin) ~= null)
		assert(string.format("%p", io.stdin) == string.format("%p", io.stdin))
		assert(string.format("%p", print) == string.format("%p", print))
		assert(string.format("%p", print) ~= string.format("%p", assert))

		assert(#string.format("%90p", {}) == 90)
		assert(#string.format("%-60p", {}) == 60)
		assert(string.format("%10p", false) == string.rep(" ", 10 - #null) .. null)
		assert(string.format("%-12p", 1.5) == null .. string.rep(" ", 12 - #null))

		do
			local t1 = {}
			local t2 = {}
			assert(topointer(t1) ~= topointer(t2))
		end

		do -- short strings are internalized
			local s1 = string.rep("a", 10)
			local s2 = string.rep("aa", 5)
			assert(topointer(s1) == topointer(s2))
		end

		do -- long strings aren't internalized
			local s1 = string.rep("a", 300)
			local s2 = string.rep("a", 300)
			assert(topointer(s1) ~= topointer(s2))
		end
	end

	local x = '"ílo"\n\\'
	assert(string.format("%q%s", x, x) == '"\\"ílo\\"\\\n\\\\""ílo"\n\\')
	assert(string.format("%q", "\0") == [["\0"]])
	assert(load(string.format("return %q", x))() == x)
	x = "\0\1\0023\5\0009"
	assert(load(string.format("return %q", x))() == x)
	assert(string.format("\0%c\0%c%x\0", string.byte("\xe4"), string.byte("b"), 140) == "\0\xe4\0b8c\0")
	assert(string.format("") == "")
	assert(
		string.format("%c", 34) .. string.format("%c", 48) .. string.format("%c", 90) .. string.format("%c", 100)
			== string.format("%1c%-c%-1c%c", 34, 48, 90, 100)
	)
	assert(string.format("%s\0 is not \0%s", "not be", "be") == "not be\0 is not \0be")
	assert(string.format("%%%d %010d", 10, 23) == "%10 0000000023")
	assert(tonumber(string.format("%f", 10.3)) == 10.3)
	assert(string.format('"%-50s"', "a") == '"a' .. string.rep(" ", 49) .. '"')

	assert(string.format("-%.20s.20s", string.rep("%", 2000)) == "-" .. string.rep("%", 20) .. ".20s")
	assert(
		string.format('"-%20s.20s"', string.rep("%", 2000))
			== string.format("%q", "-" .. string.rep("%", 2000) .. ".20s")
	)

	do
		local function checkQ(v)
			local s = string.format("%q", v)
			local nv = load("return " .. s)()
			assert(v == nv and math.type(v) == math.type(nv))
		end
		checkQ("\0\0\1\255\u{234}")
		checkQ(math.maxinteger)
		checkQ(math.mininteger)
		checkQ(math.pi)
		checkQ(0.1)
		checkQ(true)
		checkQ(nil)
		checkQ(false)
		checkQ(math.huge)
		checkQ(-math.huge)
		assert(string.format("%q", 0 / 0) == "(0/0)") -- NaN
		-- checkerror("no literal", string.format, "%q", {})
	end

	assert(string.format("\0%s\0", "\0\0\1") == "\0\0\0\1\0")
	-- checkerror("contains zeros", string.format, "%10s", "\0")

	-- format x tostring
	assert(string.format("%s %s", nil, true) == "nil true")
	assert(string.format("%s %.4s", false, true) == "false true")
	assert(string.format("%.3s %.3s", false, true) == "fal tru")
	local m = setmetatable({}, {
		__tostring = function()
			return "hello"
		end,
		__name = "hi",
	})
	assert(string.format("%s %.10s", m, m) == "hello hello")
	getmetatable(m).__tostring = nil -- will use '__name' from now on
	assert(string.format("%.4s", m) == "hi: ")

	getmetatable(m).__tostring = function()
		return {}
	end
	-- checkerror("'__tostring' must return a string", tostring, m)

	assert(string.format("%x", 0.0) == "0")
	assert(string.format("%02x", 0.0) == "00")
	assert(string.format("%08X", 0xFFFFFFFF) == "FFFFFFFF")
	assert(string.format("%+08d", 31501) == "+0031501")
	assert(string.format("%+08d", -30927) == "-0030927")

	do -- longest number that can be formatted
		local i = 1
		local j = 10000
		while i + 1 < j do -- binary search for maximum finite float
			local m = (i + j) // 2
			if 10 ^ m < math.huge then
				i = m
			else
				j = m
			end
		end
		assert(10 ^ i < math.huge and 10 ^ j == math.huge)
		local s = string.format("%.99f", -(10 ^ i))
		assert(string.len(s) >= i + 101)
		assert(tonumber(s) == -(10 ^ i))

		-- limit for floats
		assert(10 ^ 38 < math.huge)
		local s = string.format("%.99f", -(10 ^ 38))
		assert(string.len(s) >= 38 + 101)
		assert(tonumber(s) == -(10 ^ 38))
	end

	-- testing large numbers for format
	do -- assume at least 32 bits
		local max, min = 0x7fffffff, -0x80000000 -- "large" for 32 bits
		assert(string.sub(string.format("%8x", -1), -8) == "ffffffff")
		assert(string.format("%x", max) == "7fffffff")
		assert(string.sub(string.format("%x", min), -8) == "80000000")
		assert(string.format("%d", max) == "2147483647")
		assert(string.format("%d", min) == "-2147483648")
		assert(string.format("%u", 0xffffffff) == "4294967295")
		assert(string.format("%o", 0xABCD) == "125715")

		-- max, min = 0x7fffffffffffffff, -0x8000000000000000
		-- if max > 2.0 ^ 53 then -- only for 64 bits
		--	assert(string.format("%x", (2 ^ 52 | 0) - 1) == "fffffffffffff")
		--	assert(string.format("0x%8X", 0x8f000003) == "0x8F000003")
		--	assert(string.format("%d", 2 ^ 53) == "9007199254740992")
		--	assert(string.format("%i", -2 ^ 53) == "-9007199254740992")
		--	assert(string.format("%x", max) == "7fffffffffffffff")
		--	assert(string.format("%x", min) == "8000000000000000")
		--	assert(string.format("%d", max) == "9223372036854775807")
		--	assert(string.format("%d", min) == "-9223372036854775808")
		--	assert(string.format("%u", ~(-1 << 64)) == "18446744073709551615")
		--	assert(tostring(1234567890123) == "1234567890123")
		-- end
	end

	do
		local function matchhexa(n)
			local s = string.format("%a", n)
			-- result matches ISO C requirements
			assert(string.find(s, "^%-?0x[1-9a-f]%.?[0-9a-f]*p[-+]?%d+$"))
			assert(tonumber(s) == n) -- and has full precision
			s = string.format("%A", n)
			assert(string.find(s, "^%-?0X[1-9A-F]%.?[0-9A-F]*P[-+]?%d+$"))
			assert(tonumber(s) == n)
		end
		for _, n in ipairs({ 0.1, -0.1, 1 / 3, -1 / 3, 1e30, -1e30, -45 / 247, 1, -1, 2, -2, 3e-20, -3e-20 }) do
			matchhexa(n)
		end

		assert(string.find(string.format("%A", 0.0), "^0X0%.?0*P%+?0$"))
		assert(string.find(string.format("%a", -0.0), "^%-0x0%.?0*p%+?0$"))

		if not pcall(string.format, "%.3a", 0) then
			print("\n >>> modifiers for format '%a' not available <<<\n")
		else
			assert(string.find(string.format("%+.2A", 12), "^%+0X%x%.%x0P%+?%d$"))
			assert(string.find(string.format("%.4A", -12), "^%-0X%x%.%x000P%+?%d$"))
		end
	end

	-- testing some flags  (all these results are required by ISO C)
	assert(string.format("%#12o", 10) == "         012")
	assert(string.format("%#10x", 100) == "      0x64")
	assert(string.format("%#-17X", 100) == "0X64             ")
	assert(string.format("%013i", -100) == "-000000000100")
	assert(string.format("%2.5d", -100) == "-00100")
	assert(string.format("%.u", 0) == "")
	assert(string.format("%+#014.0f", 100) == "+000000000100.")
	assert(string.format("%-16c", 97) == "a               ")
	assert(string.format("%+.3G", 1.5) == "+1.5")
	assert(string.format("%.0s", "alo") == "")
	assert(string.format("%.s", "alo") == "")

	-- ISO C89 says that "The exponent always contains at least two digits",
	-- but unlike ISO C99 it does not ensure that it contains "only as many
	-- more digits as necessary".
	assert(string.match(string.format("% 1.0E", 100), "^ 1E%+0+2$"))
	assert(string.match(string.format("% .1g", 2 ^ 10), "^ 1e%+0+3$"))

	-- errors in format

	local function check(fmt, msg)
		--checkerror(msg, string.format, fmt, 10)
	end

	local aux = string.rep("0", 600)
	check("%100.3d", "invalid conversion")
	check("%1" .. aux .. ".3d", "too long")
	check("%1.100d", "invalid conversion")
	check("%10.1" .. aux .. "004d", "too long")
	check("%t", "invalid conversion")
	check("%" .. aux .. "d", "too long")
	check("%d %d", "no value")
	check("%010c", "invalid conversion")
	check("%.10c", "invalid conversion")
	check("%0.34s", "invalid conversion")
	check("%#i", "invalid conversion")
	check("%3.1p", "invalid conversion")
	check("%0.s", "invalid conversion")
	check("%10q", "cannot have modifiers")
	check("%F", "invalid conversion") -- useless and not in C89

	assert(load("return 1\n--comment without ending EOL")() == 1)

	--checkerror("table expected", table.concat, 3)
	--checkerror("at index " .. maxi, table.concat, {}, " ", maxi, maxi)
	-- '%' escapes following minus signal
	--checkerror("at index %" .. mini, table.concat, {}, " ", mini, mini)
	assert(table.concat({}) == "")
	assert(table.concat({}, "x") == "")
	assert(table.concat({ "\0", "\0\1", "\0\1\2" }, ".\0.") == "\0.\0.\0\1.\0.\0\1\2")
	local a = {}
	for i = 1, 300 do
		a[i] = "xuxu"
	end
	assert(table.concat(a, "123") .. "123" == string.rep("xuxu123", 300))
	assert(table.concat(a, "b", 20, 20) == "xuxu")
	assert(table.concat(a, "", 20, 21) == "xuxuxuxu")
	assert(table.concat(a, "x", 22, 21) == "")
	assert(table.concat(a, "3", 299) == "xuxu3xuxu")
	-- assert(table.concat({}, "x", maxi, maxi - 1) == "")
	-- assert(table.concat({}, "x", mini + 1, mini) == "")
	-- assert(table.concat({}, "x", maxi, mini) == "")
	-- assert(table.concat({ [maxi] = "alo" }, "x", maxi, maxi) == "alo")
	-- assert(table.concat({ [maxi] = "alo", [maxi - 1] = "y" }, "-", maxi - 1, maxi) == "y-alo")

	assert(not pcall(table.concat, { "a", "b", {} }))

	a = { "a", "b", "c" }
	assert(table.concat(a, ",", 1, 0) == "")
	assert(table.concat(a, ",", 1, 1) == "a")
	assert(table.concat(a, ",", 1, 2) == "a,b")
	assert(table.concat(a, ",", 2) == "b,c")
	assert(table.concat(a, ",", 3) == "c")
	assert(table.concat(a, ",", 4) == "")

	-- bug in Lua 5.3.2
	-- 'gmatch' iterator does not work across coroutines
	do
		local f = string.gmatch("1 2 3 4 5", "%d+")
		assert(f() == "1")
		local co = coroutine.wrap(f)
		assert(co() == "2")
	end
end

function stringsTest.testPatternMatching()
	t.skip("TODO")

	local function checkerror(msg, f, ...)
		local s, err = pcall(f, ...)
		assert(not s and string.find(err, msg))
	end

	local function f(s, p)
		local i, e = string.find(s, p)
		if i then
			return string.sub(s, i, e)
		end
	end

	local a, b = string.find("", "") -- empty patterns are tricky
	assert(a == 1 and b == 0)
	a, b = string.find("alo", "")
	assert(a == 1 and b == 0)
	a, b = string.find("a\0o a\0o a\0o", "a", 1) -- first position
	assert(a == 1 and b == 1)
	a, b = string.find("a\0o a\0o a\0o", "a\0o", 2) -- starts in the midle
	assert(a == 5 and b == 7)
	a, b = string.find("a\0o a\0o a\0o", "a\0o", 9) -- starts in the midle
	assert(a == 9 and b == 11)
	a, b = string.find("a\0a\0a\0a\0\0ab", "\0ab", 2) -- finds at the end
	assert(a == 9 and b == 11)
	a, b = string.find("a\0a\0a\0a\0\0ab", "b") -- last position
	assert(a == 11 and b == 11)
	assert(not string.find("a\0a\0a\0a\0\0ab", "b\0")) -- check ending
	assert(not string.find("", "\0"))
	assert(string.find("alo123alo", "12") == 4)
	assert(not string.find("alo123alo", "^12"))

	assert(string.match("aaab", ".*b") == "aaab")
	assert(string.match("aaa", ".*a") == "aaa")
	assert(string.match("b", ".*b") == "b")

	assert(string.match("aaab", ".+b") == "aaab")
	assert(string.match("aaa", ".+a") == "aaa")
	assert(not string.match("b", ".+b"))

	assert(string.match("aaab", ".?b") == "ab")
	assert(string.match("aaa", ".?a") == "aa")
	assert(string.match("b", ".?b") == "b")

	assert(f("aloALO", "%l*") == "alo")
	assert(f("aLo_ALO", "%a*") == "aLo")

	assert(f("  \n\r*&\n\r   xuxu  \n\n", "%g%g%g+") == "xuxu")

	-- Adapt a pattern to UTF-8
	local function PU(p)
		-- distribute '?' into each individual byte of a character.
		-- (For instance, "á?" becomes "\195?\161?".)
		p = string.gsub(p, "(" .. utf8.charpattern .. ")%?", function(c)
			return string.gsub(c, ".", "%0?")
		end)
		-- change '.' to utf-8 character patterns
		p = string.gsub(p, "%.", utf8.charpattern)
		return p
	end

	assert(f("aaab", "a*") == "aaa")
	assert(f("aaa", "^.*$") == "aaa")
	assert(f("aaa", "b*") == "")
	assert(f("aaa", "ab*a") == "aa")
	assert(f("aba", "ab*a") == "aba")
	assert(f("aaab", "a+") == "aaa")
	assert(f("aaa", "^.+$") == "aaa")
	assert(not f("aaa", "b+"))
	assert(not f("aaa", "ab+a"))
	assert(f("aba", "ab+a") == "aba")
	assert(f("a$a", ".$") == "a")
	assert(f("a$a", ".%$") == "a$")
	assert(f("a$a", ".$.") == "a$a")
	assert(not f("a$a", "$$"))
	assert(not f("a$b", "a$"))
	assert(f("a$a", "$") == "")
	assert(f("", "b*") == "")
	assert(not f("aaa", "bb*"))
	assert(f("aaab", "a-") == "")
	assert(f("aaa", "^.-$") == "aaa")
	assert(f("aabaaabaaabaaaba", "b.*b") == "baaabaaabaaab")
	assert(f("aabaaabaaabaaaba", "b.-b") == "baaab")
	assert(f("alo xo", ".o$") == "xo")
	assert(f(" \n isto é assim", "%S%S*") == "isto")
	assert(f(" \n isto é assim", "%S*$") == "assim")
	assert(f(" \n isto é assim", "[a-z]*$") == "assim")
	assert(f("um caracter ? extra", "[^%sa-z]") == "?")
	assert(f("", "a?") == "")
	assert(f("á", PU("á?")) == "á")
	assert(f("ábl", PU("á?b?l?")) == "ábl")
	assert(f("  ábl", PU("á?b?l?")) == "")
	assert(f("aa", "^aa?a?a") == "aa")
	assert(f("]]]áb", "[^]]+") == "áb")
	assert(f("0alo alo", "%x*") == "0a")
	assert(f("alo alo", "%C+") == "alo alo")

	local function f1(s, p)
		p = string.gsub(p, "%%([0-9])", function(s)
			return "%" .. (tonumber(s) + 1)
		end)
		p = string.gsub(p, "^(^?)", "%1()", 1)
		p = string.gsub(p, "($?)$", "()%1", 1)
		local t = { string.match(s, p) }
		return string.sub(s, t[1], t[#t] - 1)
	end

	assert(f1("alo alx 123 b\0o b\0o", "(..*) %1") == "b\0o b\0o")
	assert(f1("axz123= 4= 4 34", "(.+)=(.*)=%2 %1") == "3= 4= 4 3")
	assert(f1("=======", "^(=*)=%1$") == "=======")
	assert(not string.match("==========", "^([=]*)=%1$"))

	local function range(i, j)
		if i <= j then
			return i, range(i + 1, j)
		end
	end

	local abc = string.char(range(0, 127)) .. string.char(range(128, 255))

	assert(string.len(abc) == 256)

	local function strset(p)
		local res = { s = "" }
		string.gsub(abc, p, function(c)
			res.s = res.s .. c
		end)
		return res.s
	end

	assert(string.len(strset("[\200-\210]")) == 11)

	assert(strset("[a-z]") == "abcdefghijklmnopqrstuvwxyz")
	assert(strset("[a-z%d]") == strset("[%da-uu-z]"))
	assert(strset("[a-]") == "-a")
	assert(strset("[^%W]") == strset("[%w]"))
	assert(strset("[]%%]") == "%]")
	assert(strset("[a%-z]") == "-az")
	assert(strset("[%^%[%-a%]%-b]") == "-[]^ab")
	assert(strset("%Z") == strset("[\1-\255]"))
	assert(strset(".") == strset("[\1-\255%z]"))

	assert(string.match("alo xyzK", "(%w+)K") == "xyz")
	assert(string.match("254 K", "(%d*)K") == "")
	assert(string.match("alo ", "(%w*)$") == "")
	assert(not string.match("alo ", "(%w+)$"))
	assert(string.find("(álo)", "%(á") == 1)
	local a, b, c, d, e = string.match("âlo alo", PU("^(((.).). (%w*))$"))
	assert(a == "âlo alo" and b == "âl" and c == "â" and d == "alo" and e == nil)
	a, b, c, d = string.match("0123456789", "(.+(.?)())")
	assert(a == "0123456789" and b == "" and c == 11 and d == nil)

	assert(string.gsub("ülo ülo", "ü", "x") == "xlo xlo")
	assert(string.gsub("alo úlo  ", " +$", "") == "alo úlo") -- trim
	assert(string.gsub("  alo alo  ", "^%s*(.-)%s*$", "%1") == "alo alo") -- double trim
	assert(string.gsub("alo  alo  \n 123\n ", "%s+", " ") == "alo alo 123 ")
	local t = "abç d"
	a, b = string.gsub(t, PU("(.)"), "%1@")
	assert(a == "a@b@ç@ @d@" and b == 5)
	a, b = string.gsub("abçd", PU("(.)"), "%0@", 2)
	assert(a == "a@b@çd" and b == 2)
	assert(string.gsub("alo alo", "()[al]", "%1") == "12o 56o")
	assert(string.gsub("abc=xyz", "(%w*)(%p)(%w+)", "%3%2%1-%0") == "xyz=abc-abc=xyz")
	assert(string.gsub("abc", "%w", "%1%0") == "aabbcc")
	assert(string.gsub("abc", "%w+", "%0%1") == "abcabc")
	assert(string.gsub("áéí", "$", "\0óú") == "áéí\0óú")
	assert(string.gsub("", "^", "r") == "r")
	assert(string.gsub("", "$", "r") == "r")

	do -- new (5.3.3) semantics for empty matches
		assert(string.gsub("a b cd", " *", "-") == "-a-b-c-d-")

		local res = ""
		local sub = "a  \nbc\t\td"
		local i = 1
		for p, e in string.gmatch(sub, "()%s*()") do
			res = res .. string.sub(sub, i, p - 1) .. "-"
			i = e
		end
		assert(res == "-a-b-c-d-")
	end

	assert(string.gsub("um (dois) tres (quatro)", "(%(%w+%))", string.upper) == "um (DOIS) tres (QUATRO)")

	do
		local function setglobal(n, v)
			rawset(_G, n, v)
		end
		string.gsub("a=roberto,roberto=a", "(%w+)=(%w%w*)", setglobal)
		assert(_G.a == "roberto" and _G.roberto == "a")
		_G.a = nil
		_G.roberto = nil
	end

	function f(a, b)
		return string.gsub(a, ".", b)
	end
	assert(
		string.gsub("trocar tudo em |teste|b| é |beleza|al|", "|([^|]*)|([^|]*)|", f)
			== "trocar tudo em bbbbb é alalalalalal"
	)

	local function dostring(s)
		return load(s, "")() or ""
	end
	assert(string.gsub("alo $a='x'$ novamente $return a$", "$([^$]*)%$", dostring) == "alo  novamente x")

	local x = string.gsub("$x=string.gsub('alo', '.', string.upper)$ assim vai para $return x$", "$([^$]*)%$", dostring)
	assert(x == " assim vai para ALO")
	_G.a, _G.x = nil

	local t = {}
	local s = "a alo jose  joao"
	local r = string.gsub(s, "()(%w+)()", function(a, w, b)
		assert(string.len(w) == b - a)
		t[a] = b - a
	end)
	assert(s == r and t[1] == 1 and t[3] == 3 and t[7] == 4 and t[13] == 4)

	local function isbalanced(s)
		return not string.find(string.gsub(s, "%b()", ""), "[()]")
	end

	assert(isbalanced("(9 ((8))(\0) 7) \0\0 a b ()(c)() a"))
	assert(not isbalanced("(9 ((8) 7) a b (\0 c) a"))
	assert(string.gsub("alo 'oi' alo", "%b''", '"') == 'alo " alo')

	local t = { "apple", "orange", "lime", n = 0 }
	assert(string.gsub("x and x and x", "x", function()
		t.n = t.n + 1
		return t[t.n]
	end) == "apple and orange and lime")

	t = { n = 0 }
	string.gsub("first second word", "%w%w*", function(w)
		t.n = t.n + 1
		t[t.n] = w
	end)
	assert(t[1] == "first" and t[2] == "second" and t[3] == "word" and t.n == 3)

	t = { n = 0 }
	assert(string.gsub("first second word", "%w+", function(w)
		t.n = t.n + 1
		t[t.n] = w
	end, 2) == "first second word")
	assert(t[1] == "first" and t[2] == "second" and t[3] == undef)

	checkerror("invalid replacement value %(a table%)", string.gsub, "alo", ".", { a = {} })
	checkerror("invalid capture index %%2", string.gsub, "alo", ".", "%2")
	checkerror("invalid capture index %%0", string.gsub, "alo", "(%0)", "a")
	checkerror("invalid capture index %%1", string.gsub, "alo", "(%1)", "a")
	checkerror("invalid use of '%%'", string.gsub, "alo", ".", "%x")

	if not _soft then
		local a = string.rep("a", 300000)
		assert(string.find(a, "^a*.?$"))
		assert(not string.find(a, "^a*.?b$"))
		assert(string.find(a, "^a-.?$"))

		-- bug in 5.1.2
		a = string.rep("a", 10000) .. string.rep("b", 10000)
		assert(not pcall(string.gsub, a, "b"))
	end

	-- recursive nest of gsubs
	local function rev(s)
		return string.gsub(s, "(.)(.+)", function(c, s1)
			return rev(s1) .. c
		end)
	end

	local x = "abcdef"
	assert(rev(rev(x)) == x)

	-- gsub with tables
	assert(string.gsub("alo alo", ".", {}) == "alo alo")
	assert(string.gsub("alo alo", "(.)", { a = "AA", l = "" }) == "AAo AAo")
	assert(string.gsub("alo alo", "(.).", { a = "AA", l = "K" }) == "AAo AAo")
	assert(string.gsub("alo alo", "((.)(.?))", { al = "AA", o = false }) == "AAo AAo")

	assert(string.gsub("alo alo", "().", { "x", "yy", "zzz" }) == "xyyzzz alo")

	t = {}
	setmetatable(t, {
		__index = function(t, s)
			return string.upper(s)
		end,
	})
	assert(string.gsub("a alo b hi", "%w%w+", t) == "a ALO b HI")

	-- tests for gmatch
	local a = 0
	for i in string.gmatch("abcde", "()") do
		assert(i == a + 1)
		a = i
	end
	assert(a == 6)

	t = { n = 0 }
	for w in string.gmatch("first second word", "%w+") do
		t.n = t.n + 1
		t[t.n] = w
	end
	assert(t[1] == "first" and t[2] == "second" and t[3] == "word")

	t = { 3, 6, 9 }
	for i in string.gmatch("xuxx uu ppar r", "()(.)%2") do
		assert(i == table.remove(t, 1))
	end
	assert(#t == 0)

	t = {}
	for i, j in string.gmatch("13 14 10 = 11, 15= 16, 22=23", "(%d+)%s*=%s*(%d+)") do
		t[tonumber(i)] = tonumber(j)
	end
	a = 0
	for k, v in pairs(t) do
		assert(k + 1 == v + 0)
		a = a + 1
	end
	assert(a == 3)

	do -- init parameter in gmatch
		local s = 0
		for k in string.gmatch("10 20 30", "%d+", 3) do
			s = s + tonumber(k)
		end
		assert(s == 50)

		s = 0
		for k in string.gmatch("11 21 31", "%d+", -4) do
			s = s + tonumber(k)
		end
		assert(s == 32)

		-- there is an empty string at the end of the subject
		s = 0
		for k in string.gmatch("11 21 31", "%w*", 9) do
			s = s + 1
		end
		assert(s == 1)

		-- there are no empty strings after the end of the subject
		s = 0
		for k in string.gmatch("11 21 31", "%w*", 10) do
			s = s + 1
		end
		assert(s == 0)
	end

	-- tests for `%f' (`frontiers')

	assert(string.gsub("aaa aa a aaa a", "%f[%w]a", "x") == "xaa xa x xaa x")
	assert(string.gsub("[[]] [][] [[[[", "%f[[].", "x") == "x[]] x]x] x[[[")
	assert(string.gsub("01abc45de3", "%f[%d]", ".") == ".01abc.45de.3")
	assert(string.gsub("01abc45 de3x", "%f[%D]%w", ".") == "01.bc45 de3.")
	assert(string.gsub("function", "%f[\1-\255]%w", ".") == ".unction")
	assert(string.gsub("function", "%f[^\1-\255]", ".") == "function.")

	assert(string.find("a", "%f[a]") == 1)
	assert(string.find("a", "%f[^%z]") == 1)
	assert(string.find("a", "%f[^%l]") == 2)
	assert(string.find("aba", "%f[a%z]") == 3)
	assert(string.find("aba", "%f[%z]") == 4)
	assert(not string.find("aba", "%f[%l%z]"))
	assert(not string.find("aba", "%f[^%l%z]"))

	local i, e = string.find(" alo aalo allo", "%f[%S].-%f[%s].-%f[%S]")
	assert(i == 2 and e == 5)
	local k = string.match(" alo aalo allo", "%f[%S](.-%f[%s].-%f[%S])")
	assert(k == "alo ")

	local a = { 1, 5, 9, 14, 17 }
	for k in string.gmatch("alo alo th02 is 1hat", "()%f[%w%d]") do
		assert(table.remove(a, 1) == k)
	end
	assert(#a == 0)

	-- malformed patterns
	local function malform(p, m)
		m = m or "malformed"
		local r, msg = pcall(string.find, "a", p)
		assert(not r and string.find(msg, m))
	end

	malform("(.", "unfinished capture")
	malform(".)", "invalid pattern capture")
	malform("[a")
	malform("[]")
	malform("[^]")
	malform("[a%]")
	malform("[a%")
	malform("%b")
	malform("%ba")
	malform("%")
	malform("%f", "missing")

	-- \0 in patterns
	assert(string.match("ab\0\1\2c", "[\0-\2]+") == "\0\1\2")
	assert(string.match("ab\0\1\2c", "[\0-\0]+") == "\0")
	assert(string.find("b$a", "$\0?") == 2)
	assert(string.find("abc\0efg", "%\0") == 4)
	assert(string.match("abc\0efg\0\1e\1g", "%b\0\1") == "\0efg\0\1e\1")
	assert(string.match("abc\0\0\0", "%\0+") == "\0\0\0")
	assert(string.match("abc\0\0\0", "%\0%\0?") == "\0\0")

	-- magic char after \0
	assert(string.find("abc\0\0", "\0.") == 4)
	assert(string.find("abcx\0\0abc\0abc", "x\0\0abc\0a.") == 4)

	do -- test reuse of original string in gsub
		local s = string.rep("a", 100)
		local r = string.gsub(s, "b", "c") -- no match
		assert(string.format("%p", s) == string.format("%p", r))

		r = string.gsub(s, ".", { x = "y" }) -- no substitutions
		assert(string.format("%p", s) == string.format("%p", r))

		local count = 0
		r = string.gsub(s, ".", function(x)
			assert(x == "a")
			count = count + 1
			return nil -- no substitution
		end)
		r = string.gsub(r, ".", { b = "x" }) -- "a" is not a key; no subst.
		assert(count == 100)
		assert(string.format("%p", s) == string.format("%p", r))

		count = 0
		r = string.gsub(s, ".", function(x)
			assert(x == "a")
			count = count + 1
			return x -- substitution...
		end)
		assert(count == 100)
		-- no reuse in this case
		assert(r == s and string.format("%p", s) ~= string.format("%p", r))
	end
end

function stringsTest.testPack()
	t.assertEq(2, string.packsize("h"))
	t.assertEq(4, string.packsize("l"))
	t.assertEq(4, string.packsize("f"))
	t.assertEq(8, string.packsize("i"))
	t.assertEq(8, string.packsize("d"))
	t.assertEq(8, string.packsize("n"))
	t.assertEq(8, string.packsize("j"))
	t.assertEq(0xff, string.unpack("B", string.pack("B", 0xff)))
	t.assertEq(0x7f, string.unpack("b", string.pack("b", 0x7f)))
	t.assertEq(-0x80, string.unpack("b", string.pack("b", -0x80)))
	t.assertEq(0xffff, string.unpack("H", string.pack("H", 0xffff)))
	t.assertEq(0x7fff, string.unpack("h", string.pack("h", 0x7fff)))
	t.assertEq(-0x8000, string.unpack("h", string.pack("h", -0x8000)))
	t.assertEq(0xffffffff, string.unpack("L", string.pack("L", 0xffffffff)))
	t.assertEq(0x7fffffff, string.unpack("l", string.pack("l", 0x7fffffff)))
	t.assertEq(-0x80000000, string.unpack("l", string.pack("l", -0x80000000)))

	t.skip("TODO")

	local NB = 16
	for i = 1, NB do
		-- small numbers with signal extension ("\xFF...")
		local s = string.rep("\xff", i)
		assert(string.pack("i" .. i, -1) == s)
		assert(string.packsize("i" .. i) == #s)
		assert(string.unpack("i" .. i, s) == -1)

		-- small unsigned number ("\0...\xAA")
		s = "\xAA" .. string.rep("\0", i - 1)
		assert(string.pack("<I" .. i, 0xAA) == s)
		assert(string.unpack("<I" .. i, s) == 0xAA)
		assert(string.pack(">I" .. i, 0xAA) == s:reverse())
		assert(string.unpack(">I" .. i, s:reverse()) == 0xAA)
	end

	-- do
	--	local sizeLI = string.packsize("j")
	--	local lnum = 0x13121110090807060504030201
	--	local s = string.pack("<j", lnum)
	--	assert(string.unpack("<j", s) == lnum)
	--	assert(string.unpack("<i" .. sizeLI + 1, s .. "\0") == lnum)
	--	assert(string.unpack("<i" .. sizeLI + 1, s .. "\0") == lnum)

	--	for i = sizeLI + 1, NB do
	--		local s = string.pack("<j", -lnum)
	--		assert(string.unpack("<j", s) == -lnum)
	--		-- strings with (correct) extra bytes
	--		assert(string.unpack("<i" .. i, s .. ("\xFF"):rep(i - sizeLI)) == -lnum)
	--		assert(string.unpack(">i" .. i, ("\xFF"):rep(i - sizeLI) .. s:reverse()) == -lnum)
	--		assert(string.unpack("<I" .. i, s .. ("\0"):rep(i - sizeLI)) == -lnum)

	--		-- overflows
	--		checkerror("does not fit", string.unpack, "<I" .. i, ("\x00"):rep(i - 1) .. "\1")
	--		checkerror("does not fit", string.unpack, ">i" .. i, "\1" .. ("\x00"):rep(i - 1))
	--	end
	-- end

	-- for i = 1, sizeLI do
	--	local lstr = "\1\2\3\4\5\6\7\8\9\10\11\12\13"
	--	local lnum = 0x13121110090807060504030201
	--	local n = lnum & ~(-1 << (i * 8))
	--	local s = string.sub(lstr, 1, i)
	--	assert(string.pack("<i" .. i, n) == s)
	--	assert(string.pack(">i" .. i, n) == s:reverse())
	--	assert(string.unpack(">i" .. i, s:reverse()) == n)
	-- end

	-- sign extension
	do
		local u = 0xf0
		for i = 1, sizeLI - 1 do
			assert(string.unpack("<i" .. i, "\xf0" .. ("\xff"):rep(i - 1)) == -16)
			assert(string.unpack(">I" .. i, "\xf0" .. ("\xff"):rep(i - 1)) == u)
			u = u * 256 + 0xff
		end
	end

	-- mixed endianness
	do
		assert(string.pack(">i2 <i2", 10, 20) == "\0\10\20\0")
		local a, b = string.unpack("<i2 >i2", "\10\0\0\20")
		assert(a == 10 and b == 20)
		assert(string.pack("=i4", 2001) == string.pack("i4", 2001))
	end

	-- checkerror("out of limits", string.pack, "i0", 0)
	-- checkerror("out of limits", string.pack, "i" .. NB + 1, 0)
	-- checkerror("out of limits", string.pack, "!" .. NB + 1, 0)
	-- checkerror("%(17%) out of limits %[1,16%]", string.pack, "Xi" .. NB + 1)
	-- checkerror("invalid format option 'r'", string.pack, "i3r", 0)
	-- checkerror("16%-byte integer", string.unpack, "i16", string.rep("\3", 16))
	-- checkerror("not power of 2", string.pack, "!4i3", 0)
	-- checkerror("missing size", string.pack, "c", "")
	-- checkerror("variable%-length format", string.packsize, "s")
	-- checkerror("variable%-length format", string.packsize, "z")

	-- overflow in option size  (error will be in digit after limit)
	-- checkerror("invalid format", string.packsize, "c1" .. string.rep("0", 40))

	-- overflow in packing
	for i = 1, sizeLI - 1 do
		local umax = (1 << (i * 8)) - 1
		local max = umax >> 1
		local min = ~max
		-- checkerror("overflow", string.pack, "<I" .. i, -1)
		-- checkerror("overflow", string.pack, "<I" .. i, min)
		-- checkerror("overflow", string.pack, ">I" .. i, umax + 1)

		-- checkerror("overflow", string.pack, ">i" .. i, umax)
		-- checkerror("overflow", string.pack, ">i" .. i, max + 1)
		-- checkerror("overflow", string.pack, "<i" .. i, min - 1)

		assert(string.unpack(">i" .. i, string.pack(">i" .. i, max)) == max)
		assert(string.unpack("<i" .. i, string.pack("<i" .. i, min)) == min)
		assert(string.unpack(">I" .. i, string.pack(">I" .. i, umax)) == umax)
	end

	-- Lua integer size
	assert(string.unpack(">j", string.pack(">j", math.maxinteger)) == math.maxinteger)
	assert(string.unpack("<j", string.pack("<j", math.mininteger)) == math.mininteger)
	assert(string.unpack("<J", string.pack("<j", -1)) == -1) -- maximum unsigned integer

	if string.pack("i2", 1) == "\1\0" then
		assert(string.pack("f", 24) == string.pack("<f", 24))
	else
		assert(string.pack("f", 24) == string.pack(">f", 24))
	end

	for _, n in ipairs({ 0, -1.1, 1.9, 1 / 0, -1 / 0, 1e20, -1e20, 0.1, 2000.7 }) do
		assert(string.unpack("n", string.pack("n", n)) == n)
		assert(string.unpack("<n", string.pack("<n", n)) == n)
		assert(string.unpack(">n", string.pack(">n", n)) == n)
		assert(string.pack("<f", n) == string.pack(">f", n):reverse())
		assert(string.pack(">d", n) == string.pack("<d", n):reverse())
	end

	-- for non-native precisions, test only with "round" numbers
	for _, n in ipairs({ 0, -1.5, 1 / 0, -1 / 0, 1e10, -1e9, 0.5, 2000.25 }) do
		assert(string.unpack("<f", string.pack("<f", n)) == n)
		assert(string.unpack(">f", string.pack(">f", n)) == n)
		assert(string.unpack("<d", string.pack("<d", n)) == n)
		assert(string.unpack(">d", string.pack(">d", n)) == n)
	end

	do
		local s = string.rep("abc", 1000)
		assert(string.pack("zB", s, 247) == s .. "\0\xF7")
		local s1, b = string.unpack("zB", s .. "\0\xF9")
		assert(b == 249 and s1 == s)
		s1 = string.pack("s", s)
		assert(string.unpack("s", s1) == s)
		-- checkerror("does not fit", string.pack, "s1", s)
		-- checkerror("contains zeros", string.pack, "z", "alo\0")
		-- checkerror("unfinished string", string.unpack, "zc10000000", "alo")
		for i = 2, NB do
			local s1 = string.pack("s" .. i, s)
			assert(string.unpack("s" .. i, s1) == s and #s1 == #s + i)
		end
	end

	do
		local x = string.pack("s", "alo")
		-- checkerror("too short", string.unpack, "s", x:sub(1, -2))
		-- checkerror("too short", string.unpack, "c5", "abcd")
		-- checkerror("out of limits", string.pack, "s100", "alo")
	end

	do
		assert(string.pack("c0", "") == "")
		assert(string.packsize("c0") == 0)
		assert(string.unpack("c0", "") == "")
		assert(string.pack("<! c3", "abc") == "abc")
		assert(string.packsize("<! c3") == 3)
		assert(string.pack(">!4 c6", "abcdef") == "abcdef")
		assert(string.pack("c3", "123") == "123")
		assert(string.pack("c0", "") == "")
		assert(string.pack("c8", "123456") == "123456\0\0")
		assert(string.pack("c88 c1", "", "X") == string.rep("\0", 88) .. "X")
		assert(string.pack("c188 c2", "ab", "X\1") == "ab" .. string.rep("\0", 188 - 2) .. "X\1")
		local a, b, c = string.unpack("!4 z c3", "abcdefghi\0xyz")
		assert(a == "abcdefghi" and b == "xyz" and c == 14)
		-- checkerror("longer than", string.pack, "c3", "1234")
	end

	-- testing multiple types and sequence
	do
		local x = string.pack("<b h b f d f n i", 1, 2, 3, 4, 5, 6, 7, 8)
		assert(#x == string.packsize("<b h b f d f n i"))
		local a, b, c, d, e, f, g, h = string.unpack("<b h b f d f n i", x)
		assert(a == 1 and b == 2 and c == 3 and d == 4 and e == 5 and f == 6 and g == 7 and h == 8)
	end

	do
		assert(string.pack(" < i1 i2 ", 2, 3) == "\2\3\0") -- no alignment by default
		local x = string.pack(">!8 b Xh i4 i8 c1 Xi8", -12, 100, 200, "\xEC")
		assert(#x == string.packsize(">!8 b Xh i4 i8 c1 Xi8"))
		assert(x == "\xf4" .. "\0\0\0" .. "\0\0\0\100" .. "\0\0\0\0\0\0\0\xC8" .. "\xEC" .. "\0\0\0\0\0\0\0")
		local a, b, c, d, pos = string.unpack(">!8 c1 Xh i4 i8 b Xi8 XI XH", x)
		assert(a == "\xF4" and b == 100 and c == 200 and d == -20 and (pos - 1) == #x)

		x = string.pack(">!4 c3 c4 c2 z i4 c5 c2 Xi4", "abc", "abcd", "xz", "hello", 5, "world", "xy")
		assert(x == "abcabcdxzhello\0\0\0\0\0\5worldxy\0")
		local a, b, c, d, e, f, g, pos = string.unpack(">!4 c3 c4 c2 z i4 c5 c2 Xh Xi4", x)
		assert(
			a == "abc"
				and b == "abcd"
				and c == "xz"
				and d == "hello"
				and e == 5
				and f == "world"
				and g == "xy"
				and (pos - 1) % 4 == 0
		)

		x = string.pack(" b b Xd b Xb x", 1, 2, 3)
		assert(string.packsize(" b b Xd b Xb x") == 4)
		assert(x == "\1\2\3\0")
		a, b, c, pos = string.unpack("bbXdb", x)
		assert(a == 1 and b == 2 and c == 3 and pos == #x)

		-- only alignment
		assert(string.packsize("!8 xXi8") == 8)
		local pos = string.unpack("!8 xXi8", "0123456701234567")
		assert(pos == 9)
		assert(string.packsize("!8 xXi2") == 2)
		local pos = string.unpack("!8 xXi2", "0123456701234567")
		assert(pos == 3)
		assert(string.packsize("!2 xXi2") == 2)
		local pos = string.unpack("!2 xXi2", "0123456701234567")
		assert(pos == 3)
		assert(string.packsize("!2 xXi8") == 2)
		local pos = string.unpack("!2 xXi8", "0123456701234567")
		assert(pos == 3)
		assert(string.packsize("!16 xXi16") == 16)
		local pos = string.unpack("!16 xXi16", "0123456701234567")
		assert(pos == 17)

		-- checkerror("invalid next option", string.pack, "X")
		-- checkerror("invalid next option", string.unpack, "XXi", "")
		-- checkerror("invalid next option", string.unpack, "X i", "")
		-- checkerror("invalid next option", string.pack, "Xc1")
	end

	do -- testing initial position
		local x = string.pack("i4i4i4i4", 1, 2, 3, 4)
		for pos = 1, 16, 4 do
			local i, p = string.unpack("i4", x, pos)
			assert(i == pos // 4 + 1 and p == pos + 4)
		end

		-- with alignment
		for pos = 0, 12 do -- will always round position to power of 2
			local i, p = string.unpack("!4 i4", x, pos + 1)
			assert(i == (pos + 3) // 4 + 1 and p == i * 4 + 1)
		end

		-- negative indices
		local i, p = string.unpack("!4 i4", x, -4)
		assert(i == 4 and p == 17)
		local i, p = string.unpack("!4 i4", x, -7)
		assert(i == 4 and p == 17)
		local i, p = string.unpack("!4 i4", x, -#x)
		assert(i == 1 and p == 5)

		-- limits
		for i = 1, #x + 1 do
			assert(string.unpack("c0", x, i) == "")
		end
		-- checkerror("out of string", string.unpack, "c0", x, #x + 2)
	end
end

return stringsTest
