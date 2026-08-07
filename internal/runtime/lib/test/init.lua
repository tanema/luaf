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
local assert = require("test.assert")
local suite = require("test.suite")
local util = require("test.util")

return {
  run = suite.run,
  suite = suite.suite,
  describe = suite.suite,
  skip = util.skip,
  fail = util.fail,
  assert = assert,
}
