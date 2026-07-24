-- Basic Lua test library
--
-- How to:
-- To setup your tests your should
-- - Declare your suites
-- - Call t.run({}) to run all of the tests.
--
-- Run:
-- There is some config that can be passed in to the `t.run` to allow for changing
-- the tests.
--
-- {
--   verbose = false,
--   hooks = {
--     begin = <function>,
--     done = <function>,
--     beginSuite = <function>,
--     endSuite = <function>,
--     preTest = <function>,
--     postTest = <function>,
--   }
-- }
--
-- Test Suites
-- Suites can be defined with `t.suite` or `t.describe` and are simply a table
-- with methods defined as test.
-- Any method with the prefix name test* will be run as a test. This is done so that
-- other methods can be defined and used as helpers.
-- Also suite hooks can be defined on the table such as `suiteSetup` and `suiteTeardown`
-- Suite hook function names:  setup, teardown, suiteSetup, suiteTeardown
--
-- Test Hooks
-- Hooks are used to wrap functionality around tests or suites. They are defined
-- either on each suite or on the main test hook config. They are executed in the
-- following order.
--
-- Hook Order:
--   hooks.begin
--     hooks.beginSuite
--     suite.suiteSetup
--       hooks.preTest
--       suite.setup
--       suite.teardown
--       hooks.postTest
--     suite.suiteTeardown
--     hooks.endSuite
--   hooks.done

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
local hookNames = { "begin", "done", "beginSuite", "endSuite", "preTest", "postTest" }


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

		if table.count(r.fail) > 0 then
			print("\nFailures: ")
			for test, res in pairs(r.fail) do
				print("-> " .. test)
				print(res.msg)
				print()
			end
		end

		if table.count(r.error) > 0 then
			print("\nErrors: \n")
			for test, res in pairs(r.error) do
				print("-> " .. test)
				print(res.msg)
				print()
			end
		end

		if table.count(r.skip) > 0 then
			print("\nSkipped:")
			for test, result in pairs(r.skip) do
				print("-> " .. test .. ": " .. result.msg)
			end
		end
	end,
}

local verboseHooks = {
	beginSuite = function(suite)
		printf("== Suite: %s", suite.name)
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

	callHook(hooks.beginSuite, suite)
	callHook(suite.ssetup)
	for name, testFn in pairs(suite.tests) do
		callHook(hooks.preTest, name)
		callHook(suite.setup, name)
		local startTime = os.clock()
		local ok, result = pcall(testFn)
		local elapsed = os.clock() - startTime
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

local function fail(msg)
	error({ type = "fail", msg = msg })
end

local function skip(msg)
	error({ type = "skip", msg = msg })
end

-- addSuite will, when given a single string param, load a file at the provided path
-- which returns a table that defines the tests in the suite. If given 2 params of
-- string,table, it will define the suite by the name as the first param and the table
-- defines the suite tests.
-- Any method with the prefix name test* will be run as a test. This is done so that
-- other methods can be defined and used as helpers. Also suite hooks can be defined
-- on the table
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
	local hooks = opts.hooks or {}
	for i, key in pairs(hookNames) do
		if opts[key] then
			hooks[key] = opts[key]
		end
	end

	local systemHooks = defaultHooks
	if opts.verbose then
		systemHooks = setmetatable(verboseHooks, { __index = systemHooks })
	end

	math.randomseed(os.time())
	setmetatable(hooks, { __index = systemHooks })
	callHook(hooks.begin, suites)
	local startTime = os.clock()
	for _, suite in ipairs(suites) do
		runSuite(hooks, suite)
	end
	local elapsed = os.clock() - startTime
	callHook(hooks.done, testResults, elapsed)
	if table.count(testResults.error) + table.count(testResults.fail) > 0 then
		os.exit(1)
	end
end

-- fmtVal renders a value for a failure message. Strings are quoted (via %q) so
-- "5" and 5 are visibly distinct and embedded whitespace/control characters
-- show up, instead of both just printing as 5.
local function fmtVal(v)
	if type(v) == "string" then
		return string.format("%q", v)
	end
	return tostring(v)
end

-- withMsg builds the failure message: a "file:line: " prefix for the test code
-- that called the assertion, the generated description, and the caller's custom
-- msg if any. Level 3: level 1 is withMsg's own frame, level 2 is whichever
-- assertX function called withMsg, level 3 is what called assertX - the actual
-- test code.
local function withMsg(base, msg)
	local info = debug.getinfo(3)
	local location = info and string.format("%s:%d: ", info.short_src, info.currentline) or ""
	if msg == nil or msg == "" then
		return location .. base
	end
	return location .. base .. " (" .. tostring(msg) .. ")"
end

local function customAssert(got, msg)
	assertions = assertions + 1
	if not got then
		fail(msg)
	end
end

local function assertTrue(got, msg)
	customAssert(got, withMsg(string.format("expected a truthy value, got %s", fmtVal(got)), msg))
end

local function assertFalse(got, msg)
	customAssert(not got, withMsg(string.format("expected false, got %s", fmtVal(got)), msg))
end

local function deepEq(expected, actual)
	if expected == actual then
		return true
	elseif type(expected) == "table" and type(actual) == "table" then
		for key1, value1 in pairs(expected) do
			local value2 = actual[key1]
			if value2 == nil then
				return false
			elseif value1 ~= value2 then
				if type(value1) == "table" and type(value2) == "table" then
					if not deepEq(value1, value2) then
						return false
					end
				else
					return false
				end
			end
		end
		for key2, _ in pairs(actual) do
			if expected[key2] == nil then
				return false
			end
		end
		return true
	end
	return false
end

local function diffTables(expected, actual, path)
	path = path or ""
	local diffs = {}
	for k, v in pairs(expected) do
		local key = string.format("%s[%s]", path, fmtVal(k))
		local av = actual[k]
		if av == nil then
			table.insert(diffs, string.format("%s: missing (expected %s)", key, fmtVal(v)))
		elseif not deepEq(v, av) then
			if type(v) == "table" and type(av) == "table" then
				for _, d in ipairs(diffTables(v, av, key)) do
					table.insert(diffs, d)
				end
			else
				table.insert(diffs, string.format("%s: expected %s, got %s", key, fmtVal(v), fmtVal(av)))
			end
		end
	end
	for k, v in pairs(actual) do
		if expected[k] == nil then
			table.insert(diffs, string.format("%s[%s]: unexpected (got %s)", path, fmtVal(k), fmtVal(v)))
		end
	end
	return diffs
end

local function assertEq(expected, actual, msg)
	assertions = assertions + 1
	if deepEq(expected, actual) then
		return
	end
	local detail
	if type(expected) == "table" and type(actual) == "table" then
		detail = "expected table to equal, but found differences:\n    "
				.. table.concat(diffTables(expected, actual), "\n    ")
	else
		detail = string.format("expected %s, got %s", fmtVal(expected), fmtVal(actual))
	end
	fail(withMsg(detail, msg))
end

local function assertNotEq(expected, actual, msg)
	customAssert(
		not deepEq(expected, actual),
		withMsg(string.format("expected %s to not equal %s", fmtVal(expected), fmtVal(actual)), msg)
	)
end

local function assertNil(actual, msg)
	customAssert(actual == nil, withMsg(string.format("expected nil, got %s", fmtVal(actual)), msg))
end

local function assertNotNil(actual, msg)
	customAssert(actual ~= nil, withMsg("expected a non-nil value, got nil", msg))
end

local function assertLen(actual, expectedLen, msg)
	customAssert(
		type(actual) == "string" or type(actual) == "table",
		withMsg(string.format("assertLen: assertion failed! value is %s", type(actual)), msg)
	)

	customAssert(
		#actual == expectedLen,
		withMsg(string.format("expected length %d, got %d", expectedLen, #actual), msg)
	)
end

local function assertEmpty(actual, msg)
	customAssert(
		type(actual) == "string" or type(actual) == "table",
		withMsg(string.format("assertLen: assertion failed! value is %s", type(actual)), msg)
	)

	customAssert(#actual == 0, withMsg(string.format("expected empty got %d", #actual), msg))
end

local function assertError(fn, msg)
	customAssert(
		type(fn) == "function",
		withMsg(string.format("bad argument #1 to assertError, should be function but received %s", type(fn)), msg)
	)
	local ok, result = pcall(fn)
	if ok then
		fail(withMsg(string.format("expected function to raise an error, got %s", fmtVal(result)), msg))
	end
end

return {
	run = runTests,
	suite = addSuite,
	describe = addSuite,
	skip = skip,
	fail = fail,
	assert = {
		True = assertTrue,
		False = assertFalse,
		Eq = assertEq,
		NotEq = assertNotEq,
		Nil = assertNil,
		NotNil = assertNotNil,
		Len = assertLen,
		Empty = assertEmpty,
		Error = assertError,
	},
}
