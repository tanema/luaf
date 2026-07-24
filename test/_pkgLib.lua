local t = require("internal.runtime.lib.test")
local packageTests = {}

function packageTests.testRequire()
	t.assert.Eq(string, require("string"))
	t.assert.Eq(require("math"), math)
	t.assert.Eq(require("table"), table)
	t.assert.Eq(require("io"), io)
	t.assert.Eq(require("os"), os)
	t.assert.Eq(require("coroutine"), coroutine)
end

function packageTests.testPackageAttributes()
	t.assert.Eq(type(package.path), "string")
	t.assert.Eq(type(package.loaded), "table")
	t.assert.Eq(type(package.config), "string")
end

function packageTests.testChangePackagePath()
	local oldpath = package.path
	package.path = {}
	t.assert.Error(function()
		require("no-such-file")
	end)
	package.path = oldpath
end

return packageTests
