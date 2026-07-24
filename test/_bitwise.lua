local t = require("internal.runtime.lib.test")
local bitwiseTests = {}

function bitwiseTests.testBasic()
	t.assert.Eq(~0, -1)
	t.assert.Eq(0x123456780, 0x12345678 << 4)
	t.assert.Eq(0x1234567800, 0x12345678 << 8)
	t.assert.Eq(0x01234567, 0x12345678 << -4)
	t.assert.Eq(0x00123456, 0x12345678 << -8)
	t.assert.Eq(0x1234567800000000, 0x12345678 << 32)
	t.assert.Eq(0, 0x12345678 << -32)
	t.assert.Eq(0x01234567, 0x12345678 >> 4)
	t.assert.Eq(0x00123456, 0x12345678 >> 8)
	t.assert.Eq(0, 0x12345678 >> 32)
	t.assert.Eq(0x1234567800000000, 0x12345678 >> -32)

	local a, b, c, d = 0xF0, 0xCC, 0xAA, 0xFD
	t.assert.Eq(0xF4, a | b ~ c & d)

	a, b, c, d = 0xF0.0, 0xCC.0, 0xAA.0, 0xFD.0
	t.assert.Eq(0xF4, a | b ~ c & d)

	a, b, c, d = 0xF0000000, 0xCC000000, 0xAA000000, 0xFD000000
	t.assert.Eq(0xF4000000, a | b ~ c & d)
	t.assert.Eq(~~a, a)
	t.assert.Eq(~a, -1 ~ a)
	t.assert.Eq(-d, ~d + 1)

	a, b, c, d = a << 32, b << 32, c << 32, d << 32
	t.assert.Eq(a | b ~ c & d, 0xF4000000 << 32)
	t.assert.Eq(~~a, a)
	t.assert.Eq(~a, -1 ~ a)
	t.assert.Eq(-d, ~d + 1)
	t.assert.Eq(0, (2 ^ 30 - 1) << 2 ^ 30)
	t.assert.Eq(0, (2 ^ 30 - 1) >> 2 ^ 30)
	t.assert.Eq(1 >> -3, 1 << 3)
	t.assert.Eq(1000 >> 5, 1000 << -5)
	t.assert.Eq(73 << 1, 146)
	t.assert.Eq(0xffffffff >> 19, 0x1fff)
	t.assert.Eq(10 | 19, 27)
end

function bitwiseTests.testErrOperations()
	t.assert.Error(function()
		return 4 & "a"
	end)
	t.assert.Error(function()
		return ~"a"
	end)
	t.assert.Error(function()
		return "0xffffffffffffffff.0" | 0
	end)
	t.assert.Error(function()
		return "0xffffffffffffffff\0" | 0
	end)
end

return bitwiseTests
