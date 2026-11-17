// Package ldoc generates documentation based off of annotations in the code comments.
package ldoc

import (
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"
	"unicode"

	_ "embed"

	htmlTmpl "html/template"
	txtTmpl "text/template"
)

type (
	// Config allows for changing scope of what is shown in the documentation.
	Config struct {
		ShowPrivate bool
		ShowPackage bool
		ShowAnchors bool
	}
	// Documentation is the root struct that manages all the modules and buffering
	// incoming tags before it is known what they are attached to. Then it is the
	// main document that is returned to formatters.
	Documentation struct {
		varBuffer     *Object            `json:"-"`
		rootMod       *Object            `json:"-"`
		activeMod     *Object            `json:"-"`
		bodyStarted   bool               `json:"-"`
		lastTextChunk *string            `json:"-"`
		Config        Config             `json:"-"`
		Modules       map[string]*Object `json:"modules"`
		Anchors       []Anchor           `json:"anchors,omitempty"`
	}
	// ObjectType specifies the base type of object.
	ObjectType string
	// ScopeLevel is the documented scope visibility for an element.
	ScopeLevel string
	// Anchor captures items like TODOs so that they can have line numbers to
	// point the developer back to the items they are documenting.
	Anchor struct {
		Label   string `json:"label"`
		Address string `json:"address"`
		Message string `json:"message"`
	}
	// Object captures the documentation attributes for a module, Table, string,
	// function, etc.
	Object struct {
		doc             *Documentation    `json:"-"`
		Require         string            `json:"require,omitempty"`
		Name            string            `json:"name,omitempty"`
		Description     string            `json:"description,omitempty"`
		Version         string            `json:"version,omitempty"`
		ObjectType      ObjectType        `json:"objectType,omitempty"`
		Type            []string          `json:"type,omitempty"`
		Scope           ScopeLevel        `json:"scope,omitempty"`
		Deprecated      bool              `json:"deprecated,omitempty"`
		Author          []string          `json:"author,omitempty"`
		Copyright       string            `json:"copyright,omitempty"`
		License         string            `json:"license,omitempty"`
		Release         string            `json:"release,omitempty"`
		Usage           []string          `json:"usage,omitempty"`
		References      []string          `json:"references,omitempty"`
		Generic         []string          `json:"generic,omitempty"`
		Params          []Object          `json:"params,omitempty"`
		Returns         []Object          `json:"returns,omitempty"`
		Raises          []string          `json:"raises,omitempty"`
		Nodiscard       bool              `json:"nodiscard,omitempty"` // warn to not discard return values
		NodiscardReason string            `json:"nodiscardReason,omitempty"`
		Variables       map[string]Object `json:"variables,omitempty"` // variables in module, fields in table
		Attrs           []string          `json:"attributes,omitempty"`
		Path            string            `json:"path,omitempty"`
		owner           string
	}
)

const (
	// Public usable by everything even outside the module.
	Public ScopeLevel = "public"
	// Package only usable within a module.
	Package ScopeLevel = "package"
	// Private only usable within a class.
	Private ScopeLevel = "private"
	// Protected only usable within a class and it's descendants.
	Protected ScopeLevel = "protected"
	// Module is documentation for a lua module.
	Module ObjectType = "module"
	// Table is documentation for a lua table.
	Table ObjectType = "table"
	// Enum is documentation for a lua table that is used as an enum.
	Enum ObjectType = "enum"
	// Class is documentation for a lua table that can be considered a class.
	Class ObjectType = "class"
	// Func is documentation for a lua function.
	Func ObjectType = "function"
)

var (
	//go:embed templates/doc.txt.tmpl
	textTemplate string
	//go:embed templates/doc.md.tmpl
	markdownTemplate string
	//go:embed templates/doc.html.tmpl
	htmlTemplate string
)

// New creates a new document to generate for a lua module.
func New(cfg Config) *Documentation {
	newDoc := &Documentation{
		Config:  cfg,
		Modules: map[string]*Object{},
	}
	return newDoc
}

func (d *Documentation) ensureVarBuffer() {
	d.bodyStarted = true
	if d.varBuffer != nil {
		return
	}
	d.varBuffer = &Object{doc: d, Variables: map[string]Object{}}
}

func (d *Documentation) target() *Object {
	if d.bodyStarted {
		d.ensureVarBuffer()
		return d.varBuffer
	}
	return d.activeMod
}

func (d *Documentation) resetVarBuffer(name string) {
	obj := *d.varBuffer
	obj.Path = name
	vars := d.activeMod.Variables
	if parentVars, owner, leaf, ok := d.parentVars(name); ok {
		vars = parentVars
		obj.owner = owner
		name = leaf
		if field, exists := vars[leaf]; exists {
			obj = mergeField(obj, field)
		}
	}
	if obj.Name == "" {
		obj.Name = name
	}
	if obj.Scope == "" {
		obj.Scope = Public
	}
	vars[name] = obj
	d.varBuffer = nil
	d.lastTextChunk = nil
}

func (d *Documentation) parentVars(name string) (map[string]Object, string, string, bool) {
	idx := strings.LastIndexAny(name, ".:")
	if idx < 0 {
		return nil, "", "", false
	}
	vars := d.activeMod.Variables
	owner := ""
	for part := range strings.SplitSeq(name[:idx], ".") {
		parent, ok := vars[part]
		if !ok || !slices.Contains([]ObjectType{Table, Class, Enum}, parent.ObjectType) {
			return nil, "", "", false
		}
		vars = parent.Variables
		owner = parent.Name
	}
	return vars, owner, name[idx+1:], true
}

func mergeField(obj, field Object) Object {
	if len(obj.Type) == 0 {
		obj.Type = field.Type
	}
	if obj.Description == "" {
		obj.Description = field.Description
	}
	if obj.Scope == "" {
		obj.Scope = field.Scope
	}
	return obj
}

// AddModule adds a module definition to the documentation and sets it as the active
// module meaning that any variabled, classes, functions, etc are added to this module.
func (d *Documentation) AddModule(filename string) {
	filename = d.filenameToModule(filename)
	d.Modules[filename] = &Object{
		doc:       d,
		Name:      filename,
		Variables: map[string]Object{},
	}
	d.Modules[filename].ObjectType = Module
	d.Modules[filename].Require = filename
	d.rootMod = d.Modules[filename]
	d.activeMod = d.Modules[filename]
	d.bodyStarted = false
}

func (d *Documentation) filenameToModule(filename string) string {
	filename = strings.TrimSuffix(filename, ".lua")
	filename = strings.ReplaceAll(filename, string(os.PathSeparator), ".")
	return strings.TrimLeft(filename, ".")
}

// AddAuthor will add an author to the module documentation.
func (d *Documentation) AddAuthor(author string) {
	d.activeMod.Author = append(d.activeMod.Author, author)
}

// SetCopyright will set the copyright notice of the documentation.
func (d *Documentation) SetCopyright(copyright string) { d.activeMod.Copyright = copyright }

// SetLicense will set the license notice of the documentation.
func (d *Documentation) SetLicense(license string) { d.activeMod.License = license }

// SetRelease will set release version of the module.
func (d *Documentation) SetRelease(release string) { d.activeMod.Release = release }

// SetMeta will declare the module as type definitions only.
func (d *Documentation) SetMeta() { d.activeMod.Attrs = append(d.activeMod.Attrs, "meta") }

// SetVersion will declare the lua version support for the module.
func (d *Documentation) SetVersion(version string) { d.activeMod.Version = version }

// SetName will identify the name of the module, function, or variable.
func (d *Documentation) SetName(name string) { d.target().Name = name }

// SetDeprecated will mark the active documented object as deprecated.
func (d *Documentation) SetDeprecated() { d.target().Deprecated = true }

// AddRef adds a link to source or website to refer to.
func (d *Documentation) AddRef(ref string) {
	obj := d.target()
	obj.References = append(obj.References, ref)
}

// AddDesc adds a descroption to the mod or object.
func (d *Documentation) AddDesc(desc string) {
	obj := d.target()
	obj.Description = desc
	d.lastTextChunk = &obj.Description
}

// AddUsage adds to a list of usage examples to the mod or object.
func (d *Documentation) AddUsage(usage string) {
	obj := d.target()
	obj.Usage = append(obj.Usage, usage)
}

// AddTextChunk appends to a description already declared.
func (d *Documentation) AddTextChunk(chunk string) {
	if d.lastTextChunk == nil {
		d.AddDesc(chunk)
		return
	}

	*d.lastTextChunk += " " + chunk
}

// SetScope sets the access scope on an object, private, protected, package, public.
func (d *Documentation) SetScope(scope ScopeLevel) {
	d.ensureVarBuffer()
	d.varBuffer.Scope = scope
}

// AddType will add a type description to an or list.
func (d *Documentation) AddType(typeDesc string) {
	d.ensureVarBuffer()
	d.varBuffer.Type = append(d.varBuffer.Type, typeDesc)
}

// DeclareKind specifies if the object is class or table or enum.
func (d *Documentation) DeclareKind(kind ObjectType, extra string) {
	d.ensureVarBuffer()
	d.varBuffer.ObjectType = kind
	d.varBuffer.Description += extra
}

// SetNoDiscard declares that return values from a function should not be discarded.
func (d *Documentation) SetNoDiscard(extra string) {
	d.ensureVarBuffer()
	d.DeclareKind(Func, "")
	d.varBuffer.Nodiscard = true
	d.varBuffer.NodiscardReason = extra
}

// SetAsync indicates that a function is asynchronous.
func (d *Documentation) SetAsync() {
	d.ensureVarBuffer()
	d.DeclareKind(Func, "")
	d.varBuffer.Attrs = append(d.varBuffer.Attrs, "async")
}

func (d *Documentation) ensureVarObjectType(expectedTypes ...ObjectType) {
	defaultType := expectedTypes[0]
	if !slices.Contains(expectedTypes, d.varBuffer.ObjectType) {
		d.DeclareKind(defaultType, "")
	}
}

// AddField adds a field description to a class, enum or table.
func (d *Documentation) AddField(scope ScopeLevel, name string, typeDesc []string, desc string) {
	d.ensureVarBuffer()
	d.ensureVarObjectType(Table, Enum, Class)
	d.varBuffer.Variables[name] = Object{doc: d, Name: name, Description: desc, Type: typeDesc, Scope: scope}
}

// AddParam adds a parameter description to a function.
func (d *Documentation) AddParam(name string, typeDesc []string, desc string) {
	d.ensureVarBuffer()
	d.ensureVarObjectType(Func)
	d.varBuffer.Params = append(d.varBuffer.Params, Object{doc: d, Name: name, Description: desc, Type: typeDesc})
}

// AddReturn adds a return type description for a function.
func (d *Documentation) AddReturn(name string, typeDesc []string, desc string) {
	ret := Object{doc: d, Name: name, Description: desc, Type: typeDesc}
	if d.bodyStarted {
		d.ensureVarBuffer()
		d.ensureVarObjectType(Func)
		d.varBuffer.Returns = append(d.varBuffer.Returns, ret)
	} else {
		d.activeMod.Returns = append(d.activeMod.Returns, ret)
	}
}

// AddRaise adds an optional error that is raised during runtime.
func (d *Documentation) AddRaise(desc string) {
	d.ensureVarBuffer()
	d.ensureVarObjectType(Func)
	d.varBuffer.Raises = append(d.varBuffer.Raises, desc)
}

// SetModule sets the name of the current module.
func (d *Documentation) SetModule(modname string) {
	d.activeMod.Name = modname
}

// AddAnchor adds a new anchor (TODO, WARNING, FIXME) to the doc.
func (d *Documentation) AddAnchor(label, msg string, line, column int64) {
	anchor := Anchor{
		Label:   label,
		Address: fmt.Sprintf("%s:%v:%v", d.activeMod.Require, line, column),
		Message: msg,
	}
	d.Anchors = append(d.Anchors, anchor)
}

// AddLocalVar declares that the current var buffer is a local variable and resets the buffer.
func (d *Documentation) AddLocalVar(name string, konst, klose bool) {
	d.bodyStarted = true
	if d.varBuffer == nil {
		return
	}
	d.varBuffer.Scope = Private
	d.varBuffer.Attrs = d.generateAttrs(true, konst, klose)
	d.resetVarBuffer(name)
}

// AddLocalFunc declares that the current var buffer is a local function and resets the buffer.
func (d *Documentation) AddLocalFunc(name string) {
	d.bodyStarted = true
	if d.varBuffer == nil {
		return
	}
	d.varBuffer.Scope = Private
	d.varBuffer.ObjectType = Func
	d.varBuffer.Attrs = d.generateAttrs(true, false, false)
	d.resetVarBuffer(name)
}

// AddFunc declares that the current var buffer is a non-local function and resets the buffer.
func (d *Documentation) AddFunc(name string, self bool) {
	d.bodyStarted = true
	if d.varBuffer == nil {
		return
	}
	d.varBuffer.ObjectType = Func
	if self {
		d.varBuffer.Attrs = append(d.varBuffer.Attrs, "method")
	}
	d.resetVarBuffer(name)
}

// AddVar declares that the current var buffer is a non-local variable and resets the buffer.
func (d *Documentation) AddVar(name string) {
	d.bodyStarted = true
	if d.varBuffer == nil {
		return
	}
	if name == "" {
		d.Discard()
		return
	}
	d.resetVarBuffer(name)
}

// Discard drops pending documentation tags that are not followed by a declaration
// so that they do not attach to a later declaration.
func (d *Documentation) Discard() {
	d.bodyStarted = true
	d.varBuffer = nil
	d.lastTextChunk = nil
}

func (d *Documentation) generateAttrs(local, konst, klose bool) []string {
	attrs := []string{}
	if local {
		attrs = append(attrs, "local")
	}
	if konst {
		attrs = append(attrs, "const")
	}
	if klose {
		attrs = append(attrs, "close")
	}
	return attrs
}

// String generates a string format of the documentation.
func (d *Documentation) String() string { return dotxtTmpl(textTemplate, d) }

// GenerateHTML generates the documentation in HTML format.
func (d *Documentation) GenerateHTML() string { return dohtmlTmpl(htmlTemplate, d) }

// GenerateMarkdown generates the documentation in markdown format.
func (d *Documentation) GenerateMarkdown() string { return dotxtTmpl(markdownTemplate, d) }

// Title formats a title for a documentation object so that it captures all the
// pertinent information in a succinct way.
func (obj Object) Title() string {
	switch obj.ObjectType {
	case Module:
		return dotxtTmpl(`{{if .Deprecated}}[DEPRECATED] {{end}}{{.Name}}{{if ne .Name .Require}} ({{.Require}}){{end}}`, obj)
	default:
		return fmt.Sprintf("%s %s", obj.Scope, obj.Signature())
	}
}

// Signature returns how the object is referenced in code, including parameters
// and return types for functions and the type for variables.
func (obj Object) Signature() string {
	switch obj.ObjectType {
	case Func:
		name := obj.Name
		if obj.IsMethod() && obj.owner != "" {
			name = obj.owner + ":" + obj.Name
		} else if obj.owner != "" {
			name = obj.owner + "." + obj.Name
		}
		return name + dotxtTmpl(`({{fmtParams .Params}}){{with .Returns}}: {{fmtReturns .}}{{end}}`, obj)
	case Class:
		return obj.Name + "()"
	case Module, Enum, Table:
		return obj.Name
	default:
		return obj.Name + " " + fmtType(obj.Type)
	}
}

// IsMethod returns true if the Object is a function with a self reference.
func (obj Object) IsMethod() bool { return slices.Contains(obj.Attrs, "method") }

// Functions returns all the child objects that are functions.
func (obj Object) Functions() []Object { return obj.varOfKind(Func) }

// Classes returns all of the child objects that are declared classes.
func (obj Object) Classes() []Object { return obj.varOfKind(Class) }

// Enums returns all of the child objects that are declared enum.
func (obj Object) Enums() []Object { return obj.varOfKind(Enum) }

func (obj Object) varOfKind(kind ObjectType) []Object {
	return filter(obj.Variables, func(o Object) bool {
		return o.ObjectType == kind && o.Visible()
	})
}

// Vars returns all of the child objects that are not functions, classes, or enums.
func (obj Object) Vars() []Object {
	return filter(obj.Variables, func(o Object) bool {
		correctType := o.ObjectType != Func && o.ObjectType != Class && o.ObjectType != Enum
		return correctType && o.Visible()
	})
}

// Members returns the visible child objects that are documented as their own
// items: functions, classes, enums, and tables that have visible children.
func (obj Object) Members() []Object {
	return filter(obj.Variables, func(o Object) bool { return o.Visible() && o.isMember() })
}

// Fields returns the visible child objects that are plain values and not Members.
func (obj Object) Fields() []Object {
	return filter(obj.Variables, func(o Object) bool { return o.Visible() && !o.isMember() })
}

func (obj Object) isMember() bool {
	switch obj.ObjectType {
	case Func, Class, Enum:
		return true
	default:
		return len(filter(obj.Variables, Object.Visible)) > 0
	}
}

// Visible will return true if the object is visibile in the documentation based
// on the config.
func (obj Object) Visible() bool {
	return obj.Scope == Public || obj.Scope == Protected ||
		(obj.Scope == Private && obj.doc.Config.ShowPrivate) ||
		(obj.Scope == Package && obj.doc.Config.ShowPackage)
}

func filter(vars map[string]Object, fn func(Object) bool) []Object {
	result := []Object{}
	for _, name := range slices.Sorted(maps.Keys(vars)) {
		if fn(vars[name]) {
			result = append(result, vars[name])
		}
	}
	return result
}

var fmtType = func(t []string) string {
	if len(t) > 0 {
		return strings.Join(t, "|")
	}
	return "any"
}

var funcMap = txtTmpl.FuncMap{
	"join": strings.Join,
	"kv": func(k string, v any) string {
		if yes, _ := txtTmpl.IsTrue(v); !yes {
			return ""
		}
		return fmt.Sprintf("\n%s: %v", k, v)
	},
	"fmtParams": func(params []Object) string {
		parts := make([]string, len(params))
		for i, p := range params {
			if len(p.Type) > 0 {
				parts[i] = fmt.Sprintf("%s: %s", p.Name, strings.Join(p.Type, "|"))
			} else {
				parts[i] = p.Name
			}
		}
		return strings.Join(parts, ", ")
	},
	"fmtType": fmtType,
	"fmtField": func(o Object) string {
		parts := []string{}
		if o.Name != "" && len(o.Type) > 0 {
			parts = append(parts, o.Name+": "+fmtType(o.Type))
		} else if o.Name != "" {
			parts = append(parts, o.Name)
		} else if len(o.Type) > 0 {
			parts = append(parts, fmtType(o.Type))
		}
		if o.Description != "" {
			parts = append(parts, o.Description)
		}
		return strings.Join(parts, " -- ")
	},
	"mdcell":  strings.NewReplacer("|", "\\|", "\n", " ").Replace,
	"add":     func(a, b int) int { return a + b },
	"heading": func(level int) string { return strings.Repeat("#", min(level, 6)) },
	"level": func(level int, obj Object) map[string]any {
		return map[string]any{"Level": level, "Obj": obj}
	},
	"slug": func(s string) string {
		return strings.Map(func(r rune) rune {
			if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '.' || r == '_' || r == '-' {
				return r
			}
			return '-'
		}, s)
	},
	"section": func(title string, items []Object) map[string]any {
		return map[string]any{"Title": title, "Items": items}
	},
	"indent": func(n int, s string) string {
		pad := strings.Repeat(" ", n)
		lines := strings.Split(s, "\n")
		for i, line := range lines {
			if line != "" {
				lines[i] = pad + line
			}
		}
		return strings.Join(lines, "\n")
	},
	"fmtReturns": func(returns []Object) string {
		if len(returns) == 0 {
			return ""
		} else if len(returns) == 1 {
			return fmtType(returns[0].Type)
		}
		parts := make([]string, len(returns))
		for i, r := range returns {
			parts[i] = fmtType(r.Type)
		}
		return "(" + strings.Join(parts, ", ") + ")"
	},
}

func dotxtTmpl(tmpl string, data any) string {
	textTmpl := txtTmpl.New("quickTmpl")
	textTmpl.Funcs(funcMap).Funcs(txtTmpl.FuncMap{
		"include": func(name string, data any) (string, error) {
			var sb strings.Builder
			err := textTmpl.ExecuteTemplate(&sb, name, data)
			return sb.String(), err
		},
	})
	textTmpl, err := textTmpl.Parse(tmpl)
	if err != nil {
		return fmt.Sprintf("template error: %s", err)
	}
	var sb strings.Builder
	if err = textTmpl.Execute(&sb, data); err != nil {
		return fmt.Sprintf("template error: %s", err)
	}
	return sb.String()
}

func dohtmlTmpl(tmpl string, data any) string {
	textTmpl, err := htmlTmpl.New("quickTmpl").Funcs(funcMap).Parse(tmpl)
	if err != nil {
		return fmt.Sprintf("template error: %s", err)
	}
	var sb strings.Builder
	if err = textTmpl.Execute(&sb, data); err != nil {
		return fmt.Sprintf("template error: %s", err)
	}
	return sb.String()
}
