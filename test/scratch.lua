local tbl2 = { a = 12, b = 54, c = 99 }
local allKeys = ""
local valSums = 0
for key, val in pairs(tbl2) do
	allKeys = allKeys .. key
	valSums = valSums + val
end
assert(allKeys == "abc", "forlist keys")
assert(valSums == 165, "forlist val" .. valSums)
