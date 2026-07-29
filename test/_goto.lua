local t = require("internal.runtime.lib.test")
local gotoTests = {}

function gotoTests.testCannotSeeLabelInsideBlock()
	t.assert.SyntaxError([[ goto l1; do ::l1:: end ]], "label 'l1'")
	t.assert.SyntaxError([[ do ::l1:: end goto l1; ]], "label 'l1'")
end

function gotoTests.testRepeatedLabel()
	t.assert.SyntaxError([[ ::l1:: ::l1:: ]], "label 'l1' already defined on line 1")
	t.assert.SyntaxError([[ ::l1:: do ::l1:: end]], "label 'l1' already defined on line 1")
end

function gotoTests.testUndefinedLabel()
	t.skip("not implemented")
	-- TODO jumping scope
	-- err: <goto l1> at line 1 jumps into the scope of local 'aa'
	t.assert.SyntaxError([[ goto l1; local aa ::l1:: ::l2:: print(3) ]], "local 'aa'")
end

function gotoTests.testJumpOverVarDef()
	t.skip("not implemented")
<<<<<<< HEAD
	t.assert.SyntaxError([[
	do local bb, cc; goto l1; end
	local aa
 	::l1:: print(3)]],
		"local 'aa'"
	)
end

function gotoTests.testJumpIntoBlock()
	t.assert.SyntaxError([[ do ::l1:: end goto l1 ]], "label 'l1'")
	t.assert.SyntaxError([[ goto l1 do ::l1:: end ]], "label 'l1'")
end

function gotoTests.testJumpInsideRepeatWithVariables()
	t.skip("not implemented")
	t.assert.SyntaxError(
		[[
   repeat
     if x then goto cont end
     local xuxu = 10
     ::cont::
   until xuxu < x
 ]],
		"local 'xuxu'"
	)
end

function gotoTests.testSimpleGoto()
	local x
	do
		local y = 12
		goto l1
		::l2::
		x = x + 1
		goto l3
		::l1::
		x = y
		goto l2
	end
	::l3::
	::l3_1::
	t.assert.Eq(x, 13)
end

function gotoTests.testLongLabels()
	local prog = [[
    local a = 1
    goto l%sa; a = a + 1
   ::l%sa:: a = a + 10
    goto l%sb; a = a + 2
   ::l%sb:: a = a + 20
    return a
  ]]
	local label = string.rep("0123456789", 40)
	prog = string.format(prog, label, label, label, label)
	t.assert.Eq(load(prog)(), 31)
end

function gotoTests.testJumpOverLclDeclToEnd()
	goto l1
	local a = 23
	local x = a + 1
	::l1::
end

function gotoTests.testOutOfWhile()
	local x = 13
	while true do
		goto l4
		goto l1 -- ok to jump over local dec. to end of block
		goto l1 -- multiple uses of same label
		x = 45
		::l1::
	end
	::l4::
	t.assert.Eq(x, 13)
end

function gotoTests.testJumpInsideIfBlock()
	if true then
		goto l1 -- ok to jump over local dec. to end of block
		error("should not be here")
		goto l2 -- ok to jump over local dec. to end of block
		local x
		::l1::
		::l2::
	else
	end
end

function gotoTests.testJumpInsideFn()
	local function foo()
		local a = {}
		goto l3
		::l1::
		a[#a + 1] = 1
		goto l2
		::l2::
		a[#a + 1] = 2
		goto l5
		::l3::
		::l3a::
		a[#a + 1] = 3
		goto l1
		::l4::
		a[#a + 1] = 4
		goto l6
		::l5::
		a[#a + 1] = 5
		goto l4
		::l6::
		t.assert.Eq(a[1], 3)
		t.assert.Eq(a[2], 1)
		t.assert.Eq(a[3], 2)
		t.assert.Eq(a[4], 5)
		t.assert.Eq(a[5], 4)
		if not a[6] then
			a[6] = true
			goto l3a
		end -- do it twice
	end

	::l6::
	foo()
end

function gotoTests.testJumpSetNil()
	local x
	::L1::
	local y -- cannot join this SETNIL with previous one
	t.assert.Nil(y)
	y = true
	if x == nil then
		x = 1
		goto L1
	else
		x = x + 1
	end
	t.assert.Eq(x, 2)
	t.assert.True(y)
end

function gotoTests.testLuaBug()
	local first = true
	local a = false
	if true then
		goto LBL
		::loop::
		a = true
		::LBL::
		if first then
			first = false
			goto loop
		end
	end
	t.assert.True(a)
end

function gotoTests.testInfiniteLoops()
	do          -- compiling infinite loops
		goto escape -- do not run the infinite loops
		::a::
		goto a
		::b::
		goto c
		::c::
		goto b
	end
	::escape::
end

function gotoTests.testIfBlockOptimization()
	t.skip("not implemented")

	local function testG(a)
		if a == 1 then
			goto l1
			error("should never be here!")
		elseif a == 2 then
			goto l2
		elseif a == 3 then
			goto l3
		elseif a == 4 then
			goto l1 -- go to inside the block
			error("should never be here!")
			::l1::
			a = a + 1 -- must go to 'if' end
		else
			goto l4
			::l4a::
			a = a * 2
			goto l4b
			error("should never be here!")
			::l4::
			goto l4a
			error("should never be here!")
			::l4b::
		end
		do
			return a
		end
		::l2::
		do
			return "2"
		end
		::l3::
		do
			return "3"
		end
		::l1::
		return "1"
	end

	t.assert.Eq(testG(1), "1")
	t.assert.Eq(testG(2), "2")
	t.assert.Eq(testG(3), "3")
	t.assert.Eq(testG(4), 5)
	t.assert.Eq(testG(5), 10)
end

function gotoTests.testGotosWithTBC()
	t.skip("not implemented")
	-- set 'var' and return an object that will reset 'var' when it goes out of scope
	local function newobj(var)
		_ENV[var] = true
		return setmetatable({}, { __close = function() _ENV[var] = nil end })
||||||| parent of 3eaa38e (Working on gotos)
=======
	t.assert.SyntaxError(
		[[
	do local bb, cc; goto l1; end
	local aa
 	::l1:: print(3)]],
		"local 'aa'"
	)
end

function gotoTests.testJumpIntoBlock()
	t.assert.SyntaxError([[ do ::l1:: end goto l1 ]], "label 'l1'")
	t.assert.SyntaxError([[ goto l1 do ::l1:: end ]], "label 'l1'")
end

function gotoTests.testJumpInsideRepeatWithVariables()
	t.skip("not implemented")
	t.assert.SyntaxError(
		[[
   repeat
     if x then goto cont end
     local xuxu = 10
     ::cont::
   until xuxu < x
 ]],
		"local 'xuxu'"
	)
end

function gotoTests.testSimpleGoto()
	local x
	do
		local y = 12
		goto l1
		::l2::
		x = x + 1
		goto l3
		::l1::
		x = y
		goto l2
	end
	::l3::
	::l3_1::
	t.assert.Eq(x, 13)
end

function gotoTests.testLongLabels()
	local prog = [[
    local a = 1
    goto l%sa; a = a + 1
   ::l%sa:: a = a + 10
    goto l%sb; a = a + 2
   ::l%sb:: a = a + 20
    return a
  ]]
	local label = string.rep("0123456789", 40)
	prog = string.format(prog, label, label, label, label)
	t.assert.Eq(load(prog)(), 31)
end

function gotoTests.testJumpOverLclDeclToEnd()
	goto l1
	local a = 23
	local x = a + 1
	::l1::
end

function gotoTests.testOutOfWhile()
	local x = 13
	while true do
		goto l4
		goto l1 -- ok to jump over local dec. to end of block
		goto l1 -- multiple uses of same label
		x = 45
		::l1::
	end
	::l4::
	t.assert.Eq(x, 13)
end

function gotoTests.testJumpInsideIfBlock()
	if true then
		goto l1 -- ok to jump over local dec. to end of block
		error("should not be here")
		goto l2 -- ok to jump over local dec. to end of block
		local x
		::l1::
		::l2::
	else
	end
end

function gotoTests.testJumpInsideFn()
	local function foo()
		local a = {}
		goto l3
		::l1::
		a[#a + 1] = 1
		goto l2
		::l2::
		a[#a + 1] = 2
		goto l5
		::l3::
		::l3a::
		a[#a + 1] = 3
		goto l1
		::l4::
		a[#a + 1] = 4
		goto l6
		::l5::
		a[#a + 1] = 5
		goto l4
		::l6::
		t.assert.Eq(a[1], 3)
		t.assert.Eq(a[2], 1)
		t.assert.Eq(a[3], 2)
		t.assert.Eq(a[4], 5)
		t.assert.Eq(a[5], 4)
		if not a[6] then
			a[6] = true
			goto l3a
		end -- do it twice
	end

	::l6::
	foo()
end

function gotoTests.testJumpSetNil()
	local x
	::L1::
	local y -- cannot join this SETNIL with previous one
	t.assert.Nil(y)
	y = true
	if x == nil then
		x = 1
		goto L1
	else
		x = x + 1
	end
	t.assert.Eq(x, 2)
	t.assert.True(y)
end

function gotoTests.testLuaBug()
	local first = true
	local a = false
	if true then
		goto LBL
		::loop::
		a = true
		::LBL::
		if first then
			first = false
			goto loop
		end
	end
	t.assert.True(a)
end

function gotoTests.testInfiniteLoops()
	do -- compiling infinite loops
		goto escape -- do not run the infinite loops
		::a::
		goto a
		::b::
		goto c
		::c::
		goto b
	end
	::escape::
end

function gotoTests.testIfBlockOptimization()
	t.skip("not implemented")

	local function testG(a)
		if a == 1 then
			goto l1
			error("should never be here!")
		elseif a == 2 then
			goto l2
		elseif a == 3 then
			goto l3
		elseif a == 4 then
			goto l1 -- go to inside the block
			error("should never be here!")
			::l1::
			a = a + 1 -- must go to 'if' end
		else
			goto l4
			::l4a::
			a = a * 2
			goto l4b
			error("should never be here!")
			::l4::
			goto l4a
			error("should never be here!")
			::l4b::
		end
		do
			return a
		end
		::l2::
		do
			return "2"
		end
		::l3::
		do
			return "3"
		end
		::l1::
		return "1"
	end

	t.assert.Eq(testG(1), "1")
	t.assert.Eq(testG(2), "2")
	t.assert.Eq(testG(3), "3")
	t.assert.Eq(testG(4), 5)
	t.assert.Eq(testG(5), 10)
end

function gotoTests.testGotosWithTBC()
	t.skip("not implemented")
	-- set 'var' and return an object that will reset 'var' when it goes out of scope
	local function newobj(var)
		_ENV[var] = true
		return setmetatable({}, {
			__close = function()
				_ENV[var] = nil
			end,
		})
>>>>>>> 3eaa38e (Working on gotos)
	end

	goto L1

	::L4::
	t.assert.False(X)
	goto L5 -- varX dead here

	::L1::
	local varX <close> = newobj("X")
	assert(X)
	goto L2 -- varX alive here

	::L3::
	assert(X)
	goto L4 -- varX alive here

	::L2::
	assert(X)
	goto L3 -- varX alive here

	::L5:: -- return
end

return gotoTests
