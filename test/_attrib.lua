local t = require("src.runtime.lib.test")
local attribTests = {}

function attribTests.testRequire()
	t.assert(string, require("string"))
	t.assert(require("math"), math)
	t.assert(require("table"), table)
	t.assert(require("io"), io)
	t.assert(require("os"), os)
	t.assert(require("coroutine"), coroutine)
end

function attribTests.testPackageAttributes()
	t.assertEq(type(package.path), "string")
	t.assertEq(type(package.loaded), "table")
	t.assertEq(type(package.config), "string")
end

function attribTests.testChangePackagePath()
	local oldpath = package.path
	package.path = {}
	t.assertError(function()
		require("no-such-file")
	end)
	package.path = oldpath
end

return attribTests
