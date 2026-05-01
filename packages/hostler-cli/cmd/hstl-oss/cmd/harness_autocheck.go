// Package cmd - the harness auto-check subcommand.
//
// In Go projects, runs the deterministic harness items
// (build_passed / tests_passed / lint_passed) through the Go toolchain and
// auto-checks them on success. Items that need human judgement
// (criteria_checked / reproduction / root_cause / code_review /
// deploy_verified) are left untouched (preserves Harness Gate safety).
//
// Intended usage:
//
//	hstl harness auto-check task T100
//	hstl harness check task T100 criteria_checked --evidence "..."
//	hstl harness check task T100 reproduction --evidence "..."
//	hstl harness check task T100 root_cause --evidence "..."
//	hstl task complete T100 --with-ceremony
//
// Reduces the 5-6 manual harness check calls per Task down to about three.
package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/app"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/brand"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/domain"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/output"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/apperr"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/db"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/store"
)

// autoCheckGoItems lists harness item IDs auto-checked when go build/test/vet succeeds.
var autoCheckGoItems = map[string]struct {
	command []string
	name    string
}{
	"build_passed": {command: []string{"go", "build", "./..."}, name: "go build ./..."},
	"tests_passed": {command: []string{"go", "test", "./..."}, name: "go test ./..."},
	"lint_passed":  {command: []string{"go", "vet", "./..."}, name: "go vet ./..."},
}

// Non-Go items that admit deterministic auto-verification.
// context_acknowledged is purely deterministic - the work_ticket <-> hash
// round-trip can be put on the auto-check whitelist without compromising
// Harness Gate safety (per KB A01). Human-judgement items
// (criteria_checked / code_review / reproduction / root_cause /
// deploy_verified) remain manual.
var autoCheckDeterministicItems = map[string]string{
	"context_acknowledged": "context --ticket <WT> round-trip",
}

// detectGoModuleDir searches for the directory containing go.mod under the
// project root. Priority: root/go.mod -> root/cli/go.mod ->
// root/packages/hostler-cli/go.mod (monorepo layout).
func detectGoModuleDir(root string) string {
	for _, sub := range []string{".", "cli", "packages/hostler-cli"} {
		if _, err := os.Stat(filepath.Join(root, sub, "go.mod")); err == nil {
			return filepath.Join(root, sub)
		}
	}
	return ""
}

var harnessAutoCheckCmd = &cobra.Command{
	Use:   "auto-check <entity-type> <entity-id>",
	Short: "Auto-verify Go-side deterministic harness items and check them",
	Long: `In Go projects, runs build_passed / tests_passed / lint_passed via
the Go toolchain and, on success, auto-invokes harness check. Items that
need human judgement (criteria_checked / reproduction / root_cause /
code_review / deploy_verified) are left untouched.

See also: harness check (clear a single item), harness check-all (clear all with the same evidence)`,
	Example: fmt.Sprintf(`  %s harness auto-check task T137
  %s harness auto-check task T137 && %s task complete T137 --with-ceremony`,
		brand.ShortName, brand.ShortName, brand.ShortName),
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		entityType := args[0]
		entityID := args[1]

		if entityType != "task" && entityType != "sprint" {
			Out.Error("entity-type must be task or sprint", apperr.CategoryInvalidInput.String(), "")
			os.Exit(exitError)
		}

		mustInitDB()
		defer db.Close()

		summary, err := runAutoCheckGo(entityType, entityID, true)
		if err != nil {
			Out.Error(err.Error(), "", "")
			os.Exit(exitError)
		}

		Out.Success(fmt.Sprintf("%d auto-checked / %d skipped", summary.CheckedCount, summary.SkippedCount), map[string]any{
			"entity_type":   entityType,
			"entity_id":     entityID,
			"checked_count": summary.CheckedCount,
			"skipped_count": summary.SkippedCount,
			"results":       summary.Results,
		})
		return nil
	},
}

// AutoCheckGoSummary aggregates the runAutoCheckGo execution result.
type AutoCheckGoSummary struct {
	CheckedCount int
	SkippedCount int
	Results      map[string]string
}

// runAutoCheckGo applies the Go deterministic auto-check logic to the
// entity harness items. Both the harness_autocheck subcommand and the
// internal task complete call use this function. With verbose=true, prints
// progress to stderr.
func runAutoCheckGo(entityType, entityID string, verbose bool) (AutoCheckGoSummary, error) {
	summary := AutoCheckGoSummary{Results: map[string]string{}}

	raw, err := app.HarnessGet(entityType, entityID)
	if err != nil {
		return summary, err
	}
	state, ok := raw.(*domain.HarnessGetResult)
	if !ok {
		return summary, fmt.Errorf("harness_get response type mismatch")
	}

	root := app.ProjectRoot()
	goDir := detectGoModuleDir(root)

	for _, item := range state.Items {
		if !item.Required || item.Done {
			continue
		}
		// context_acknowledged deterministic auto-processing (task only).
		if _, ok := autoCheckDeterministicItems[item.ID]; ok && entityType == "task" {
			if item.ID == "context_acknowledged" {
				evidence, err := autoCheckContextAck(entityID)
				if err != nil {
					summary.Results[item.ID] = "skipped: " + err.Error()
					summary.SkippedCount++
					if verbose {
						fmt.Fprintf(output.Stderr(), "  [warn] %s - %s\n", item.ID, err.Error())
					}
					continue
				}
				if _, checkErr := app.HarnessCheck(entityType, entityID, item.ID, evidence, "cli"); checkErr != nil {
					summary.Results[item.ID] = fmt.Sprintf("check failed: %v", checkErr)
					continue
				}
				summary.Results[item.ID] = "auto-checked: context-ack round-trip"
				summary.CheckedCount++
				if verbose {
					fmt.Fprintf(output.Stderr(), "  [OK] %s - context --ticket round-trip\n", item.ID)
				}
				continue
			}
		}
		cfg, autoEligible := autoCheckGoItems[item.ID]
		if !autoEligible {
			summary.Results[item.ID] = "skipped: manual required"
			summary.SkippedCount++
			continue
		}
		if goDir == "" {
			summary.Results[item.ID] = "skipped: go.mod not found"
			summary.SkippedCount++
			continue
		}
		runCmd := exec.Command(cfg.command[0], cfg.command[1:]...)
		runCmd.Dir = goDir
		out, runErr := runCmd.CombinedOutput()
		if runErr != nil {
			summary.Results[item.ID] = fmt.Sprintf("FAILED: %s\n%s", runErr, trimOutput(string(out), autoCheckOutputTrimMax))
			if verbose {
				fmt.Fprintf(output.Stderr(), "  [X] %s - %s failed\n", item.ID, cfg.name)
			}
			continue
		}
		evidence := fmt.Sprintf("auto-check: %s PASS (dir=%s)", cfg.name, goDir)
		if _, checkErr := app.HarnessCheck(entityType, entityID, item.ID, evidence, "cli"); checkErr != nil {
			summary.Results[item.ID] = fmt.Sprintf("check failed: %v", checkErr)
			continue
		}
		summary.Results[item.ID] = "auto-checked: " + cfg.name
		summary.CheckedCount++
		if verbose {
			fmt.Fprintf(output.Stderr(), "  [OK] %s - %s\n", item.ID, cfg.name)
		}
	}
	return summary, nil
}

// autoCheckOutputTrimMax is the maximum number of bytes preserved when trimming command output.
const autoCheckOutputTrimMax = 500

// autoCheckContextAck reads the Task work_ticket, collects the context,
// computes the canonical JSON SHA256, records it in the
// context_acknowledgments table, and returns the evidence string
// "sha256:<hex>". That evidence deterministically passes the
// verifyContextAckEvidence check inside harness.HarnessCheck.
func autoCheckContextAck(taskID string) (string, error) {
	gs := store.Get()
	if gs == nil {
		return "", fmt.Errorf("DB not initialized")
	}
	ticket, err := gs.GetTaskWorkTicket(taskID)
	if err != nil {
		return "", fmt.Errorf("failed to look up work_ticket: %w", err)
	}
	if ticket == "" {
		return "", fmt.Errorf("work_ticket missing - run '%s task start %s' first", brand.ShortName, taskID)
	}
	ctx := collectContext()
	hash, size, err := computeContextHash(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to compute hash: %w", err)
	}
	if ierr := gs.InsertContextAck(ticket, hash, size); ierr != nil {
		return "", fmt.Errorf("failed to store ack: %w", ierr)
	}
	return "sha256:" + hash, nil
}

func trimOutput(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "…(truncated)"
}

func init() {
	harnessCmd.AddCommand(harnessAutoCheckCmd)
}
