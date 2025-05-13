package luaf

import (
	"math"
)

const (
	// LUASIGNATURE is an artifact to put at the beginning of a dumped fnproto so that we can detect binary data.
	LUASIGNATURE = "\x1bLuaf"
	// LUACOPYRIGHT is the copyright to be written out in the CLI.
	LUACOPYRIGHT = "Copyright (C) 2025"
	// LUAVERSION is the version of the luaf application.
	LUAVERSION = "Luaf 0.1.0"
	// LUAVERSIONMAJORN is the major version.
	LUAVERSIONMAJORN = 0
	// LUAVERSIONMINORN is the minor version.
	LUAVERSIONMINORN = 1
	// LUAVERSIONPATCHN is the patch version.
	LUAVERSIONPATCHN = 0
	// LUAFORMAT dump/undump format incase it ever changes.
	LUAFORMAT = 0
	// INITIALSTACKSIZE  stack size at vm startup.
	INITIALSTACKSIZE = 128
	// MAXSTACKSIZE  max stack size.
	MAXSTACKSIZE = math.MaxInt64
	// MAXUPVALUES max allowed upvals referred in a fn scope.
	MAXUPVALUES = 255
	// MAXLOCALS max allowed vars defined in a fn scope.
	MAXLOCALS = 200
	// MAXCONST max amount of consts that a fnproto can store.
	MAXCONST = 64_536
	// MAXINLINECONST max index that we can index constants with iABC.
	MAXINLINECONST = 255
	// MAXRESULTS max amount of return values.
	MAXRESULTS = 250
	// GCPAUSE minimum number of objects before calling collection.
	GCPAUSE = 200
)
