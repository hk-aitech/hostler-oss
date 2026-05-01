// trac: HAR-CM003
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/audit"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/ceremony"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/fileutil"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/store"
)

// ceremonyCmd is the root of the ceremony subcommand.
var ceremonyCmd = &cobra.Command{
	Use:   "ceremony",
	Short: "Helper commands for ceremonies (sprint:start/complete, task:start/complete)",
	Long:  "Auxiliary verification commands for the Sprint and Task ceremony stages, including verify-registration.",
}

var (
	ceremonyVerifyKBIDs    string
	ceremonyVerifySprintID string
	ceremonyVerifyTaskIDs  string
)

// ceremonyVerifyRegistrationCmd
//
// Called right after Phase 6/9 KB/Task registration in the sprint:complete
// ceremony. It read-verifies that each registered ID actually exists as a DB
// row. A single NOT_FOUND triggers BLOCK + an audit
// ceremony.verification.failed entry + exit code 1. When everything passes,
// records ceremony.verification.passed + exit code 0.
//
// Empty input is treated as zero items verified, i.e. passed.
var ceremonyVerifyRegistrationCmd = &cobra.Command{
	Use:   "verify-registration",
	Short: "Read-verify the IDs registered in Phase 6/9",
	Long: `Read-verifies that each KB / Task ID registered in Phase 6/9 exists as a real DB row.

Follows up KB `+"`mistakes/governance.md`"+` M003 - prevents the recurrence
of "registration was reported but no DB row exists" cases.

Example:
  hstl ceremony verify-registration --sprint-id sprint-69 \
    --kb-ids M001,A001 --task-ids T100,T101

Empty input is treated as zero items verified = passed.`,
	RunE: runCeremonyVerifyRegistration,
}

// dbRegistrationLookup is the hostler DB implementation of RegistrationLookup.
type dbRegistrationLookup struct {
	projectRoot string
}

func (l *dbRegistrationLookup) KBExists(id string) (bool, error) {
	// KB cards live as .md files under docs/07-knowledge/ and embed the ID in
	// their body. We check INDEX.md first and grep the card files second to
	// minimize false negatives.
	knowledgeRoot := filepath.Join(l.projectRoot, "docs", "07-knowledge")
	// Word-boundary check so that, e.g., M001 does not match M001x.
	idMarker := id // simple substring; card bodies usually look like "## M001" or "KB M001 ..."
	found := false
	walkErr := filepath.Walk(knowledgeRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".md") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		if strings.Contains(string(data), idMarker) {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	if walkErr != nil && !os.IsNotExist(walkErr) {
		return false, walkErr
	}
	return found, nil
}

func (l *dbRegistrationLookup) TaskExists(id string) (bool, error) {
	// Direct query against the tasks table in the hostler DB.
	gs := store.Get()
	if gs == nil {
		return false, fmt.Errorf("graph store not initialized")
	}
	_, err := gs.GetTaskFilePath(id)
	if err != nil {
		// NOT_FOUND -> false; propagate other errors.
		if strings.Contains(err.Error(), "NOT_FOUND") || strings.Contains(err.Error(), "not found") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func runCeremonyVerifyRegistration(cmd *cobra.Command, _ []string) error {
	if err := initDB(); err != nil {
		return fmt.Errorf("DB initialization failed: %w", err)
	}

	kbIDs := splitCSV(ceremonyVerifyKBIDs)
	taskIDs := splitCSV(ceremonyVerifyTaskIDs)

	root := fileutil.GetProjectRoot()
	if root == "" {
		var err error
		root, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("cwd: %w", err)
		}
	}

	lookup := &dbRegistrationLookup{projectRoot: root}
	res, err := ceremony.VerifyRegistration(kbIDs, taskIDs, lookup)
	if err != nil {
		return fmt.Errorf("verification lookup failed: %w", err)
	}

	// Record an audit event.
	eventType := "ceremony.verification.passed"
	if !res.Passed {
		eventType = "ceremony.verification.failed"
	}
	details := map[string]any{
		"checked_kbs":   res.CheckedKBs,
		"checked_tasks": res.CheckedTasks,
		"missing_kbs":   res.MissingKBs,
		"missing_tasks": res.MissingTasks,
	}
	entityID := ceremonyVerifySprintID
	if entityID == "" {
		entityID = "unknown"
	}
	_ = audit.LogEvent(eventType, "sprint", entityID, "claude", details, "")

	// JSON output.
	envelope := map[string]any{
		"status":         map[bool]string{true: "ok", false: "blocked"}[res.Passed],
		"sprint_id":      ceremonyVerifySprintID,
		"verification":   res,
		"audit_event":    eventType,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(envelope); err != nil {
		return err
	}

	if !res.Passed {
		os.Exit(1)
	}
	return nil
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func init() {
	ceremonyVerifyRegistrationCmd.Flags().StringVar(&ceremonyVerifyKBIDs, "kb-ids", "", "KB IDs to verify (comma-separated, e.g., M001,A001)")
	ceremonyVerifyRegistrationCmd.Flags().StringVar(&ceremonyVerifyTaskIDs, "task-ids", "", "Task IDs to verify (comma-separated, e.g., T100,T101)")
	ceremonyVerifyRegistrationCmd.Flags().StringVar(&ceremonyVerifySprintID, "sprint-id", "", "audit event entity_id (sprint-XX)")
	ceremonyCmd.AddCommand(ceremonyVerifyRegistrationCmd)
	rootCmd.AddCommand(ceremonyCmd)
}
