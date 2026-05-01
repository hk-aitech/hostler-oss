package cmd

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/domain"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/apperr"
)

// TestT434_printBlockedErrorTo_CauseMessageAlwaysVisible verifies that the
// BlockedError cause message is always shown first instead of a blank
// "(no unchecked items)" notice (regression guard).
func TestT434_printBlockedErrorTo_CauseMessageAlwaysVisible(t *testing.T) {
	var buf bytes.Buffer
	be := &apperr.BlockedError{
		Message: "hotfix rollback verification failed - missing rollback section",
		SuggestedActions: []string{
			"add a ## Rollback section to the hotfix Task body",
		},
	}
	// Simulate the case where harness state is empty.
	stubGetter := func(t, id string) (domain.HarnessGetResult, error) {
		return domain.HarnessGetResult{}, errors.New("not found")
	}

	printBlockedErrorTo(&buf, "task", "T999", be, stubGetter)

	out := buf.String()
	if !strings.Contains(out, "Task BLOCKED - T999") {
		t.Errorf("missing BLOCKED banner: %q", out)
	}
	if !strings.Contains(out, "hotfix rollback verification failed") {
		t.Errorf("missing cause message: %q", out)
	}
	if !strings.Contains(out, "add a ## Rollback section") {
		t.Errorf("missing SuggestedActions: %q", out)
	}
}

// TestT434_printBlockedErrorTo_SprintTitleCase verifies that
// entityType = "sprint" renders as "Sprint BLOCKED".
func TestT434_printBlockedErrorTo_SprintTitleCase(t *testing.T) {
	var buf bytes.Buffer
	be := &apperr.BlockedError{
		Message: "some Tasks are not done",
	}
	stub := func(t, id string) (domain.HarnessGetResult, error) {
		return domain.HarnessGetResult{}, errors.New("nope")
	}
	printBlockedErrorTo(&buf, "sprint", "sprint-51", be, stub)

	out := buf.String()
	if !strings.Contains(out, "Sprint BLOCKED - sprint-51") {
		t.Errorf("missing sprint title: %q", out)
	}
}

// TestT434_printBlockedErrorTo_EmptySuggestedActions verifies that the
// "How to resolve:" section is not emitted when SuggestedActions is empty.
func TestT434_printBlockedErrorTo_EmptySuggestedActions(t *testing.T) {
	var buf bytes.Buffer
	be := &apperr.BlockedError{Message: "not done"}
	stub := func(t, id string) (domain.HarnessGetResult, error) {
		return domain.HarnessGetResult{}, errors.New("x")
	}
	printBlockedErrorTo(&buf, "task", "T1", be, stub)

	if strings.Contains(buf.String(), "How to resolve:") {
		t.Errorf("'How to resolve:' section emitted with empty SuggestedActions")
	}
}
