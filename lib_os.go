package luaf

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
)

var startTime time.Time

func init() {
	startTime = time.Now()
}

func createOSLib() *Table {
	return &Table{
		hashtable: map[any]any{
			"clock":     Fn("os.clock", stdOSClock),
			"execute":   Fn("os.execute", stdOSExecute),
			"exit":      Fn("os.exit", stdOSExit),
			"getenv":    Fn("os.getenv", stdOSGetenv),
			"remove":    Fn("os.remove", stdOSRemove),
			"rename":    Fn("os.rename", stdOSRename),
			"setlocale": Fn("os.setlocale", stdOSSetlocale),
			"tmpname":   Fn("os.tmpname", stdOSTmpname),
			"time":      Fn("os.time", stdOSTime),
			"date":      Fn("os.date", stdOSDate),
			"difftime":  Fn("os.difftime", stdOSDifftime),
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
	if err := popenCommand(args[0].(string)).Run(); err != nil {
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
	return []any{os.Getenv(args[0].(string))}, nil
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

func stdOSSetlocale(_ *VM, _ []any) ([]any, error) {
	return []any{false}, nil
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
		return []any{time.Now().Unix()}, nil
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
	return []any{t.Unix()}, nil
}

func stdOSDifftime(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "os.difftime", "number", "number"); err != nil {
		return nil, err
	}
	return []any{toInt(args[0]) - toInt(args[1])}, nil
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
