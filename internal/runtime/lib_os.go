package runtime

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/lestrrat-go/strftime"

	"github.com/tanema/luaf/internal/i18n"
)

var startTime time.Time

func init() {
	startTime = time.Now()
}

func createOSLib() *Table {
	return &Table{
		hashtable: map[any]any{
			"clock":      Fn("os.clock", stdOSClock),
			"execute":    Fn("os.execute", stdOSExecute),
			"exit":       Fn("os.exit", stdOSExit),
			"getenv":     Fn("os.getenv", stdOSGetenv),
			"chdir":      Fn("os.chdir", stdOSChdir),
			"currentdir": Fn("os.currentdir", stdOSCurrentDir),
			"link":       Fn("os.link", stdOSLink),
			"mkdir":      Fn("os.mkdir", stdOSMkDir),
			"rmdir":      Fn("os.rmdir", stdOSRmDir),
			"remove":     Fn("os.remove", stdOSRemove),
			"rename":     Fn("os.rename", stdOSRename),
			"setlocale":  Fn("os.setlocale", stdOSSetlocale),
			"tmpname":    Fn("os.tmpname", stdOSTmpname),
			"time":       Fn("os.time", stdOSTime),
			"date":       Fn("os.date", stdOSDate),
			"difftime":   Fn("os.difftime", stdOSDifftime),
		},
	}
}

func stdOSClock(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "os.clock"); err != nil {
		return nil, err
	}
	return []any{time.Since(startTime).Seconds()}, nil
}

func stdOSExecute(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "os.execute", "~string"); err != nil {
		return nil, err
	}
	if len(args) == 0 {
		return []any{true}, nil
	}
	if err := PopenCommand(args[0].(string)).Run(); err != nil {
		var execErr *exec.ExitError
		if errors.As(err, &execErr) {
			code := execErr.ExitCode()
			if execErr.Exited() {
				return []any{nil, "exit", int64(code)}, nil
			}
			return []any{nil, "signal", int64(code)}, nil
		}
		return []any{false, "exit", int64(1)}, nil
	}
	return []any{true, "exit", int64(0)}, nil
}

func stdOSExit(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "os.exit", "~nil|boolean|number", "~boolean"); err != nil {
		return nil, err
	}
	code := 0
	closeAll := false
	if len(args) > 0 {
		switch args[0].(type) {
		case int64, float64:
			code = int(toInt(args[0]))
		case nil:
		default:
			if !toBool(args[0]) {
				code = 1
			}
		}
	}
	if len(args) > 1 {
		closeAll = toBool(args[1])
	}
	return nil, &Interrupt{kind: InterruptExit, code: code, flag: closeAll}
}

func stdOSGetenv(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "os.getenv", "string"); err != nil {
		return nil, err
	}
	envVar := os.Getenv(args[0].(string))
	if envVar == "" {
		return []any{nil}, nil
	}
	return []any{envVar}, nil
}

func stdOSChdir(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "os.chdir", "string"); err != nil {
		return nil, err
	}
	newDir := args[0].(string)
	if err := os.Chdir(newDir); err != nil {
		return []any{nil, toLuaError(err)}, nil
	}
	return []any{true, nil}, nil
}

func stdOSCurrentDir(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "os.currentdir"); err != nil {
		return nil, err
	}
	dir, err := os.Getwd()
	if err != nil {
		return []any{nil, toLuaError(err)}, nil
	}
	return []any{dir, nil}, nil
}

func stdOSLink(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "os.mkdir", "string", "string", "~boolean"); err != nil {
		return nil, err
	}
	oldname := args[0].(string)
	newname := args[1].(string)
	symlink := false
	if len(args) > 2 {
		symlink = args[2].(bool)
	}

	if symlink {
		if err := os.Symlink(oldname, newname); err != nil {
			return []any{nil, toLuaError(err)}, nil
		}
	} else if err := os.Link(oldname, newname); err != nil {
		return []any{nil, toLuaError(err)}, nil
	}
	return []any{true, nil}, nil
}

func stdOSMkDir(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "os.mkdir", "string"); err != nil {
		return nil, err
	}

	dirName := args[0].(string)
	if err := os.MkdirAll(dirName, os.ModeDir); err != nil {
		return []any{nil, toLuaError(err)}, nil
	}

	return []any{true, nil}, nil
}

func stdOSRmDir(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "os.rmdir", "string"); err != nil {
		return nil, err
	}
	dirName := args[0].(string)
	if err := os.RemoveAll(dirName); err != nil {
		return []any{nil, toLuaError(err)}, nil
	}
	return []any{true, nil}, nil
}

func stdOSRemove(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "os.remove", "string"); err != nil {
		return nil, err
	}
	var retVals []any
	if err := os.Remove(args[0].(string)); err != nil {
		retVals = []any{nil, err.Error()}
	} else {
		retVals = []any{true}
	}
	return retVals, nil
}

func stdOSRename(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "os.rename", "string", "string"); err != nil {
		return nil, err
	}
	var retVals []any
	if err := os.Rename(args[0].(string), args[1].(string)); err != nil {
		retVals = []any{nil, err.Error()}
	} else {
		retVals = []any{true}
	}
	return retVals, nil
}

func stdOSSetlocale(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "os.setlocale", "~string", "~string"); err != nil {
		return nil, err
	} else if len(args) == 0 {
		return []any{i18n.GetLocale(i18n.CategoryALL).String()}, nil
	}
	locale, err := i18n.ParseLocale(args[0].(string))
	if err != nil {
		return nil, err
	}
	category := i18n.CategoryALL
	if len(args) > 1 {
		category, err = i18n.ParseCategory(args[1].(string))
		if err != nil {
			return nil, err
		}
	}
	i18n.SetLocale(locale, category)
	return []any{locale.String()}, nil
}

func stdOSTmpname(_ *VM, _ []any) ([]any, error) {
	pathname := filepath.Join(os.TempDir(), strconv.Itoa(int(randSource.Uint32())))
	return []any{pathname}, nil
}

func stdOSTime(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "os.time", "~table"); err != nil {
		return nil, err
	}
	if len(args) == 0 {
		return []any{float64(time.Now().Unix()) / 1000}, nil
	}
	timeTable := args[0].(*Table).hashtable
	if timeTable["year"] == nil {
		return nil, errors.New("field 'year' missing in the time table")
	} else if timeTable["month"] == nil {
		return nil, errors.New("field 'month' missing in the time table")
	} else if timeTable["day"] == nil {
		return nil, errors.New("field 'day' missing in the time table")
	}
	year := toInt(timeTable["year"])
	month := toInt(timeTable["month"])
	day := toInt(timeTable["day"])
	hour := toIntWithDefault(timeTable["hour"], 12)
	minute := toIntWithDefault(timeTable["min"], 0)
	sec := toIntWithDefault(timeTable["sec"], 0)
	t := time.Date(int(year), time.Month(month), int(day), int(hour), int(minute), int(sec), 0, time.Local)
	return []any{float64(t.Unix()) / 1000}, nil
}

func stdOSDifftime(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "os.difftime", "number", "number"); err != nil {
		return nil, err
	}
	return []any{toFloat(args[0]) - toFloat(args[1])}, nil
}

func stdOSDate(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "os.date", "~string", "~number"); err != nil {
		return nil, err
	}
	format := "%c"
	if len(args) > 0 {
		format = args[0].(string)
	}
	fmtTime := time.Now()
	if len(args) > 1 {
		fmtTime = time.Unix(toInt(args[1]), 0)
	}
	if strings.HasPrefix(format, "!") {
		fmtTime = fmtTime.UTC()
	}
	format = strings.TrimPrefix(format, "!")
	if strings.TrimSpace(format) == "*t" {
		tbl := NewTable(nil, map[any]any{
			"year":   int64(fmtTime.Year()),
			"month":  int64(fmtTime.Month()),
			"day":    int64(fmtTime.Day()),
			"hour":   int64(fmtTime.Hour()),
			"min":    int64(fmtTime.Minute()),
			"second": int64(fmtTime.Second()),
			"wday":   int64(fmtTime.Weekday() + 1),
			"yday":   int64(fmtTime.YearDay()),
			"isdst":  fmtTime.IsDST(),
		})
		return []any{tbl}, nil
	}
	strf, err := strftime.New(format)
	if err != nil {
		return nil, fmt.Errorf("invalid time format '%vs'", format)
	}
	return []any{strf.FormatString(fmtTime)}, nil
}

func toIntWithDefault(val any, def int64) int64 {
	if val == nil {
		return def
	}
	return toInt(val)
}
