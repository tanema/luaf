return {
	fail = function(msg)
		error({ type = "fail", msg = msg })
	end,
	skip = function(msg)
		error({ type = "skip", msg = msg or "" })
	end,
	tableCount = function(tbl)
		local count = 0
		for _ in pairs(tbl) do
			count = count + 1
		end
		return count
	end
}
