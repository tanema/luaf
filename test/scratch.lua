local function test1()
	return 42
end

local function test2()
	return test1()
end

print(test2())
