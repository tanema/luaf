-- this is really only used for development of the test library because it
-- messes up test total if run on it's own
local t = require("src.runtime.lib.test")
local testLib = {
	testVarDoesntWork = 42,
}

function testLib.testItWorks()
	t.assertTrue(true, "yes")
end

function testLib.testItFails()
	t.assertTrue(false, "this is supposed to fail")
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
