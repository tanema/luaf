package conf

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"

	"github.com/tanema/luaf/internal/i18n"
)

const (
	appName              = "luaf"
	systemConfigFilename = "config.lua"
)

// Config not fully implemented yet. It is a configuration for how the Parser
// behaves on each file that it parses. It may allow for the program to be more
// strict.
type Config struct {
	StringArith      bool         `lua:"string_arith"`  // disallow string coersion in arith
	RequireOnly      bool         `lua:"require_only"`  // require std libs instead of available by default
	ReadOnlyEnv      bool         `lua:"read_only_env"` // not allowed to change _ENV
	Globals          bool         `lua:"globals"`       // not allowed to define globals only locals
	Strict           bool         `lua:"strict"`        // type checking and throw parsing errors if types are bad
	Locale           *i18n.Locale `lua:"locale"`        // change locale for just this file.
	InitialStackSize int          `lua:"initial_stack_size"`
	MaxStackSize     int          `lua:"max_stack_size"`
	MaxCallDepth     int          `lua:"max_call_depth"`
	MaxCallChain     int          `lua:"max_call_chain"`
	MaxUpvalues      int          `lua:"max_upvalues"`
	MaxLocals        int          `lua:"max_locals"`
	MaxConst         int          `lua:"max_const"`
	MaxInlineConst   uint16       `lua:"max_inline_const"`
	MaxResults       int          `lua:"max_results"`
	MaxRegs          uint8        `lua:"max_regs"`
	MaxCalls         int          `lua:"max_calls"`
	GCPause          int          `lua:"gc_pause"`
}

// New creates a new config for parsing and running lua.
func New() Config {
	return Config{
		StringArith:      true,
		Globals:          true,
		Locale:           i18n.GetLocale(i18n.CategoryALL),
		InitialStackSize: INITIALSTACKSIZE,
		MaxStackSize:     MAXSTACKSIZE,
		MaxCallDepth:     MAXCALLDEPTH,
		MaxCallChain:     MAXCALLCHAIN,
		MaxUpvalues:      MAXUPVALUES,
		MaxLocals:        MAXLOCALS,
		MaxConst:         MAXCONST,
		MaxInlineConst:   MAXINLINECONST,
		MaxResults:       MAXRESULTS,
		MaxRegs:          MAXREGS,
		MaxCalls:         MAXCALLS,
		GCPause:          GCPAUSE,
	}
}

func (c *Config) String() string {
	return fmt.Sprintf(`-- Luaf config file that changes how parsing and runtime operate
-- Parsing
config.strict        = %v -- type checking and throw parsing errors if types are bad.
config.string_arith  = %v -- allow/disallow string coersion in arith.
config.require_only  = %v -- allow/disallow usage of std libs without requiring them. 
config.globals       = %v -- allow/disallow defining globals.
config.read_only_env = %v -- allow/disallow changes made to _G/_ENV including changing existing globals.
config.locale        = "%v" -- set locale
-- Runtime (Be careful)
config.initial_stack_size = %v
config.max_stack_size = %v
config.max_call_depth = %v
config.max_call_chain = %v
config.max_upvalues = %v
config.max_locals = %v
config.max_const = %v
config.max_inline_const = %v
config.max_results = %v
config.max_regs = %v
config.max_calls = %v
config.gc_pause = %v`,
		c.Strict,
		c.StringArith,
		c.RequireOnly,
		c.ReadOnlyEnv,
		c.Globals,
		c.Locale,
		c.InitialStackSize,
		c.MaxStackSize,
		c.MaxCallDepth,
		c.MaxCallChain,
		c.MaxUpvalues,
		c.MaxLocals,
		c.MaxConst,
		c.MaxInlineConst,
		c.MaxResults,
		c.MaxRegs,
		c.MaxCalls,
		c.GCPause,
	)
}

// SystemConfigFile returns the config file and ensures that it exists.
func SystemConfigFile() (*os.File, error) {
	return ensureFile(xdg.ConfigFile(filepath.Join(appName, systemConfigFilename)))
}

// SystemDataFile returns the data file and ensures that it exists.
func SystemDataFile(filename string) (*os.File, error) {
	return ensureFile(xdg.DataFile(filepath.Join(appName, filename)))
}

// SystemStateFile returns the state file and ensures that it exists.
func SystemStateFile(filename string) (*os.File, error) {
	return ensureFile(xdg.StateFile(filepath.Join(appName, filename)))
}

// SystemCacheFile returns the cache file and ensures that it exists.
func SystemCacheFile(filename string) (*os.File, error) {
	return ensureFile(xdg.CacheFile(filepath.Join(appName, filename)))
}

// SystemRuntimeFile returns the runtime file and ensures that it exists.
func SystemRuntimeFile(filename string) (*os.File, error) {
	return ensureFile(xdg.CacheFile(filepath.Join(appName, filename)))
}

func ensureFile(path string, err error) (*os.File, error) {
	var file *os.File
	if err != nil {
		return nil, err
	} else if file, err = os.Open(path); err == nil {
		return file, nil
	} else if file, err = os.Create(path); err != nil {
		return nil, err
	}
	return file, nil
}
