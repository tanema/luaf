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

local util = require("test.util")
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

local function printf(msg, ...) print(string.format(msg, ...)) end

local function callHook(fn, ...)
  if fn and type(fn) == "function" then fn(...) end
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
  postTest = function(_, res) io.write(dotCh[res.type]) end,
  done = function(r, elapsed)
    local ps, fs, ss, es =
      util.tableCount(r.pass), util.tableCount(r.fail), util.tableCount(r.skip), util.tableCount(r.error)
    printf("\nFinished in %s with %d assertions", fmtDuration(elapsed), assertions)
    printf("%d passed, %d failed, %d error(s), %d skipped.", ps, fs, es, ss)

    if util.tableCount(r.fail) > 0 then
      print("\nFailures: ")
      for test, res in pairs(r.fail) do
        print("-> " .. test)
        print(res.msg)
        print()
      end
    end

    if util.tableCount(r.error) > 0 then
      print("\nErrors: \n")
      for test, res in pairs(r.error) do
        print("-> " .. test)
        print(res.msg)
        print()
      end
    end

    if util.tableCount(r.skip) > 0 then
      print("\nSkipped:")
      for test, result in pairs(r.skip) do
        print("-> " .. test .. ": " .. result.msg)
      end
    end
  end,
}

local verboseHooks = {
  beginSuite = function(suite) printf("== Suite: %s", suite.name) end,
  preTest = function(name) printf("  RUN\t%s", name) end,
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
  if util.tableCount(suite.tests) == 0 then return end

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

  if not mod then mod = require(modname) end

  assert(type(mod) == "table", string.format("bad argument #2 to testing.suite (table expected, got %s)", type(mod)))

  local tests = {}
  for k, v in pairs(mod) do
    if type(k) == "string" and (k:match("^test.*") or k:match("test$")) and type(v) == "function" then tests[k] = v end
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
    if opts[key] then hooks[key] = opts[key] end
  end

  local systemHooks = defaultHooks
  if opts.verbose then systemHooks = setmetatable(verboseHooks, { __index = systemHooks }) end

  math.randomseed(os.time())
  setmetatable(hooks, { __index = systemHooks })
  callHook(hooks.begin, suites)
  local startTime = os.clock()
  for _, suite in ipairs(suites) do
    runSuite(hooks, suite)
  end
  local elapsed = os.clock() - startTime
  callHook(hooks.done, testResults, elapsed)
  if util.tableCount(testResults.error) + util.tableCount(testResults.fail) > 0 then os.exit(1) end
end

return {
  run = runTests,
  suite = addSuite,
  describe = addSuite,
}
