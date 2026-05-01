package cmd

// Interactive prompt for task complete.
// Minimal implementation using only bufio.Scanner, with no external dependencies.

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/app"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/output"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/apperr"
)

// isInteractiveTTY reports whether stdin/stdout are actual terminals.
// Returns false in pipelines (CI, | jq ...) so interactive prompts are skipped.
func isInteractiveTTY() bool {
	stdinStat, err1 := os.Stdin.Stat()
	stdoutStat, err2 := os.Stdout.Stat()
	if err1 != nil || err2 != nil {
		return false
	}
	// character device (tty) vs pipe/regular file
	return (stdinStat.Mode()&os.ModeCharDevice) != 0 &&
		(stdoutStat.Mode()&os.ModeCharDevice) != 0
}

// runInteractiveHarnessResolve interactively works through the unchecked items
// of a BlockedError. Each item shows a [y/n/s] prompt - y = enter evidence,
// n = skip, s = abort the whole flow. Returns the number of items processed
// plus whether the user chose to abort.
func runInteractiveHarnessResolve(entityType, entityID string, be *apperr.BlockedError) (checked int, aborted bool) {
	if !isInteractiveTTY() {
		fmt.Fprintln(output.Stderr(), "[--interactive] not a TTY - skipping interactive prompt")
		return 0, false
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Fprintf(output.Stderr(), "\n%s %s: %d unchecked Harness items - interactive resolve\n",
		strings.ToUpper(entityType), entityID, len(be.UncheckedItems))

	for i, item := range be.UncheckedItems {
		fmt.Fprintf(output.Stderr(), "\n[%d/%d] %s - %s (required=%v)\n",
			i+1, len(be.UncheckedItems), item.ID, item.Name, item.Required)
		fmt.Fprintf(output.Stderr(), "  [y] check (enter evidence)  [n] skip  [s] abort: ")

		line, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintf(output.Stderr(), "[--interactive] failed to read input: %v\n", err)
			return checked, true
		}
		choice := strings.ToLower(strings.TrimSpace(line))
		switch choice {
		case "y", "yes":
			fmt.Fprintf(output.Stderr(), "  evidence (one line, press Enter to finish): ")
			evLine, err := reader.ReadString('\n')
			if err != nil {
				fmt.Fprintf(output.Stderr(), "[--interactive] failed to read evidence\n")
				continue
			}
			ev := strings.TrimSpace(evLine)
			if ev == "" {
				fmt.Fprintln(output.Stderr(), "  evidence empty - skipping")
				continue
			}
			if _, err := app.HarnessCheck(entityType, entityID, item.ID, ev, "cli-interactive"); err != nil {
				fmt.Fprintf(output.Stderr(), "  harness_check failed: %v\n", err)
				continue
			}
			fmt.Fprintf(output.Stderr(), "  OK %s checked\n", item.ID)
			checked++
		case "n", "no", "":
			fmt.Fprintln(output.Stderr(), "  skipped")
		case "s", "skip", "abort", "q":
			fmt.Fprintln(output.Stderr(), "  aborted")
			return checked, true
		default:
			fmt.Fprintf(output.Stderr(), "  unknown input %q - skipping\n", choice)
		}
	}

	fmt.Fprintf(output.Stderr(), "\n[--interactive] done: %d/%d items checked\n", checked, len(be.UncheckedItems))
	return checked, false
}
