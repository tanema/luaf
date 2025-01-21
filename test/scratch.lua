local function fnone(x, ...)
	print(...)
	return 42+x
end

local function fntwo(...)
	return fnone(...)
end

print(fntwo(22, "hello", "world"))
