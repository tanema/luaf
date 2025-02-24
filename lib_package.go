package luaf

import (
	"embed"
	"fmt"
	"os"
	"strings"
)

const (
	PkgPathSeparator     = string(os.PathSeparator)
	PkgTemplateSeparator = ";"
	PkgSubstitutionPoint = "?"
	PkgExecutableDirWin  = "!"
	PkgIgnoreMark        = "-"
)

var (
	//go:embed lib
	stdLib         embed.FS
	PkgPathDefault = []string{
		"./?.lua",
		"./?/init.lua",
	}
	searchPaths    = &String{val: strings.Join(PkgPathDefault, PkgTemplateSeparator)}
	loadedPackages = &Table{
		hashtable: map[any]Value{
			"string":    libString,
			"table":     libTable,
			"math":      libMath,
			"utf8":      libUtf8,
			"os":        libOS,
			"io":        libIO,
			"coroutine": libCoroutine,
			"debug":     libDebug,
		},
	}
)

var libPackage = &Table{
	hashtable: map[any]Value{
		"config": &String{val: strings.Join([]string{
			PkgPathSeparator,
			PkgTemplateSeparator,
			PkgSubstitutionPoint,
			PkgExecutableDirWin,
			PkgIgnoreMark,
		}, "\n")},
		"loaded":     loadedPackages,
		"path":       searchPaths,
		"searchers":  NewTable([]Value{&ExternFunc{stdPkgSearchPath}}, nil),
		"searchpath": &ExternFunc{stdPkgSearchPath},
	},
}

func stdRequire(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(vm, args, "require", "string"); err != nil {
		return nil, err
	}

	dir, err := os.Getwd()
	if err != nil {
		return nil, vm.err("trouble getting pwd: %v", err)
	}
	dirStr := &String{val: dir}

	modNameStr := args[0].(*String)
	modName := modNameStr.val
	loadedCache := loadedPackages.hashtable
	if lib, found := loadedCache[modName]; found {
		return []Value{lib}, nil
	}

	libPath := "lib/" + strings.ReplaceAll(modName, ".", PkgPathSeparator) + ".lua"
	if f, err := stdLib.ReadFile(libPath); err == nil {
		fn, err := Parse(modName, strings.NewReader(string(f)), ModeBinary&ModeText)
		if err != nil {
			return nil, err
		}
		res, err := vm.Eval(fn)
		if err != nil {
			return nil, err
		}
		if len(res) > 0 {
			loadedCache[modName] = res[0]
			return res, nil
		}
		loadedCache[modName] = &Nil{}
		return []Value{&Nil{}}, nil
	}

	var foundPath string
	var lastErr error
	searchers := libPackage.hashtable["searchers"].(*Table).val
	for _, search := range searchers {
		res, err := vm.Call("require", search, []Value{modNameStr, dirStr})
		if len(res) == 1 {
			foundPath = res[0].(*String).val
			break
		} else if len(res) == 2 {
			requireErr := res[1].(*Error).val.String()
			err = fmt.Errorf("%v", requireErr)
		}
		lastErr = err
	}
	if foundPath == "" && lastErr != nil {
		return []Value{}, lastErr
	}

	fn, err := ParseFile(foundPath, ModeText)
	if err != nil {
		return nil, err
	}
	res, err := vm.Eval(fn)
	if err != nil {
		return nil, err
	}

	if len(res) > 0 {
		loadedCache[modName] = res[0]
		return res, nil
	}
	// don't need to load again but nothing to save
	loadedCache[modName] = &Nil{}
	return []Value{&Nil{}}, nil
}

func stdPkgSearchPath(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(vm, args, "package.searchpath", "string", "string", "~string", "~string"); err != nil {
		return nil, err
	}
	searchedPaths := []string{}
	dirPath := args[1].(*String).val
	sep := "."
	if len(args) > 2 {
		sep = args[2].(*String).val
	}
	rep := PkgPathSeparator
	if len(args) > 3 {
		sep = args[3].(*String).val
	}

	modName := strings.ReplaceAll(args[0].(*String).val, sep, rep)
	paths := strings.Split(searchPaths.val, PkgTemplateSeparator)
	for _, pathTmpl := range paths {
		if strings.HasPrefix(pathTmpl, "./") {
			pathTmpl = fmt.Sprintf("%v%v", dirPath, strings.TrimPrefix(pathTmpl, "."))
		}
		modPath := strings.ReplaceAll(pathTmpl, PkgSubstitutionPoint, modName)
		searchedPaths = append(searchedPaths, modPath)
		info, err := os.Stat(modPath)
		if err != nil || info.IsDir() {
			continue
		}
		return []Value{&String{val: modPath}}, nil
	}
	err := fmt.Sprintf("could not find module %v\nin paths:\n%v", args[0].(*String).val, strings.Join(searchedPaths, "\n"))
	return []Value{&Nil{}, &Error{val: &String{val: err}}}, nil
}
