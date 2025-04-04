local tmpl = require("tmpl")

local helloTmpl = tmpl.parse("Hello <%= name %>")

print(helloTmpl({ name = "tim" }))
