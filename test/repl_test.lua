local t = require("internal.runtime.lib.test")
local repl = __LUAF__BuiltinSupport__.repl
local replTests = {}

local function silenced(fn)
  local stderr, print = io.stderr, print
  io.stderr = { write = function() end }
  _G.print = function() end
  local ok, err = pcall(fn)
  io.stderr, _G.print = stderr, print
  if not ok then error(err, 0) end
end

function replTests.testIncompleteInputEndsWithEOF()
  local cases = { "if true then", "x = ", "return 1 +", "foo(", "local s = [[a", "function f()" }
  for _, src in ipairs(cases) do
    local f, err = load(src, "=REPL", "t")
    t.assert.Nil(f, src)
    t.assert.Eq("<eof>", err:sub(-5), src)
  end
end

function replTests.testCompleteStatementRuns()
  repl._buffer = ""
  repl:handleline("__replTestValue = 42")
  t.assert.Eq(42, __replTestValue)
  t.assert.Eq("", repl._buffer)
  __replTestValue = nil
end

function replTests.testMultilineStatementBuffers()
  repl._buffer = ""
  repl:handleline("if true then")
  t.assert.Eq("if true then\n", repl._buffer)
  repl:handleline("__replTestValue = 'multi'")
  t.assert.Eq("if true then\n__replTestValue = 'multi'\n", repl._buffer)
  repl:handleline("end")
  t.assert.Eq("multi", __replTestValue)
  t.assert.Eq("", repl._buffer)
  __replTestValue = nil
end

function replTests.testSyntaxErrorClearsBuffer()
  repl._buffer = "if true then\n"
  silenced(function() repl:handleline("= bad") end)
  t.assert.Eq("", repl._buffer)
end

function replTests.testRuntimeErrorDoesNotRaise()
  repl._buffer = ""
  t.assert.NoError(function()
    silenced(function() repl:handleline("error('boom')") end)
  end)
  t.assert.Eq("", repl._buffer)
end

return replTests
