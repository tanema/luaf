package luaf

import (
	"math"
	"os"
)

const (
	LUASIGNATURE     = "\x1bLuaf"
	LUAVERSION       = "Luaf 0.1.0"
	LUACOPYRIGHT     = "Copyright (C) 2024"
	LUAVERSIONMAJORN = 0
	LUAVERSIONMINORN = 1
	LUAVERSIONPATCHN = 0
	LUAFORMAT        = 0             // dump/undump format incase it ever changes
	INITIALSTACKSIZE = 128           // stack size at vm startup
	MAXSTACKSIZE     = math.MaxInt64 // max stack size
	MAXUPVALUES      = 255           // max allowed upvals referred in a fn scope
	MAXLOCALS        = 200           // max allowed vars defined in a fn scope
	MAXCONST         = 64_536        // max amount of consts that a fnproto can store
	MAXINLINECONST   = 255           // max index that we can index constants with iABC
	MAXRESULTS       = 250           // max amount of return values
	MAXARGA          = math.MaxUint8
	MAXARGB          = math.MaxUint8
	MAXARGC          = math.MaxUint8
	MAXARGBx         = math.MaxUint16
	MAXARGSsBx       = math.MaxInt16
	GCPAUSE          = 200 // minimum number of objects before calling collection

	PkgPathSeparator     = string(os.PathSeparator)
	PkgTemplateSeparator = ";"
	PkgSubstitutionPoint = "?"
	PkgExecutableDirWin  = "!"
	PkgIgnoreMark        = "-"
	charPattern          = "[--][-]*"
)
