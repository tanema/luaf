local forNumSum = 0
for i = 10, 1, -1 do
	print(i, forNumSum)
	forNumSum = forNumSum + i
end
print(forNumSum)
assert(forNumSum == 55, "for num" .. forNumSum)
