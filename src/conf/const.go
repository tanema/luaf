// Package conf contains the constants that are used across packages for configuring
// versions and stack sizes.
package conf

import (
	"fmt"
	"math"
	"time"
)

const (
	// LUASIGNATURE is an artifact to put at the beginning of a dumped fnproto so that we can detect binary data.
	LUASIGNATURE = "\x1bLuaf"
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

// FullVersion returns the version and copyright.
func FullVersion() string {
	return fmt.Sprintf("%v Copyright (C) %v", LUAVERSION, time.Now().Year())
}

// Copyright is the copyright to be written out in the CLI.
func Copyright() string {
	return fmt.Sprintf("Copyright (C) %v", time.Now().Year())
}
