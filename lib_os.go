package luaf

import (
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

var libOS = &Table{
	hashtable: map[any]Value{
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

func stdOSClock(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(vm, args, "os.clock"); err != nil {
		return nil, err
	}
	return []Value{&Float{val: time.Since(startTime).Seconds()}}, nil
}

func stdOSExecute(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(vm, args, "os.execute", "~string"); err != nil {
		return nil, err
	}
	if len(args) == 0 {
		return []Value{&Boolean{val: true}}, nil
	}
	if err := popenCommand(args[0].(*String).val).Run(); err != nil {
		if execErr, ok := err.(*exec.ExitError); ok {
			code := execErr.ExitCode()
			if execErr.ProcessState.Exited() {
				return []Value{&Nil{}, &String{val: "exit"}, &Integer{val: int64(code)}}, nil
			}
			return []Value{&Nil{}, &String{val: "signal"}, &Integer{val: int64(code)}}, nil
		}
		return []Value{&Boolean{val: false}, &String{val: "exit"}, &Integer{val: 1}}, nil
	}
	return []Value{&Boolean{val: true}, &String{val: "exit"}, &Integer{val: 0}}, nil
}

func stdOSExit(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(vm, args, "os.exit", "~nil|boolean|number", "~boolean"); err != nil {
		return nil, err
	}
	code := 0
	closeAll := false
	if len(args) > 0 {
		switch args[0].(type) {
		case *Integer, *Float:
			code = int(toInt(args[0]))
		case *Nil:
		default:
			if !toBool(args[0]).val {
				code = 1
			}
		}
	}
	if len(args) > 1 {
		closeAll = toBool(args[1]).val
	}
	return nil, &Interrupt{kind: InterruptExit, code: code, flag: closeAll}
}

func stdOSGetenv(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(vm, args, "os.getenv", "string"); err != nil {
		return nil, err
	}
	return []Value{&String{val: os.Getenv(args[0].(*String).val)}}, nil
}

func stdOSRemove(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(vm, args, "os.remove", "string"); err != nil {
		return nil, err
	}
	if err := os.Remove(args[0].(*String).val); err != nil {
		return []Value{&Nil{}, &String{val: err.Error()}}, nil
	}
	return []Value{&Boolean{val: true}}, nil
}

func stdOSRename(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(vm, args, "os.rename", "string", "string"); err != nil {
		return nil, err
	}
	if err := os.Rename(args[0].(*String).val, args[1].(*String).val); err != nil {
		return []Value{&Nil{}, &String{val: err.Error()}}, nil
	}
	return []Value{&Boolean{val: true}}, nil
}

func stdOSSetlocale(vm *VM, args []Value) ([]Value, error) {
	return []Value{&Boolean{val: false}}, nil
}

func stdOSTmpname(vm *VM, args []Value) ([]Value, error) {
	pathname := filepath.Join(os.TempDir(), strconv.Itoa(int(randSource.Uint32())))
	return []Value{&String{val: pathname}}, nil
}

func stdOSTime(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(vm, args, "os.time", "~table"); err != nil {
		return nil, err
	}
	if len(args) == 0 {
		return []Value{&Integer{val: time.Now().Unix()}}, nil
	}
	timeTable := args[0].(*Table).hashtable
	if isNil(timeTable["year"]) {
		return nil, fmt.Errorf("field 'year' missing in the time table")
	} else if isNil(timeTable["month"]) {
		return nil, fmt.Errorf("field 'month' missing in the time table")
	} else if isNil(timeTable["day"]) {
		return nil, fmt.Errorf("field 'day' missing in the time table")
	}
	year := toInt(timeTable["year"])
	month := toInt(timeTable["month"])
	day := toInt(timeTable["day"])
	hour := toIntWithDefault(timeTable["hour"], 12)
	min := toIntWithDefault(timeTable["min"], 0)
	sec := toIntWithDefault(timeTable["sec"], 0)
	t := time.Date(int(year), time.Month(month), int(day), int(hour), int(min), int(sec), 0, time.Local)
	return []Value{&Integer{val: t.Unix()}}, nil
}

func stdOSDifftime(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(vm, args, "os.difftime", "number", "number"); err != nil {
		return nil, err
	}
	return []Value{&Integer{val: toInt(args[0]) - toInt(args[1])}}, nil
}

func stdOSDate(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(vm, args, "os.date", "~string", "~number"); err != nil {
		return nil, err
	}
	format := "%c"
	if len(args) > 0 {
		format = args[0].(*String).val
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
		tbl := NewTable(nil, map[any]Value{
			"year":   &Integer{val: int64(fmtTime.Year())},
			"month":  &Integer{val: int64(fmtTime.Month())},
			"day":    &Integer{val: int64(fmtTime.Day())},
			"hour":   &Integer{val: int64(fmtTime.Hour())},
			"min":    &Integer{val: int64(fmtTime.Minute())},
			"second": &Integer{val: int64(fmtTime.Second())},
			"wday":   &Integer{val: int64(fmtTime.Weekday() + 1)},
			"yday":   &Integer{val: int64(fmtTime.YearDay())},
			"isdst":  &Boolean{val: fmtTime.IsDST()},
		})
		return []Value{tbl}, nil
	}
	strf, err := strftime.New(format)
	if err != nil {
		return nil, fmt.Errorf("invalid time format '%vs'", format)
	}
	return []Value{&String{val: strf.FormatString(fmtTime)}}, nil
}

func toIntWithDefault(val Value, def int64) int64 {
	if isNil(val) {
		return def
	}
	return toInt(val)
}
