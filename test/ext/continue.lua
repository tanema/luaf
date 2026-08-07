local t = require("test")
local continueTests = {}

function continueTests.testContinueInWhileLoop()
  local seen = {}
  local i = 0
  while i < 5 do
    i = i + 1
    if i == 2 then continue end
    seen[#seen + 1] = i
  end
  t.assert.Eq(seen, { 1, 3, 4, 5 })
end

function continueTests.testContinueInForNumLoop()
  local seen = {}
  for i = 1, 6 do
    if i % 2 == 0 then continue end
    seen[#seen + 1] = i
    if i == 5 then break end
  end
  t.assert.Eq(seen, { 1, 3, 5 })
end

function continueTests.testContinueInForInLoop()
  local seen = {}
  for i, v in ipairs({ 10, 20, 30, 40, 50 }) do
    if i % 2 == 0 then continue end
    seen[#seen + 1] = v
  end
  t.assert.Eq(seen, { 10, 30, 50 })
end

function continueTests.testContinueInRepeatLoop()
  -- repeat's until condition is evaluated after the body, so continue must
  -- jump to the condition check, not back to the top of the body.
  local seen = {}
  local i = 0
  repeat
    i = i + 1
    if i == 2 then continue end
    seen[#seen + 1] = i
  until i >= 5
  t.assert.Eq(seen, { 1, 3, 4, 5 })
end

function continueTests.testContinueOnlyAffectsInnermostLoop()
  local seen = {}
  for a = 1, 3 do
    for b = 1, 3 do
      if b == 2 then continue end
      seen[#seen + 1] = a * 10 + b
    end
  end
  t.assert.Eq(seen, { 11, 13, 21, 23, 31, 33 })
end

function continueTests.testContinuePreservesPerIterationClosuresForNum()
  local fns = {}
  for i = 1, 4 do
    local x = i * 10
    fns[i] = function() return x end
    if i % 2 == 0 then continue end
  end
  t.assert.Eq(fns[1](), 10)
  t.assert.Eq(fns[2](), 20)
  t.assert.Eq(fns[3](), 30)
  t.assert.Eq(fns[4](), 40)
end

function continueTests.testContinuePreservesPerIterationClosuresForIn()
  local fns = {}
  for i, v in ipairs({ 10, 20, 30, 40 }) do
    local x = v
    fns[i] = function() return x end
    if i % 2 == 0 then continue end
  end
  t.assert.Eq(fns[1](), 10)
  t.assert.Eq(fns[2](), 20)
  t.assert.Eq(fns[3](), 30)
  t.assert.Eq(fns[4](), 40)
end

return continueTests
