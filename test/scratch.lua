local a = {a = 1, b = 2, c = 3, d= 4}
a.test = 1
print(a)
for k, v in pairs(a) do
	print(k, v)
end
print("done.")
