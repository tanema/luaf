local t = require("internal.runtime.lib.test")
local coroutineTests = {}

function coroutineTests.testMain()
  t.skip()
  local main, ismain = coroutine.running()
  t.assert.Eq(type(main), "thread")
  t.assert.True(ismain)
  t.assert.False(coroutine.resume(main))
  t.assert.False(coroutine.isyieldable(main))
  t.assert.False(coroutine.isyieldable())
  t.assert.False(pcall(coroutine.yield))
end

function coroutineTests.testTrivialErrors()
  assert(not pcall(coroutine.resume, 0))
  assert(not pcall(coroutine.status, 0))
end

function coroutineTests.testMultipleYeildArguments()
  t.skip()
  _G.x = nil
  local f
  local function foo(a, ...)
    local x, y = coroutine.running()
    t.assert.Eq(x, f)
    t.assert.Eq(y, false)
    -- next call should not corrupt coroutine (but must fail, as it attempts to resume the running coroutine)
    t.assert.False(coroutine.resume(f))
    t.assert.Eq(coroutine.status(f), "running")
    local arg = { ... }
    t.assert.True(coroutine.isyieldable(x))
    for i = 1, #arg do
      _G.x = { coroutine.yield(table.unpack(arg[i])) }
    end
    return table.unpack(a)
  end

  f = coroutine.create(foo)
  t.assert.True(coroutine.isyieldable(f))
  t.assert.Eq(type(f), "thread")
  t.assert.Eq(coroutine.status(f), "suspended")
  t.assert.True(string.find(tostring(f), "thread"))

  local s, a, b, c, d
  s, a, b, c, d = coroutine.resume(f, { 1, 2, 3 }, {}, { 1 }, { "a", "b", "c" })
  t.assert.True(coroutine.isyieldable(f))
  t.assert.True(s)
  t.assert.Nil(a)
  t.assert.Eq(coroutine.status(f) == "suspended")

  s, a, b, c, d = coroutine.resume(f)
  t.assert.Eq({}, _G.x)
  t.assert.True(s)
  t.assert.Eq(1, a)
  t.assert.Nil(b)
  t.assert.True(coroutine.isyieldable(f))

  s, a, b, c, d = coroutine.resume(f, 1, 2, 3)
  t.assert.Eq({ 1, 2, 3 }, _G.x)
  t.assert.True(s)
  t.assert.Eq("a", a)
  t.assert.Eq("b", b)
  t.assert.Eq("c", c)
  t.assert.Nil(d)

  s, a, b, c, d = coroutine.resume(f, "xuxu")
  t.assert.Eq({ "xuxu" }, _G.x)
  t.assert.True(s)
  t.assert.Eq(1, a)
  t.assert.Eq(2, b)
  t.assert.Eq(3, c)
  t.assert.Nil(d)
  t.assert.Eq(coroutine.status(f), "dead")

  s, a = coroutine.resume(f, "xuxu")
  t.assert.False(s)
  t.assert.True(string.find(a, "dead"))
  t.assert.Eq(coroutine.status(f) == "dead")
end

function coroutineTests.testYieldsInTailCalls()
  t.skip()
  local function foo(i) return coroutine.yield(i) end
  local f = coroutine.wrap(function()
    for i = 1, 10 do
      t.assert.Eq(_G.x, foo(i))
    end
    return "a"
  end)
  for i = 1, 10 do
    _G.x = i
    t.assert.Eq(i, f(i))
  end
  _G.x = "xuxu"
  t.assert.Eq(f("xuxu"), "a")
  _G.x = nil
end

function coroutineTests.testRecursive()
  t.skip()
  local function pf(n, i)
    coroutine.yield(n)
    pf(n * i, i + 1)
  end

  local f, s = coroutine.wrap(pf), 1
  for i = 1, 10 do
    t.assert.Eq(s, f(1, 1))
    s = s * i
  end
end

return coroutineTests
