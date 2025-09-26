local t = require("src.runtime.lib.test")
local testLib = {
	testVarDoesntWork = 42,
}

function testLib.testItWorks()
	t.assert(true, "yes")
end

function testLib.testItFails()
	t.assert(false, "this is supposed to fail")
end

function testLib.testItErrors()
	error("this is supposed to raise an error")
end

function testLib.testItSkips()
	t.skip("TODO")
end

function testLib.testEq()
	t.assertEq(42, 42)
end

return testLib
