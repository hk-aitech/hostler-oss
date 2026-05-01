// Package cmd - emit a stdout JSON payload when task complete returns BLOCKED.
//
// Previous behavior: when `hstl task complete --with-ceremony` was BLOCKED
// by strict result_check policy or the Harness Gate, `printBlockedError`
// emitted a banner to stderr and `os.Exit(exitBlocked)` terminated. In JSON
// mode (`-o json`) `output.Stderr()` returns `io.Discard`, suppressing
// stderr; stdout was empty too, leaving exit 3 silent and forcing autonomous
// AI runs into a blind flight without a BLOCK cause.
//
// This file ports the sprint-complete countermeasure (`printSprintBlockedJSON`)
// to task complete. `printTaskBlockedJSON` only emits the BLOCKED payload to
// stdout when JSON mode is active (in console/text mode the original stderr
// banner is still effective, so this is a no-op). The caller continues to
// exit with code 3.
package cmd

import (
	"time"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/brand"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/output"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/apperr"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/ceremony"
)

// taskBlockedPayload is the stdout JSON shape for a BLOCKED `hstl task
// complete`. Conforms to the CLI Output Contract
// (docs/08-references/standards/cli-output-contract.md, sec. 3) requirement
// that "every terminal state carries a JSON payload."
//
// reason values (derived from BlockedError.Extra["reason"] or Category):
//   - "task_result_unverified": strict result_check missing_files
//   - "hotfix_rollback_missing": hotfix rollback section verification failed
//   - "harness_unchecked": Harness Gate has unchecked items (default)
type taskBlockedPayload struct {
	Status              string               `json:"status"`
	TaskID              string               `json:"task_id"`
	Reason              string               `json:"reason"`
	Message             string               `json:"message"`
	VerificationResult  map[string]any       `json:"verification_result,omitempty"`
	UncheckedItems      []blockedItemView    `json:"unchecked_items,omitempty"`
	SuggestedActions    []string             `json:"suggested_actions,omitempty"`
	SuggestedNextAction string               `json:"suggested_next_action,omitempty"`
	Ceremony            *blockedCeremonyView `json:"ceremony,omitempty"`
	Timestamp           string               `json:"timestamp"`
}

// blockedItemView is the JSON representation of an unchecked Harness item;
// a cmd-layer projection of apperr.UncheckedItem.
type blockedItemView struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Required bool   `json:"required"`
}

// blockedCeremonyView is the ceremony subset emitted on BLOCKED.
// We do not emit the full ceremony because BLOCKED means "not yet complete":
// Harness Gate status is meaningless in that state, and only ChangedFiles
// and ResultSection.MissingFiles are immediately useful.
type blockedCeremonyView struct {
	ChangedFiles         []string `json:"changed_files,omitempty"`
	ResultSectionMissing []string `json:"result_section_missing_files,omitempty"`
}

// printTaskBlockedJSON emits the payload on stdout in JSON mode when task
// complete returns BLOCKED. In console/text mode, the existing stderr banner
// from `printBlockedError` is the user-visible output, so this is a no-op
// (printing the Go default dump on stdout would just be noise).
//
// The caller still invokes `os.Exit(exitBlocked)` after this function.
//
// be: apperr.BlockedError (assumed non-nil)
// cer: ceremony collected when --with-ceremony was set (may be nil)
func printTaskBlockedJSON(taskID string, be *apperr.BlockedError, cer *ceremony.TaskCompleteCeremony) {
	if !output.IsJSONMode() {
		return
	}
	payload := buildTaskBlockedPayload(taskID, be, cer, time.Now().UTC())
	Out.Print(payload)
}

// buildTaskBlockedPayload is the pure-function portion of
// printTaskBlockedJSON. Split out so unit tests can inject the time and
// verify deterministically (per the dependency-injection guidance).
func buildTaskBlockedPayload(
	taskID string,
	be *apperr.BlockedError,
	cer *ceremony.TaskCompleteCeremony,
	now time.Time,
) taskBlockedPayload {
	payload := taskBlockedPayload{
		Status:           "blocked",
		TaskID:           taskID,
		Reason:           deriveBlockedReason(be),
		Message:          be.Error(),
		SuggestedActions: be.SuggestedActions,
		Timestamp:        now.Format(time.RFC3339),
	}

	// verification_result - strict result_check specific fields.
	payload.VerificationResult = extractVerificationResult(be)

	// unchecked_items - Harness Gate BLOCKED only.
	if len(be.UncheckedItems) > 0 {
		items := make([]blockedItemView, 0, len(be.UncheckedItems))
		for _, u := range be.UncheckedItems {
			items = append(items, blockedItemView{ID: u.ID, Name: u.Name, Required: u.Required})
		}
		payload.UncheckedItems = items
	}

	payload.SuggestedNextAction = deriveSuggestedNextAction(taskID, be)

	// ceremony subset - only the parts immediately useful while BLOCKED.
	if cer != nil {
		view := &blockedCeremonyView{ChangedFiles: cer.ChangedFiles}
		if cer.ResultSection != nil {
			view.ResultSectionMissing = cer.ResultSection.MissingFiles
		}
		if len(view.ChangedFiles) > 0 || len(view.ResultSectionMissing) > 0 {
			payload.Ceremony = view
		}
	}

	return payload
}

// deriveBlockedReason picks a reason string from BlockedError.Extra or
// Category. Extra["reason"] takes precedence, then Category, then the
// default "blocked".
func deriveBlockedReason(be *apperr.BlockedError) string {
	if be.Extra != nil {
		if r, ok := be.Extra["reason"].(string); ok && r != "" {
			return r
		}
	}
	if cat := be.Category(); cat != "" {
		return cat
	}
	return "blocked"
}

// extractVerificationResult reconstructs the verification_result fields
// populated by the strict result_check BLOCKED path from BlockedError.Extra.
// Only present values among missing_files / policy / recovery_hint /
// section_titles_used / checked_files are included (an empty map returns nil
// so JSON omits the field).
func extractVerificationResult(be *apperr.BlockedError) map[string]any {
	if be.Extra == nil {
		return nil
	}
	verificationKeys := []string{
		"missing_files",
		"policy",
		"recovery_hint",
		"section_titles_used",
		"checked_files",
		"available_item_ids",
		"template_key",
	}
	result := map[string]any{}
	for _, k := range verificationKeys {
		if v, ok := be.Extra[k]; ok {
			result[k] = v
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// deriveSuggestedNextAction builds an example recovery command from the
// first unchecked item, matching the printSprintBlockedJSON pattern.
func deriveSuggestedNextAction(taskID string, be *apperr.BlockedError) string {
	if len(be.UncheckedItems) > 0 {
		first := be.UncheckedItems[0]
		return brand.ShortName + " harness check task " + taskID + " " + first.ID + " --evidence \"...\""
	}
	// result_check / hotfix paths surface the first SuggestedActions entry as-is.
	if len(be.SuggestedActions) > 0 {
		return be.SuggestedActions[0]
	}
	return ""
}
