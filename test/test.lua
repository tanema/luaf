local t = require("test")

t.describe("Test Library", {
	test_it_works = function()
		t.assert(true, "yes")
	end,
	test_it_fails = function()
		t.assert(false, "this is supposed to fail")
	end,
	test_it_errors = function()
		error("yikes")
	end,
	test_it_skips = function()
		t.skip("TODO")
	end,
	testMore = function()
		t.assert(42 == 42, "expected 42 but got %v", 42)
	end,
	test_var_doesnt_work = 42,
})

t.run({ verbose = true })
