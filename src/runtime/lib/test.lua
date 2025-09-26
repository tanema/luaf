local suites = {}
local assertions = 0
local testResults = {
	pass = {},
	fail = {},
	skip = {},
	error = {},
}
local dotCh = {
	pass = ".",
	fail = "F",
	skip = "S",
	error = "E",
}

function table.count(tbl)
	local count = 0
	for _ in pairs(tbl) do
		count = count + 1
	end
	return count
end

local function printf(msg, ...)
	print(string.format(msg, ...))
end

local function callHook(fn, ...)
	if fn and type(fn) == "function" then
		fn(...)
	end
end

local function fmtDuration(t)
	assert(type(t) == "number", string.format("bad argument #1 to fmtDuration (number expected, got %s)", type(t)))
	local unit = "s"
	if t < 1 then
		unit, t = "ms", t * 1000
	end
	return string.format("%.2f %s", t, unit)
end

local defaultHooks = {
	postTest = function(_, res)
		io.write(dotCh[res.type])
	end,
	done = function(r, elapsed)
		local ps, fs, ss, es = table.count(r.pass), table.count(r.fail), table.count(r.skip), table.count(r.error)
		printf("\nFinished in %s with %d assertions", fmtDuration(elapsed), assertions)
		printf("%d passed, %d failed, %d error(s), %d skipped.", ps, fs, es, ss)
	end,
}

local verboseHooks = {
	beginSuite = function(suite)
		printf("== %s", suite.name)
	end,
	postTest = function(name, res)
		printf(
			"  %s\t%s\t(%s)\t%s",
			string.upper(res.type),
			name,
			fmtDuration(res.elapsed),
			res.msg and tostring(res.msg) or ""
		)
	end,
}

local function runSuite(hooks, suite)
	if table.count(suite.tests) == 0 then
		return
	end

	callHook(suite.ssetup)
	callHook(hooks.beginSuite, suite)
	for name, testFn in pairs(suite.tests) do
		callHook(hooks.preTest, name)
		callHook(suite.setup, name)
		local startTime = os.time()
		local ok, result = pcall(testFn)
		local elapsed = os.time() - startTime
		local isTestResult = type(result) == "table" and result.type and dotCh[result.type]
		if ok then
			result = { type = "pass" }
		elseif not ok and not isTestResult then
			result = { type = "error", msg = tostring(result) }
		end
		callHook(suite.teardown, name, elapsed)
		result.elapsed = elapsed
		testResults[result.type][suite.name .. "." .. name] = result
		callHook(hooks.postTest, name, result)
	end
	callHook(suite.steardown)
	callHook(hooks.endSuite, testResults)
end

local function fail(msg, ...)
	error({ type = "fail", msg = string.format(msg, ...) })
end

local function skip(msg)
	error({ type = "skip", msg = msg })
end

local function addSuite(modname, mod)
	assert(
		type(modname) == "string",
		string.format("bad argument #1 to testing.suite (string expected, got %s)", type(modname))
	)

	if not mod then
		mod = require(modname)
	end

	assert(type(mod) == "table", string.format("bad argument #2 to testing.suite (table expected, got %s)", type(mod)))

	local tests = {}
	for k, v in pairs(mod) do
		if type(k) == "string" and (k:match("^test.*") or k:match("test$")) and type(v) == "function" then
			tests[k] = v
		end
	end

	table.insert(suites, {
		name = modname,
		tests = tests,
		setup = rawget(mod, "setup"),
		teardown = rawget(mod, "teardown"),
		ssetup = rawget(mod, "suiteSetup"),
		steardown = rawget(mod, "suiteTeardown"),
	})
end

local function runTests(cfg)
	local opts = cfg or {}
	local hooks = cfg.hooks or {}
	if opts.verbose then
		hooks = verboseHooks
	end
	setmetatable(hooks, { __index = defaultHooks })
	callHook(hooks.begin, suites)
	local startTime = os.time()
	for _, suite in ipairs(suites) do
		runSuite(hooks, suite)
	end
	local elapsed = os.time() - startTime
	callHook(hooks.done, testResults, elapsed)
	if table.count(testResults.error) + table.count(testResults.fail) > 0 then
		os.exit(1)
	end
end

local function customAssert(got, msg, ...)
	assertions = assertions + 1
	if not got then
		fail("assertion failed!" .. msg, ...)
	end
end

local function assertEq(expected, actual, msg, ...)
	customAssert(expected == actual, "expected %v to equal %v " .. msg, expected, actual, ...)
end

local function assertNotEq(expected, actual, msg, ...)
	customAssert(expected ~= actual, "expected %v to not equal %v " .. msg, expected, actual, ...)
end

local function assertNil(actual, msg, ...)
	customAssert(actual == nil, "expected %v to equal nil" .. msg, actual, ...)
end

local function assertNotNil(actual, msg, ...)
	customAssert(actual ~= nil, "expected %v to not equal nil" .. msg, actual, ...)
end

local function assertLen(actual, expectedLen, msg, ...)
	if type(actual) ~= "string" and type(actual) ~= "table" then
		error({ type = "error", msg = string.format("assertion failed! value is %v" .. msg, type(actual), ...) })
	end
	customAssert(
		#actual == expectedLen,
		"expected length to be equal to %v but got %v" .. msg,
		expectedLen,
		#actual,
		...
	)
end

local function assertError(fn, msg, ...)
	error({ type = "error", msg = "bad argument #1 to assertError, should be function" })
	local ok, result = pcall(fn)
	if ok then
		fail("expected error from function but it succeeded" .. msg, ...)
	end
end

return {
	run = runTests,
	suite = addSuite,
	describe = addSuite,
	skip = skip,
	fail = fail,
	-- assertion helpers
	assert = customAssert,
	assertEq = assertEq,
	assertNotEq = assertNotEq,
	assertNil = assertNil,
	assertNotNil = assertNotNil,
	assertLen = assertLen,
	assertError = assertError,
}
