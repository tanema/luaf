package parse

type (
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
		DocTypeDesc
		Author    []string
		Copyright string
		License   string
		Release   string
		Generic   []string
		Params    []DocTypeDesc
		Returns   []DocTypeDesc
		Aliases   []DocTypeDesc
		Variables []DocVariable
		Classes   []DocVariable
		TODOs     []DocAnchor
		FixMes    []DocAnchor
		Warnings  []DocAnchor
		Meta      bool
	}
	// DocVariable captures the documentation for any value with attributes.
	// Table, string, function, etc. If it is a function the Func field is defined
	// with the function attributes. If it is a table the Table doc will be defined.
	DocVariable struct {
		DocTypeDesc
		Local    bool // if the variable has been defined as local or global.
		Const    bool // if the variable has been defined as const.
		Language string
		Func     *DocFunc
		Table    *DocTable
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
