local a = setmetatable({}, {
	__index = function(table, key)
		print("indexed")
		return print
	end
})

function a.what()
	print("what")
end

a.what()
a.fun("hi")
