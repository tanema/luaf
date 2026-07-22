local t = require("internal.runtime.lib.test")
local tmpl = require("internal.runtime.lib.tmpl")
local tmplTests = {}

function tmplTests.testSimpleTemplate()
	local render = tmpl.parse("Hello <%= name %>")
	local actual = render({ name = "tim" })
	local expected = "Hello tim"
	t.assertEq(expected, actual)
end

function tmplTests.testRender()
	local actual = tmpl.render("Hello <%= name %>", { name = "tim" })
	local expected = "Hello tim"
	t.assertEq(expected, actual)
end

function tmplTests.testLogicInTmpl()
	local render = tmpl.parse("Hello <% if showName then %><%= name %><% else %><%= anonName %><% end %>")
	local actual = render({
		showName = false,
		name = "tim",
		anonName = "buddy",
	})
	local expected = "Hello buddy"
	t.assertEq(expected, actual)
end

return tmplTests
