local t = require("src.runtime.lib.test")
local packageTests = {}

function packageTests.testRequire()
	t.assertEq(string, require("string"))
	t.assertEq(require("math"), math)
	t.assertEq(require("table"), table)
	t.assertEq(require("io"), io)
	t.assertEq(require("os"), os)
	t.assertEq(require("coroutine"), coroutine)
end

function packageTests.testPackageAttributes()
	t.assertEq(type(package.path), "string")
	t.assertEq(type(package.loaded), "table")
	t.assertEq(type(package.config), "string")
end

function packageTests.testChangePackagePath()
	local oldpath = package.path
	package.path = {}
	t.assertError(function()
		require("no-such-file")
	end)
	package.path = oldpath
end

return packageTests
