local repeatSum = 0
repeat
	repeatSum = repeatSum + 1
	print(repeatSum)
until repeatSum >= 10
assert(repeatSum == 10, "repeat stat")
