package parse

import (
	"fmt"
	"os"
	"strings"
)

type (
	// Documentation is the root struct that manages all the modules and buffering
	// incoming tags before it is known what they are attached to. Then it is the
	// main document that is returned to formatters.
	Documentation struct {
		filename            string
		descriptionChunks   []string
		currentActiveModule string
		Modules             map[string]*DocModule
	}
	// DocScopeLevel is the documented scope visibility for an element.
	DocScopeLevel int
	// DocTypeDesc captures common attributes across all documentatable elements.
	DocTypeDesc struct {
		Name        string
		Description string
		Type        []string
		Scope       DocScopeLevel
		Deprecated  bool
	}
	// DocModule captures the documentation attributes for a module.
	DocModule struct {
		Name        string
		Description string
		Type        []string
		Scope       DocScopeLevel
		Deprecated  bool
		Author      []string
		Copyright   string
		License     string
		Release     string
		Generic     []string
		Params      []DocTypeDesc
		Returns     []DocTypeDesc
		Aliases     []DocTypeDesc
		Variables   []DocVariable
		Classes     []DocVariable
		TODOs       []DocAnchor
		FixMes      []DocAnchor
		Warnings    []DocAnchor
		Meta        bool
	}
	// DocVariable captures the documentation for any value with attributes.
	// Table, string, function, etc. If it is a function the Func field is defined
	// with the function attributes. If it is a table the Table doc will be defined.
	DocVariable struct {
		Name        string
		Description string
		Type        []string
		Scope       DocScopeLevel
		Deprecated  bool
		Local       bool // if the variable has been defined as local or global.
		Const       bool // if the variable has been defined as const.
		Language    string
		Func        *DocFunc
		Table       *DocTable
	}
	// DocAnchor captures items like TODOs so that they can have line numbers to
	// point the developer back to the items they are documenting.
	DocAnchor struct {
		Label    string // TODO, FIXME, WARN
		LineInfo LineInfo
		Message  string
	}
	// DocFunc captures the documentation attributes of a function variable.
	DocFunc struct {
		Version   string
		Params    []DocTypeDesc
		Returns   []DocTypeDesc
		Raises    []string
		Nodiscard bool
		Async     bool
	}
	// DocTable captures the documentation attributes of a table variable.
	DocTable struct {
		Enum   bool
		Fields map[string]DocVariable
	}
)

const (
	// AccessPrivate only usable within a class.
	AccessPrivate DocScopeLevel = iota
	// AccessPackage only usable within a module.
	AccessPackage
	// AccessPublic usable by everything even outside the module.
	AccessPublic
)

func newDocumentation(filename string) *Documentation {
	newDoc := &Documentation{
		filename: filename,
		Modules:  map[string]*DocModule{},
	}
	newDoc.addModule(filename, "")
	return newDoc
}

func (d *Documentation) handleTag(tagName, arg string, li LineInfo) {
	switch tagName {
	// Module tags
	case "module":
		d.addModule(d.filename, arg)
	case "author":
		d.addAuthor(arg)
	case "copyright":
		d.Modules[d.currentActiveModule].Copyright = arg
	case "license":
		d.Modules[d.currentActiveModule].License = arg
	case "meta":
		d.Modules[d.currentActiveModule].Meta = true
	case "release":
		d.Modules[d.currentActiveModule].Release = arg
	case "alias":
	case "type":
	case "generic":
	// Variable Tags
	case "class": // declared above a variable but adds to module
	case "enum": // declared above a variable but adds to module
	case "field":
	case "nodiscard":
	case "usage":
	case "operator":
	case "package":
	case "private":
	case "protected":
	case "description":
	case "name":
	case "deprecated":
	case "see":
	case "source":
	case "language":
	// function only tags
	case "overload":
	case "version":
	case "raise":
	case "async":
	case "param":
	case "return":
	// Misc tags
	case "todo":
		d.addTODO(arg, li)
	case "fixme":
		d.addFixMe(arg, li)
	case "warning":
		d.addWarn(arg, li)
	}
}

func (d *Documentation) addDescriptionChunk(chunk string) {
	d.descriptionChunks = append(d.descriptionChunks, chunk)
}

func (d *Documentation) addModule(filename, extra string) {
	filename = strings.TrimSuffix(filename, ".lua")
	filename = strings.ReplaceAll(filename, string(os.PathSeparator), ".")
	filename = strings.TrimLeft(filename, ".")
	if extra != "" {
		filename += "." + extra
	}
	d.currentActiveModule = filename
	d.Modules[d.currentActiveModule] = &DocModule{
		Name: d.currentActiveModule,
	}
}

func (d *Documentation) addAuthor(author string) {
	d.Modules[d.currentActiveModule].Author = append(d.Modules[d.currentActiveModule].Author, author)
}

func (d *Documentation) addTODO(msg string, li LineInfo) {
	d.Modules[d.currentActiveModule].TODOs = append(
		d.Modules[d.currentActiveModule].TODOs,
		DocAnchor{Label: "TODO", LineInfo: li, Message: msg},
	)
}

func (d *Documentation) addFixMe(msg string, li LineInfo) {
	d.Modules[d.currentActiveModule].FixMes = append(
		d.Modules[d.currentActiveModule].FixMes,
		DocAnchor{Label: "FIXME", LineInfo: li, Message: msg},
	)
}

func (d *Documentation) addWarn(msg string, li LineInfo) {
	d.Modules[d.currentActiveModule].Warnings = append(
		d.Modules[d.currentActiveModule].Warnings,
		DocAnchor{Label: "WARN", LineInfo: li, Message: msg},
	)
}

func (d *Documentation) String() string {
	var sb strings.Builder
	for _, mod := range d.Modules {
		fmt.Fprintf(&sb, "Module: %s\n", mod.Name)
		if len(mod.Author) > 0 {
			fmt.Fprintf(&sb, "  Author: %s\n", strings.Join(mod.Author, ", "))
		}
		if mod.License != "" {
			fmt.Fprintf(&sb, "  License: %s\n", mod.License)
		}
		if mod.Copyright != "" {
			fmt.Fprintf(&sb, "  Copyright: © %s\n", mod.Copyright)
		}
		if mod.Release != "" {
			fmt.Fprintf(&sb, "  Release: %s\n", mod.Release)
		}
	}
	return sb.String()
}
