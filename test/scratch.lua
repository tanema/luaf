local tmpl = require("tmpl")
print(tmpl.parse("put name here:<%= name %>")({ name = "tim" }))
