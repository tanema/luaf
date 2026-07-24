local t = require("internal.runtime.lib.test")
local stringsTest = {}

function stringsTest.testStringComparision()
	t.assert.True("alo" < "alo1")
	t.assert.True("" < "a")
	t.assert.True("alo\0alo" < "alo\0b")
	t.assert.True("alo\0alo\0\0" > "alo\0alo\0")
	t.assert.True("alo" < "alo\0")
	t.assert.True("alo\0" > "alo")
	t.assert.True("\0" < "\1")
	t.assert.True("\0\0" < "\0\1")
	t.assert.True("\1\0a\0a" <= "\1\0a\0a")
	t.assert.True(not ("\1\0a\0b" <= "\1\0a\0a"))
	t.assert.True("\0\0\0" < "\0\0\0\0")
	t.assert.True(not ("\0\0\0\0" < "\0\0\0"))
	t.assert.True("\0\0\0" <= "\0\0\0\0")
	t.assert.True(not ("\0\0\0\0" <= "\0\0\0"))
	t.assert.True("\0\0\0" <= "\0\0\0")
	t.assert.True("\0\0\0" >= "\0\0\0")
	t.assert.True(not ("\0\0b" < "\0\0a\0"))
end

function stringsTest.testStringSub()
	t.assert.Eq("234", string.sub("123456789", 2, 4))
	t.assert.Eq("789", string.sub("123456789", 7))
	t.assert.Eq("", string.sub("123456789", 7, 6))
	t.assert.Eq("7", string.sub("123456789", 7, 7))
	t.assert.Eq("", string.sub("123456789", 0, 0))
	t.assert.Eq("123456789", string.sub("123456789", -10, 10))
	t.assert.Eq("123456789", string.sub("123456789", 1, 9))
	t.assert.Eq("", string.sub("123456789", -10, -20))
	t.assert.Eq("9", string.sub("123456789", -1))
	t.assert.Eq("6789", string.sub("123456789", -4))
	t.assert.Eq("456", string.sub("123456789", -6, -4))
	t.assert.Eq("234", string.sub("\000123456789", 3, 5))
	t.assert.Eq("789", ("\000123456789"):sub(8))
end

function stringsTest.testStringFind()
	t.assert.Eq(3, string.find("123456789", "345"))
	local a, b = string.find("123456789", "345")
	t.assert.Eq("345", string.sub("123456789", a, b))
	t.assert.Eq(3, string.find("1234567890123456789", "345", 3))
	t.assert.Eq(13, string.find("1234567890123456789", "345", 4))
	t.assert.Nil(string.find("1234567890123456789", "346", 4))
	t.assert.Eq(13, string.find("1234567890123456789", ".45", -9))
	t.assert.Nil(string.find("abcdefg", "\0", 5))
	t.assert.Nil(string.find("", "", 2))
	t.assert.Nil(string.find("", "aaa", 1))
	t.assert.Eq(4, ("alo(.)alo"):find("(.)", 1, true))

	local function f(s, p)
		local i, e = string.find(s, p)
		if i then
			return string.sub(s, i, e)
		end
	end

	a, b = string.find("", "") -- empty patterns are tricky
	t.assert.Eq(1, a)
	t.assert.Eq(0, b)
	a, b = string.find("alo", "")
	t.assert.Eq(1, a)
	t.assert.Eq(0, b)
	a, b = string.find("a\0o a\0o a\0o", "a", 1) -- first position
	t.assert.Eq(1, a)
	t.assert.Eq(1, b)
	a, b = string.find("a\0o a\0o a\0o", "a\0o", 2) -- starts in the midle
	t.assert.Eq(5, a)
	t.assert.Eq(7, b)
	a, b = string.find("a\0o a\0o a\0o", "a\0o", 9) -- starts in the midle
	t.assert.Eq(9, a)
	t.assert.Eq(11, b)
	a, b = string.find("a\0a\0a\0a\0\0ab", "\0ab", 2) -- finds at the end
	t.assert.Eq(9, a)
	t.assert.Eq(11, b)
	a, b = string.find("a\0a\0a\0a\0\0ab", "b") -- last position
	t.assert.Eq(11, a)
	t.assert.Eq(11, b)

	t.assert.False(string.find("a\0a\0a\0a\0\0ab", "b\0")) -- check ending
	t.assert.False(string.find("", "\0"))
	t.assert.Eq(4, string.find("alo123alo", "12"))
	t.assert.False(string.find("alo123alo", "^12"))
	t.assert.Eq("alo", f("aloALO", "%l*"))
	t.assert.Eq("aLo", f("aLo_ALO", "%a*"))
	t.assert.Eq("aaa", f("aaab", "a*"))
	t.assert.Eq("aaa", f("aaa", "^.*$"))
	t.assert.Eq("aa", f("aaa", "ab*a"))
	t.assert.Eq("aba", f("aba", "ab*a"))
	t.assert.Eq("aaa", f("aaab", "a+"))
	t.assert.Eq("aaa", f("aaa", "^.+$"))
	t.assert.False(f("aaa", "b+"))
	t.assert.False(f("aaa", "ab+a"))
	t.assert.Eq("aba", f("aba", "ab+a"))
	t.assert.Eq("a", f("a$a", ".$"))
	t.assert.Eq("a$", f("a$a", ".%$"))
	t.assert.Eq("a$a", f("a$a", ".$."))
	t.assert.False(f("a$a", "$$"))
	t.assert.False(f("a$b", "a$"))
	t.assert.False(f("aaa", "bb*"))
	t.assert.Eq("aaa", f("aaa", "^.-$"))
	t.assert.Eq("baaabaaabaaab", f("aabaaabaaabaaaba", "b.*b"))
	t.assert.Eq("baaab", f("aabaaabaaabaaaba", "b.-b"))
	t.assert.Eq("xo", f("alo xo", ".o$"))
	t.assert.Eq("isto", f(" \n isto é assim", "%S%S*"))
	t.assert.Eq("?", f("um caracter ? extra", "[^%sa-z]"))
	t.assert.Eq("á", f("á", "á?"))
	t.assert.Eq("ábl", f("ábl", "á?b?l?"))
	t.assert.Eq("aa", f("aa", "^aa?a?a"))
	t.assert.Eq("áb", f("]]]áb", "[^]]+"))
	t.assert.Eq("0a", f("0alo alo", "%x*"))
	t.assert.Eq("alo alo", f("alo alo", "%C+"))

	t.assert.Eq("xuxu", f("  \n\r*&\n\r   xuxu  \n\n", "%g%g%g+"))
	-- these patterns can only ever match the empty string, so find still reports a
	-- (zero-length) match rather than failing outright: f() returns "", not nil.
	t.assert.Eq("", f("aaa", "b*"))
	t.assert.Eq("", f("a$a", "$"))
	t.assert.Eq("", f("", "b*"))
	t.assert.Eq("", f("aaab", "a-"))
	t.assert.Eq("", f("", "a?"))
	t.assert.Eq("ábl", f("  ábl", "á?b?l?"))

	t.assert.Eq("assim", f(" \n isto é assim", "[a-z]*$"))
	t.assert.Eq("assim", f(" \n isto é assim", "%S*$"))
end

function stringsTest.testStringLen()
	t.assert.Eq(0, string.len(""))
	t.assert.Eq(3, string.len("\0\0\0"))
	t.assert.Eq(10, string.len("1234567890"))
	t.assert.Eq(0, #"")
	t.assert.Eq(3, #"\0\0\0")
	t.assert.Eq(10, #"1234567890")
end

function stringsTest.testStringByte()
	t.assert.Eq(97, string.byte("a"))
	t.assert.Eq(92, string.byte("\x5c"))
	t.assert.Eq(255, string.byte("\255"))
	t.assert.Eq(255, string.byte(string.char(255)))
	t.assert.Eq(0, string.byte(string.char(0)))
	t.assert.Eq(0, string.byte("\0"))
	t.assert.Eq(120, string.byte("\0\0alo\0x", -1))
	t.assert.Eq(120, string.byte("x"))
	t.assert.Eq(string.byte("\0\0alo\0x", -1), string.byte("x"))
	t.assert.Eq(97, string.byte("ba", 2))
	t.assert.Eq(97, string.byte("ba", 2, -1))
	t.assert.Eq(97, string.byte("ba", 2, 2))
	t.assert.Nil(string.byte(""))
	t.assert.Nil(string.byte("hi", -3))
	t.assert.Nil(string.byte("hi", 3))
	t.assert.Nil(string.byte("hi", 9, 10))
	t.assert.Nil(string.byte("hi", 2, 1))
end

function stringsTest.testStringChar()
	t.assert.Eq("", string.char())
	t.assert.Eq("a", string.char(97))
	t.assert.Eq("\xff", string.char(255))
	t.assert.Eq("\0\xe4\0", string.char(0, string.byte("\xe4"), 0))
	t.assert.Eq("\xe4l\0óu", string.char(string.byte("\xe4l\0óu", 1, -1)))
	t.assert.Eq("", string.char(string.byte("\xe4l\0óu", 1, 0)))
	t.assert.Eq("\xe4l\0óu", string.char(string.byte("\xe4l\0óu", 1, 100)))
end

function stringsTest.testStringUpper()
	t.assert.Eq("AB\0C", string.upper("ab\0c"))
end

function stringsTest.testStringLower()
	t.assert.Eq("\0abcc%$", string.lower("\0ABCc%$"))
end

function stringsTest.testStringRep()
	t.assert.Eq("", string.rep("teste", 0))
	t.assert.Eq("tés\00têtés\00tê", string.rep("tés\00tê", 2))
	t.assert.Eq("", string.rep("", 10))
	t.assert.Eq("", string.rep("teste", 0, "xuxu"))
	t.assert.Eq("teste", string.rep("teste", 1, "xuxu"))
	t.assert.Eq("\1\0\1\0\0\1\0\1", string.rep("\1\0\1", 2, "\0\0"))
	t.assert.Eq(string.rep("", 10, "."), string.rep(".", 9))
end

function stringsTest.testStringReverse()
	t.assert.Eq("", string.reverse(""))
	t.assert.Eq("43210", string.reverse("01234"))
end

function stringsTest.testToString()
	t.assert.Eq("string", type(tostring(nil)))
	t.assert.Eq("string", type(tostring(12)))
	t.assert.Eq("table:", string.sub(tostring({}), 1, 6))
	t.assert.Eq("function:", string.sub(tostring(print), 1, 9))
	t.assert.Eq(1, #tostring("\0"))
	t.assert.Eq("true", tostring(true))
	t.assert.Eq("false", tostring(false))
	t.assert.Eq("-1203", tostring(-1203))
	t.assert.Eq("1203.125", tostring(1203.125))
	t.assert.Eq("-0.5", tostring(-0.5))
	t.assert.Eq("-32767", tostring(-32767))
	t.assert.Eq("0.1", tostring(0.1))
	t.assert.Eq("12", "" .. 12)
	t.assert.Eq("12.1", 12.1 .. "")
	t.assert.Eq("-1203.1", tostring(-1203 + -0.1))
end

function stringsTest.testStringFormat()
	local x = '"ílo"\n\\'
	t.assert.Eq([["\x00"]], string.format("%q", "\0"))
	t.assert.Eq(x, load(string.format("return %q", x))())

	x = "\0\1\0023\5\0009"
	t.assert.Eq(x, load(string.format("return %q", x))())
	t.assert.Eq("\0\xe4\0b8c\0", string.format("\0%c\0%c%x\0", string.byte("\xe4"), string.byte("b"), 140))
	t.assert.Eq("", string.format(""))
	t.assert.Eq(
		string.format("%c", 34) .. string.format("%c", 48) .. string.format("%c", 90) .. string.format("%c", 100),
		string.format("%1c%-c%-1c%c", 34, 48, 90, 100)
	)
	t.assert.Eq("not be\0 is not \0be", string.format("%s\0 is not \0%s", "not be", "be"))
	t.assert.Eq("%10 0000000023", string.format("%%%d %010d", 10, 23))
	t.assert.Eq(10.3, tonumber(string.format("%f", 10.3)))
	t.assert.Eq('"a' .. string.rep(" ", 49) .. '"', string.format('"%-50s"', "a"))
	t.assert.Eq("-" .. string.rep("%", 20) .. ".20s", string.format("-%.20s.20s", string.rep("%", 2000)))
	t.assert.Eq(
		string.format("%q", "-" .. string.rep("%", 2000) .. ".20s"),
		string.format('"-%20s.20s"', string.rep("%", 2000))
	)
	t.assert.Eq("0.1", string.format("%q", 0.1))
	t.assert.Eq("nil", string.format("%q", nil))
	t.assert.Eq("true", string.format("%q", true))
	t.assert.Eq("false", string.format("%q", false))
	t.assert.Eq('"test"', string.format("%q", "test"))
	t.assert.Eq("\0\0\0\1\0", string.format("\0%s\0", "\0\0\1"))
	t.assert.Eq("nil true", string.format("%s %s", nil, true))
	t.assert.Eq("false true", string.format("%s %.4s", false, true))
	t.assert.Eq("fal tru", string.format("%.3s %.3s", false, true))
	t.assert.Eq("0", string.format("%x", 0.0))
	t.assert.Eq("00", string.format("%02x", 0.0))
	t.assert.Eq("FFFFFFFF", string.format("%08X", 0xFFFFFFFF))
	t.assert.Eq("+0031501", string.format("%+08d", 31501))
	t.assert.Eq("-0030927", string.format("%+08d", -30927))
	t.assert.Eq("7fffffff", string.format("%x", 0x7fffffff))
	t.assert.Eq("80000000", string.sub(string.format("%x", -0x80000000), -8))
	t.assert.Eq("2147483647", string.format("%d", 0x7fffffff))
	t.assert.Eq("-2147483648", string.format("%d", -0x80000000))
	t.assert.Eq("4294967295", string.format("%u", 0xffffffff))
	t.assert.Eq("125715", string.format("%o", 0xABCD))
	t.assert.Eq("         012", string.format("%#12o", 10))
	t.assert.Eq("      0x64", string.format("%#10x", 100))
	t.assert.Eq("0X64             ", string.format("%#-17X", 100))
	t.assert.Eq("-000000000100", string.format("%013i", -100))
	t.assert.Eq("-00100", string.format("%2.5d", -100))
	t.assert.Eq("0", string.format("%.u", 0))
	t.assert.Eq("+000000000100.", string.format("%+#014.0f", 100))
	t.assert.Eq("a               ", string.format("%-16c", 97))
	t.assert.Eq("+1.5", string.format("%+.3G", 1.5))
	t.assert.Eq("", string.format("%.0s", "alo"))
	t.assert.Eq("", string.format("%.s", "alo"))
end

function stringsTest.testGmatchCoroutines()
	-- bug in Lua 5.3.2
	-- 'gmatch' iterator does not work across coroutines
	local f = string.gmatch("1 2 3 4 5", "%d+")
	t.assert.Eq("1", f())
	local co = coroutine.wrap(f)
	t.assert.Eq("2", co())
end

function stringsTest.testStringMatch()
	t.assert.Eq("xyz", string.match("alo xyzK", "(%w+)K"))
	t.assert.Eq("", string.match("254 K", "(%d*)K"))
	t.assert.Eq("", string.match("alo ", "(%w*)$"))
	t.assert.False(string.match("alo ", "(%w+)$"))

	-- "\xe2" (not "â"): patterns match bytes, not UTF-8 runes, so each "." here must
	-- line up with a single byte for the nested capture structure below to hold.
	local a, b, c, d, e = string.match("\xe2lo alo", "^(((.).). (%w*))$")
	t.assert.Eq("\xe2lo alo", a)
	t.assert.Eq("\xe2l", b)
	t.assert.Eq("\xe2", c)
	t.assert.Eq("alo", d)
	t.assert.Nil(e)

	a, b, c, d = string.match("0123456789", "(.+(.?)())")
	t.assert.Eq("0123456789", a)
	t.assert.Eq("", b)
	t.assert.Eq(11, c)
	t.assert.Eq(nil, d)

	t.assert.Eq("alo ", string.match(" alo aalo allo", "%f[%S](.-%f[%s].-%f[%S])"))
	t.assert.Eq("\0\1\2", string.match("ab\0\1\2c", "[\0-\2]+"))
	t.assert.Eq("\0", string.match("ab\0\1\2c", "[\0-\0]+"))
	t.assert.Eq("\0efg\0\1e\1", string.match("abc\0efg\0\1e\1g", "%b\0\1"))
	t.assert.Eq("\0\0\0", string.match("abc\0\0\0", "%\0+"))
	t.assert.Eq("\0\0", string.match("abc\0\0\0", "%\0%\0?"))
	t.assert.Eq("aaab", string.match("aaab", ".*b"))
	t.assert.Eq("aaa", string.match("aaa", ".*a"))
	t.assert.Eq("b", string.match("b", ".*b"))
	t.assert.Eq("aaab", string.match("aaab", ".+b"))
	t.assert.Eq("aaa", string.match("aaa", ".+a"))
	t.assert.False(string.match("b", ".+b"))
	t.assert.Eq("ab", string.match("aaab", ".?b"))
	t.assert.Eq("aa", string.match("aaa", ".?a"))
	t.assert.Eq("b", string.match("b", ".?b"))
end

function stringsTest.testStringGMatch()
	local function range(i, j)
		if i < j then
			return i, range(i + 1, j)
		end
		return i
	end

	local abc = string.char(range(0, 127)) .. string.char(range(128, 255))
	t.assert.Len(abc, 256)

	local function strset(p)
		local res = { s = "" }
		string.gsub(abc, p, function(c)
			res.s = res.s .. c
		end)
		return res.s
	end

	t.assert.Len(strset("[\200-\210]"), 11)
	t.assert.Eq(strset("[a-z]"), "abcdefghijklmnopqrstuvwxyz")
	t.assert.Eq(strset("[a-z%d]"), strset("[%da-uu-z]"))
	t.assert.Eq(strset("[a-]"), "-a")
	t.assert.Eq(strset("[^%W]"), strset("[%w]"))
	t.assert.Eq(strset("[]%%]"), "%]")
	t.assert.Eq(strset("[a%-z]"), "-az")
	t.assert.Eq(strset("[%^%[%-a%]%-b]"), "-[]^ab")
	t.assert.Eq(strset("%Z"), strset("[\1-\255]"))
	t.assert.Eq(strset("."), strset("[\1-\255%z]"))
	t.assert.Eq(string.find("(álo)", "%(á"), 1)
	t.assert.Eq(string.gsub("ülo ülo", "ü", "x"), "xlo xlo")
	t.assert.Eq(string.gsub("alo úlo  ", " +$", ""), "alo úlo") -- trim
	t.assert.Eq(string.gsub("  alo alo  ", "^%s*(.-)%s*$", "%1"), "alo alo") -- double trim
	t.assert.Eq(string.gsub("alo  alo  \n 123\n ", "%s+", " "), "alo alo 123 ")
	t.assert.Eq(string.gsub("alo alo", "()[al]", "%1"), "12o 56o")
	t.assert.Eq(string.gsub("abc=xyz", "(%w*)(%p)(%w+)", "%3%2%1-%0"), "xyz=abc-abc=xyz")
	t.assert.Eq(string.gsub("abc", "%w", "%1%0"), "aabbcc")
	t.assert.Eq(string.gsub("abc", "%w+", "%0%1"), "abcabc")
	t.assert.Eq(string.gsub("áéí", "$", "\0óú"), "áéí\0óú")
	t.assert.Eq(string.gsub("", "^", "r"), "r")
	t.assert.Eq(string.gsub("", "$", "r"), "r")

	do -- new (5.3.3) semantics for empty matches
		t.assert.Eq(string.gsub("a b cd", " *", "-"), "-a-b-c-d-")

		local res = ""
		local sub = "a  \nbc\t\td"
		local i = 1
		for p, e in string.gmatch(sub, "()%s*()") do
			res = res .. string.sub(sub, i, p - 1) .. "-"
			i = e
		end
		t.assert.Eq(res, "-a-b-c-d-")
	end

	t.assert.Eq(string.gsub("um (dois) tres (quatro)", "(%(%w+%))", string.upper), "um (DOIS) tres (QUATRO)")

	do
		local function setglobal(n, v)
			rawset(_G, n, v)
		end
		string.gsub("a=roberto,roberto=a", "(%w+)=(%w%w*)", setglobal)
		t.assert.Eq(_G.a, "roberto")
		t.assert.Eq(_G.roberto, "a")
		_G.a = nil
		_G.roberto = nil
	end

	function f(a, b)
		return string.gsub(a, ".", b)
	end

	t.assert.Eq(
		string.gsub("trocar tudo em |teste|b| é |beleza|al|", "|([^|]*)|([^|]*)|", f),
		"trocar tudo em bbbbb é alalalalalal"
	)

	local function dostring(s)
		return load(s, "")() or ""
	end
	t.assert.Eq(string.gsub("alo $a='x'$ novamente $return a$", "$([^$]*)%$", dostring), "alo  novamente x")

	local x = string.gsub("$x=string.gsub('alo', '.', string.upper)$ assim vai para $return x$", "$([^$]*)%$", dostring)
	t.assert.Eq(x, " assim vai para ALO")
	_G.a, _G.x = nil

	local positions = {}
	local s = "a alo jose  joao"
	local r = string.gsub(s, "()(%w+)()", function(a, w, b)
		t.assert.Eq(string.len(w), b - a)
		positions[a] = b - a
	end)
	assert(s == r and positions[1] == 1 and positions[3] == 3 and positions[7] == 4 and positions[13] == 4)

	local function isbalanced(s)
		return not string.find(string.gsub(s, "%b()", ""), "[()]")
	end

	t.assert.True(isbalanced("(9 ((8))(\0) 7) \0\0 a b ()(c)() a"))
	t.assert.False(isbalanced("(9 ((8) 7) a b (\0 c) a"))
	t.assert.Eq(string.gsub("alo 'oi' alo", "%b''", '"'), 'alo " alo')

	local tbl = { "apple", "orange", "lime", n = 0 }
	t.assert.Eq(
		string.gsub("x and x and x", "x", function()
			tbl.n = tbl.n + 1
			return tbl[tbl.n]
		end),
		"apple and orange and lime"
	)

	tbl = { n = 0 }
	string.gsub("first second word", "%w%w*", function(w)
		tbl.n = tbl.n + 1
		tbl[tbl.n] = w
	end)
	t.assert.Eq(tbl[1], "first")
	t.assert.Eq(tbl[2], "second")
	t.assert.Eq(tbl[3], "word")
	t.assert.Eq(tbl.n, 3)

	tbl = { n = 0 }
	t.assert.Eq(
		string.gsub("first second word", "%w+", function(w)
			tbl.n = tbl.n + 1
			tbl[tbl.n] = w
		end, 2),
		"first second word"
	)
	t.assert.Eq(tbl[1], "first")
	t.assert.Eq(tbl[2], "second")
	t.assert.Nil(tbl[3])

	t.assert.Error(function()
		string.gsub("alo", ".", { a = {} })
	end)
	t.assert.Error(function()
		string.gsub("alo", "(%0)", "a")
	end)
	t.assert.Error(function()
		string.gsub("alo", "(%1)", "a")
	end)
	t.assert.Error(function()
		string.gsub("alo", ".", "%x")
	end)

	local a = string.rep("a", 300000)
	t.assert.True(string.find(a, "^a*.?$"))
	t.assert.False(string.find(a, "^a*.?b$"))
	t.assert.True(string.find(a, "^a-.?$"))

	-- bug in 5.1.2
	a = string.rep("a", 10000) .. string.rep("b", 10000)
	t.assert.False(pcall(string.gsub, a, "b"))

	-- recursive nest of gsubs
	local function rev(s)
		return string.gsub(s, "(.)(.+)", function(c, s1)
			return rev(s1) .. c
		end)
	end

	local x = "abcdef"
	t.assert.Eq(rev(rev(x)), x)

	-- gsub with tables
	t.assert.Eq(string.gsub("alo alo", ".", {}), "alo alo")
	t.assert.Eq(string.gsub("alo alo", "(.)", { a = "AA", l = "" }), "AAo AAo")
	t.assert.Eq(string.gsub("alo alo", "(.).", { a = "AA", l = "K" }), "AAo AAo")
	t.assert.Eq(string.gsub("alo alo", "((.)(.?))", { al = "AA", o = false }), "AAo AAo")
	t.assert.Eq(string.gsub("alo alo", "().", { "x", "yy", "zzz" }), "xyyzzz alo")

	local replTbl = {}
	setmetatable(replTbl, {
		__index = function(_, s)
			return string.upper(s)
		end,
	})
	t.assert.Eq(string.gsub("a alo b hi", "%w%w+", replTbl), "a ALO b HI")

	-- tests for gmatch
	local a = 0
	for i in string.gmatch("abcde", "()") do
		t.assert.Eq(i, a + 1)
		a = i
	end
	t.assert.Eq(a, 6)

	tbl = { n = 0 }
	for w in string.gmatch("first second word", "%w+") do
		tbl.n = tbl.n + 1
		tbl[tbl.n] = w
	end
	t.assert.Eq(tbl[1], "first")
	t.assert.Eq(tbl[2], "second")
	t.assert.Eq(tbl[3], "word")

	tbl = { 3, 6, 9 }
	for i in string.gmatch("xuxx uu ppar r", "()(.)%2") do
		t.assert.Eq(i, table.remove(tbl, 1))
	end
	t.assert.Eq(#tbl, 0)

	tbl = {}
	for i, j in string.gmatch("13 14 10 = 11, 15= 16, 22=23", "(%d+)%s*=%s*(%d+)") do
		tbl[tonumber(i)] = tonumber(j)
	end
	a = 0
	for k, v in pairs(tbl) do
		t.assert.Eq(k + 1, v + 0)
		a = a + 1
	end
	t.assert.Eq(a, 3)

	do -- init parameter in gmatch
		local s = 0
		for k in string.gmatch("10 20 30", "%d+", 3) do
			s = s + tonumber(k)
		end
		t.assert.Eq(s, 50)

		s = 0
		for k in string.gmatch("11 21 31", "%d+", -4) do
			s = s + tonumber(k)
		end
		t.assert.Eq(s, 32)

		-- there is an empty string at the end of the subject
		s = 0
		for k in string.gmatch("11 21 31", "%w*", 9) do
			s = s + 1
		end
		t.assert.Eq(s, 1)

		-- there are no empty strings after the end of the subject
		s = 0
		for k in string.gmatch("11 21 31", "%w*", 10) do
			s = s + 1
		end
		t.assert.Eq(s, 0)
	end

	-- tests for `%f' (`frontiers')
	t.assert.Eq(string.gsub("aaa aa a aaa a", "%f[%w]a", "x"), "xaa xa x xaa x")
	t.assert.Eq(string.gsub("[[]] [][] [[[[", "%f[[].", "x"), "x[]] x]x] x[[[")
	t.assert.Eq(string.gsub("01abc45de3", "%f[%d]", "."), ".01abc.45de.3")
	t.assert.Eq(string.gsub("01abc45 de3x", "%f[%D]%w", "."), "01.bc45 de3.")
	t.assert.Eq(string.gsub("function", "%f[\1-\255]%w", "."), ".unction")
	t.assert.Eq(string.gsub("function", "%f[^\1-\255]", "."), "function.")

	t.assert.Eq(string.find("a", "%f[a]"), 1)
	t.assert.Eq(string.find("a", "%f[^%z]"), 1)
	t.assert.Eq(string.find("a", "%f[^%l]"), 2)
	t.assert.Eq(string.find("aba", "%f[a%z]"), 3)
	t.assert.Eq(string.find("aba", "%f[%z]"), 4)
	t.assert.False(string.find("aba", "%f[%l%z]"))
	t.assert.False(string.find("aba", "%f[^%l%z]"))

	local i, e = string.find(" alo aalo allo", "%f[%S].-%f[%s].-%f[%S]")
	t.assert.Eq(i, 2)
	t.assert.Eq(e, 5)

	local a = { 1, 5, 9, 14, 17 }
	for k in string.gmatch("alo alo th02 is 1hat", "()%f[%w%d]") do
		t.assert.Eq(table.remove(a, 1), k)
	end
	t.assert.Eq(#a, 0)

	-- malformed patterns
	local function malform(p, m)
		m = m or "malformed"
		local r, msg = pcall(string.find, "a", p)
		t.assert.True(not r and string.find(msg, m))
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
	t.assert.Eq(string.find("b$a", "$\0?"), 2)
	t.assert.Eq(string.find("abc\0efg", "%\0"), 4)

	-- magic char after \0
	t.assert.Eq(string.find("abc\0\0", "\0."), 4)
	t.assert.Eq(string.find("abcx\0\0abc\0abc", "x\0\0abc\0a."), 4)

	do -- test reuse of original string in gsub
		local s = string.rep("a", 100)
		local r = string.gsub(s, "b", "c") -- no match
		t.assert.Eq(string.format("%p", s), string.format("%p", r))

		r = string.gsub(s, ".", { x = "y" }) -- no substitutions
		t.assert.Eq(string.format("%p", s), string.format("%p", r))

		local count = 0
		r = string.gsub(s, ".", function(x)
			t.assert.Eq(x, "a")
			count = count + 1
			return nil -- no substitution
		end)
		r = string.gsub(r, ".", { b = "x" }) -- "a" is not a key; no subst.
		t.assert.Eq(count, 100)
		t.assert.Eq(string.format("%p", s), string.format("%p", r))

		count = 0
		r = string.gsub(s, ".", function(x)
			t.assert.Eq(x, "a")
			count = count + 1
			return x -- substitution...
		end)
		t.assert.Eq(count, 100)
		-- no reuse in this case
		t.assert.Eq(r, s)
		t.assert.NotEq(string.format("%p", s), string.format("%p", r))
	end
end

function stringsTest.testPackSize()
	t.assert.Eq(2, string.packsize("h"))
	t.assert.Eq(4, string.packsize("l"))
	t.assert.Eq(4, string.packsize("f"))
	t.assert.Eq(8, string.packsize("i"))
	t.assert.Eq(8, string.packsize("d"))
	t.assert.Eq(8, string.packsize("n"))
	t.assert.Eq(8, string.packsize("j"))
end

function stringsTest.testPackUnpack()
	t.assert.Eq(0xff, string.unpack("B", string.pack("B", 0xff)))
	t.assert.Eq(0x7f, string.unpack("b", string.pack("b", 0x7f)))
	t.assert.Eq(-0x80, string.unpack("b", string.pack("b", -0x80)))
	t.assert.Eq(0xffff, string.unpack("H", string.pack("H", 0xffff)))
	t.assert.Eq(0x7fff, string.unpack("h", string.pack("h", 0x7fff)))
	t.assert.Eq(-0x8000, string.unpack("h", string.pack("h", -0x8000)))
	t.assert.Eq(0xffffffff, string.unpack("L", string.pack("L", 0xffffffff)))
	t.assert.Eq(0x7fffffff, string.unpack("l", string.pack("l", 0x7fffffff)))
	t.assert.Eq(-0x80000000, string.unpack("l", string.pack("l", -0x80000000)))
end

function stringsTest.testPack()
	local NB = 16
	-- for i = 1, NB do
	-- -- small numbers with signal extension ("\xFF...")
	-- local s = string.rep("\xff", i)
	-- t.assert.Eq(string.pack("i" .. i, -1), s)
	-- t.assert.Eq(string.packsize("i" .. i), #s)
	-- t.assert.Eq(string.unpack("i" .. i, s), -1)

	-- -- small unsigned number ("\0...\xAA")
	-- s = "\xAA" .. string.rep("\0", i - 1)
	-- t.assert.Eq(string.pack("<I" .. i, 0xAA), s)
	-- t.assert.Eq(string.unpack("<I" .. i, s), 0xAA)
	-- t.assert.Eq(string.pack(">I" .. i, 0xAA), s:reverse())
	-- t.assert.Eq(string.unpack(">I" .. i, s:reverse()), 0xAA)
	-- end

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
	-- local sizeLI = string.packsize("j")
	-- do
	--	local u = 0xf0
	--	for i = 1, sizeLI - 1 do
	--		t.assert.Eq(string.unpack("<i" .. i, "\xf0" .. ("\xff"):rep(i - 1)), -16)
	--		t.assert.Eq(string.unpack(">I" .. i, "\xf0" .. ("\xff"):rep(i - 1)), u)
	--		u = u * 256 + 0xff
	--	end
	-- end

	-- mixed endianness
	-- do
	--	t.assert.Eq(string.pack(">i2 <i2", 10, 20) == "\0\10\20\0")
	--	local a, b = string.unpack("<i2 >i2", "\10\0\0\20")
	--	t.assert.Eq(a, 10)
	--	t.assert.Eq(b == 20)
	--	t.assert.Eq(string.pack("=i4", 2001), string.pack("i4", 2001))
	-- end

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
	-- if string.pack("i2", 1) == "\1\0" then
	--	t.assert.Eq(string.pack("f", 24), string.pack("<f", 24))
	-- else
	--	t.assert.Eq(string.pack("f", 24), string.pack(">f", 24))
	-- end

	-- for _, n in ipairs({ 0, -1.1, 1.9, 1 / 0, -1 / 0, 1e20, -1e20, 0.1, 2000.7 }) do
	--	t.assert.Eq(string.unpack("n", string.pack("n", n)), n)
	--	t.assert.Eq(string.unpack("<n", string.pack("<n", n)), n)
	--	t.assert.Eq(string.unpack(">n", string.pack(">n", n)), n)
	--	t.assert.Eq(string.pack("<f", n), string.pack(">f", n):reverse())
	--	t.assert.Eq(string.pack(">d", n), string.pack("<d", n):reverse())
	-- end

	-- for non-native precisions, test only with "round" numbers
	-- for _, n in ipairs({ 0, -1.5, 1 / 0, -1 / 0, 1e10, -1e9, 0.5, 2000.25 }) do
	--	t.assert.Eq(string.unpack("<f", string.pack("<f", n)), n)
	--	t.assert.Eq(string.unpack(">f", string.pack(">f", n)), n)
	--	t.assert.Eq(string.unpack("<d", string.pack("<d", n)), n)
	--	t.assert.Eq(string.unpack(">d", string.pack(">d", n)), n)
	-- end

	-- do
	--	local s = string.rep("abc", 1000)
	--	t.assert.Eq(string.pack("zB", s, 247), s .. "\0\xF7")
	--	local s1, b = string.unpack("zB", s .. "\0\xF9")
	--	t.assert.Eq(b, 249)
	--	t.assert.Eq(s1, s)
	--	s1 = string.pack("s", s)
	--	t.assert.Eq(string.unpack("s", s1), s)
	--	-- checkerror("does not fit", string.pack, "s1", s)
	--	-- checkerror("contains zeros", string.pack, "z", "alo\0")
	--	-- checkerror("unfinished string", string.unpack, "zc10000000", "alo")
	--	for i = 2, NB do
	--		local s1 = string.pack("s" .. i, s)
	--		t.assert.Eq(string.unpack("s" .. i, s1), s)
	--		t.assert.Eq(#s1, #s + i)
	--	end
	-- end

	-- do
	--	local x = string.pack("s", "alo")
	--	-- checkerror("too short", string.unpack, "s", x:sub(1, -2))
	--	-- checkerror("too short", string.unpack, "c5", "abcd")
	--	-- checkerror("out of limits", string.pack, "s100", "alo")
	-- end

	-- do
	--	assert(string.pack("c0", "") == "")
	--	assert(string.packsize("c0") == 0)
	--	assert(string.unpack("c0", "") == "")
	--	assert(string.pack("<! c3", "abc") == "abc")
	--	assert(string.packsize("<! c3") == 3)
	--	assert(string.pack(">!4 c6", "abcdef") == "abcdef")
	--	assert(string.pack("c3", "123") == "123")
	--	assert(string.pack("c0", "") == "")
	--	assert(string.pack("c8", "123456") == "123456\0\0")
	--	assert(string.pack("c88 c1", "", "X") == string.rep("\0", 88) .. "X")
	--	assert(string.pack("c188 c2", "ab", "X\1") == "ab" .. string.rep("\0", 188 - 2) .. "X\1")
	--	local a, b, c = string.unpack("!4 z c3", "abcdefghi\0xyz")
	--	assert(a == "abcdefghi" and b == "xyz" and c == 14)
	--	-- checkerror("longer than", string.pack, "c3", "1234")
	-- end

	-- -- testing multiple types and sequence
	-- do
	--	local x = string.pack("<b h b f d f n i", 1, 2, 3, 4, 5, 6, 7, 8)
	--	assert(#x == string.packsize("<b h b f d f n i"))
	--	local a, b, c, d, e, f, g, h = string.unpack("<b h b f d f n i", x)
	--	assert(a == 1 and b == 2 and c == 3 and d == 4 and e == 5 and f == 6 and g == 7 and h == 8)
	-- end

	-- do
	--	assert(string.pack(" < i1 i2 ", 2, 3) == "\2\3\0") -- no alignment by default
	--	local x = string.pack(">!8 b Xh i4 i8 c1 Xi8", -12, 100, 200, "\xEC")
	--	assert(#x == string.packsize(">!8 b Xh i4 i8 c1 Xi8"))
	--	assert(x == "\xf4" .. "\0\0\0" .. "\0\0\0\100" .. "\0\0\0\0\0\0\0\xC8" .. "\xEC" .. "\0\0\0\0\0\0\0")
	--	local a, b, c, d, pos = string.unpack(">!8 c1 Xh i4 i8 b Xi8 XI XH", x)
	--	assert(a == "\xF4" and b == 100 and c == 200 and d == -20 and (pos - 1) == #x)

	--	x = string.pack(">!4 c3 c4 c2 z i4 c5 c2 Xi4", "abc", "abcd", "xz", "hello", 5, "world", "xy")
	--	assert(x == "abcabcdxzhello\0\0\0\0\0\5worldxy\0")
	--	local a, b, c, d, e, f, g, pos = string.unpack(">!4 c3 c4 c2 z i4 c5 c2 Xh Xi4", x)
	--	assert(
	--		a == "abc"
	--			and b == "abcd"
	--			and c == "xz"
	--			and d == "hello"
	--			and e == 5
	--			and f == "world"
	--			and g == "xy"
	--			and (pos - 1) % 4 == 0
	--	)

	--	x = string.pack(" b b Xd b Xb x", 1, 2, 3)
	--	assert(string.packsize(" b b Xd b Xb x") == 4)
	--	assert(x == "\1\2\3\0")
	--	a, b, c, pos = string.unpack("bbXdb", x)
	--	assert(a == 1 and b == 2 and c == 3 and pos == #x)

	--	-- only alignment
	--	assert(string.packsize("!8 xXi8") == 8)
	--	local pos = string.unpack("!8 xXi8", "0123456701234567")
	--	assert(pos == 9)
	--	assert(string.packsize("!8 xXi2") == 2)
	--	local pos = string.unpack("!8 xXi2", "0123456701234567")
	--	assert(pos == 3)
	--	assert(string.packsize("!2 xXi2") == 2)
	--	local pos = string.unpack("!2 xXi2", "0123456701234567")
	--	assert(pos == 3)
	--	assert(string.packsize("!2 xXi8") == 2)
	--	local pos = string.unpack("!2 xXi8", "0123456701234567")
	--	assert(pos == 3)
	--	assert(string.packsize("!16 xXi16") == 16)
	--	local pos = string.unpack("!16 xXi16", "0123456701234567")
	--	assert(pos == 17)

	--	-- checkerror("invalid next option", string.pack, "X")
	--	-- checkerror("invalid next option", string.unpack, "XXi", "")
	--	-- checkerror("invalid next option", string.unpack, "X i", "")
	--	-- checkerror("invalid next option", string.pack, "Xc1")
	-- end

	-- do -- testing initial position
	--	local x = string.pack("i4i4i4i4", 1, 2, 3, 4)
	--	for pos = 1, 16, 4 do
	--		local i, p = string.unpack("i4", x, pos)
	--		assert(i == pos // 4 + 1 and p == pos + 4)
	--	end

	--	-- with alignment
	--	for pos = 0, 12 do -- will always round position to power of 2
	--		local i, p = string.unpack("!4 i4", x, pos + 1)
	--		assert(i == (pos + 3) // 4 + 1 and p == i * 4 + 1)
	--	end

	--	-- negative indices
	--	local i, p = string.unpack("!4 i4", x, -4)
	--	assert(i == 4 and p == 17)
	--	local i, p = string.unpack("!4 i4", x, -7)
	--	assert(i == 4 and p == 17)
	--	local i, p = string.unpack("!4 i4", x, -#x)
	--	assert(i == 1 and p == 5)

	--	-- limits
	--	for i = 1, #x + 1 do
	--		assert(string.unpack("c0", x, i) == "")
	--	end
	--	-- checkerror("out of string", string.unpack, "c0", x, #x + 2)
	-- end
end

return stringsTest
