local function fib(n)
	if n < 2 then
		return n
	end
	return fib(n - 2) + fib(n - 1)
end
fib(35)
