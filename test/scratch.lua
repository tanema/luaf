local boo = {
	result = {},
}

function table.count(tbl)
	local count = 0
	for _ in pairs(tbl) do
		count = count + 1
	end
	return count
end

function table.print(tbl)
	local str = ""
	for key, val in pairs(tbl) do
		str = str .. ", " .. tostring(key) .. ":" .. tostring(val)
	end
	print(str)
end

local function add(data, name)
	data["result"]["done" .. name] = 1
end

local function callAdd(name)
	add(boo, name)
end

callAdd("a")
callAdd("b")
callAdd("c")

print(table.count(boo.result))
table.print(boo.result)
