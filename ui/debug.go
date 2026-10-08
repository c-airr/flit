package ui

import (
	"log"
	"os"
)

// Debug warnings point out layouts that are allowed but probably not what
// the author meant. They are off unless the environment variable
// FLIT_DEBUG is "1"; Go has no separate debug build like C++ with NDEBUG,
// so a run-time switch does the job.

// Both are variables so tests can switch warnings on and capture them.
// os.Getenv runs once, when the package is initialized at program start.
var (
	debugEnabled = os.Getenv("FLIT_DEBUG") == "1"
	debugLogf    = log.Printf
)

const (
	warnNotInFlex = "flit: Expanded is not a child of a Row or Column; it lays out like its child"
	warnUnbounded = "flit: Expanded in a Row/Column with an unbounded main axis; it lays out like an ordinary child"
)

// warnOnce logs msg the first time it is raised for n, so a layout that
// runs every frame does not flood the log.
func warnOnce(n *node, msg string) {
	if !debugEnabled || n.warned {
		return
	}
	n.warned = true
	debugLogf("%s", msg)
}
