local t = require("src.runtime.lib.test")
local bitwiseTests = {}

function bitwiseTests.testBasic()
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

	a, b, c, d = 0xF0.0, 0xCC.0, 0xAA.0, 0xFD.0
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
	t.assertEq(0, (2 ^ 30 - 1) << 2 ^ 30)
	t.assertEq(0, (2 ^ 30 - 1) >> 2 ^ 30)
	t.assertEq(1 >> -3, 1 << 3)
	t.assertEq(1000 >> 5, 1000 << -5)
	t.assertEq(73 << 1, 146)
	t.assertEq(0xffffffff >> 19, 0x1fff)
	t.assertEq(10 | 19, 27)
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
