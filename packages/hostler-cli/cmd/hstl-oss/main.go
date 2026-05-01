// Package main is the hstl CLI entry point.
// rootCmd has SilenceErrors/SilenceUsage=true, which suppresses cobra's
// default error output. The wrapper below explicitly prints the err
// returned by Execute() to stderr so users can immediately see the cause
// when a command fails.
//
// Output goes through `output.Stderr()`: in JSON mode the stream is
// redirected to io.Discard to keep the stdout JSON payload clean. When
// Out (Formatter) has been initialized we prefer structured error output
// via Out.Error; otherwise we fall back to os.Stderr.
package main

import (
	"fmt"
	"os"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/cmd/hstl-oss/cmd"
)

// No magic numbers - exit codes extracted to constants.
const (
	exitSuccess = 0
	exitFailure = 1
)

func main() {
	if err := cmd.Execute(); err != nil {
		// If Out was initialized in PersistentPreRun, route the error
		// through the Formatter (JSON mode = payload, console/text = stderr).
		// For uninitialized paths (e.g. unknown flag parse failures), fall
		// back to os.Stderr.
		if cmd.Out != nil {
			cmd.Out.Error(err.Error(), "", "")
		} else {
			fmt.Fprintln(os.Stderr, "Error:", err)
		}
		os.Exit(exitFailure)
	}
	os.Exit(exitSuccess)
}
