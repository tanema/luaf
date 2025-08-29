local suites = {}
local assertions = 0
local dotCh = {
	pass = ".",
	fail = "F",
	skip = "S",
	error = "E",
}

local function printf(...)
	print(string.format(...))
end

local function callHook(fn, ...)
	if fn and type(fn) == "function" then
		fn(...)
	end
end

local function count(t)
	local ct = 0
	for _ in pairs(t) do
		ct = ct + 1
	end
	return ct
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
	done = function(r)
		local ps, fs, ss, es = count(r.pass), count(r.fail), count(r.skip), count(r.error)
		printf("\nFinished in %s with %d assertions", fmtDuration(os.time() - r.startTime), assertions)
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

local function errHandler(e)
	if type(e) == "table" and e.type and dotCh[e.type] then
		return e
	end
	return { type = "error", msg = tostring(e) }
end

local function runSuite(hooks, results, suite)
	if count(suite.tests) == 0 then
		return
	end

	callHook(suite.ssetup)
	callHook(hooks.beginSuite, suite)
	for name, testFn in pairs(suite.tests) do
		callHook(hooks.preTest, name)
		callHook(suite.setup, name)
		local startTime = os.time()
		local ok, result = xpcall(testFn, errHandler)
		local elapsed = os.time() - startTime
		callHook(suite.teardown, name, elapsed)
		if ok then
			result = { type = "pass" }
		end
		result.elapsed = elapsed
		results[result.type][suite.name .. "." .. name] = result
		callHook(hooks.postTest, name, result)
	end
	callHook(suite.steardown)
	callHook(hooks.endSuite, results)
end

local function fail(msg)
	error({ type = "fail", msg = msg })
end

local function skip(msg)
	error({ type = "skip", msg = msg })
end

local function testAssertion(got, msg, ...)
	assertions = assertions + 1
	if not got then
		error({ type = "fail", msg = string.format(msg, ...) })
	end
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
	local results = { pass = {}, fail = {}, skip = {}, error = {}, startTime = os.time() }
	callHook(hooks.begin, suites)
	for _, suite in ipairs(suites) do
		runSuite(hooks, results, suite)
	end
	callHook(hooks.done, results)
	if count(results.error) + count(results.fail) > 0 then
		os.exit(1)
	end
end

return {
	run = runTests,
	suite = addSuite,
	describe = addSuite,
	assert = testAssertion,
	skip = skip,
	fail = fail,
}
