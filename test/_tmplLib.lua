local t = require("test")
local tmpl = require("tmpl")
local tmplTests = {}

function tmplTests.testSimpleTemplate()
	local render = tmpl.parse("Hello <%= name %>")
	local actual = render({ name = "tim" })
	local expected = "Hello tim"
	t.assert(actual == expected, "simple template %s does not match %s", actual, expected)
end

function tmplTests.testRender()
	local actual = tmpl.render("Hello <%= name %>", { name = "tim" })
	local expected = "Hello tim"
	t.assert(actual == expected, "simple template %s does not match %s", actual, expected)
end

function tmplTests.testLogicInTmpl()
	t.skip("breaking")
	local render = tmpl.parse("Hello <% if showName then %><%= name %><% else %><%= anonName %><% end %>")
	local actual = render({
		showName = false,
		name = "tim",
		anonName = "buddy",
	})
	local expected = "Hello buddy"
	t.assert(actual == expected, "simple template %s does not match %s", actual, expected)
end

return tmplTests
