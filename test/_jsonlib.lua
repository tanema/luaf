local t = require("test")
local json = require("encoding.json")
local jsonTests = {}

function jsonTests.testMarshal()
	local data = { name = { first = "bobby", last = "tables" }, age = 22 }
	local expected = '{"name":{"first":"bobby","last":"tables"},"age":22}'
	t.assert.Eq(expected, json.marshal(data))

	data.trick = data
	t.assert.NoError(function()
		json.marshal(data)
	end)

	data = { [data] = "test" }
	t.assert.Error(function()
		json.marshal(data)
	end, "cannot marshal a key value that is a table")

	data = { 1, 2, 3, 4, age = 22 }
	t.assert.Error(function()
		json.marshal(data)
	end, "mixed keyed and indexed table")

	data = { 1, 2, 3, 4 }
	expected = "[1,2,3,4]"
	t.assert.Eq(expected, json.marshal(data))

	t.assert.Eq("null", json.marshal(nil))
	t.assert.Eq("22", json.marshal(22))
	t.assert.Eq('"test"', json.marshal("test"))
	t.assert.Eq("true", json.marshal(true))
	t.assert.Eq("false", json.marshal(false))
end

function jsonTests.testUnmarshal()
	local data = '{"name":{"first":"bobby","last":"tables"},"age":22}'
	local expected = { name = { first = "bobby", last = "tables" }, age = 22 }
	t.assert.Eq(expected, json.unmarshal(data))

	data = "[1, 2, 3, 4]"
	expected = { 1, 2, 3, 4 }
	t.assert.Eq(expected, json.unmarshal(data))

	t.assert.Nil(json.unmarshal("null"))
	t.assert.Eq(22, json.unmarshal("22"))
	t.assert.Eq("test", json.unmarshal('"test"'))
	t.assert.True(json.unmarshal("true"))
	t.assert.False(json.unmarshal("false"))
end

return jsonTests
