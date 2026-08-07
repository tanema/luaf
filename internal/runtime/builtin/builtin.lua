-- These are standard library functions that can be expressed in plain lua. This
-- makes more of the stdlib portable as it minimizes the native calls that would
-- need to be handled in the vm.
-- required libs: io, string, coroutine, table

function assert(...)
  if select("#", ...) < 1 then error("bad argument #1 to 'assert' (value expected)", 2) end
  if not (...) then
    if select("#", ...) >= 2 then error((select(2, ...)), 2) end
    error("assertion failed!", 2)
  end
  return ...
end

function loadfile(filename, mode, env)
  local file <close>, err = io.open(filename, "r")
  if not file then return nil, err end
  return load(file:read("a"), "@" .. filename, mode, env)
end

function dofile(filename) return assert(loadfile(filename))() end

function pairs(t)
  local mt = getmetatable(t)
  if mt and mt.__pairs then
    return mt.__pairs(t)
  else
    return next, t, nil
  end
end

local function ipairs_iterator(t, i)
  if i ~= nil then
    i = i + 1
    local v = t[i]
    if v ~= nil then return i, v end
  end
  return nil
end

function ipairs(t) return ipairs_iterator, t, 0 end

function print(...)
  local args = { ... }
  local num_args = select("#", ...)

  for i = 1, num_args do
    io.write(tostring(args[i]))
    if i < num_args then io.write("\t") end
  end
  io.write("\n")
end

function select(index, ...)
  if index == "#" then return table.pack(...).n end

  local n = tonumber(index)
  if not n or n % 1 ~= 0 or n == 0 then
    error("bad argument #1 to 'select' (number expected, got " .. type(index) .. ")", 2)
  end

  local args = table.pack(...)
  if n < 0 then n = args.n + n + 1 end

  if n < 1 or n > args.n + 1 then error("bad argument #1 to 'select' (index out of range)", 2) end

  return table.unpack(args, n, args.n)
end

function pcall(f, ...)
  return xpcall(f, function(err) return err end, ...)
end
