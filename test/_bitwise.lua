local t = require("src.runtime.lib.test")
local bitwiseTests = {}

function bitwiseTests.testBasic()
	t.skip("TODO")
	t.assertEq(~0, -1)
	t.assertEq(0x123456780, 0x12345678 << 4)
	t.assertEq(0x1234567800, 0x12345678 << 8)
	t.assertEq(0x01234567, 0x12345678 << -4)
	t.assertEq(0x00123456, 0x12345678 << -8)
	t.assertEq(0x1234567800000000, 0x12345678 << 32)
	t.assertEq(0, 0x12345678 << -32)
	t.assertEq(0x01234567, 0x12345678 >> 4)
	t.assertEq(0x00123456, 0x12345678 >> 8)
	t.assertEq(0, 0x12345678 >> 32)
	t.assertEq(0x1234567800000000, 0x12345678 >> -32)

	local a, b, c, d = 0xF0, 0xCC, 0xAA, 0xFD
	t.assertEq(0xF4, a | b ~ c & d)

	a, b, c, d = 0xF0.0, 0xCC.0, "0xAA.0", "0xFD.0"
	t.assertEq(0xF4, a | b ~ c & d)

	a, b, c, d = 0xF0000000, 0xCC000000, 0xAA000000, 0xFD000000
	t.assertEq(0xF4000000, a | b ~ c & d)
	t.assertEq(~~a, a)
	t.assertEq(~a, -1 ~ a)
	t.assertEq(-d, ~d + 1)

	a, b, c, d = a << 32, b << 32, c << 32, d << 32
	t.assertEq(a | b ~ c & d, 0xF4000000 << 32)
	t.assertEq(~~a, a)
	t.assertEq(~a, -1 ~ a)
	t.assertEq(-d, ~d + 1)

	local numbits = string.packsize("j") * 8
	t.assertEq(-1 >> 1, (1 << (numbits - 1)) - 1)
	t.assertEq(1 << 31, 0x80000000)
	t.assertEq(1, -1 >> (numbits - 1))
	t.assertEq(0, -1 >> numbits)
	t.assertEq(0, -1 >> -numbits)
	t.assertEq(0, -1 << numbits)
	t.assertEq(0, -1 << -numbits)
	t.assertEq(0, (2 ^ 30 - 1) << 2 ^ 30)
	t.assertEq(0, (2 ^ 30 - 1) >> 2 ^ 30)

	t.assertEq(1 >> -3, 1 << 3)
	t.assertEq(1000 >> 5, 1000 << -5)

	t.assertEq("0xffffffffffffffff" | 0, -1)
	t.assertEq("0xfffffffffffffffe" & "-1", -2)
	t.assertEq(" \t-0xfffffffffffffffe\n\t" & "-1", 2)
	t.assertEq("   \n  -45  \t " >> "  -2  ", -45 * 4)
	t.assertEq("1234.0" << "5.0", 1234 * 32)
	t.assertEq("0xffff.0" ~ "0xAAAA", 0x5555)
	t.assertEq(~"0x0.000p4", -1)

	t.assertEq(("7" .. 3) << 1, 146)
	t.assertEq(0xffffffff >> (1 .. "9"), 0x1fff)
	t.assertEq(10 | (1 .. "9"), 27)
end

function bitwiseTests.testErrOperations()
	t.assertError(function()
		return 4 & "a"
	end)
	t.assertError(function()
		return ~"a"
	end)
	t.assertError(function()
		return "0xffffffffffffffff.0" | 0
	end)
	t.assertError(function()
		return "0xffffffffffffffff\0" | 0
	end)
end

return bitwiseTests
