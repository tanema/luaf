local util = require("test.util")
local dotCh = {
  pass = ".",
  fail = "F",
  skip = "S",
  error = "E",
}

local function printf(msg, ...) print(string.format(msg, ...)) end

local function fmtDuration(t)
  assert(type(t) == "number", string.format("bad argument #1 to fmtDuration (number expected, got %s)", type(t)))
  local unit = "s"
  if t < 1 then
    unit, t = "ms", t * 1000
  end
  return string.format("%.2f %s", t, unit)
end

return {
  default = {
    postTest = function(_, res) io.write(dotCh[res.type]) end,
    done = function(r, elapsed)
      local ps, fs, ss, es =
        util.tableCount(r.pass), util.tableCount(r.fail), util.tableCount(r.skip), util.tableCount(r.error)
      printf("\nFinished in %s with %d assertions", fmtDuration(elapsed), (_G["__LUA_TEST_ASSERTION_TOTAL"] or 0))
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
  },
  verbose = {
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
  },
}
