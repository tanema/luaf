package runtime

import (
	"embed"
	"fmt"
	"os"
	"strings"

	"github.com/tanema/luaf/internal/lerrors"
	"github.com/tanema/luaf/internal/parse"
)

const (
	pkgPathSeparator     = string(os.PathSeparator)
	pkgTemplateSeparator = ";"
	pkgSubstitutionPoint = "?"
	pkgExecutableDirWin  = "!"
	pkgIgnoreMark        = "-"
	charPattern          = "[--][-]*"
)

type resolveStrategy = func(*VM, string) (bool, any, error)

var (
	//go:embed lib
	stdLib embed.FS
	//go:embed builtin/builtin.lua
	builtinLib      string
	pkgpathdefault  = []string{"./?.lua", "./?/init.lua"}
	pkgBuiltinPaths = []string{"lib/?.lua", "lib/?/init.lua"}
	pkgSearchers    = NewTable([]any{Fn("package.searchpath", stdPkgSearchPath)}, nil)
	searchPaths     = strings.Join(pkgpathdefault, pkgTemplateSeparator)
	loadedPackages  = &Table{hashtable: map[any]any{}}
	stdPackageLib   = &Table{
		hashtable: map[any]any{
			"config": strings.Join([]string{
				pkgPathSeparator,
				pkgTemplateSeparator,
				pkgSubstitutionPoint,
				pkgExecutableDirWin,
				pkgIgnoreMark,
			}, "\n"),
			"loaded":     loadedPackages,
			"path":       searchPaths,
			"searchers":  pkgSearchers,
			"searchpath": Fn("package.searchpath", stdPkgSearchPath),
		},
	}
)

func stdRequire(vm *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "require", "string"); err != nil {
		return nil, err
	}

	modName := args[0].(string)
	moduleResolutionStrategies := []resolveStrategy{searchLibCache, searchStdLib, searchBuiltinLib, searchUserModules}
	for _, strategy := range moduleResolutionStrategies {
		found, lib, err := strategy(vm, modName)
		if err != nil {
			return nil, err
		} else if found {
			loadedPackages.hashtable[modName] = lib
			return []any{lib}, nil
		}
	}

	return nil, newModuleNotFoundErr(modName)
}

func newModuleNotFoundErr(modName string) error {
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("trouble getting pwd: %w", err)
	}
	searchedPaths := []string{
		fmt.Sprintf("\tno field package.preload[%q]", modName),
	}
	for _, path := range generateUserSearchPaths(modName, dir, ".", pkgPathSeparator) {
		searchedPaths = append(searchedPaths, fmt.Sprintf("\tno file %q", path))
	}
	return fmt.Errorf("module %q not found:\n%v", modName, strings.Join(searchedPaths, "\n"))
}

func searchLibCache(_ *VM, modName string) (bool, any, error) {
	loadedCache := loadedPackages.hashtable
	lib, found := loadedCache[modName]
	return found, lib, nil
}

func searchStdLib(_ *VM, modName string) (bool, any, error) {
	std := map[string]func() *Table{
		"coroutine": createCoroutineLib,
		"debug":     createDebugLib,
		"io":        createIOLib,
		"math":      createMathLib,
		"os":        createOSLib,
		"string":    createStringLib,
		"table":     createTableLib,
		"utf8":      createUtf8Lib,
	}
	mod, found := std[modName]
	if !found {
		return false, nil, nil
	}
	lib := mod()
	return true, lib, nil
}

func generateBuiltinSearchPaths(modName string) []string {
	searchedPaths := make([]string, len(pkgBuiltinPaths))
	modName = strings.ReplaceAll(modName, ".", pkgPathSeparator)
	for i, pathTmpl := range pkgBuiltinPaths {
		searchedPaths[i] = strings.ReplaceAll(pathTmpl, pkgSubstitutionPoint, modName)
	}
	return searchedPaths
}

func searchBuiltinLib(vm *VM, modName string) (bool, any, error) {
	for _, modPath := range generateBuiltinSearchPaths(modName) {
		if f, err := stdLib.ReadFile(modPath); err != nil {
			continue
		} else if fn, err := parse.Parse(modName, strings.NewReader(string(f)), parse.ModeBinary|parse.ModeText); err != nil {
			return false, nil, err
		} else if res, err := vm.Eval(fn); err != nil {
			return false, nil, err
		} else if len(res) > 0 {
			return true, res[0], nil
		}
	}
	return false, nil, nil
}

func searchUserModules(vm *VM, modName string) (bool, any, error) {
	dir, err := os.Getwd()
	if err != nil {
		return false, nil, fmt.Errorf("trouble getting pwd: %w", err)
	}

	var foundPath string
	searchers := pkgSearchers.val
	for _, search := range searchers {
		if res, err := vm.call(search, []any{modName, dir}); err != nil {
			return false, nil, err
		} else if len(res) == 1 {
			foundPath = res[0].(string)
			break
		}
	}
	if foundPath == "" {
		return false, nil, nil
	}

	if fn, err := parse.File(foundPath, parse.ModeText); err != nil {
		return false, nil, err
	} else if res, err := vm.Eval(fn); err != nil {
		return false, nil, err
	} else if len(res) > 0 {
		return true, res[0], nil
	}
	return true, nil, nil
}

func generateUserSearchPaths(modName, dirPath, sep, rep string) []string {
	searchedPaths := []string{}
	modName = strings.ReplaceAll(modName, sep, rep)
	for pathTmpl := range strings.SplitSeq(searchPaths, pkgTemplateSeparator) {
		if strings.HasPrefix(pathTmpl, "./") {
			pathTmpl = fmt.Sprintf("%v%v", dirPath, strings.TrimPrefix(pathTmpl, "."))
		}
		searchedPaths = append(searchedPaths, strings.ReplaceAll(pathTmpl, pkgSubstitutionPoint, modName))
	}
	return searchedPaths
}

func stdPkgSearchPath(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "package.searchpath", "string", "string", "~string", "~string"); err != nil {
		return nil, err
	}
	modName := args[0].(string)
	sep := "."
	if len(args) > 2 {
		sep = args[2].(string)
	}
	rep := pkgPathSeparator
	if len(args) > 3 {
		sep = args[3].(string)
	}

	paths := generateUserSearchPaths(args[0].(string), args[1].(string), sep, rep)
	for _, modPath := range paths {
		info, err := os.Stat(modPath)
		if err != nil || info.IsDir() {
			continue
		}
		return []any{modPath}, nil
	}
	return []any{nil, &lerrors.Error{Kind: lerrors.RuntimeErr, Err: newModuleNotFoundErr(modName)}}, nil
}
