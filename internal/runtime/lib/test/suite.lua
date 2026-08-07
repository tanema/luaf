local testHooks = require("test.hooks")
local util = require("test.util")
local suites = {}
local hookNames = { "begin", "done", "beginSuite", "endSuite", "preTest", "postTest" }
local testResults = {
  pass = {},
  fail = {},
  skip = {},
  error = {},
}

local function callHook(fn, ...)
  if fn and type(fn) == "function" then fn(...) end
end

local function runSuite(hooks, suite)
  callHook(hooks.beginSuite, suite)
  callHook(suite.ssetup)
  for name, testFn in pairs(suite.tests) do
    callHook(hooks.preTest, name)
    callHook(suite.setup, name)
    local startTime = os.clock()
    local ok, result = pcall(testFn)
    local elapsed = os.clock() - startTime
    local isTestResult = type(result) == "table" and result.type
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
  for _, key in pairs(hookNames) do
    if opts[key] then hooks[key] = opts[key] end
  end

  local systemHooks = testHooks.default
  if opts.verbose then systemHooks = setmetatable(testHooks.verbose, { __index = systemHooks }) end

  math.randomseed(os.time())
  setmetatable(hooks, { __index = systemHooks })
  callHook(hooks.begin, suites)
  local startTime = os.clock()
  for _, suite in ipairs(suites) do
    if util.tableCount(suite.tests) > 0 then runSuite(hooks, suite) end
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
