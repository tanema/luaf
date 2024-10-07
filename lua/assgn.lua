-- ensure b can use a, and the final value is discarded

local a, b = 22 + 45 / 2 *87, 22 - a, 32
print(a, ", ", b)
