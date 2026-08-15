//go:build !darwin

package main

import (
	"fmt"
	"io"
)

var (
	version = "dev"
	commit  = "unknown"
)

// run is a compile-only non-Darwin refusal. U9 makes no non-Darwin runtime
// claim; only help and version are inertly available.
func run(args []string, _ io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 1 && (args[0] == "version" || args[0] == "--version") {
		fmt.Fprintf(stdout, "countershape %s (%s)\n", version, commit)
		return 0
	}
	if len(args) == 1 && (args[0] == "help" || args[0] == "--help" || args[0] == "-h") {
		fmt.Fprintln(stdout, "Countershape: Darwin runtime required; this binary is compilation evidence only.")
		return 0
	}
	fmt.Fprintln(stderr, "countershape: Darwin runtime required; Linux runtime is UNRECEIPTED")
	return 1
}
