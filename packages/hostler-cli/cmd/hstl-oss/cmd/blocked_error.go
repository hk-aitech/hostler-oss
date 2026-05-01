// Package cmd - shared BlockedError output helper.
//
// The BlockedError output in `hstl task complete` was reorganized so that the
// cause message is always shown first, even for BLOCKED outcomes that did not
// originate from harness (hotfix rollback verification, strict result_verify,
// etc.). This file is the single entry point that lets other CLI paths such
// as sprint complete reuse the same pattern.
package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/app"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/domain"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/output"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/apperr"
)

// blockedSeparatorWidth is the width of the BLOCKED banner. Extracted as a
// const per the no-magic-number rule.
const blockedSeparatorWidth = 45

// printBlockedError emits the BlockedError cause message and SuggestedActions
// to stderr in a consistent banner format.
//
// It also queries harness state and, if any unchecked items remain, appends
// the detailed printHarnessBlocked output. When harness state is missing or
// the unchecked count is zero, the detail block is omitted to avoid printing
// a blank "(no unchecked items)" notice.
//
// entityType: harness entity kind such as "task" or "sprint"
// entityID:   target ID (e.g., "T422", "sprint-51")
// be:         apperr.BlockedError (assumed non-nil)
func printBlockedError(entityType, entityID string, be *apperr.BlockedError) {
	// Routed through output.Stderr() so JSON mode can suppress via io.Discard.
	printBlockedErrorTo(output.Stderr(), entityType, entityID, be, appHarnessGet)
}

// appHarnessGet adapts app.HarnessGet (which returns any) to the concrete
// signature blocked_error needs (domain.HarnessGetResult). Keeps unit test
// stubs typed against the concrete type even after the port conversion.
func appHarnessGet(entityType, entityID string) (domain.HarnessGetResult, error) {
	raw, err := app.HarnessGet(entityType, entityID)
	if err != nil {
		return domain.HarnessGetResult{}, err
	}
	if p, ok := raw.(*domain.HarnessGetResult); ok && p != nil {
		return *p, nil
	}
	return domain.HarnessGetResult{}, nil
}

// printBlockedErrorTo is the injectable variant of printBlockedError.
// Unit tests use it to swap the writer for a buffer and replace harnessGetter
// with a stub.
func printBlockedErrorTo(
	w io.Writer,
	entityType, entityID string,
	be *apperr.BlockedError,
	harnessGetter func(string, string) (domain.HarnessGetResult, error),
) {
	separator := strings.Repeat("━", blockedSeparatorWidth)
	title := entityType
	if len(title) > 0 {
		title = strings.ToUpper(title[:1]) + title[1:]
	}

	_, _ = fmt.Fprintln(w, separator)
	_, _ = fmt.Fprintf(w, "  %s BLOCKED - %s\n", title, entityID)
	_, _ = fmt.Fprintln(w, separator)
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintf(w, "  Cause: %s\n", be.Error())
	if len(be.SuggestedActions) > 0 {
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, "  How to resolve:")
		for _, sa := range be.SuggestedActions {
			_, _ = fmt.Fprintf(w, "    - %s\n", sa)
		}
	}
	_, _ = fmt.Fprintln(w, separator)

	// Append the detail block only when harness state still has unchecked items.
	if state, hErr := harnessGetter(entityType, entityID); hErr == nil {
		hasUnchecked := false
		for _, item := range state.Items {
			if item.Required && !item.Done {
				hasUnchecked = true
				break
			}
		}
		if hasUnchecked {
			// printHarnessBlocked always writes to stderr and does not accept w;
			// buffer-injection tests only cover the hasUnchecked=false path.
			printHarnessBlocked(entityType, entityID, &state)
		}
	}
}
