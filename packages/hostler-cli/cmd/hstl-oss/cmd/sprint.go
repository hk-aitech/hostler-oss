// trac: HAR-CM001,HAR-CM002,HAR-CM003,HAR-CM004,HAR-CM005,HAR-CM006
// trac: HAR-CM007,TRC-CM007,TRC-CM008,TRC-QR005,WRK-QR006,WRK-QR007,WRK-QR008
package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/app"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/brand"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/domain"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/output"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/apperr"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/ceremony"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/config"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/db"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/envalias"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/sprint"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/template"
)

// ---------------------------------------------------------------------------
// Flag variables
// ---------------------------------------------------------------------------

var (
	sprintStatusFilter         string
	sprintCreateID             string
	sprintCreateTitle          string
	sprintCreateGoal           string
	sprintCreateAutoVerify     bool
	sprintCreateWithCeremony   bool
	sprintUpdateTitle          string
	sprintUpdateGoal           string
	sprintUpdateStatus         string
	sprintUpdateFolderPath     string
	sprintUpdateForce          bool
	sprintCreateForce          bool
	sprintCreateRequireReview  bool // goal scope heuristic BLOCK
	sprintReconcileDryRun      bool
	sprintStartWithCeremony    bool
	sprintCompleteWithCeremony bool
	// --dry-run flags
	sprintStartDryRun    bool
	sprintCompleteDryRun bool
	// sprint start --force - bypass the Sprint Tier BLOCK
	sprintStartForce bool
	// sprint complete --lite - opt in to auto-stamp when KB+Task candidates are empty
	sprintCompleteLite bool
	// --interactive - Phase 5 retro TTY capture
	sprintCompleteInteractive bool
)

// ---------------------------------------------------------------------------
// Top-level sprint command
// ---------------------------------------------------------------------------

var sprintCmd = &cobra.Command{
	Use:   "sprint",
	Short: "Manage Sprints",
	Long:  "List, create, start, complete, inspect progress, and update fields of Sprints.",
}

// ---------------------------------------------------------------------------
// sprint list
// ---------------------------------------------------------------------------

var (
	sprintListSince string
	sprintListUntil string
	sprintListLast  int
	sprintListLimit int
)

var sprintListCmd = &cobra.Command{
	Use:   "list",
	Short: "List Sprints",
	Long: `List Sprints from the DB. Filter flags can be combined.

Filters:
  --status             omitted: excludes discarded (default)
                       active|completed|backlog|discarded: that status only
                       all: everything (including discarded)
  --since / --until    ISO-8601 date (uses completed_at, falls back to started_at)
  --last N             most recent N (forces time-desc sort)
  --limit N            truncate (sort-independent)

When --last and --limit are both specified, --last wins.
Semantic SSOT: docs/08-references/standards/cli-filter-schema.md.`,
	Example: brand.Examplef(
		`sprint list --status active`,
		`sprint list --status completed --last 3`,
		`sprint list --since 2026-04-01`,
		`sprint list -o json | jq '.sprints[] | select(.status == "completed")'`,
	),
	RunE: func(cmd *cobra.Command, args []string) error {
		mustInitDB()
		defer db.Close()

		rawSprints, err := app.SprintList(sprintStatusFilter)
		if err != nil {
			Out.Error("failed to list Sprints", apperr.CategoryInvalidState.String(), err.Error())
			os.Exit(exitError)
		}
		sprints, _ := rawSprints.([]domain.SprintRecord)

		// Client-side filter + sort + truncate. Sprint counts are small so we
		// do this client-side (no DB-level optimization needed, unlike tasks).
		sprints = applyT237Filters(sprints, sprintListSince, sprintListUntil, sprintListLast, sprintListLimit)

		if outputFormat == "json" {
			Out.Print(map[string]any{
				"sprints": sprints,
				"count":   len(sprints),
			})
			return nil
		}

		headers := []string{"ID", "Title", "Status", "Started", "Completed"}
		rows := make([][]string, 0, len(sprints))
		for _, s := range sprints {
			started := s.StartedAt
			if started == "" {
				started = "-"
			}
			completed := s.CompletedAt
			if completed == "" {
				completed = "-"
			}
			rows = append(rows, []string{s.SprintID, s.Title, s.Status, started, completed})
		}

		Out.Table(headers, rows)
		return nil
	},
}

// sprintSortKey returns the time string used as the recency key for a Sprint.
// SSOT: completed_at takes precedence, falling back to started_at.
func sprintSortKey(s domain.SprintRecord) string {
	if s.CompletedAt != "" {
		return s.CompletedAt
	}
	return s.StartedAt
}

// applyT237Filters applies since/until/last/limit filters to a sprint list.
// Semantic SSOT: docs/08-references/standards/cli-filter-schema.md.
func applyT237Filters(sprints []domain.SprintRecord, since, until string, last, limit int) []domain.SprintRecord {
	// since/until filtering (completed_at preferred, started_at fallback).
	if since != "" || until != "" {
		out := sprints[:0:0]
		for _, s := range sprints {
			key := sprintSortKey(s)
			if since != "" && (key == "" || key < since) {
				continue
			}
			if until != "" && (key == "" || key > until) {
				continue
			}
			out = append(out, s)
		}
		sprints = out
	}

	// --last (forces time-desc) and --limit (truncate by existing order) are separate.
	if last > 0 {
		// Sort time-desc and take the top N.
		// Bubble sort keeps things simple; the data is small.
		sorted := append([]domain.SprintRecord(nil), sprints...)
		for i := 0; i < len(sorted); i++ {
			for j := i + 1; j < len(sorted); j++ {
				if sprintSortKey(sorted[i]) < sprintSortKey(sorted[j]) {
					sorted[i], sorted[j] = sorted[j], sorted[i]
				}
			}
		}
		if len(sorted) > last {
			sorted = sorted[:last]
		}
		return sorted
	}
	if limit > 0 && len(sprints) > limit {
		sprints = sprints[:limit]
	}
	return sprints
}

// ---------------------------------------------------------------------------
// sprint create
// ---------------------------------------------------------------------------

var sprintCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a Sprint",
	Long: `Create a new Sprint folder + SPRINT.md and register it in the DB.

See also: sprint create -> sprint start -> sprint complete (workflow order)`,
	Example: brand.Examplef(
		`sprint create --id sprint-19 --title "Quality debt cleanup" --goal "Remove N+1 + switch hook"`,
		`sprint create --id sprint-20 --title "Test coverage 80%"`,
	),
	RunE: func(cmd *cobra.Command, args []string) error {
		if sprintCreateID == "" {
			Out.Error("--id is required", apperr.CategoryInvalidInput.String(), "Pass the Sprint ID as --id sprint-XX.")
			os.Exit(exitError)
		}
		if sprintCreateTitle == "" {
			Out.Error("--title is required", apperr.CategoryInvalidInput.String(), "Pass the Sprint title with --title.")
			os.Exit(exitError)
		}

		// With --require-review, run the goal-scope heuristic before DB init.
		// --force overrides. No signals -> pass.
		if sprintCreateRequireReview && !sprintCreateForce {
			signals := sprint.AnalyzeScope(sprintCreateGoal)
			if len(signals) > 0 {
				msg := sprint.ScopeReviewBlocked(signals)
				Out.Error(msg, apperr.CategoryRejected.String(), "Override with --force after user approval, or narrow --goal and retry")
				os.Exit(exitError)
			}
		}

		mustInitDB()
		defer db.Close()

		// Guard: existing DB record with a different title.
		if !sprintCreateForce {
			if guardErr := app.SprintCheckCreateGuard(sprintCreateID, sprintCreateTitle); guardErr != nil {
				Out.Error("Sprint create guard", apperr.CategoryRejected.String(), guardErr.Error())
				os.Exit(exitError)
			}
		}

		// Multi-worktree sprint ID conflict check.
		// Override with --force or HSTL_SPRINT_ID_CONFLICT_CHECK=off.
		if !sprintCreateForce {
			if conflictErr := sprint.CheckMultiWorktreeIDConflict(sprintCreateID); conflictErr != nil {
				Out.Error("Sprint ID conflict", apperr.CategoryRejected.String(), conflictErr.Error())
				os.Exit(exitError)
			}
		}

		rawCreate, err := app.SprintCreate(sprintCreateID, sprintCreateTitle, sprintCreateGoal, nil)
		if err != nil {
			Out.Error("failed to create Sprint", apperr.CategoryInvalidState.String(), err.Error())
			os.Exit(exitError)
		}
		createResult, _ := rawCreate.(*domain.SprintCreateResult)

		// With --auto-verify, automatically create a Dev verification Task
		// and inject the body template.
		var autoVerifyID string
		if sprintCreateAutoVerify {
			rawVerify, verifyErr := app.SprintAppendAutoVerifyTask(sprintCreateID)
			verifyResult, _ := rawVerify.(*domain.CreateResult)
			if verifyErr != nil {
				fmt.Fprintf(output.Stderr(), "warning: failed to create auto-verify Task: %v\n", verifyErr)
			} else if verifyResult != nil {
				autoVerifyID = verifyResult.TaskID
				fmt.Fprintf(output.Stderr(), "auto-verify Task created: %s (body template injected)\n", verifyResult.TaskID)
			}
		}

		// Display size_mode info (info only - actual BLOCK validation runs on
		// sprint start or a separate validate-tier command; at create time
		// task count is 0, so a BLOCK has no meaning).
		// SSOT: docs/08-references/standards/sprint-tier-spec.md
		if cfg, cfgErr := config.LoadProjectConfig(); cfgErr == nil && cfg != nil && cfg.Sprints != nil {
			tier, _ := sprint.ParseTier(cfg.Sprints.SizeMode)
			defaults := sprint.DefaultsFor(tier)
			fmt.Fprintf(output.Stderr(), "info: size_mode=%s (min=%d max=%d capacity<=%dpt) - SSOT: docs/08-references/standards/sprint-tier-spec.md\n",
				tier, defaults.MinTaskCount, defaults.MaxTaskCount, defaults.MaxCapacity)
		}

		// When --with-ceremony is set, attach the ceremony briefing.
		var result any = createResult
		if sprintCreateWithCeremony {
			result = wrapSprintCreateCeremony(createResult, sprintCreateID, sprintCreateTitle, autoVerifyID)
		}

		Out.Success(fmt.Sprintf("Sprint %s created", sprintCreateID), result)
		return nil
	},
}

// sprintCreateCeremonyWrapper is the --with-ceremony response shape.
type sprintCreateCeremonyWrapper struct {
	Inner    any                      `json:"sprint"`
	Ceremony sprintCreateCeremonyInfo `json:"ceremony"`
}

// loadConfigCompleteMode reads the sprints.complete_mode field from
// .hostler/project-config.yaml. Returns the empty string when missing
// (callers fall back to "confirm").
func loadConfigCompleteMode() string {
	cfg, err := config.LoadProjectConfig()
	if err != nil || cfg == nil || cfg.Sprints == nil {
		return ""
	}
	return cfg.Sprints.CompleteMode
}

type sprintCreateCeremonyInfo struct {
	Briefing     string   `json:"briefing"`
	Reminders    []string `json:"reminders"`
	AutoVerifyID string   `json:"auto_verify_task_id,omitempty"`
	NextActions  []string `json:"next_actions"`
}

// wrapSprintCreateCeremony attaches a ceremony briefing to the sprint create result.
func wrapSprintCreateCeremony(result any, sprintID, title, autoVerifyID string) any {
	reminders := []string{
		fmt.Sprintf("Task assignment - run `%s task assign` to attach existing backlog Tasks to this Sprint", brand.ShortName),
		"Automatic side effects: works/sprints/active/CURRENT-FOCUS.md is updated when sprint start runs",
		"The 10-Phase Cascade is mandatory before sprint:complete (doc/code/work-audit, retro, learned, Dev verification, etc.)",
	}
	nextActions := []string{
		fmt.Sprintf("%s task assign <ids> --sprint %s", brand.ShortName, sprintID),
		fmt.Sprintf("%s sprint start %s", brand.ShortName, sprintID),
	}
	briefing := fmt.Sprintf(
		"Sprint %s (%q) created. Assign Tasks and run sprint start to proceed.",
		sprintID, title,
	)
	if autoVerifyID != "" {
		briefing += fmt.Sprintf(" Dev verification Task %s created automatically.", autoVerifyID)
	}
	return &sprintCreateCeremonyWrapper{
		Inner: result,
		Ceremony: sprintCreateCeremonyInfo{
			Briefing:     briefing,
			Reminders:    reminders,
			AutoVerifyID: autoVerifyID,
			NextActions:  nextActions,
		},
	}
}

// ---------------------------------------------------------------------------
// sprint start
// ---------------------------------------------------------------------------

var sprintStartCmd = &cobra.Command{
	Use:   "start <id>",
	Short: "Start a Sprint",
	Long: `Transition the Sprint to active and move it from backlog -> active.
With --with-ceremony, the response includes briefing, design readiness, and reminder data.

Includes the size_mode BLOCK check. When the Sprint Tier limit is violated,
returns BLOCK + recovery_hint. Override with --force or
HSTL_SPRINT_TIER_POLICY=warn|off.

See also: sprint create -> sprint start -> sprint complete (workflow order)`,
	Example: brand.Examplef(
		`sprint start sprint-19`,
		`sprint start sprint-19 -o json --with-ceremony`,
		`sprint start sprint-19 --force  # bypass tier violation`,
	),
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]

		mustInitDB()
		defer db.Close()

		if sprintStartDryRun {
			Out.Print(buildSprintStartDryRun(id))
			return nil
		}

		// Sprint Tier BLOCK check.
		// SSOT: docs/08-references/standards/sprint-tier-spec.md, sec. 6
		// Override with --force or env HSTL_SPRINT_TIER_POLICY=warn|off.
		policy := envalias.Lookup("SPRINT_TIER_POLICY")
		if !sprintStartForce && policy != "off" {
			cfg, _ := config.LoadProjectConfig()
			sizeMode := ""
			override := sprint.TierDefaults{}
			if cfg != nil && cfg.Sprints != nil {
				sizeMode = cfg.Sprints.SizeMode
				if cfg.Sprints.SizeOverride != nil {
					if cfg.Sprints.SizeOverride.MinTaskCount != nil {
						override.MinTaskCount = *cfg.Sprints.SizeOverride.MinTaskCount
					}
					if cfg.Sprints.SizeOverride.MaxTaskCount != nil {
						override.MaxTaskCount = *cfg.Sprints.SizeOverride.MaxTaskCount
					}
					if cfg.Sprints.SizeOverride.MaxCapacity != nil {
						override.MaxCapacity = *cfg.Sprints.SizeOverride.MaxCapacity
					}
				}
			}
			tierResult := sprint.RunTierCheck(id, sizeMode, override)
			if tierResult.IsBlocked() {
				if policy == "warn" {
					fmt.Fprintf(output.Stderr(), "warning: Sprint Tier (warn policy): %s\n",
						sprint.FormatTierViolationHint(tierResult))
				} else {
					Out.Error("Sprint Tier violation",
						apperr.CategoryRejected.String(),
						sprint.FormatTierViolationHint(tierResult))
					os.Exit(exitError)
				}
			}
		}

		// --with-ceremony: collect ceremony data before Start (looked up in the backlog state).
		var cer *ceremony.SprintStartCeremony
		if sprintStartWithCeremony {
			cer, _ = app.CeremonyCollectSprintStart(id).(*ceremony.SprintStartCeremony)
		}

		rawStart, err := app.SprintStart(id)
		if err != nil {
			Out.Error(fmt.Sprintf("failed to start Sprint %s", id), "", err.Error())
			os.Exit(exitError)
		}
		result := rawStart.(*domain.SprintStartResult)

		if sprintStartWithCeremony && cer != nil {
			Out.Print(map[string]any{
				"sprint_id":   result.SprintID,
				"status":      result.Status,
				"started_at":  result.StartedAt,
				"folder_path": result.FolderPath,
				"ceremony":    cer,
			})
		} else {
			Out.Success(fmt.Sprintf("Sprint %s started", id), result)
		}
		return nil
	},
}

// ---------------------------------------------------------------------------
// sprint complete
// ---------------------------------------------------------------------------

var sprintCompleteCmd = &cobra.Command{
	Use:   "complete <id>",
	Short: "Complete a Sprint (status change only)",
	Long: `Transition the Sprint to completed and move it from active -> completed.
With --with-ceremony, the response includes the 10-Phase Harness state and KB candidates.

See also: harness get/check (clear gates), sprint progress (inspect progress)`,
	Example: brand.Examplef(
		`sprint complete sprint-19`,
		`sprint complete sprint-19 -o json --with-ceremony`,
	),
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]

		mustInitDB()
		defer db.Close()

		if sprintCompleteDryRun {
			Out.Print(buildSprintCompleteDryRun(id))
			return nil
		}

		// lite mode recognition stub.
		// Priority: CLI flag --lite > env HSTL_SPRINT_COMPLETE_MODE > config value > "confirm".
		// The actual auto-stamp behavior is a follow-up; this call only enters the stub and prints guidance.
		completeMode := sprint.ResolveCompleteMode(
			sprintCompleteLite,
			envalias.Lookup("SPRINT_COMPLETE_MODE"),
			loadConfigCompleteMode(),
		)
		if completeMode == "lite" {
			fmt.Fprintf(output.Stderr(),
				"warning: sprint complete --lite mode detected - actual auto-stamp behavior is a follow-up. Falling back to confirm behavior (zero regression).\n")
		}


		// Harness Gate verification.
		rawHS, hErr := app.HarnessGet("sprint", id)
		if hErr == nil {
			harnessState := rawHS.(*domain.HarnessGetResult)
			if harnessState.Blocked {
				// stderr diagnostic + stdout JSON.
				// CLI Output Contract: stdout is JSON-only; stderr carries
				// the human-readable diagnostic box. Previously only stderr
				// was emitted, so `hstl sprint complete > out 2>err` left
				// out at 0 bytes and the user could not see the BLOCKED cause.
				printHarnessBlocked("sprint", id, harnessState)
				printSprintBlockedJSON(id, harnessState)
				os.Exit(exitBlocked)
			}
		}

		// --with-ceremony: collect ceremony data before Complete.
		var cer *ceremony.SprintCompleteCeremony
		if sprintCompleteWithCeremony {
			cer, _ = app.CeremonyCollectSprintComplete(id).(*ceremony.SprintCompleteCeremony)
		}

		// With --interactive (MVP), capture missing RETRO.md from the TTY
		// before calling Sprint.Complete to pre-empt the Phase 5 BLOCK.
		// Skips for non-TTY or when the file already exists.
		if sprintCompleteInteractive {
			if rawRec, gErr := app.SprintGet(id); gErr == nil {
				if rec, _ := rawRec.(*domain.SprintRecord); rec != nil && rec.FolderPath != "" {
					if _, rErr := runInteractiveSprintRetro(id, rec.FolderPath); rErr != nil {
						fmt.Fprintf(output.Stderr(), "[--interactive] retro capture failed (continuing anyway): %v\n", rErr)
					}
				}
			}
		}

		rawComplete, err := app.SprintComplete(id)
		if err != nil {
			// Routed through the shared helper so sprint.Complete can later
			// surface BlockedError. Currently it only returns a generic
			// error, but we keep the same pattern as task complete.
			var be *apperr.BlockedError
			if errors.As(err, &be) {
				printBlockedError("sprint", id, be)
				os.Exit(exitBlocked)
			}
			Out.Error(fmt.Sprintf("failed to complete Sprint %s", id), "", err.Error())
			os.Exit(exitError)
		}
		result := rawComplete.(*domain.SprintCompleteResult)

		// Pacing Guard - inject the rest reminder into the ceremony response
		// right after a successful Sprint complete (only when config allows).
		if sprintCompleteWithCeremony && cer != nil && result.Status == "completed" {
			if cfg, cfgErr := config.LoadProjectConfig(); cfgErr == nil && cfg.GetPacingGuardReminderOnSprintComplete() {
				if items, _, loadErr := template.LoadReminderContent("sprint.complete.pacing-guard"); loadErr == nil {
					cer.PacingGuard = items
				}
			}
		}

		if sprintCompleteWithCeremony && cer != nil {
			// side_effects transparency.
			Out.Print(map[string]any{
				"sprint_id":    result.SprintID,
				"status":       result.Status,
				"completed_at": result.CompletedAt,
				"folder_path":  result.FolderPath,
				"side_effects": result.SideEffects,
				"ceremony":     cer,
			})
		} else {
			Out.Success(fmt.Sprintf("Sprint %s completed", id), result)
		}
		return nil
	},
}

// ---------------------------------------------------------------------------
// sprint discard (T424, Sprint-36)
// ---------------------------------------------------------------------------

var (
	sprintDiscardReason      string
	sprintDiscardReturnTasks bool
	sprintDiscardDBOnly      bool // --db-only flag
)

var sprintDiscardCmd = &cobra.Command{
	Use:   "discard <id>",
	Short: "Discard a Sprint (scope duplicated / low progress / stopped)",
	Long: `Officially discard a Sprint.

Transition rules:
  - backlog -> discarded  (explicit user discard)
  - active  -> discarded  (stop in progress)
  - completed -> discarded is rejected (KB O003 - history is immutable)

Side effects:
  - folder move: works/sprints/<cur>/<id>/ -> works/sprints/discarded/<id>/
  - SPRINT.md frontmatter status: discarded
  - audit: sprint.discarded event (from_status, reason, returned_tasks)
  - CURRENT-FOCUS returns to idle (when discarding from active)
  - no HMAC signature (preserves the completed chain's continuity)

reason must be at least 10 characters (matches the reopen/delete rule).`,
	Example: brand.Examplef(
		`sprint discard sprint-37 --reason "scope overlapped with sprint-36, absorbed"`,
		`sprint discard sprint-50 --reason "initial design error required full redesign" --return-tasks`,
	),
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]

		if strings.TrimSpace(sprintDiscardReason) == "" {
			Out.Error("--reason is required", apperr.CategoryInvalidInput.String(),
				"Provide a discard reason of at least 10 characters. Example: --reason \"absorbed sprint-37 due to scope overlap\"")
			os.Exit(exitError)
		}

		mustInitDB()
		defer db.Close()

		raw, err := app.SprintDiscard(id, sprintDiscardReason, sprintDiscardReturnTasks, sprintDiscardDBOnly)
		if err != nil {
			var re *apperr.RejectedError
			if errors.As(err, &re) {
				Out.Error(fmt.Sprintf("Sprint %s discard rejected", id), re.Category(), re.RecoveryHint)
				os.Exit(exitError)
			}
			Out.Error(fmt.Sprintf("failed to discard Sprint %s", id), apperr.CategoryInvalidState.String(), err.Error())
			os.Exit(exitError)
		}
		result := raw.(*domain.SprintDiscardResult)

		Out.Success(fmt.Sprintf("Sprint %s discarded (%s -> %s)",
			result.SprintID, result.PreviousStatus, result.Status), result)
		return nil
	},
}

// buildSprintStartDryRun produces the sprint start --dry-run response. It
// reports the current state, attached Task count, and design-readiness warnings.
func buildSprintStartDryRun(sprintID string) map[string]any {
	report := map[string]any{
		"dry_run":   true,
		"sprint_id": sprintID,
		"action":    "start",
		"decision":  "proceed",
	}
	checks := []map[string]any{}
	rawRec, err := app.SprintGet(sprintID)
	rec, _ := rawRec.(*domain.SprintRecord)
	if err != nil {
		checks = append(checks, map[string]any{"check": "sprint_get", "status": "fail", "detail": err.Error()})
		report["decision"] = "blocked"
		report["checks"] = checks
		return report
	}
	checks = append(checks, map[string]any{"check": "current_status", "status": "pass", "detail": rec.Status})
	if rec.Status == "active" || rec.Status == "completed" {
		checks = append(checks, map[string]any{"check": "state_allows_start", "status": "fail", "detail": fmt.Sprintf("cannot start from %s state", rec.Status)})
		report["decision"] = "blocked"
	}
	// Reuse design readiness.
	readiness, _ := app.CeremonyCollectSprintStart(sprintID).(*ceremony.SprintStartCeremony)
	if readiness != nil {
		report["design_readiness"] = readiness.DesignReadiness
		for _, r := range readiness.DesignReadiness {
			if r.Status == "fail" || r.Status == "warn" {
				if report["decision"] != "blocked" {
					report["decision"] = "warn"
				}
			}
		}
	}
	report["checks"] = checks
	return report
}

// printSprintBlockedJSON writes a JSON payload to stdout when sprint
// complete returns BLOCKED.
//
// printHarnessBlocked writes an ASCII box to stderr only, leaving stdout
// empty - so `hstl sprint complete > out` left the user without the BLOCKED
// cause. Per the CLI Output Contract (stdout = JSON payload only), we also
// emit the JSON so it can be parsed with jq.
//
// The stderr human diagnostic is preserved (printHarnessBlocked is still
// invoked first). exit code 3 (exitBlocked) is unchanged.
//
// Example output:
//
//	{
//	  "status": "blocked",
//	  "sprint_id": "sprint-28",
//	  "missing_phases": [
//	    {"id": "phase1_doc_review", "name": "Phase 1: doc-review"},
//	    {"id": "phase5_retro", "name": "Phase 5: retro (KPT)"}
//	  ],
//	  "suggested_next_action": "hstl harness check sprint sprint-28 phase1_doc_review --evidence \"...\""
//	}
func printSprintBlockedJSON(sprintID string, state *domain.HarnessGetResult) {
	missing := []map[string]string{}
	for _, item := range state.Items {
		if item.Required && !item.Done {
			missing = append(missing, map[string]string{
				"id":   item.ID,
				"name": item.Name,
			})
		}
	}
	suggested := ""
	if len(missing) > 0 {
		suggested = fmt.Sprintf(
			"%s harness check sprint %s %s --evidence \"...\"",
			brand.ShortName, sprintID, missing[0]["id"],
		)
	}
	Out.Print(map[string]any{
		"status":                "blocked",
		"sprint_id":             sprintID,
		"missing_phases":        missing,
		"suggested_next_action": suggested,
	})
}

// buildSprintCompleteDryRun produces the sprint complete --dry-run response.
// Validates the Harness Gate without performing the completion.
func buildSprintCompleteDryRun(sprintID string) map[string]any {
	report := map[string]any{
		"dry_run":   true,
		"sprint_id": sprintID,
		"action":    "complete",
		"decision":  "proceed",
	}
	checks := []map[string]any{}
	rawState, err := app.HarnessGet("sprint", sprintID)
	var state *domain.HarnessGetResult
	if err == nil {
		state = rawState.(*domain.HarnessGetResult)
		if state.Blocked {
			unchecked := []string{}
			for _, it := range state.Items {
				if it.Required && !it.Done {
					unchecked = append(unchecked, it.ID)
				}
			}
			checks = append(checks, map[string]any{"check": "harness_gate", "status": "fail", "detail": fmt.Sprintf("required unchecked: %v", unchecked)})
			report["decision"] = "blocked"
		} else {
			checks = append(checks, map[string]any{"check": "harness_gate", "status": "pass", "detail": "all required checked"})
		}
	}
	// Aggregate Task done counts.
	sp := sprintID
	raw, lerr := app.TaskList(&sp, nil)
	if lerr == nil {
		if lr, ok := raw.(*domain.ListResult); ok && lr != nil {
			done := 0
			for _, t := range lr.Tasks {
				if t.Status == "done" {
					done++
				}
			}
			checks = append(checks, map[string]any{"check": "task_progress", "status": "info", "detail": fmt.Sprintf("%d/%d done", done, lr.Count)})
			if done != lr.Count && lr.Count > 0 {
				if report["decision"] == "proceed" {
					report["decision"] = "warn"
				}
			}
		}
	}
	report["checks"] = checks
	return report
}

// ---------------------------------------------------------------------------
// sprint progress
// ---------------------------------------------------------------------------

var sprintProgressCmd = &cobra.Command{
	Use:   "progress <id>",
	Short: "Show Sprint progress",
	Long: `Aggregates Task statuses for the Sprint and renders an ASCII progress bar.
With --output json, the done/total/percent values are emitted as structured data.`,
	Example: brand.Examplef(
		`sprint progress sprint-19`,
		`sprint progress sprint-19 -o json`,
	),
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]

		mustInitDB()
		defer db.Close()

		rawProg, err := app.SprintAggregateProgress(id)
		if err != nil {
			Out.Error(fmt.Sprintf("failed to query progress for Sprint %s", id), apperr.CategoryInvalidState.String(), err.Error())
			os.Exit(exitError)
		}
		result := rawProg.(*domain.ProgressResult)

		// Look up the Sprint title (fall back to just the ID).
		title := ""
		if rawRec, recErr := app.SprintGet(id); recErr == nil {
			if rec, ok := rawRec.(*domain.SprintRecord); ok && rec != nil {
				title = rec.Title
			}
		}

		// Render the ASCII progress bar.
		progressLine := renderProgressBar(result.Done, result.Total, result.Percent)

		header := result.SprintID
		if title != "" {
			header = fmt.Sprintf("%s - %s", result.SprintID, title)
		}

		sep := strings.Repeat("=", 49)

		// json mode: serialize ProgressResult directly.
		if outputFormat == "json" {
			Out.Print(result)
			return nil
		}

		// text mode: dashboard output.
		Out.Print(sep)
		Out.Print(fmt.Sprintf("  %s", header))
		Out.Print(sep)
		Out.Print("")
		Out.Print(fmt.Sprintf("  %s  %d/%d  (%d%%)", progressLine, result.Done, result.Total, result.Percent))
		Out.Print("")

		// List Tasks.
		sprintPtr := id
		raw, listErr := app.TaskList(&sprintPtr, nil)
		if listErr == nil {
			if lr, ok := raw.(*domain.ListResult); ok && len(lr.Tasks) > 0 {
				Out.Print("  Tasks:")
				for _, t := range lr.Tasks {
					icon := "[ ]"
					switch t.Status {
					case "done":
						icon = "[x]"
					case "in-progress":
						icon = "[~]"
					}
					Out.Print(fmt.Sprintf("  %s %-6s %-40s %s", icon, t.TaskID, t.Title, t.Status))
				}
				Out.Print("")
			}
		}

		Out.Print(sep)
		return nil
	},
}

// renderProgressBar returns a 20-cell ASCII progress bar using the done/total ratio.
func renderProgressBar(done, total, percent int) string {
	const barWidth = 20
	filled := 0
	if total > 0 {
		filled = done * barWidth / total
	}
	bar := strings.Repeat("#", filled) + strings.Repeat("-", barWidth-filled)
	return "[" + bar + "]"
}

// ---------------------------------------------------------------------------
// sprint update
// ---------------------------------------------------------------------------

var sprintUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update Sprint fields",
	Long: `Update title/goal/status/folder_path in SPRINT.md frontmatter and the DB.
--status and --folder-path are also supported. Status transitions require --force.`,
	Example: brand.Examplef(
		`sprint update sprint-19 --title "Quality debt + CLI UX"`,
		`sprint update sprint-19 --goal "N+1 removal complete"`,
		`sprint update sprint-70 --status backlog --folder-path works/sprints/backlog/sprint-70 --force`,
	),
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]

		if sprintUpdateTitle == "" && sprintUpdateGoal == "" && sprintUpdateStatus == "" && sprintUpdateFolderPath == "" {
			Out.Error("no fields to update", apperr.CategoryInvalidInput.String(),
				"Specify at least one of --title, --goal, --status, --folder-path.")
			os.Exit(exitError)
		}
		if sprintUpdateStatus != "" && !sprintUpdateForce {
			Out.Error("status transition requires --force", apperr.CategoryRejected.String(),
				"Pass --force explicitly to acknowledge bypassing the lifecycle rule.")
			os.Exit(exitError)
		}

		mustInitDB()
		defer db.Close()

		// Drop the prior folder_path alignment requirement: when the DB
		// lookup fails, fall back to the filesystem.
		rawRec, err := app.SprintGet(id)
		rec, _ := rawRec.(*domain.SprintRecord)
		var sprintMDPath string
		if err != nil || rec == nil {
			rawLoc, locErr := app.SprintFindOnDisk(id)
			if locErr != nil {
				Out.Error(fmt.Sprintf("failed to look up Sprint %s", id), apperr.CategoryNotFound.String(), err.Error())
				os.Exit(exitError)
			}
			located := rawLoc.(*domain.LocatedSprint)
			sprintMDPath = located.SprintMD
		} else {
			sprintMDPath = filepath.Join(rec.FolderPath, "SPRINT.md")
			if _, statErr := os.Stat(sprintMDPath); statErr != nil {
				// DB folder_path is stale - rescan the filesystem.
				if rawLoc, locErr := app.SprintFindOnDisk(id); locErr == nil {
					if located, ok := rawLoc.(*domain.LocatedSprint); ok {
						sprintMDPath = located.SprintMD
					}
				}
			}
		}

		if sprintUpdateTitle != "" {
			if err := app.SprintUpdateMDField(sprintMDPath, "title", fmt.Sprintf("%q", sprintUpdateTitle)); err != nil {
				Out.Error("failed to update title", apperr.CategoryInvalidState.String(), err.Error())
				os.Exit(exitError)
			}
		}
		if sprintUpdateGoal != "" {
			if err := app.SprintUpdateMDField(sprintMDPath, "goal", fmt.Sprintf("%q", sprintUpdateGoal)); err != nil {
				Out.Error("failed to update goal", apperr.CategoryInvalidState.String(), err.Error())
				os.Exit(exitError)
			}
		}

		// Update DB fields (title/goal/status/folder_path).
		var titlePtr, goalPtr, statusPtr, folderPtr *string
		if sprintUpdateTitle != "" {
			titlePtr = &sprintUpdateTitle
		}
		if sprintUpdateGoal != "" {
			goalPtr = &sprintUpdateGoal
		}
		if sprintUpdateStatus != "" {
			statusPtr = &sprintUpdateStatus
		}
		if sprintUpdateFolderPath != "" {
			folderPtr = &sprintUpdateFolderPath
		}
		if err := app.SprintUpdateFields(id, titlePtr, goalPtr, statusPtr, folderPtr); err != nil {
			Out.Error("failed to update DB fields", apperr.CategoryInvalidState.String(), err.Error())
			os.Exit(exitError)
		}

		updated := map[string]any{"sprint_id": id}
		if sprintUpdateTitle != "" {
			updated["title"] = sprintUpdateTitle
		}
		if sprintUpdateGoal != "" {
			updated["goal"] = sprintUpdateGoal
		}
		if sprintUpdateStatus != "" {
			updated["status"] = sprintUpdateStatus
		}
		if sprintUpdateFolderPath != "" {
			updated["folder_path"] = sprintUpdateFolderPath
		}

		Out.Success(fmt.Sprintf("Sprint %s updated", id), updated)
		return nil
	},
}

// ---------------------------------------------------------------------------
// sprint reconcile
// ---------------------------------------------------------------------------

var sprintReconcileCmd = &cobra.Command{
	Use:   "reconcile <id>",
	Short: "Re-sync the Sprint DB to match the filesystem",
	Long: `When parallel worktree work leaves the DB record out of sync with the
filesystem, treat the files (SPRINT.md frontmatter + folder location) as SSOT
and re-sync the DB. By default emits the diff without making changes. Use
--dry-run=false to actually apply.`,
	Example: brand.Examplef(
		`sprint reconcile sprint-70`,
		`sprint reconcile sprint-70 --dry-run=false`,
	),
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		mustInitDB()
		defer db.Close()

		result, err := app.SprintReconcile(id, sprintReconcileDryRun)
		if err != nil {
			Out.Error(fmt.Sprintf("failed to reconcile Sprint %s", id), apperr.CategoryInvalidState.String(), err.Error())
			os.Exit(exitError)
		}
		Out.Success(fmt.Sprintf("Sprint %s reconciled", id), result)
		return nil
	},
}

// ---------------------------------------------------------------------------
// init - register commands and flags
// ---------------------------------------------------------------------------

func init() {
	// sprint list flags
	sprintListCmd.Flags().StringVar(&sprintStatusFilter, "status", "", "Status filter (active|completed|backlog|discarded|all). Omitting it excludes discarded by default.")
	sprintListCmd.Flags().StringVar(&sprintListSince, "since", "", "Since this point (ISO-8601, prefers completed_at)")
	sprintListCmd.Flags().StringVar(&sprintListUntil, "until", "", "Until this point (ISO-8601)")
	sprintListCmd.Flags().IntVar(&sprintListLast, "last", 0, "Most recent N (forces time-desc sort)")
	sprintListCmd.Flags().IntVar(&sprintListLimit, "limit", 0, "Maximum number of rows (truncate)")

	// sprint create flags
	sprintCreateCmd.Flags().StringVar(&sprintCreateID, "id", "", "Sprint ID (e.g. sprint-19) [required]")
	sprintCreateCmd.Flags().StringVar(&sprintCreateTitle, "title", "", "Sprint title [required]")
	sprintCreateCmd.Flags().StringVar(&sprintCreateGoal, "goal", "", "Sprint goal")
	sprintCreateCmd.Flags().BoolVar(&sprintCreateAutoVerify, "auto-verify", false, "Auto-create a Dev deployment verification Task")
	sprintCreateCmd.Flags().BoolVar(&sprintCreateWithCeremony, "with-ceremony", false,
		"Include ceremony data (briefing, reminders, next_actions)")
	sprintCreateCmd.Flags().BoolVar(&sprintCreateForce, "force", false,
		"Override the title-mismatch guard against existing DB records")
	sprintCreateCmd.Flags().BoolVar(&sprintCreateRequireReview, "require-review", false,
		"Enable the goal-scope heuristic BLOCK - blocks when persona_boundary/external_integration/new_binary signals are detected; override with --force")

	// sprint start flags
	sprintStartCmd.Flags().BoolVar(&sprintStartDryRun, "dry-run", false,
		"Validate without performing the state transition")
	sprintCompleteCmd.Flags().BoolVar(&sprintCompleteDryRun, "dry-run", false,
		"Validate the Harness Gate without performing complete")
	sprintStartCmd.Flags().BoolVar(&sprintStartWithCeremony, "with-ceremony", false,
		"Include ceremony data (briefing, design_readiness, reminders)")
	sprintStartCmd.Flags().BoolVar(&sprintStartForce, "force", false,
		"Bypass the Sprint Tier BLOCK")

	// sprint complete flags
	sprintCompleteCmd.Flags().BoolVar(&sprintCompleteWithCeremony, "with-ceremony", false,
		"Include ceremony data (harness, kb_candidates)")
	sprintCompleteCmd.Flags().BoolVar(&sprintCompleteInteractive, "interactive", false,
		"MVP: when Phase 5 retro is missing, capture K/P/T from the TTY and write RETRO.md")
	sprintCompleteCmd.Flags().BoolVar(&sprintCompleteLite, "lite", false,
		"Lite-mode auto-stamp when KB+Task candidates are empty (real behavior is follow-up)")

	// sprint update flags
	sprintUpdateCmd.Flags().StringVar(&sprintUpdateTitle, "title", "", "New Sprint title")
	sprintUpdateCmd.Flags().StringVar(&sprintUpdateGoal, "goal", "", "New Sprint goal")
	sprintUpdateCmd.Flags().StringVar(&sprintUpdateStatus, "status", "", "New Sprint status (backlog|active|completed|discarded) - requires --force")
	sprintUpdateCmd.Flags().StringVar(&sprintUpdateFolderPath, "folder-path", "", "New folder_path (relative path)")
	sprintUpdateCmd.Flags().BoolVar(&sprintUpdateForce, "force", false, "Bypass the lifecycle rule (required for status transitions)")

	// sprint reconcile flags
	sprintReconcileCmd.Flags().BoolVar(&sprintReconcileDryRun, "dry-run", true,
		"Print the diff without applying (default true). Use --dry-run=false to actually apply.")

	// Register subcommands
	sprintCmd.AddCommand(sprintListCmd)
	sprintCmd.AddCommand(sprintCreateCmd)
	sprintCmd.AddCommand(sprintStartCmd)
	sprintCmd.AddCommand(sprintCompleteCmd)
	sprintCmd.AddCommand(sprintProgressCmd)
	sprintCmd.AddCommand(sprintUpdateCmd)
	// sprint discard
	sprintDiscardCmd.Flags().StringVar(&sprintDiscardReason, "reason", "", "Discard reason (>= 10 chars, required)")
	sprintDiscardCmd.Flags().BoolVar(&sprintDiscardReturnTasks, "return-tasks", false, "Automatically return unfinished (todo/in-progress) Tasks to backlog")
	sprintDiscardCmd.Flags().BoolVar(&sprintDiscardDBOnly, "db-only", false, "DB-only drift cleanup when the folder is missing - update DB status to discarded without moving folders")
	sprintCmd.AddCommand(sprintDiscardCmd)
	sprintCmd.AddCommand(sprintReconcileCmd)

	// Register on the root command.
	rootCmd.AddCommand(sprintCmd)
}
