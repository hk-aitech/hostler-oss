// trac: HAR-CM008,HAR-CM009,HAR-CM010,HAR-CM011,HAR-CM012,HAR-CM013
// trac: HAR-CM014,HAR-CM015,HAR-CM016,HAR-CM017,TRC-CM009,TRC-CM010,WRK-QR001
// trac: WRK-QR002,WRK-QR005
package cmd

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/app"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/brand"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/domain"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/output"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/ports"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/apperr"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/audit"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/ceremony"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/db"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/envalias"
	pkglog "github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/log"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/task"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/template"
)

// cmdTaskLogger is the pkg/log instance dedicated to cmd/task.go.
var cmdTaskLogger = pkglog.NewDefault()

// --with-ceremony flag
var (
	taskStartWithCeremony    bool
	taskCompleteWithCeremony bool
	// --dry-run flag - validate without performing the state transition.
	taskStartDryRun    bool
	taskCompleteDryRun bool
	// --auto-check-criteria flag - parses the checkboxes in the Task body
	// `## Done Criteria` section. When all are `- [x]`, the
	// criteria_checked harness item is auto-checked; if any `- [ ]` remain,
	// the command BLOCKs.
	taskCompleteAutoCheckCriteria bool
	// --interactive flag - present a [y/n/s] prompt for each unchecked
	// BLOCKED item.
	taskCompleteInteractive bool
	// --auto-check-go flag - before task complete, automatically run and
	// check the Go-side deterministic items (build_passed / tests_passed /
	// lint_passed). Defaults to true. Can also be disabled via the env var
	// HSTL_TASK_AUTOCHECK_GO=off.
	taskCompleteAutoCheckGo bool
)

// initDB / mustInitDB live in cmd/composition.go.
// Composition root pattern - adapter wiring belongs only in cmd/composition.go.

// taskCmd is the root of the task subcommand.
var taskCmd = &cobra.Command{
	Use:   "task",
	Short: "Manage Tasks (list/get/create/start/complete/next/delete/reopen)",
}

// ---------------------------------------------------------------------------
// task list
// ---------------------------------------------------------------------------

var (
	taskListSprint   string
	taskListStatus   string
	taskListPriority string
	taskListType     string
	taskListSince    string
	taskListUntil    string
	taskListLast     int
	taskListLimit    int
)

var taskListCmd = &cobra.Command{
	Use:   "list",
	Short: "List Tasks",
	Long: `List Tasks. Filter flags can be combined.

Filters:
  --sprint / --status   existing compatibility
  --priority p0~p3      priority
  --type <t>            Task type (feature/bugfix/...)
  --since / --until     ISO-8601 date (against tasks.updated_at)
  --last N              time-desc N rows (forces sort order)
  --limit N             truncate (sort-independent)

When --last and --limit are both specified, --last wins.
Semantic SSOT: docs/08-references/standards/cli-filter-schema.md.`,
	Example: brand.Examplef(
		`task list --sprint sprint-19 --status todo`,
		`task list --priority p1 --last 5`,
		`task list --type bugfix --since 2026-04-01`,
		`task list -o json -q | jq '.tasks[] | select(.priority == "p0")'`,
	),
	RunE: func(cmd *cobra.Command, args []string) error {
		mustInitDB()
		defer db.Close()

		// --status alias normalization.
		// Unify variants like "in-progress" / "in_progress" / "InProgress" /
		// " in-progress " to the canonical form so DB queries match
		// consistently. Empty input keeps the "no filter" meaning.
		if taskListStatus != "" {
			canonical, err := task.Canonicalize(taskListStatus)
			if err != nil {
				Out.Error(err.Error(), "INVALID_STATUS",
					"Allowed values: todo / in-progress / done / blocked / reopened")
				os.Exit(exitError)
			}
			taskListStatus = canonical
		}

		// Use the extended path when any of the new filters are set; otherwise legacy.
		usingFilter := taskListPriority != "" || taskListType != "" ||
			taskListSince != "" || taskListUntil != "" ||
			taskListLast > 0 || taskListLimit > 0

		var result any
		var err error
		if usingFilter {
			filter := ports.TaskListFilter{Last: taskListLast, Limit: taskListLimit}
			if taskListSprint != "" {
				filter.Sprint = &taskListSprint
			}
			if taskListStatus != "" {
				filter.Status = &taskListStatus
			}
			if taskListPriority != "" {
				filter.Priority = &taskListPriority
			}
			if taskListType != "" {
				filter.Type = &taskListType
			}
			if taskListSince != "" {
				filter.Since = &taskListSince
			}
			if taskListUntil != "" {
				filter.Until = &taskListUntil
			}
			result, err = app.TaskListFiltered(filter)
		} else {
			var sprintPtr, statusPtr *string
			if taskListSprint != "" {
				sprintPtr = &taskListSprint
			}
			if taskListStatus != "" {
				statusPtr = &taskListStatus
			}
			result, err = app.TaskList(sprintPtr, statusPtr)
		}
		if err != nil {
			Out.Error(err.Error(), "", "")
			os.Exit(exitError)
		}

		if outputFormat == "json" {
			Out.Print(result)
			return nil
		}

		lr, ok := result.(*domain.ListResult)
		if !ok {
			Out.Print(result)
			return nil
		}

		headers := []string{"ID", "Title", "Type", "Status", "Priority", "Sprint"}
		rows := make([][]string, 0, len(lr.Tasks))
		for _, t := range lr.Tasks {
			sprint := ""
			if t.Sprint != nil {
				sprint = fmt.Sprintf("%v", t.Sprint)
			}
			rows = append(rows, []string{
				t.TaskID,
				t.Title,
				t.Type,
				t.Status,
				t.Priority,
				sprint,
			})
		}
		Out.Table(headers, rows)
		return nil
	},
}

// ---------------------------------------------------------------------------
// task get
// ---------------------------------------------------------------------------

var taskGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Look up a single Task",
	Example: brand.Examplef(
		`task get T137`,
		`task get T137 -o json | jq '.depends_on'`,
	),
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mustInitDB()
		defer db.Close()

		result, err := app.TaskGet(args[0])
		if err != nil {
			var nfe *apperr.NotFoundError
			if errors.As(err, &nfe) {
				Out.Error(nfe.Error(), nfe.Category(), nfe.RecoveryHint)
			} else {
				Out.Error(err.Error(), "", "")
			}
			os.Exit(exitError)
		}

		Out.Print(result)
		return nil
	},
}

// ---------------------------------------------------------------------------
// task create
// ---------------------------------------------------------------------------

// Minimum summary length constant - can be overridden via the env var
// HSTL_SUMMARY_MIN_LEN. The max bound was removed in a 2026-04 hotfix:
// summary quality is delegated to the heuristic plus the LLM judge, and the
// length cap conflicted with the WHAT/WHY/SUCCESS three-element requirement.
const defaultMinSummaryLen = 10

// SummaryLenHardMax is the absolute upper bound on env-var overrides for
// getMinSummaryLen. Setting min to an extreme value (e.g. 999999) would mark
// every summary invalid and effectively block Task creation. When the
// override exceeds the bound, fall back to defaultMinSummaryLen and emit a
// warning.
const SummaryLenHardMax = 9999

func getMinSummaryLen() int {
	v := envalias.Lookup("SUMMARY_MIN_LEN")
	if v == "" {
		return defaultMinSummaryLen
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		cmdTaskLogger.Warn("failed to parse HSTL_SUMMARY_MIN_LEN - using default",
			"value", v, "default", defaultMinSummaryLen)
		return defaultMinSummaryLen
	}
	if n <= 0 {
		cmdTaskLogger.Warn("HSTL_SUMMARY_MIN_LEN out of range (n <= 0) - using default",
			"n", n, "default", defaultMinSummaryLen)
		return defaultMinSummaryLen
	}
	if n > SummaryLenHardMax {
		cmdTaskLogger.Warn("HSTL_SUMMARY_MIN_LEN exceeds hard max - using default",
			"n", n, "max", SummaryLenHardMax, "default", defaultMinSummaryLen)
		return defaultMinSummaryLen
	}
	return n
}

var (
	taskCreateTitle                 string
	taskCreateType                  string
	taskCreatePriority              string
	taskCreateEstimate              string
	taskCreateDepends               string
	taskCreateSummary               string
	taskCreateWithCeremony          bool
	taskCreateSkipSummaryValidation bool
	taskCreateAllowDuplicateTitle   bool
	taskCreateDryRun                bool
)

// Detection of keywords that imply Go code changes during task create.
// When a title in the docs/chore type contains Go-related keywords, emit a WARN.
var goCodeKeywords = []string{".go", "func ", "struct ", "cli/pkg/", "cli/cmd/", "interface "}

// goCodeWarnTypes is the set of Task types for which Go keyword detection
// warnings are emitted.
var goCodeWarnTypes = map[string]bool{"docs": true, "chore": true}

// containsGoCodeKeyword returns the first matching keyword and true when the
// title contains a Go-related keyword (case-insensitive).
func containsGoCodeKeyword(title string) (string, bool) {
	lower := strings.ToLower(title)
	for _, kw := range goCodeKeywords {
		if strings.Contains(lower, strings.ToLower(kw)) {
			return kw, true
		}
	}
	return "", false
}

// auditSummaryHeadLimit caps the length (in characters) of the summary
// excerpt stored in audit_log. Too long bloats the audit DB row; too short
// loses failure context.
const auditSummaryHeadLimit = 120

// truncateForAudit builds an audit_log-friendly summary excerpt.
// Preserves UTF-8 rune boundaries and terminates with "..." when the limit is exceeded.
func truncateForAudit(s string) string {
	rs := []rune(s)
	if len(rs) <= auditSummaryHeadLimit {
		return s
	}
	return string(rs[:auditSummaryHeadLimit]) + "..."
}

var taskCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a Task",
	Long: fmt.Sprintf(`Create a new Task and register it in BACKLOG.
With --with-ceremony, the response includes briefing data (body authoring guide,
automatic side effects, next steps).
With --dry-run, only the template rendering is previewed - no DB / file / counter changes.

Task IDs are auto-allocated: the 'task' counter in id_counters is atomically
incremented and returned as "T###". Placeholder filename scanning plus
self-heal ensures no collision with existing IDs even under DB drift. Manual
ID assignment is not supported.

Tasks are always created in BACKLOG. After writing the body, run
'%s task assign <ID> --sprint sprint-NN' separately to attach to a Sprint.
The 3-step ceremony prevents placeholder-body Tasks from being attached to a Sprint.

See also: task create -> (write body) -> task assign -> task start -> task complete`, brand.ShortName),
	Example: brand.Examplef(
		`task create --title "Add API endpoint" --type feature`,
		`task create --title "Fix N+1 query" --type refactor --priority p1 --depends T100,T101`,
		`task create --title "hotfix test" --type hotfix -o json --with-ceremony`,
		`task create --title "validation only" --type chore --summary "..." --dry-run`,
	),
	RunE: func(cmd *cobra.Command, args []string) error {
		// INVALID_STATE / MISSING_SUMMARY / INVALID_SUMMARY_LENGTH all map to
		// argument problems (missing required field, format error) so we
		// unify them under apperr.CategoryInvalidInput.String().
		if taskCreateTitle == "" {
			Out.Error("--title flag is required", apperr.CategoryInvalidInput.String(), "")
			os.Exit(exitError)
		}
		if taskCreateType == "" {
			Out.Error("--type flag is required", apperr.CategoryInvalidInput.String(), "")
			os.Exit(exitError)
		}
		if taskCreateSummary == "" {
			Out.Error("--summary flag is required", apperr.CategoryInvalidInput.String(), "Summarize what / why / success criteria in a single line")
			os.Exit(exitError)
		}

		// Move the three normalize steps **before** the --dry-run branch.
		// An earlier change inserted normalization into the create/update real
		// paths, but the dry-run branch ran first, so previews showed raw
		// input. Fix it so previews also reflect the canonical values.
		//
		// priority normalization plus enum validation.
		if normalized, err := normalizePriority(taskCreatePriority); err != nil {
			Out.Error(err.Error(), apperr.CategoryInvalidInput.String(), "--priority must be one of p0|p1|p2|p3")
			os.Exit(exitError)
		} else {
			taskCreatePriority = normalized
		}

		// type alias normalization plus enum validation.
		if normalized, err := normalizeType(taskCreateType); err != nil {
			Out.Error(err.Error(), apperr.CategoryInvalidInput.String(), "--type must be one of feature|bugfix|docs|refactor|infra|test|chore|spike|hotfix")
			os.Exit(exitError)
		} else {
			taskCreateType = normalized
		}

		// estimate normalization plus enum validation.
		if normalized, err := normalizeEstimate(taskCreateEstimate); err != nil {
			Out.Error(err.Error(), apperr.CategoryInvalidInput.String(), "--estimate must be one of XS|S|M|L|XL")
			os.Exit(exitError)
		} else if normalized != "" {
			taskCreateEstimate = normalized
		}

		// migrate/cutover/rename heuristic.
		// Tasks containing these keywords show a structural underestimation
		// bias. When the keyword appears in title or summary together with
		// estimate=XS, emit a stderr WARN.
		if reason := detectMigrationEstimateRisk(taskCreateTitle, taskCreateSummary, taskCreateEstimate); reason != "" {
			fmt.Fprintf(cmd.ErrOrStderr(),
				"[warn] estimate heuristic - %s. Verification checklist: (1) grep at the code level to identify the writer, (2) inspect recent mtime via find, (3) trace path references in audit logs to re-judge scope.\n",
				reason)
		}

		// --dry-run early branch.
		// Skip the summary heuristic (avoid subagent invocation) and only
		// render the template. No DB init / pending ticket / placeholder
		// scanner calls -> zero counter drift. The branch sits **after** the
		// three normalizers above so previews use canonical values
		// (type=bugfix / priority=p1 / estimate=XS, etc.).
		if taskCreateDryRun {
			var depends []string
			if taskCreateDepends != "" {
				depends = strings.Split(taskCreateDepends, ",")
				for i := range depends {
					depends[i] = strings.TrimSpace(depends[i])
				}
			}
			preview := template.RenderTaskTemplate(
				"T???", taskCreateTitle, taskCreateType, "backlog",
				taskCreatePriority, taskCreateEstimate,
				depends, taskCreateSummary,
			)
			if outputFormat == "json" {
				Out.Print(map[string]any{
					"dry_run": true,
					"preview": preview,
					"meta": map[string]any{
						"title":    taskCreateTitle,
						"type":     taskCreateType,
						"priority": taskCreatePriority,
						"estimate": taskCreateEstimate,
					},
				})
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), preview)
				fmt.Fprintln(cmd.ErrOrStderr(), "[dry-run] no DB / file / counter changes - preview only")
			}
			return nil
		}

		minLen := getMinSummaryLen()
		summaryLen := len([]rune(taskCreateSummary))
		if summaryLen < minLen {
			Out.Error(
				fmt.Sprintf("--summary below minimum length (current %d chars, minimum %d)", summaryLen, minLen),
				apperr.CategoryInvalidInput.String(),
				fmt.Sprintf("Write at least %d characters - cover what / why / success criteria concretely", minLen),
			)
			os.Exit(exitError)
		}

		// Even the summary-validation failure path records a PENDING
		// work_ticket in audit_log so create failures (reject /
		// subprocess_fail) remain traceable. The DB must be initialized
		// before app.AuditAppend can run; defer db.Close() handles cleanup
		// even when we early-exit.
		mustInitDB()
		defer db.Close()

		pendingTicket, ptErr := db.GeneratePendingTicket()
		if ptErr != nil {
			// Extremely rare system error - audit cannot be recorded; keep the original flow.
			fmt.Fprintf(output.Stderr(), "warning: failed to allocate pending work_ticket (audit skipped): %v\n", ptErr)
			pendingTicket = ""
		}
		logCreateFailure := func(reason, summary, details string) {
			if pendingTicket == "" {
				return
			}
			_ = app.AuditAppend("task.create_failed", "task", "", "claude", map[string]any{
				"pending_ticket": pendingTicket,
				"failure_reason": reason,
				"title":          taskCreateTitle,
				"type":           taskCreateType,
				"summary_head":   summary,
				"details":        details,
			}, "")
		}

		// Validate summary quality through a single CLI path.
		//   1) Static heuristic - length/words/banned terms/connectors - fail-closed gate.
		//      On failure, exit immediately and skip the subagent (saves cost).
		//   2) LLM subagent - only invoked when the heuristic passes; calls
		//      claude -p --system-prompt-file agents/summary-judge.md and
		//      reuses the session UUID via --session-id/--resume, so each
		//      call costs roughly $0.005.
		//
		// The command body no longer pre-validates via the Task tool (single
		// CLI path). After the AI writes the summary it invokes the CLI
		// directly; the two stages inside the CLI form the quality gate.
		//
		// Bypasses:
		//   --skip-summary-validation  - skip both layers (CI emergency).
		//   HSTL_SUMMARY_POLICY=off   - disable only the heuristic.
		//   HSTL_SUMMARY_SUBAGENT=off - disable only the subagent (heuristic stays).
		if !taskCreateSkipSummaryValidation {
			rawCfg := app.TaskResolvePreset(envalias.Lookup("SUMMARY_POLICY"))
			heuristicCfg := rawCfg.(task.HeuristicConfig)
			if issues, ok := app.TaskHeuristicCheck(taskCreateSummary, heuristicCfg); !ok {
				Out.Error(
					fmt.Sprintf("summary static heuristic failed (preset=%s). Fix the items below.", heuristicCfg.Preset),
					"INVALID_SUMMARY_HEURISTIC",
					"Adjust strictness with HSTL_SUMMARY_POLICY=moderate|lenient|off, or bypass with --skip-summary-validation (CI exception only)",
				)
				for _, iss := range issues {
					fmt.Fprintf(output.Stderr(), "  X %s\n", iss)
				}
				logCreateFailure("summary_heuristic_reject", truncateForAudit(taskCreateSummary), strings.Join(issues, "; "))
				os.Exit(exitError)
			}

		}
		if taskCreateSkipSummaryValidation {
			fmt.Fprintf(output.Stderr(), "warning: --skip-summary-validation in use - summary two-layer gate is bypassed (CI exception).\n")
			// Audit record - preserves trace evidence of preset heuristic bypass.
			_ = audit.LogEvent("task.create.summary_skipped", "task", "(pre-create)",
				"", map[string]any{"title": taskCreateTitle, "type": taskCreateType}, "")
		}

		// Pass the --allow-duplicate-title flag through an env var so
		// task.Create applies the same bypass rule regardless of the call
		// path (direct API / CLI).
		if taskCreateAllowDuplicateTitle {
			_ = os.Setenv(task.TaskCreateAllowDuplicateEnv, "1")
			fmt.Fprintf(output.Stderr(), "warning: --allow-duplicate-title in use - title duplicate detection is skipped.\n")
		}

		// mustInitDB / defer db.Close are above, before summary validation.

		var dependsOn []string
		if taskCreateDepends != "" {
			for _, d := range strings.Split(taskCreateDepends, ",") {
				d = strings.TrimSpace(d)
				if d != "" {
					dependsOn = append(dependsOn, d)
				}
			}
		}

		result, err := app.TaskCreate(
			taskCreateTitle,
			taskCreateType,
			"", // --sprint deprecated - always create in BACKLOG; attach with task assign after writing the body.
			taskCreatePriority,
			taskCreateEstimate,
			taskCreateSummary,
			dependsOn,
		)
		if err != nil {
			Out.Error(err.Error(), "", "")
			os.Exit(exitError)
		}

		// When --with-ceremony is set, wrap the result with the ceremony field.
		if taskCreateWithCeremony {
			result = wrapTaskCreateCeremony(result)
		}

		Out.Success("Task created", result)

		// For docs/chore types, warn if the title contains Go-related keywords.
		if goCodeWarnTypes[taskCreateType] {
			if kw, ok := containsGoCodeKeyword(taskCreateTitle); ok {
				fmt.Fprintf(output.Stderr(), "warning: title contains Go code keyword %q. Consider --type feature or --type refactor.\n", kw)
			}
		}

		// In text mode, surface the body authoring reminder on stderr explicitly.
		// JSON mode already includes the same items in result.warnings.
		if cr, ok := result.(*domain.CreateResult); ok && len(cr.Warnings) > 0 {
			for _, w := range cr.Warnings {
				fmt.Fprintf(output.Stderr(), "warning: %s\n", w)
			}
		}
		return nil
	},
}

// taskCreateCeremonyWrapper is the --with-ceremony response shape.
type taskCreateCeremonyWrapper struct {
	*domain.CreateResult
	Ceremony taskCreateCeremonyInfo `json:"ceremony"`
}

type taskCreateCeremonyInfo struct {
	Briefing    string   `json:"briefing"`
	Reminders   []string `json:"reminders"`
	NextActions []string `json:"next_actions"`
}

// wrapTaskCreateCeremony attaches a ceremony briefing to the task create result.
func wrapTaskCreateCeremony(result any) any {
	cr, ok := result.(*domain.CreateResult)
	if !ok {
		return result
	}
	return &taskCreateCeremonyWrapper{
		CreateResult: cr,
		Ceremony: taskCreateCeremonyInfo{
			Briefing: fmt.Sprintf(
				"Task %s created. File: %s. Write the body (purpose / requirements / done criteria / type) and then run task start.",
				cr.TaskID, cr.FilePath,
			),
			Reminders: []string{
				"Body authoring is required - running task start while still a placeholder triggers a design-readiness warning",
				"Automatic side effects: works/tasks/BACKLOG.md (and CURRENT-FOCUS.md when an active Sprint exists) are updated automatically",
				"Do not edit BACKLOG.md / CURRENT-FOCUS.md directly with Write/Edit",
			},
			NextActions: []string{
				fmt.Sprintf("%s task start %s", brand.ShortName, cr.TaskID),
			},
		},
	}
}

// ---------------------------------------------------------------------------
// task start
// ---------------------------------------------------------------------------

var taskStartCmd = &cobra.Command{
	Use:   "start <id>",
	Short: "Start a Task (todo -> in-progress)",
	Long: `Transition the Task status from todo to in-progress.
With --with-ceremony, the response includes the Task briefing and reminders.

See also: task create -> task start -> task complete (workflow order)`,
	Example: brand.Examplef(
		`task start T137`,
		`task start T137 -o json --with-ceremony`,
	),
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]
		mustInitDB()
		defer db.Close()

		if taskStartDryRun {
			report := buildTaskStartDryRun(taskID)
			Out.Print(report)
			return nil
		}

		// origin/dev sync check + WARN (graceful skip).
		checkDevSyncWarn()

		result, err := app.TaskStart(taskID)
		if err != nil {
			var nfe *apperr.NotFoundError
			var rje *apperr.RejectedError
			if errors.As(err, &nfe) {
				Out.Error(nfe.Error(), nfe.Category(), nfe.RecoveryHint)
			} else if errors.As(err, &rje) {
				Out.Error(rje.Error(), rje.Category(), "")
			} else {
				Out.Error(err.Error(), "", "")
			}
			os.Exit(exitError)
		}

		if taskStartWithCeremony {
			cer, _ := app.CeremonyCollectTaskStart(taskID).(*ceremony.TaskStartCeremony)
			Out.Print(map[string]any{
				"result":   result,
				"ceremony": cer,
			})
		} else {
			Out.Success("Task started", result)
		}
		return nil
	},
}

// ---------------------------------------------------------------------------
// task complete
// ---------------------------------------------------------------------------

var taskCompleteCmd = &cobra.Command{
	Use:   "complete <id>",
	Short: "Complete a Task (in-progress -> done)",
	Long: `Transition the Task status from in-progress to done. Verifies the Harness Gate;
exits BLOCKED (exit 3) when any required items remain unchecked.
With --with-ceremony, the response includes changed files, Harness Gate, and Result section status.

See also: task start -> task complete, harness check (clear Gate items)`,
	Example: brand.Examplef(
		`task complete T137`,
		`task complete T137 -o json --with-ceremony`,
	),
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]
		mustInitDB()
		defer db.Close()

		// Preview the current git-changed files on stderr before the user
		// writes the Result section. Works without --with-ceremony to reduce
		// BLOCKED false positives. Uses the same source as strict
		// verification (task.GetGitChangedFiles).
		printChangedFilesPreview(taskID)

		if taskCompleteDryRun {
			report := buildTaskCompleteDryRun(taskID)
			Out.Print(report)
			return nil
		}

		// --auto-check-go flag (defaults to on).
		// Auto-runs the Go-side deterministic items
		// (build_passed / tests_passed / lint_passed / context_acknowledged)
		// before task complete to remove the "skip and re-run with manual
		// patches" pattern. Failing items stay unchecked, so the subsequent
		// TaskComplete naturally returns BLOCKED. Disable via the env var
		// HSTL_TASK_AUTOCHECK_GO=off or --auto-check-go=false.
		autoCheckGoEnabled := taskCompleteAutoCheckGo && !strings.EqualFold(envalias.Lookup("TASK_AUTOCHECK_GO"), "off")
		if autoCheckGoEnabled {
			summary, acErr := runAutoCheckGo("task", taskID, true)
			if acErr != nil {
				fmt.Fprintf(output.Stderr(), "[--auto-check-go] auto-check failed to run: %v (manual checks may continue)\n", acErr)
			} else if summary.CheckedCount > 0 || summary.SkippedCount > 0 {
				fmt.Fprintf(output.Stderr(), "[--auto-check-go] %d auto-checked / %d skipped\n", summary.CheckedCount, summary.SkippedCount)
			}
		}

		// --auto-check-criteria flag.
		// Parses the `## Done Criteria` checkboxes in the Task body:
		//   - all checked: auto-check the criteria_checked harness item
		//   - any unchecked: print the list and BLOCK (abort complete)
		//   - section absent: emit one warning, no other effect (proceed)
		if taskCompleteAutoCheckCriteria {
			rawCR, _ := app.HarnessAutoCheckCriteria(taskID)
			cr, _ := rawCR.(domain.CriteriaResult)
			switch {
			case cr.Total == 0:
				fmt.Fprintf(output.Stderr(), "[--auto-check-criteria] Task %s has no `## Done Criteria` section or checkboxes - flag has no effect\n", taskID)
			case !cr.AllChecked:
				Out.Error(
					fmt.Sprintf("done criteria unchecked: %d remaining (%d/%d checked)", len(cr.Unchecked), cr.Checked, cr.Total),
					"BLOCKED",
					"Switch every `- [ ]` to `- [x]` in the Task file's `## Done Criteria` section, then re-run.",
				)
				for i, item := range cr.Unchecked {
					fmt.Fprintf(output.Stderr(), "  [%d] %s\n", i+1, item)
				}
				os.Exit(exitBlocked)
			default:
				// All checked: auto-check criteria_checked.
				evidence := fmt.Sprintf("auto-check-criteria: %d/%d checked", cr.Checked, cr.Total)
				if _, cherr := app.HarnessCheck("task", taskID, "criteria_checked", evidence, "cli-auto"); cherr != nil {
					fmt.Fprintf(output.Stderr(), "[--auto-check-criteria] harness_check failed: %v (continuing complete)\n", cherr)
				} else {
					fmt.Fprintf(output.Stderr(), "[--auto-check-criteria] criteria_checked auto-checked (%d/%d)\n", cr.Checked, cr.Total)
				}
			}
		}

		// --with-ceremony: collect ceremony data before Complete.
		var cer *ceremony.TaskCompleteCeremony
		if taskCompleteWithCeremony {
			cer, _ = app.CeremonyCollectTaskComplete(taskID).(*ceremony.TaskCompleteCeremony)
		}

		result, err := app.TaskComplete(taskID, false)
		if err != nil {
			var be *apperr.BlockedError
			var nfe *apperr.NotFoundError
			var rje *apperr.RejectedError
			if errors.As(err, &be) {
				// With --interactive, walk through unchecked items via prompt and retry.
				if taskCompleteInteractive {
					n, aborted := runInteractiveHarnessResolve("task", taskID, be)
					if !aborted && n > 0 {
						// Retry - succeed when every required item is checked, otherwise BLOCK again.
						result, err = app.TaskComplete(taskID, false)
						if err == nil {
							goto taskCompleteSuccess
						}
						var be2 *apperr.BlockedError
						if errors.As(err, &be2) {
							printBlockedError("task", taskID, be2)
							// Avoid silent exit 3 in JSON mode.
							printTaskBlockedJSON(taskID, be2, cer)
							os.Exit(exitBlocked)
						}
					}
				}
				// Routed through the shared printBlockedError helper.
				printBlockedError("task", taskID, be)
				// In JSON mode, emit the BLOCKED payload to stdout. In
				// console/text mode this is a noop (the printBlockedError
				// stderr banner is the user-visible output).
				printTaskBlockedJSON(taskID, be, cer)
				os.Exit(exitBlocked)
			} else if errors.As(err, &nfe) {
				Out.Error(nfe.Error(), nfe.Category(), nfe.RecoveryHint)
				os.Exit(exitError)
			} else if errors.As(err, &rje) {
				Out.Error(rje.Error(), rje.Category(), "")
				os.Exit(exitError)
			} else {
				Out.Error(err.Error(), "", "")
				os.Exit(exitError)
			}
		}

	taskCompleteSuccess:
		if taskCompleteWithCeremony && cer != nil {
			Out.Print(map[string]any{
				"result":   result,
				"ceremony": cer,
			})
		} else {
			Out.Success("Task complete", result)
		}
		return nil
	},
}

// buildTaskStartDryRun produces the task start --dry-run response. It runs
// the DB state lookup, dependency check, and placeholder body detection
// without performing any state transition. Exit code is always 0
// (informational only).
func buildTaskStartDryRun(taskID string) map[string]any {
	report := map[string]any{
		"dry_run":  true,
		"task_id":  taskID,
		"action":   "start",
		"checks":   []map[string]any{},
		"decision": "proceed",
	}
	checks := []map[string]any{}
	gr, err := app.TaskGet(taskID)
	if err != nil {
		checks = append(checks, map[string]any{"check": "task_get", "status": "fail", "detail": err.Error()})
		report["checks"] = checks
		report["decision"] = "blocked"
		return report
	}
	t, _ := gr.(*domain.GetResult)
	if t == nil {
		checks = append(checks, map[string]any{"check": "task_get", "status": "fail", "detail": "result nil"})
		report["checks"] = checks
		report["decision"] = "blocked"
		return report
	}
	checks = append(checks, map[string]any{"check": "current_status", "status": "pass", "detail": t.Status})
	if t.Status != "todo" {
		checks = append(checks, map[string]any{"check": "transition_todo_to_inprogress", "status": "warn", "detail": fmt.Sprintf("current status %s (not todo)", t.Status)})
		report["decision"] = "warn"
	}
	if app.TaskHasPlaceholderBody(taskID) {
		checks = append(checks, map[string]any{"check": "placeholder_body", "status": "warn", "detail": "body placeholder still present"})
		report["decision"] = "warn"
	}
	for _, dep := range t.DependsOn {
		depRaw, derr := app.TaskGet(dep)
		if derr != nil {
			continue
		}
		if d, ok := depRaw.(*domain.GetResult); ok && d.Status != "done" {
			checks = append(checks, map[string]any{"check": "dependency", "status": "fail", "detail": fmt.Sprintf("%s not done (status=%s)", dep, d.Status)})
			report["decision"] = "blocked"
		}
	}
	report["checks"] = checks
	return report
}

// buildTaskCompleteDryRun produces the task complete --dry-run response.
// Validates the Harness Gate, AutoCheckCriteria, and Result section.
func buildTaskCompleteDryRun(taskID string) map[string]any {
	report := map[string]any{
		"dry_run":  true,
		"task_id":  taskID,
		"action":   "complete",
		"checks":   []map[string]any{},
		"decision": "proceed",
	}
	checks := []map[string]any{}

	rawCriteria, _ := app.HarnessAutoCheckCriteria(taskID)
	criteria, _ := rawCriteria.(domain.CriteriaResult)
	if !criteria.AllChecked {
		checks = append(checks, map[string]any{"check": "completion_criteria", "status": "warn", "detail": fmt.Sprintf("%d/%d checked (%d unchecked)", criteria.Checked, criteria.Total, len(criteria.Unchecked))})
		report["decision"] = "warn"
	} else {
		checks = append(checks, map[string]any{"check": "completion_criteria", "status": "pass", "detail": fmt.Sprintf("%d/%d checked", criteria.Checked, criteria.Total)})
	}

	rawHR, herr := app.HarnessCheckAll("task", taskID)
	hr, _ := rawHR.(domain.HarnessResult)
	if herr == nil {
		if hr.Blocked {
			unchecked := []string{}
			for _, it := range hr.UncheckedRequired {
				unchecked = append(unchecked, it.ID)
			}
			checks = append(checks, map[string]any{"check": "harness_gate", "status": "fail", "detail": fmt.Sprintf("required unchecked: %v", unchecked)})
			report["decision"] = "blocked"
		} else {
			checks = append(checks, map[string]any{"check": "harness_gate", "status": "pass", "detail": "all required checked"})
		}
	}

	changed := app.TaskGetGitChangedFiles()
	checks = append(checks, map[string]any{"check": "git_changed_files", "status": "info", "detail": fmt.Sprintf("%d", len(changed))})

	report["checks"] = checks
	return report
}

// printChangedFilesPreview previews the current git-changed file list on
// stderr at task complete entry. Lets the user see which files are
// staged/modified before writing the Result section, reducing strict
// false-positive recovery. Uses the same source as strict verification
// (task.GetGitChangedFiles) to avoid drift. Always written to stderr so JSON
// stdout stays clean.
func printChangedFilesPreview(taskID string) {
	files := app.TaskGetGitChangedFiles()
	if len(files) == 0 {
		fmt.Fprintf(output.Stderr(), "[%s] changed-files preview: (0) - finish actual work before drafting the Result section.\n", taskID)
		return
	}
	fmt.Fprintf(output.Stderr(), "[%s] changed-files preview (%d) - candidates for the Result section:\n", taskID, len(files))
	for _, f := range files {
		fmt.Fprintf(output.Stderr(), "  - %s\n", f)
	}
}

// ---------------------------------------------------------------------------
// task next
// ---------------------------------------------------------------------------

var taskNextCmd = &cobra.Command{
	Use:   "next",
	Short: "Suggest the next Task to work on",
	Long: `Suggests the highest-priority todo Task whose dependencies are satisfied.

See also: task start (start the suggested Task immediately)`,
	Example: brand.Examplef(
		`task next`,
		`task next -o json | jq '.task_id'`,
	),
	RunE: func(cmd *cobra.Command, args []string) error {
		mustInitDB()
		defer db.Close()

		result, err := app.TaskNext()
		if err != nil {
			Out.Error(err.Error(), "", "")
			os.Exit(exitError)
		}

		Out.Print(result)
		return nil
	},
}

// ---------------------------------------------------------------------------
// task update
// ---------------------------------------------------------------------------

var (
	taskUpdateTitle    string
	taskUpdateType     string
	taskUpdatePriority string
	taskUpdateEstimate string
	taskUpdateStatus   string
	taskUpdateSprint   string
	taskUpdateDepends  string
)

var taskUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Partial update of Task fields (title/type/priority/estimate/sprint/depends)",
	Example: brand.Examplef(
		`task update T100 --priority p1`,
		`task update T100 --title "new title" --estimate L`,
		`task update T100 --sprint sprint-37  # repair DB drift (no file move)`,
		`task update T100 --depends T001,T002`,
		`task update T100 --depends "" # clear all dependencies`,
	),
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		req := domain.UpdateRequest{TaskID: args[0]}

		// Apply only flags whose Changed bit is set (omitted = nil = no change).
		if cmd.Flags().Changed("title") {
			req.Title = &taskUpdateTitle
		}
		if cmd.Flags().Changed("type") {
			// type alias normalization.
			normalized, err := normalizeType(taskUpdateType)
			if err != nil {
				Out.Error(err.Error(), apperr.CategoryInvalidInput.String(), "--type must be one of feature|bugfix|docs|refactor|infra|test|chore|spike|hotfix")
				os.Exit(exitError)
			}
			taskUpdateType = normalized
			req.Type = &taskUpdateType
		}
		if cmd.Flags().Changed("priority") {
			// priority normalization plus enum validation.
			normalized, err := normalizePriority(taskUpdatePriority)
			if err != nil {
				Out.Error(err.Error(), apperr.CategoryInvalidInput.String(), "--priority must be one of p0|p1|p2|p3")
				os.Exit(exitError)
			}
			taskUpdatePriority = normalized
			req.Priority = &taskUpdatePriority
		}
		if cmd.Flags().Changed("estimate") {
			// estimate normalization.
			normalized, err := normalizeEstimate(taskUpdateEstimate)
			if err != nil {
				Out.Error(err.Error(), apperr.CategoryInvalidInput.String(), "--estimate must be one of XS|S|M|L|XL")
				os.Exit(exitError)
			}
			taskUpdateEstimate = normalized
			req.Estimate = &taskUpdateEstimate
		}
		if cmd.Flags().Changed("status") {
			req.Status = &taskUpdateStatus
		}
		if cmd.Flags().Changed("depends") {
			deps := parseCSV(taskUpdateDepends)
			req.DependsOn = &deps
		}
		if cmd.Flags().Changed("sprint") {
			req.Sprint = &taskUpdateSprint
		}

		if !req.HasUpdates() {
			Out.Error("specify at least one field to update", apperr.CategoryInvalidState.String(),
				"choose from --title, --type, --priority, --estimate, --status, --sprint, --depends")
			os.Exit(exitError)
		}

		mustInitDB()
		defer db.Close()

		result, err := app.TaskUpdate(&req)
		if err != nil {
			var nfe *apperr.NotFoundError
			var ise *apperr.InvalidStateError
			if errors.As(err, &nfe) {
				Out.Error(nfe.Error(), nfe.Category(), nfe.RecoveryHint)
			} else if errors.As(err, &ise) {
				Out.Error(ise.Error(), ise.Category(), "")
			} else {
				Out.Error(err.Error(), "", "")
			}
			os.Exit(exitError)
		}

		Out.Success("Task updated", result)
		return nil
	},
}

// parseCSV splits a comma-separated string into a slice. Returns an empty slice for empty input.
func parseCSV(s string) []string {
	if s == "" {
		return []string{}
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// ---------------------------------------------------------------------------
// task delete
// ---------------------------------------------------------------------------

var (
	taskDeleteReason     string
	taskDeleteOrphanFile bool // tidy up DB drift orphan files
)

var taskDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a Task (only when status is todo)",
	Example: brand.Examplef(
		`task delete T099 --reason "duplicate Task merged into T100"`,
		`task delete T099 --reason "no longer needed after requirement change"`,
		`task delete T706 --reason "DB drift cleanup" --orphan-file`,
	),
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if taskDeleteReason == "" {
			Out.Error("--reason flag is required", apperr.CategoryInvalidState.String(), "")
			os.Exit(exitError)
		}

		mustInitDB()
		defer db.Close()

		result, err := app.TaskDeleteWithOptions(args[0], taskDeleteReason, task.DeleteOptions{
			OrphanFile: taskDeleteOrphanFile,
		})
		if err != nil {
			var nfe *apperr.NotFoundError
			var rje *apperr.RejectedError
			if errors.As(err, &nfe) {
				Out.Error(nfe.Error(), nfe.Category(), nfe.RecoveryHint)
			} else if errors.As(err, &rje) {
				Out.Error(rje.Error(), rje.Category(), "")
			} else {
				Out.Error(err.Error(), "", "")
			}
			os.Exit(exitError)
		}

		Out.Success("Task deleted", result)
		return nil
	},
}

// ---------------------------------------------------------------------------
// completed-Sprint guard helpers
// ---------------------------------------------------------------------------

// sprintStatusCompleted is the Sprint status constant used by the KB M001 guard.
const sprintStatusCompleted = "completed"

// sprintGetForGuard is the Sprint lookup dependency - overridable in tests.
var sprintGetForGuard = app.SprintGet

// taskGetForGuard is the Task lookup dependency - overridable in tests.
var taskGetForGuard = app.TaskGet

// rejectIfCompletedSprint returns an error when the Sprint with the given ID
// is in the completed state. Prevents recurrence of KB M001: editing
// SPRINT.md / tasks under a completed Sprint triggers a chained block in
// precommit.sprint.hmac for legacy mismatch, so we reject by default. Use
// --force to override.
func rejectIfCompletedSprint(sprintID, action string) error {
	raw, err := sprintGetForGuard(sprintID)
	if err != nil {
		// Missing Sprint is out of scope for this guard - upstream pkg/task handles it.
		return nil
	}
	rec, ok := raw.(*domain.SprintRecord)
	if !ok || rec == nil {
		return nil
	}
	if rec.Status == sprintStatusCompleted {
		return fmt.Errorf("%s rejected: Sprint %q is in completed state (KB M001 guard)",
			action, sprintID)
	}
	return nil
}

// rejectUnassignIfCompletedSprint returns an error when the Task's current
// Sprint is completed. Tasks not attached to a Sprint pass through (treated
// as BACKLOG).
func rejectUnassignIfCompletedSprint(taskID string) error {
	raw, err := taskGetForGuard(taskID)
	if err != nil {
		return nil
	}
	tr, ok := raw.(*domain.GetResult)
	if !ok || tr == nil {
		return nil
	}
	sprintID, _ := tr.Sprint.(string)
	if sprintID == "" {
		return nil
	}
	return rejectIfCompletedSprint(sprintID, "unassign")
}

// ---------------------------------------------------------------------------
// task assign
// ---------------------------------------------------------------------------

var (
	taskAssignSprint string
	taskAssignForce  bool // escape hatch for the completed-Sprint guard
)

var taskAssignCmd = &cobra.Command{
	Use:   "assign <id>",
	Short: "Assign a Task to a Sprint (BACKLOG -> Sprint)",
	Example: brand.Examplef(
		`task assign T100 --sprint sprint-19`,
		`task assign T100,T101,T102 --sprint sprint-19`,
		`task assign T100 --sprint sprint-19 --force  # force-assign to a completed Sprint`,
	),
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if taskAssignSprint == "" {
			Out.Error("--sprint flag is required", apperr.CategoryInvalidState.String(), "")
			os.Exit(exitError)
		}

		taskIDs := parseCSV(args[0])
		if len(taskIDs) == 0 {
			Out.Error("specify at least one Task ID", apperr.CategoryInvalidState.String(), "")
			os.Exit(exitError)
		}

		mustInitDB()
		defer db.Close()

		// completed-Sprint guard - prevents recurrence of KB M001.
		// Editing SPRINT.md / tasks/ under an active Sprint triggers
		// precommit.sprint.hmac to chain-block on legacy mismatch.
		// --force explicitly overrides.
		if !taskAssignForce {
			if err := rejectIfCompletedSprint(taskAssignSprint, "assign"); err != nil {
				Out.Error(err.Error(), apperr.CategoryInvalidState.String(),
					"See KB M001 - completed Sprints must not be modified. Assign to a new Sprint or use --force.")
				os.Exit(exitError)
			}
		}

		result, err := app.TaskAssignSprint(taskIDs, taskAssignSprint)
		if err != nil {
			var nfe *apperr.NotFoundError
			var rje *apperr.RejectedError
			if errors.As(err, &nfe) {
				Out.Error(nfe.Error(), nfe.Category(), nfe.RecoveryHint)
			} else if errors.As(err, &rje) {
				Out.Error(rje.Error(), rje.Category(), "")
			} else {
				Out.Error(err.Error(), "", "")
			}
			os.Exit(exitError)
		}

		// Avoid showing a success message when every assignment failed.
		if ar, ok := result.(*domain.AssignSprintResult); ok && len(ar.Assigned) == 0 && len(ar.Failed) > 0 {
			reasons := make([]string, 0, len(ar.Failed))
			hint := ""
			for _, f := range ar.Failed {
				reasons = append(reasons, fmt.Sprintf("%s(%s)", f.TaskID, f.Reason))
				if hint == "" && f.Reason == "placeholder_body" {
					hint = "Write the Purpose/Requirements/Done Criteria sections in the Task file, then retry"
				}
			}
			Out.Error(
				fmt.Sprintf("Task Sprint assignment failed - %s", strings.Join(reasons, ", ")),
				"ASSIGN_FAILED",
				hint,
			)
			os.Exit(exitError)
		}
		Out.Success("Task assigned to Sprint", result)
		return nil
	},
}

// ---------------------------------------------------------------------------
// task unassign
// ---------------------------------------------------------------------------

var taskUnassignForce bool // escape hatch for the completed-Sprint guard

var taskUnassignCmd = &cobra.Command{
	Use:   "unassign <id>",
	Short: "Detach a Task from its Sprint (Sprint -> BACKLOG)",
	Example: brand.Examplef(
		`task unassign T100`,
		`task unassign T100,T101,T102`,
		`task unassign T100 --force  # force-detach from a completed Sprint`,
	),
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskIDs := parseCSV(args[0])
		if len(taskIDs) == 0 {
			Out.Error("specify at least one Task ID", apperr.CategoryInvalidState.String(), "")
			os.Exit(exitError)
		}

		mustInitDB()
		defer db.Close()

		// completed-Sprint guard.
		// If the current Sprint of any Task is completed, reject. --force overrides.
		if !taskUnassignForce {
			for _, tid := range taskIDs {
				if err := rejectUnassignIfCompletedSprint(tid); err != nil {
					Out.Error(err.Error(), apperr.CategoryInvalidState.String(),
						"See KB M001 - Tasks under a completed Sprint are fixed. To change, allocate a replacement Task or use --force.")
					os.Exit(exitError)
				}
			}
		}

		result, err := app.TaskUnassignSprint(taskIDs)
		if err != nil {
			var nfe *apperr.NotFoundError
			var rje *apperr.RejectedError
			if errors.As(err, &nfe) {
				Out.Error(nfe.Error(), nfe.Category(), nfe.RecoveryHint)
			} else if errors.As(err, &rje) {
				Out.Error(rje.Error(), rje.Category(), "")
			} else {
				Out.Error(err.Error(), "", "")
			}
			os.Exit(exitError)
		}

		// Avoid showing a success message when every detach failed.
		if ur, ok := result.(*domain.UnassignSprintResult); ok && len(ur.Unassigned) == 0 && len(ur.Failed) > 0 {
			reasons := make([]string, 0, len(ur.Failed))
			for _, f := range ur.Failed {
				reasons = append(reasons, fmt.Sprintf("%s(%s)", f.TaskID, f.Reason))
			}
			Out.Error(
				fmt.Sprintf("Task Sprint detach failed - %s", strings.Join(reasons, ", ")),
				"UNASSIGN_FAILED",
				"",
			)
			os.Exit(exitError)
		}
		Out.Success("Task detached from Sprint", result)
		return nil
	},
}

// ---------------------------------------------------------------------------
// task checkpoint
// ---------------------------------------------------------------------------

var taskCheckpointReason string

var taskCheckpointCmd = &cobra.Command{
	Use:   "checkpoint <id>",
	Short: "Issue a fresh work ticket mid-task (invalidates the existing ack)",
	Long: fmt.Sprintf(`If context shifts significantly during a long Task, the existing ack may be stale.
checkpoint issues a new work ticket and automatically invalidates the prior
ticket-based ack. After running it, fetch the new hash with
'%s context --ticket <new>' and ack again so task complete can pass.`, brand.ShortName),
	Example: brand.Examplef(
		`task checkpoint T549 --reason "context revisit after requirement change"`,
	),
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mustInitDB()
		defer db.Close()

		rawResult, err := app.TaskCheckpoint(args[0], taskCheckpointReason)
		if err != nil {
			var nfe *apperr.NotFoundError
			var rje *apperr.RejectedError
			if errors.As(err, &nfe) {
				Out.Error(nfe.Error(), nfe.Category(), nfe.RecoveryHint)
			} else if errors.As(err, &rje) {
				Out.Error(rje.Error(), rje.Category(), "")
			} else {
				Out.Error(err.Error(), "", "")
			}
			os.Exit(exitError)
		}
		result := rawResult.(*domain.CheckpointResult)

		Out.Success(fmt.Sprintf("work ticket reissued - run '%s context --ticket %s' to obtain a new ack", brand.ShortName, result.NewTicket), result)
		return nil
	},
}

// ---------------------------------------------------------------------------
// task reopen
// ---------------------------------------------------------------------------

var (
	taskReopenReason string
	taskReopenStatus string
)

var taskReopenCmd = &cobra.Command{
	Use:   "reopen <id>",
	Short: "Reopen a Task (done/in-progress -> todo/in-progress)",
	Example: brand.Examplef(
		`task reopen T050 --reason "rework required after additional requirements"`,
		`task reopen T050 --reason "tests failed, reopening" --status in-progress`,
	),
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if taskReopenReason == "" {
			Out.Error("--reason flag is required", apperr.CategoryInvalidState.String(), "")
			os.Exit(exitError)
		}

		mustInitDB()
		defer db.Close()

		newStatus := taskReopenStatus
		if newStatus == "" {
			newStatus = "todo"
		}
		// status alias normalization - accept inputs like `--status in_progress`.
		// On error, surface the allowed values to the user.
		if canonical, err := task.Canonicalize(newStatus); err == nil && canonical != "" {
			newStatus = canonical
		} else if err != nil {
			Out.Error(err.Error(), "INVALID_STATUS",
				"Allowed values: todo / in-progress / done / blocked / reopened")
			os.Exit(exitError)
		}

		result, err := app.TaskReopen(args[0], taskReopenReason, newStatus)
		if err != nil {
			var nfe *apperr.NotFoundError
			var rje *apperr.RejectedError
			if errors.As(err, &nfe) {
				Out.Error(nfe.Error(), nfe.Category(), nfe.RecoveryHint)
			} else if errors.As(err, &rje) {
				Out.Error(rje.Error(), rje.Category(), "")
			} else {
				Out.Error(err.Error(), "", "")
			}
			os.Exit(exitError)
		}

		Out.Success("Task reopened", result)
		return nil
	},
}

// ---------------------------------------------------------------------------
// init
// ---------------------------------------------------------------------------

func init() {
	// task list flags
	taskListCmd.Flags().StringVar(&taskListSprint, "sprint", "", "Sprint ID filter (e.g. sprint-01)")
	taskListCmd.Flags().StringVar(&taskListStatus, "status", "", "Status filter (todo|in-progress|done)")
	taskListCmd.Flags().StringVar(&taskListPriority, "priority", "", "Priority (p0|p1|p2|p3)")
	taskListCmd.Flags().StringVar(&taskListType, "type", "", "Task type (feature|bugfix|refactor|infra|docs|test|chore|hotfix|spike)")
	taskListCmd.Flags().StringVar(&taskListSince, "since", "", "Since this point (ISO-8601, against updated_at)")
	taskListCmd.Flags().StringVar(&taskListUntil, "until", "", "Until this point (ISO-8601, against updated_at)")
	taskListCmd.Flags().IntVar(&taskListLast, "last", 0, "Most recent N (forces time-desc sort)")
	taskListCmd.Flags().IntVar(&taskListLimit, "limit", 0, "Maximum number of rows (truncate)")

	// task create flags
	taskCreateCmd.Flags().StringVar(&taskCreateTitle, "title", "", "Task title (required)")
	taskCreateCmd.Flags().StringVar(&taskCreateType, "type", "", "Task type: feature|bugfix|docs|refactor|infra|test|chore|spike (required)")
	// --sprint flag deprecated - Tasks are always created in BACKLOG.
	// After writing the body, run
	// 'hstl task assign <ID> --sprint sprint-NN' to attach to a Sprint.
	taskCreateCmd.Flags().StringVar(&taskCreatePriority, "priority", "p2", "Priority: p0|p1|p2|p3")
	taskCreateCmd.Flags().StringVar(&taskCreateEstimate, "estimate", "M", "Estimate: XS|S|M|L|XL")
	taskCreateCmd.Flags().StringVar(&taskCreateDepends, "depends", "", "Comma-separated dependency Task IDs (e.g. T001,T002)")
	taskCreateCmd.Flags().StringVar(&taskCreateSummary, "summary", "", "One-line summary - what / why / draft success criteria (required, 10-200 chars)")
	taskCreateCmd.Flags().BoolVar(&taskCreateWithCeremony, "with-ceremony", false,
		"Include ceremony data (briefing, reminders, next_actions)")
	// --dry-run: preview the template render without DB / file / counter side effects.
	taskCreateCmd.Flags().BoolVar(&taskCreateDryRun, "dry-run", false,
		"Preview the template render with no DB / file / counter side effects")
	taskCreateCmd.Flags().BoolVar(&taskCreateSkipSummaryValidation, "skip-summary-validation", false,
		"Skip the summary semantic validation (subagent) - CI / emergency only; recorded in the audit log")
	taskCreateCmd.Flags().BoolVar(&taskCreateAllowDuplicateTitle, "allow-duplicate-title", false,
		"Skip title-duplicate detection - use when an intentional duplicate is desired")

	// task start flags
	taskStartCmd.Flags().BoolVar(&taskStartWithCeremony, "with-ceremony", false,
		"Include ceremony data (briefing, reminders)")
	taskStartCmd.Flags().BoolVar(&taskStartDryRun, "dry-run", false,
		"Validate without performing the state transition")

	// task complete flags
	taskCompleteCmd.Flags().BoolVar(&taskCompleteAutoCheckGo, "auto-check-go", true,
		"Auto-run build/test/vet and check harness items before task complete in Go projects (default true; also disabled by HSTL_TASK_AUTOCHECK_GO=off)")
	taskCompleteCmd.Flags().BoolVar(&taskCompleteAutoCheckCriteria, "auto-check-criteria", false,
		"Parse the Task body's `## Done Criteria` checkboxes; if all checked, auto-check criteria_checked, otherwise BLOCK")
	taskCompleteCmd.Flags().BoolVar(&taskCompleteInteractive, "interactive", false,
		"Prompt [y/n/s] for each unchecked BLOCKED item (auto-disabled when not a TTY)")
	taskCompleteCmd.Flags().BoolVar(&taskCompleteWithCeremony, "with-ceremony", false,
		"Include ceremony data (changed_files, harness_gate, result_section)")
	taskCompleteCmd.Flags().BoolVar(&taskCompleteDryRun, "dry-run", false,
		"Validate Harness / Result section without performing complete")

	// task update flags
	taskUpdateCmd.Flags().StringVar(&taskUpdateTitle, "title", "", "New title")
	taskUpdateCmd.Flags().StringVar(&taskUpdateType, "type", "", "Task type: feature|bugfix|hotfix|refactor|infra|docs|test|chore|spike")
	taskUpdateCmd.Flags().StringVar(&taskUpdatePriority, "priority", "", "Priority: p0|p1|p2|p3")
	taskUpdateCmd.Flags().StringVar(&taskUpdateEstimate, "estimate", "", "Estimate: XS|S|M|L|XL")
	taskUpdateCmd.Flags().StringVar(&taskUpdateStatus, "status", "", "Escape hatch for drift recovery only: todo|in-progress|done|absorbed. Normal path: task start/complete/reopen. When invoked, an audit event task.status.force_updated is recorded.")
	taskUpdateCmd.Flags().StringVar(&taskUpdateSprint, "sprint", "", "Edit the Sprint ID directly (updates only DB+frontmatter; no file move) - drift recovery path")
	taskUpdateCmd.Flags().StringVar(&taskUpdateDepends, "depends", "", "Comma-separated dependency Task IDs (empty string clears all)")

	// task delete flags
	taskDeleteCmd.Flags().StringVar(&taskDeleteReason, "reason", "", "Deletion reason (>= 10 chars, required)")
	taskDeleteCmd.Flags().BoolVar(&taskDeleteOrphanFile, "orphan-file", false, "Clean up drift Tasks that exist as files only (no DB row) from files and BACKLOG (audit task.orphan_deleted)")

	// task reopen flags
	taskReopenCmd.Flags().StringVar(&taskReopenReason, "reason", "", "Reopen reason (>= 10 chars, required)")
	taskReopenCmd.Flags().StringVar(&taskReopenStatus, "status", "todo", "Target status: todo|in-progress")

	// task assign flags
	taskAssignCmd.Flags().StringVar(&taskAssignSprint, "sprint", "", "Sprint ID to assign to (e.g. sprint-19, required)")
	taskAssignCmd.Flags().BoolVar(&taskAssignForce, "force", false, "Override the completed-Sprint guard (see KB M001)")

	// task unassign flags
	taskUnassignCmd.Flags().BoolVar(&taskUnassignForce, "force", false, "Override the completed-Sprint guard (see KB M001)")

	// Register subcommands
	taskCmd.AddCommand(taskListCmd)
	taskCmd.AddCommand(taskGetCmd)
	taskCmd.AddCommand(taskCreateCmd)
	taskCmd.AddCommand(taskStartCmd)
	taskCmd.AddCommand(taskCompleteCmd)
	taskCmd.AddCommand(taskNextCmd)
	taskCmd.AddCommand(taskUpdateCmd)
	taskCmd.AddCommand(taskDeleteCmd)
	taskCmd.AddCommand(taskReopenCmd)
	taskCmd.AddCommand(taskAssignCmd)
	taskCmd.AddCommand(taskUnassignCmd)

	// task checkpoint (T549)
	taskCheckpointCmd.Flags().StringVar(&taskCheckpointReason, "reason", "", "Checkpoint reason (optional, recorded in audit log)")
	taskCmd.AddCommand(taskCheckpointCmd)

	rootCmd.AddCommand(taskCmd)
}
