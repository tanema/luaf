package ldoc

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestDoc() *Documentation {
	doc := New(Config{})
	doc.AddModule("./lib/test.lua")
	return doc
}

func TestAddModule(t *testing.T) {
	t.Parallel()

	doc := newTestDoc()
	mod, ok := doc.Modules["lib.test"]
	require.True(t, ok)
	assert.Equal(t, "lib.test", mod.Name)
	assert.Equal(t, "lib.test", mod.Require)
	assert.Equal(t, Module, mod.ObjectType)
	assert.Same(t, mod, doc.activeMod)
}

func TestModuleAttributes(t *testing.T) {
	t.Parallel()

	doc := newTestDoc()
	doc.SetModule("mymod")
	doc.AddAuthor("a")
	doc.AddAuthor("b")
	doc.SetCopyright("c")
	doc.SetLicense("MIT")
	doc.SetRelease("1.0")
	doc.SetVersion("2.0")
	doc.SetMeta()
	doc.AddDesc("module desc")
	doc.AddTextChunk("continued")
	doc.AddReturn("", []string{"table"}, "")

	mod := doc.Modules["lib.test"]
	assert.Equal(t, "mymod", mod.Name)
	assert.Equal(t, []string{"a", "b"}, mod.Author)
	assert.Equal(t, "c", mod.Copyright)
	assert.Equal(t, "MIT", mod.License)
	assert.Equal(t, "1.0", mod.Release)
	assert.Equal(t, "2.0", mod.Version)
	assert.Equal(t, []string{"meta"}, mod.Attrs)
	assert.Equal(t, "module desc continued", mod.Description)
	require.Len(t, mod.Returns, 1)
	assert.Empty(t, mod.Variables)
}

func TestAddLocalVar(t *testing.T) {
	t.Parallel()

	t.Run("with docs", func(t *testing.T) {
		t.Parallel()

		doc := newTestDoc()
		doc.AddDesc("ignored module desc")
		doc.AddType("number")
		doc.AddLocalVar("x", true, true)

		v, ok := doc.activeMod.Variables["x"]
		require.True(t, ok)
		assert.Equal(t, Private, v.Scope)
		assert.Equal(t, []string{"local", "const", "close"}, v.Attrs)
		assert.Equal(t, []string{"number"}, v.Type)
		assert.Nil(t, doc.varBuffer)
	})

	t.Run("without docs", func(t *testing.T) {
		t.Parallel()

		doc := newTestDoc()
		doc.AddLocalVar("x", false, false)
		assert.Empty(t, doc.activeMod.Variables)
		assert.True(t, doc.bodyStarted)
	})
}

func TestAddLocalFunc(t *testing.T) {
	t.Parallel()

	doc := newTestDoc()
	doc.AddParam("a", []string{"string"}, "")
	doc.AddLocalFunc("fn")

	v := doc.activeMod.Variables["fn"]
	assert.Equal(t, Func, v.ObjectType)
	assert.Equal(t, Private, v.Scope)
	assert.Equal(t, []string{"local"}, v.Attrs)
	require.Len(t, v.Params, 1)
}

func TestAddFunc(t *testing.T) {
	t.Parallel()

	t.Run("defaults to public", func(t *testing.T) {
		t.Parallel()

		doc := newTestDoc()
		doc.AddRaise("err")
		doc.AddFunc("M.fn", false)

		v := doc.activeMod.Variables["M.fn"]
		assert.Equal(t, Func, v.ObjectType)
		assert.Equal(t, Public, v.Scope)
		assert.Equal(t, []string{"err"}, v.Raises)
	})

	t.Run("keeps explicit scope", func(t *testing.T) {
		t.Parallel()

		doc := newTestDoc()
		doc.SetScope(Protected)
		doc.AddFunc("M:fn", true)
		assert.Equal(t, Protected, doc.activeMod.Variables["M:fn"].Scope)
	})
}

func TestAddVar(t *testing.T) {
	t.Parallel()

	t.Run("documents named var", func(t *testing.T) {
		t.Parallel()

		doc := newTestDoc()
		doc.DeclareKind(Class, "a class")
		doc.AddVar("config")

		v, ok := doc.activeMod.Variables["config"]
		require.True(t, ok)
		assert.Equal(t, "config", v.Name)
		assert.Equal(t, Class, v.ObjectType)
		assert.Equal(t, Public, v.Scope)
		assert.Empty(t, v.Attrs)
		assert.Equal(t, "a class", v.Description)
		assert.Nil(t, doc.varBuffer)
	})

	t.Run("keeps explicit scope", func(t *testing.T) {
		t.Parallel()

		doc := newTestDoc()
		doc.SetScope(Package)
		doc.AddVar("M.x")
		assert.Equal(t, Package, doc.activeMod.Variables["M.x"].Scope)
	})

	t.Run("empty name discards buffer", func(t *testing.T) {
		t.Parallel()

		doc := newTestDoc()
		doc.DeclareKind(Class, "")
		doc.AddDesc("should not leak")
		doc.AddVar("")

		assert.Empty(t, doc.activeMod.Variables)
		assert.Nil(t, doc.varBuffer)
		assert.Nil(t, doc.lastTextChunk)

		doc.AddParam("a", nil, "")
		doc.AddFunc("fn", false)
		v := doc.activeMod.Variables["fn"]
		assert.Empty(t, v.Description)
		assert.Equal(t, Func, v.ObjectType)
	})

	t.Run("without docs", func(t *testing.T) {
		t.Parallel()

		doc := newTestDoc()
		doc.AddVar("x")
		assert.Empty(t, doc.activeMod.Variables)
		assert.True(t, doc.bodyStarted)
	})
}

func TestDescriptionAfterBodyStarted(t *testing.T) {
	t.Parallel()

	doc := newTestDoc()
	doc.AddLocalVar("first", false, false)
	doc.AddDesc("var desc")
	doc.AddTextChunk("more")
	doc.AddVar("x")

	assert.Empty(t, doc.activeMod.Description)
	assert.Equal(t, "var desc more", doc.activeMod.Variables["x"].Description)
}

func TestEnsureVarObjectType(t *testing.T) {
	t.Parallel()

	t.Run("field on class stays class", func(t *testing.T) {
		t.Parallel()

		doc := newTestDoc()
		doc.DeclareKind(Class, "")
		doc.AddField(Public, "name", []string{"string"}, "")
		doc.AddVar("Person")

		v := doc.activeMod.Variables["Person"]
		assert.Equal(t, Class, v.ObjectType)
		assert.Contains(t, v.Variables, "name")
	})

	t.Run("field without kind becomes table", func(t *testing.T) {
		t.Parallel()

		doc := newTestDoc()
		doc.AddField(Public, "name", nil, "")
		doc.AddVar("t")
		assert.Equal(t, Table, doc.activeMod.Variables["t"].ObjectType)
	})

	t.Run("param becomes function", func(t *testing.T) {
		t.Parallel()

		doc := newTestDoc()
		doc.AddParam("a", nil, "")
		doc.AddVar("M.fn")
		assert.Equal(t, Func, doc.activeMod.Variables["M.fn"].ObjectType)
	})
}

func TestFunctionAttributes(t *testing.T) {
	t.Parallel()

	doc := newTestDoc()
	doc.SetNoDiscard("important")
	doc.SetAsync()
	doc.AddFunc("fn", false)

	v := doc.activeMod.Variables["fn"]
	assert.Equal(t, Func, v.ObjectType)
	assert.True(t, v.Nodiscard)
	assert.Equal(t, "important", v.NodiscardReason)
	assert.Equal(t, []string{"async"}, v.Attrs)
}

func TestAddAnchor(t *testing.T) {
	t.Parallel()

	doc := newTestDoc()
	doc.AddAnchor("TODO", "fix this", 3, 7)
	assert.Equal(t, []Anchor{{Label: "TODO", Address: "lib.test:3:7", Message: "fix this"}}, doc.Anchors)
}

func TestVisible(t *testing.T) {
	t.Parallel()

	tests := []struct {
		scope ScopeLevel
		cfg   Config
		want  bool
	}{
		{Public, Config{}, true},
		{Protected, Config{}, true},
		{Private, Config{}, false},
		{Private, Config{ShowPrivate: true}, true},
		{Package, Config{}, false},
		{Package, Config{ShowPackage: true}, true},
		{Package, Config{ShowPrivate: true}, false},
	}
	for _, tt := range tests {
		doc := New(tt.cfg)
		obj := Object{doc: doc, Scope: tt.scope}
		assert.Equal(t, tt.want, obj.Visible(), "scope %s with %+v", tt.scope, tt.cfg)
	}
}

func TestVariableFilters(t *testing.T) {
	t.Parallel()

	doc := newTestDoc()
	doc.DeclareKind(Class, "")
	doc.AddVar("C")
	doc.DeclareKind(Enum, "")
	doc.AddVar("E")
	doc.AddParam("a", nil, "")
	doc.AddFunc("f", false)
	doc.AddType("string")
	doc.AddVar("v")
	doc.AddType("string")
	doc.AddLocalVar("hidden", false, false)

	mod := doc.activeMod
	assert.Len(t, mod.Classes(), 1)
	assert.Len(t, mod.Enums(), 1)
	assert.Len(t, mod.Functions(), 1)
	vars := mod.Vars()
	require.Len(t, vars, 1)
	assert.Equal(t, "v", vars[0].Name)
}

func TestTitle(t *testing.T) {
	t.Parallel()

	doc := New(Config{})
	tests := []struct {
		obj  Object
		want string
	}{
		{Object{doc: doc, ObjectType: Module, Name: "a", Require: "a"}, "a"},
		{Object{doc: doc, ObjectType: Module, Name: "a", Require: "lib.a", Deprecated: true}, "[DEPRECATED] a (lib.a)"},
		{
			Object{
				doc: doc, ObjectType: Func, Scope: Public, Name: "f",
				Params:  []Object{{Name: "a", Type: []string{"string", "nil"}}, {Name: "b"}},
				Returns: []Object{{Type: []string{"number"}}, {}},
			},
			"public f(a: string|nil, b): (number, any)",
		},
		{
			Object{doc: doc, ObjectType: Func, Scope: Public, Name: "m", Attrs: []string{"method"}},
			"public m()",
		},
		{
			Object{doc: doc, ObjectType: Func, Scope: Public, Name: "m", Attrs: []string{"method"}, owner: "C"},
			"public C:m()",
		},
		{Object{doc: doc, ObjectType: Table, Scope: Public, Name: "t"}, "public t"},
		{Object{doc: doc, ObjectType: Class, Scope: Public, Name: "C"}, "public C()"},
		{Object{doc: doc, ObjectType: Enum, Scope: Private, Name: "E"}, "private E"},
		{Object{doc: doc, Scope: Public, Name: "v"}, "public v any"},
		{Object{doc: doc, Scope: Public, Name: "v", Type: []string{"string"}}, "public v string"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, tt.obj.Title())
	}
}

func TestGenerators(t *testing.T) {
	t.Parallel()

	doc := newTestDoc()
	doc.AddDesc("module desc")
	doc.AddParam("a", []string{"string"}, "")
	doc.AddFunc("M.fn", false)
	doc.DeclareKind(Class, "")
	doc.AddField(Public, "visible_field", nil, "")
	doc.AddField(Private, "hidden_field", nil, "")
	doc.AddVar("C")
	doc.AddParam("b", nil, "")
	doc.AddFunc("C:meth", true)
	doc.DeclareKind(Table, "")
	doc.AddVar("T")
	doc.AddType("string")
	doc.AddVar("T.tfield")

	outputs := map[string]string{
		"text":     doc.String(),
		"markdown": doc.GenerateMarkdown(),
		"html":     doc.GenerateHTML(),
	}
	for name, out := range outputs {
		assert.NotContains(t, out, "template error", name)
		assert.Contains(t, out, "M.fn(a: string)", name)
		assert.Contains(t, out, "visible_field", name)
		assert.NotContains(t, out, "hidden_field", name)
		assert.Contains(t, out, "C:meth(b)", name)
		assert.Contains(t, out, "tfield", name)
	}
	assert.Contains(t, outputs["text"], "public C:meth(b)")
	assert.Contains(t, outputs["text"], "\n  public T\n")
	for _, kind := range []string{"METHOD", "FUNCTION", "CLASS", "TABLE", "ENUM"} {
		assert.NotContains(t, outputs["text"], kind)
	}

	t.Run("text nests members and has no trailing whitespace", func(t *testing.T) {
		t.Parallel()

		out := outputs["text"]
		assert.Contains(t, out, "\n  public C()\n")
		assert.Contains(t, out, "\n      public C:meth(b)\n")
		assert.Contains(t, out, "\n      ● tfield: string\n")
		assert.False(t, strings.HasPrefix(out, "\n"))
		assert.True(t, strings.HasSuffix(out, "\n"))
		for i, line := range strings.Split(out, "\n") {
			assert.Equal(t, strings.TrimRight(line, " \t"), line, "trailing whitespace on line %d", i+1)
		}
	})

	t.Run("markdown has valid structure", func(t *testing.T) {
		t.Parallel()

		out := outputs["markdown"]
		assert.Contains(t, out, "\n## Classes\n\n### `C()`\n\n_public_\n")
		assert.Contains(t, out, "\n#### `C:meth(b)`\n\n_public_\n")
		assert.Contains(t, out, "\n### `T`\n\n_public_\n")
		assert.Contains(t, out, "| `tfield` | `string` |  |")
		assert.NotContains(t, out, "●")
		assert.NotContains(t, out, "\n\n\n")
		assert.True(t, strings.HasSuffix(out, "\n"))
		lines := strings.Split(out, "\n")
		for i, line := range lines {
			assert.Equal(t, strings.TrimRight(line, " \t"), line, "trailing whitespace on line %d", i+1)
			if strings.HasPrefix(line, "| ---") {
				require.Positive(t, i)
				assert.True(t, strings.HasPrefix(lines[i-1], "| Name"), "table separator without header on line %d", i+1)
			}
		}
	})

	t.Run("html is standalone with matching anchors", func(t *testing.T) {
		t.Parallel()

		out := outputs["html"]
		assert.Contains(t, out, "<style>")
		assert.NotContains(t, out, "<link")
		assert.NotContains(t, out, "src=")
		assert.Contains(t, out, `<code class="sig">C:meth(b)</code>`)
		assert.Contains(t, out, `<code class="sig">T</code>`)
		assert.NotContains(t, out, "kind-")
		assert.Contains(t, out, `id="C-meth"`)
		assert.Contains(t, out, `href="#C-meth"`)
	})
}

func TestMarkdownHelpers(t *testing.T) {
	t.Parallel()

	mdcell, ok := funcMap["mdcell"].(func(string) string)
	require.True(t, ok)
	assert.Equal(t, `a \| b c`, mdcell("a | b\nc"))

	heading, ok := funcMap["heading"].(func(int) string)
	require.True(t, ok)
	assert.Equal(t, "###", heading(3))
	assert.Equal(t, "######", heading(9))
}

func TestMarkdownUsageAndEscaping(t *testing.T) {
	t.Parallel()

	doc := newTestDoc()
	doc.AddParam("a", []string{"string", "nil"}, "either | or")
	doc.AddUsage("f(1)")
	doc.AddUsage("f(2)")
	doc.AddFunc("f", false)

	out := doc.GenerateMarkdown()
	assert.Contains(t, out, "```lua\nf(1)\nf(2)\n```")
	assert.Contains(t, out, "| `a` | `string \\| nil` | either \\| or |")
}

func TestSlug(t *testing.T) {
	t.Parallel()

	slug, ok := funcMap["slug"].(func(string) string)
	require.True(t, ok)
	assert.Equal(t, "config-read", slug("config:read"))
	assert.Equal(t, "a.b_c-d", slug("a.b_c-d"))
	assert.Equal(t, "a-b", slug("a b"))
}

func TestFmtField(t *testing.T) {
	t.Parallel()

	fmtField, ok := funcMap["fmtField"].(func(Object) string)
	require.True(t, ok)
	tests := []struct {
		obj  Object
		want string
	}{
		{Object{Name: "a", Type: []string{"string", "nil"}, Description: "desc"}, "a: string|nil -- desc"},
		{Object{Name: "a"}, "a"},
		{Object{Type: []string{"number"}}, "number"},
		{Object{Description: "desc"}, "desc"},
		{Object{}, ""},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, fmtField(tt.obj))
	}
}

func TestSignature(t *testing.T) {
	t.Parallel()

	tests := []struct {
		obj Object
		sig string
	}{
		{Object{ObjectType: Module, Name: "m"}, "m"},
		{Object{ObjectType: Func, Name: "f"}, "f()"},
		{Object{ObjectType: Func, Name: "m", Attrs: []string{"method"}}, "m()"},
		{Object{ObjectType: Func, Name: "m", Attrs: []string{"method"}, owner: "C"}, "C:m()"},
		{Object{ObjectType: Func, Name: "f", owner: "T"}, "T.f()"},
		{Object{ObjectType: Class, Name: "C"}, "C()"},
		{Object{ObjectType: Enum, Name: "E"}, "E"},
		{Object{ObjectType: Table, Name: "T"}, "T"},
		{Object{Name: "v", Type: []string{"string"}}, "v string"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.sig, tt.obj.Signature())
	}
}

func TestPath(t *testing.T) {
	t.Parallel()

	doc := newTestDoc()
	doc.DeclareKind(Class, "")
	doc.SetName("Config")
	doc.AddVar("config")
	doc.AddParam("a", nil, "")
	doc.AddFunc("config:read", true)

	c := doc.activeMod.Variables["config"]
	assert.Equal(t, "config", c.Path)
	assert.Equal(t, "config:read", c.Variables["read"].Path)
}

func TestNesting(t *testing.T) {
	t.Parallel()

	t.Run("nests under documented class", func(t *testing.T) {
		t.Parallel()

		doc := newTestDoc()
		doc.DeclareKind(Class, "")
		doc.AddVar("C")
		doc.AddType("number")
		doc.AddVar("C.x")
		doc.AddParam("a", nil, "")
		doc.AddFunc("C:m", true)

		vars := doc.activeMod.Variables
		assert.Len(t, vars, 1)
		c := vars["C"]
		assert.Equal(t, "x", c.Variables["x"].Name)
		assert.Equal(t, Public, c.Variables["x"].Scope)
		m := c.Variables["m"]
		assert.Equal(t, "m", m.Name)
		assert.True(t, m.IsMethod())
	})

	t.Run("nests through multiple levels", func(t *testing.T) {
		t.Parallel()

		doc := newTestDoc()
		doc.DeclareKind(Table, "")
		doc.AddVar("a")
		doc.DeclareKind(Enum, "")
		doc.AddVar("a.b")
		doc.AddType("string")
		doc.AddVar("a.b.c")

		c, ok := doc.activeMod.Variables["a"].Variables["b"].Variables["c"]
		require.True(t, ok)
		assert.Equal(t, []string{"string"}, c.Type)
	})

	t.Run("falls back to flat when parent is not documented", func(t *testing.T) {
		t.Parallel()

		doc := newTestDoc()
		doc.AddType("string")
		doc.AddVar("M.x")
		assert.Contains(t, doc.activeMod.Variables, "M.x")
	})

	t.Run("falls back to flat when parent is not a container", func(t *testing.T) {
		t.Parallel()

		doc := newTestDoc()
		doc.AddParam("a", nil, "")
		doc.AddFunc("f", false)
		doc.AddType("string")
		doc.AddVar("f.x")
		assert.Contains(t, doc.activeMod.Variables, "f.x")
	})

	t.Run("falls back to flat when middle of path is missing", func(t *testing.T) {
		t.Parallel()

		doc := newTestDoc()
		doc.DeclareKind(Table, "")
		doc.AddVar("a")
		doc.AddType("string")
		doc.AddVar("a.missing.c")
		assert.Contains(t, doc.activeMod.Variables, "a.missing.c")
	})

	t.Run("merges with existing field", func(t *testing.T) {
		t.Parallel()

		doc := newTestDoc()
		doc.DeclareKind(Class, "")
		doc.AddField(Private, "x", []string{"number"}, "field desc")
		doc.AddField(Public, "y", []string{"number"}, "y desc")
		doc.AddVar("C")
		doc.AddDesc("new desc")
		doc.AddVar("C.x")
		doc.AddType("string")
		doc.SetScope(Package)
		doc.AddVar("C.y")

		x := doc.activeMod.Variables["C"].Variables["x"]
		assert.Equal(t, []string{"number"}, x.Type)
		assert.Equal(t, "new desc", x.Description)
		assert.Equal(t, Private, x.Scope)

		y := doc.activeMod.Variables["C"].Variables["y"]
		assert.Equal(t, []string{"string"}, y.Type)
		assert.Equal(t, "y desc", y.Description)
		assert.Equal(t, Package, y.Scope)
	})

	t.Run("function without self is not a method", func(t *testing.T) {
		t.Parallel()

		doc := newTestDoc()
		doc.DeclareKind(Table, "")
		doc.AddVar("T")
		doc.AddParam("a", nil, "")
		doc.AddFunc("T.f", false)
		f := doc.activeMod.Variables["T"].Variables["f"]
		assert.False(t, f.IsMethod())
	})
}

func TestDocumentedName(t *testing.T) {
	t.Parallel()

	t.Run("keeps documented name and keys by variable", func(t *testing.T) {
		t.Parallel()

		doc := newTestDoc()
		doc.DeclareKind(Class, "")
		doc.SetName("Config")
		doc.AddVar("config")
		doc.AddParam("a", nil, "")
		doc.AddFunc("config:read", true)

		c, ok := doc.activeMod.Variables["config"]
		require.True(t, ok)
		assert.Equal(t, "Config", c.Name)
		assert.Equal(t, "read", c.Variables["read"].Name)
	})

	t.Run("keeps documented name on local", func(t *testing.T) {
		t.Parallel()

		doc := newTestDoc()
		doc.DeclareKind(Enum, "")
		doc.SetName("States")
		doc.AddLocalVar("states", false, false)
		assert.Equal(t, "States", doc.activeMod.Variables["states"].Name)
	})

	t.Run("keeps documented name on nested member", func(t *testing.T) {
		t.Parallel()

		doc := newTestDoc()
		doc.DeclareKind(Table, "")
		doc.AddVar("T")
		doc.AddParam("a", nil, "")
		doc.SetName("alias")
		doc.AddFunc("T.f", false)
		assert.Equal(t, "alias", doc.activeMod.Variables["T"].Variables["f"].Name)
	})
}

func TestTargetBeforeAndAfterBody(t *testing.T) {
	t.Parallel()

	t.Run("module level before body", func(t *testing.T) {
		t.Parallel()

		doc := newTestDoc()
		doc.SetDeprecated()
		doc.AddRef("ref")
		doc.AddUsage("require('m')")
		doc.SetName("renamed")

		mod := doc.activeMod
		assert.True(t, mod.Deprecated)
		assert.Equal(t, []string{"ref"}, mod.References)
		assert.Equal(t, []string{"require('m')"}, mod.Usage)
		assert.Equal(t, "renamed", mod.Name)
		assert.Nil(t, doc.varBuffer)
		assert.False(t, doc.bodyStarted)
	})

	t.Run("item level after body", func(t *testing.T) {
		t.Parallel()

		doc := newTestDoc()
		doc.AddLocalVar("first", false, false)
		doc.SetDeprecated()
		doc.AddRef("ref")
		doc.AddUsage("f()")
		doc.AddFunc("f", false)

		mod := doc.activeMod
		assert.False(t, mod.Deprecated)
		assert.Empty(t, mod.References)
		assert.Empty(t, mod.Usage)
		f := mod.Variables["f"]
		assert.True(t, f.Deprecated)
		assert.Equal(t, []string{"ref"}, f.References)
		assert.Equal(t, []string{"f()"}, f.Usage)
	})
}

func TestDiscard(t *testing.T) {
	t.Parallel()

	doc := newTestDoc()
	doc.DeclareKind(Class, "")
	doc.SetName("Thing")
	doc.AddDesc("leaked")
	doc.Discard()

	assert.Nil(t, doc.varBuffer)
	assert.Nil(t, doc.lastTextChunk)
	assert.True(t, doc.bodyStarted)

	doc.AddParam("a", nil, "")
	doc.AddFunc("f", false)
	f := doc.activeMod.Variables["f"]
	assert.Equal(t, "f", f.Name)
	assert.Empty(t, f.Description)
	assert.Equal(t, Func, f.ObjectType)
}

func TestModuleUsageRendered(t *testing.T) {
	t.Parallel()

	doc := newTestDoc()
	doc.AddUsage(`require("m")`)
	assert.Contains(t, doc.String(), "  Usage:\n    require(\"m\")")
	assert.Contains(t, doc.GenerateMarkdown(), "**Usage**\n\n```lua\nrequire(\"m\")\n```")
	assert.Contains(t, doc.GenerateHTML(), "<h4>Usage</h4>")
}

func TestMembersAndFields(t *testing.T) {
	t.Parallel()

	doc := newTestDoc()
	doc.DeclareKind(Class, "")
	doc.AddField(Public, "x", []string{"number"}, "")
	doc.AddVar("C")
	doc.AddParam("a", nil, "")
	doc.AddFunc("C:m", true)
	doc.DeclareKind(Table, "")
	doc.AddVar("C.sub")
	doc.AddParam("a", nil, "")
	doc.AddFunc("C.sub.go", false)
	doc.DeclareKind(Table, "")
	doc.AddVar("C.empty")
	doc.DeclareKind(Table, "")
	doc.AddVar("C.hidden")
	doc.SetScope(Private)
	doc.AddType("string")
	doc.AddVar("C.hidden.secret")
	doc.DeclareKind(Enum, "")
	doc.AddVar("C.Kinds")

	names := func(objs []Object) []string {
		out := make([]string, 0, len(objs))
		for _, o := range objs {
			out = append(out, o.Name)
		}
		return out
	}
	c := doc.activeMod.Variables["C"]
	assert.Equal(t, []string{"Kinds", "m", "sub"}, names(c.Members()))
	assert.Equal(t, []string{"empty", "hidden", "x"}, names(c.Fields()))

	doc.Config.ShowPrivate = true
	assert.Equal(t, []string{"Kinds", "hidden", "m", "sub"}, names(c.Members()))
}

func TestNestedMembersRendered(t *testing.T) {
	t.Parallel()

	doc := newTestDoc()
	doc.DeclareKind(Class, "")
	doc.AddVar("P")
	doc.DeclareKind(Table, "")
	doc.AddVar("P.meta")
	doc.AddParam("a", []string{"string"}, "")
	doc.AddFunc("P.meta.go", false)

	text := doc.String()
	assert.Contains(t, text, "\n      public meta\n        Members:\n\n          public meta.go(a: string)\n")
	assert.Contains(t, doc.GenerateMarkdown(), "\n#### `meta`\n\n_public_\n\n##### `meta.go(a: string)`\n")
	html := doc.GenerateHTML()
	assert.Contains(t, html, `id="P.meta.go"`)
	assert.Contains(t, html, `href="#P.meta.go"`)
}

func TestMethodOwner(t *testing.T) {
	t.Parallel()

	t.Run("uses documented class name", func(t *testing.T) {
		t.Parallel()

		doc := newTestDoc()
		doc.DeclareKind(Class, "")
		doc.SetName("Config")
		doc.AddVar("config")
		doc.AddParam("a", []string{"string"}, "")
		doc.AddFunc("config:read", true)

		read := doc.activeMod.Variables["config"].Variables["read"]
		assert.Equal(t, "public Config:read(a: string)", read.Title())
	})

	t.Run("uses innermost owner for nested tables", func(t *testing.T) {
		t.Parallel()

		doc := newTestDoc()
		doc.DeclareKind(Table, "")
		doc.AddVar("a")
		doc.DeclareKind(Class, "")
		doc.SetName("B")
		doc.AddVar("a.b")
		doc.AddParam("x", nil, "")
		doc.AddFunc("a.b:run", true)

		run := doc.activeMod.Variables["a"].Variables["b"].Variables["run"]
		assert.Equal(t, "public B:run(x)", run.Title())
	})

	t.Run("undocumented owner keeps full name", func(t *testing.T) {
		t.Parallel()

		doc := newTestDoc()
		doc.AddParam("x", nil, "")
		doc.AddFunc("M:run", true)
		assert.Equal(t, "public M:run(x)", doc.activeMod.Variables["M:run"].Title())
	})

	t.Run("dot functions use owner name", func(t *testing.T) {
		t.Parallel()

		doc := newTestDoc()
		doc.DeclareKind(Table, "")
		doc.AddVar("T")
		doc.AddParam("x", nil, "")
		doc.AddFunc("T.f", false)
		assert.Equal(t, "public T.f(x)", doc.activeMod.Variables["T"].Variables["f"].Title())
	})

	t.Run("dot function on undocumented owner keeps full name", func(t *testing.T) {
		t.Parallel()

		doc := newTestDoc()
		doc.AddParam("x", nil, "")
		doc.AddFunc("M.fn", false)
		assert.Equal(t, "public M.fn(x)", doc.activeMod.Variables["M.fn"].Title())
	})

	t.Run("top level functions use their name", func(t *testing.T) {
		t.Parallel()

		doc := newTestDoc()
		doc.AddParam("x", nil, "")
		doc.AddFunc("f", false)
		assert.Equal(t, "public f(x)", doc.activeMod.Variables["f"].Title())
	})
}

func TestAddFieldScope(t *testing.T) {
	t.Parallel()

	doc := newTestDoc()
	doc.AddField(Protected, "x", nil, "")
	assert.Equal(t, Protected, doc.varBuffer.Variables["x"].Scope)
}

func TestFiltersAreSorted(t *testing.T) {
	t.Parallel()

	doc := newTestDoc()
	for _, name := range []string{"c", "a", "b", "e", "d"} {
		doc.AddType("string")
		doc.AddVar(name)
		doc.AddParam("p", nil, "")
		doc.AddFunc("f"+name, false)
	}

	names := func(objs []Object) []string {
		out := make([]string, 0, len(objs))
		for _, o := range objs {
			out = append(out, o.Name)
		}
		return out
	}
	assert.Equal(t, []string{"a", "b", "c", "d", "e"}, names(doc.activeMod.Vars()))
	assert.Equal(t, []string{"fa", "fb", "fc", "fd", "fe"}, names(doc.activeMod.Functions()))
}
