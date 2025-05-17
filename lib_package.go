package luaf

import (
	"embed"
	"fmt"
	"os"
	"strings"

	"github.com/tanema/luaf/src/lerrors"
	"github.com/tanema/luaf/src/parse"
)

const (
	pkgPathSeparator     = string(os.PathSeparator)
	pkgTemplateSeparator = ";"
	pkgSubstitutionPoint = "?"
	pkgExecutableDirWin  = "!"
	pkgIgnoreMark        = "-"
	charPattern          = "[--][-]*"
)

var (
	//go:embed lib
	stdLib         embed.FS
	pkgpathdefault = []string{
		"./?.lua",
		"./?/init.lua",
	}
	searchPaths    = strings.Join(pkgpathdefault, pkgTemplateSeparator)
	loadedPackages = &Table{hashtable: map[any]any{}}
)

var libPackage = &Table{
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
		"searchers":  NewTable([]any{Fn("package.searchpath", stdPkgSearchPath)}, nil),
		"searchpath": Fn("package.searchpath", stdPkgSearchPath),
	},
}

func stdRequire(vm *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "require", "string"); err != nil {
		return nil, err
	}

	dir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("trouble getting pwd: %w", err)
	}
	dirStr := dir

	modName := args[0].(string)
	loadedCache := loadedPackages.hashtable
	if lib, found := loadedCache[modName]; found {
		return []any{lib}, nil
	}

	libPath := "lib/" + strings.ReplaceAll(modName, ".", pkgPathSeparator) + ".lua"
	if f, err := stdLib.ReadFile(libPath); err == nil {
		fn, err := parse.Parse(modName, strings.NewReader(string(f)), parse.ModeBinary&parse.ModeText)
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
		loadedCache[modName] = nil
		return []any{nil}, nil
	}

	var foundPath string
	var lastErr error
	searchers := libPackage.hashtable["searchers"].(*Table).val
	for _, search := range searchers {
		res, err := vm.call(search, []any{modName, dirStr})
		if len(res) == 1 {
			foundPath = res[0].(string)
			break
		} else if len(res) == 2 {
			requireErr := res[1].(error).Error()
			err = fmt.Errorf("%v", requireErr)
		}
		lastErr = err
	}
	if foundPath == "" && lastErr != nil {
		return []any{}, lastErr
	}

	fn, err := parse.File(foundPath, parse.ModeText)
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
	loadedCache[modName] = nil
	return []any{nil}, nil
}

func stdPkgSearchPath(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "package.searchpath", "string", "string", "~string", "~string"); err != nil {
		return nil, err
	}
	searchedPaths := []string{}
	dirPath := args[1].(string)
	sep := "."
	if len(args) > 2 {
		sep = args[2].(string)
	}
	rep := pkgPathSeparator
	if len(args) > 3 {
		sep = args[3].(string)
	}

	modName := strings.ReplaceAll(args[0].(string), sep, rep)
	paths := strings.Split(searchPaths, pkgTemplateSeparator)
	for _, pathTmpl := range paths {
		if strings.HasPrefix(pathTmpl, "./") {
			pathTmpl = fmt.Sprintf("%v%v", dirPath, strings.TrimPrefix(pathTmpl, "."))
		}
		modPath := strings.ReplaceAll(pathTmpl, pkgSubstitutionPoint, modName)
		searchedPaths = append(searchedPaths, modPath)
		info, err := os.Stat(modPath)
		if err != nil || info.IsDir() {
			continue
		}
		return []any{modPath}, nil
	}
	err := fmt.Errorf("could not find module %v\nin paths:\n%v", ToString(args[0]), strings.Join(searchedPaths, "\n"))
	return []any{nil, &lerrors.Error{Kind: lerrors.RuntimeErr, Err: err}}, nil
}
