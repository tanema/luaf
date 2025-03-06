package luaf

import (
	"math"
	"os"
)

const (
	LUA_SIGNATURE       = "\x1bLuaf"
	LUA_VERSION         = "Luaf 0.1.0"
	LUA_COPYRIGHT       = "Copyright (C) 2024"
	LUA_VERSION_MAJOR_N = 0
	LUA_VERSION_MINOR_N = 1
	LUA_VERSION_PATCH_N = 0
	LUA_FORMAT          = 0             // dump/undump format incase it ever changes
	INITIALSTACKSIZE    = 128           // stack size at vm startup, smaller for smaller footprint, larger for less allocation
	MAXSTACKSIZE        = math.MaxInt64 // max stack size
	MAXUPVALUES         = 255           // max allowed upvals referred in a fn scope
	MAXLOCALS           = 200           // max allowed vars defined in a fn scope
	MAXCONST            = 64_536        // max amount of consts that a fnproto can store
	MAXINLINECONST      = 255           // max index that we can index constants with iABC if larger we need LOADK with iABx
	MAXRESULTS          = 250           // max amount of return values
	MAXARG_A            = math.MaxUint8
	MAXARG_B            = math.MaxUint8
	MAXARG_C            = math.MaxUint8
	MAXARG_Bx           = math.MaxUint16
	MAXARGS_sBx         = math.MaxInt16
	GCPAUSE             = 200 // minimum number of objects before calling collection

	PkgPathSeparator     = string(os.PathSeparator)
	PkgTemplateSeparator = ";"
	PkgSubstitutionPoint = "?"
	PkgExecutableDirWin  = "!"
	PkgIgnoreMark        = "-"
	charPattern          = "[--][-]*"
)
