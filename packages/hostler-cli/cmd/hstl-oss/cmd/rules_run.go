package cmd

// rules_run.go - the `hstl rules run --phase <phase>` subcommand.
//
// For each phase, collects the inputs the rules need (commit-msg body /
// staged diff), runs the rules registered for that phase, and reports the
// exit code with evidence. This is the entry point invoked from git hooks.
//
// Phases currently implemented:
//   - commit-msg : commit message injected via stdin or --msg-file
//                  -> runs commit.message.korean / commit.task_id.present
//   - precommit  : staged diff collected from `git diff --cached`
//                  -> runs commit.files.no_secrets
//
// Exit code contract:
//   0 - every Rule OK or skipped
//   3 - at least one BLOCK / HARD_BLOCK violation
//   other - internal error

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"sync"

	"github.com/spf13/cobra"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/app"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/brand"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/ports"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/envalias"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/rules"
)

// recordRuleJudgement records a rule's execution result in the gate log.
// Maps the Rule's Status x Severity combination to a gate Verdict.
func recordRuleJudgement(ruleID string, res *rules.RuleResult) {
	if res == nil {
		return
	}
	var v ports.Verdict
	switch res.Status {
	case rules.StatusOK:
		v = ports.VerdictPass
	case rules.StatusViolated:
		switch res.Severity {
		case rules.SeverityHardBlock:
			v = ports.VerdictHardBlock
		case rules.SeverityBlock:
			v = ports.VerdictBlock
		case rules.SeverityWarn:
			v = ports.VerdictWarn
		default:
			v = ports.VerdictWarn
		}
	case rules.StatusSkipped, rules.StatusError:
		v = ports.VerdictSkip
	}
	app.GateRecord(ports.JudgementRecord{
		GateType:     "rules",
		ItemID:       ruleID,
		Verdict:      v,
		RuleID:       ruleID,
		RuleSeverity: res.Severity.String(),
		InputSnippet: res.Message,
		EvidenceRefs: res.Evidence,
	})
}

const (
	// "--phase all" sentinel.
	rulesPhaseAll = "all"
	// Env var that controls parallelism for the precommit phase.
	// When unset, falls back to runtime.NumCPU(). Set to 1 to force sequential.
	envPrecommitParallelism = "HSTL_PRECOMMIT_PARALLELISM"
	// sequentialParallelism is the fixed value for phases like commit-msg
	// where parallelism gives no benefit.
	sequentialParallelism = 1
)

var (
	rulesRunPhase   string
	rulesRunMsgFile string
	rulesRunAudit   bool   // non-blocking audit mode
	rulesRunReport  string // path to a JSON report file
)

var rulesRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the Rule Engine for git hooks / audit",
	Long: `Run the Rules registered for the specified Phase. This is the git hook
entry point and also supports an audit mode.

--phase commit-msg : run commit.* Rules against the message at --msg-file
--phase precommit  : run pre-commit Rules against the diff from git diff --cached
--phase all        : iterate every supported phase (recommended with audit mode)

--audit            : non-blocking observation mode. Even on block/hard-block
                     violations, exit 0 and only emit a report. Useful as a
                     before/after baseline when migrating bridge rules.
--report <path>    : write the JSON report to <path>. Without it, only the
                     summary is emitted to stdout.

When a Rule whose severity is block / hard-block is violated, the default
mode exits 3. In audit mode, the same condition exits 0.`,
	Example: brand.Examplef(
		`rules run --phase commit-msg --msg-file .git/COMMIT_EDITMSG`,
		`rules run --phase precommit`,
		`rules run --phase all --audit --report /tmp/rules-audit.json`,
	),
	RunE: runRulesRun,
}

func runRulesRun(cmd *cobra.Command, args []string) error {
	phases, err := resolvePhases(rulesRunPhase)
	if err != nil {
		return err
	}
	cfgRaw, err := app.RulesLoadConfig(app.ProjectRoot())
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	cfg, _ := cfgRaw.(*rules.EngineConfig)
	effRaw, err := app.RulesResolveEffective(cfg, app.ProjectRoot())
	if err != nil {
		return fmt.Errorf("failed to resolve effective rules: %w", err)
	}
	eff, _ := effRaw.(map[string]*rules.EffectiveRule)

	type phaseResult struct {
		phase  rules.Phase
		result *rules.RuleResult
	}
	var all []phaseResult

	for _, phase := range phases {
		ctx := &rules.RuleContext{
			Phase:       phase,
			ProjectRoot: app.ProjectRoot(),
			Config:      cfg,
			Extra:       map[string]any{},
		}
		if err := populateExtra(ctx, phase, rulesRunMsgFile); err != nil {
			if rulesRunAudit {
				continue
			}
			return err
		}
		ids := rulesForPhase(phase)
		parallelism := parallelismForPhase(phase)
		phaseResults := executeRules(ctx, ids, eff, parallelism)
		for _, res := range phaseResults {
			all = append(all, phaseResult{phase: phase, result: res})
		}
	}

	// In audit + phase=all, also run Rules that rulesForPhase does not map
	// (precommit/skill/harness/sprint categories) on a best-effort basis from
	// the full Registry so the audit report includes them. Rules already
	// processed in the loop above are skipped via an ID set to avoid
	// duplicates.
	if rulesRunAudit && rulesRunPhase == rulesPhaseAll {
		seen := make(map[string]bool, len(all))
		for _, e := range all {
			seen[e.result.RuleID] = true
		}
		// Best-effort context - only fills the project root (each Rule may use ProjectRoot).
		auditCtx := &rules.RuleContext{
			ProjectRoot: app.ProjectRoot(),
			Config:      cfg,
			Extra:       map[string]any{},
		}
		allRules, _ := app.RulesList().([]rules.Rule)
		for _, r := range allRules {
			if seen[r.ID()] {
				continue
			}
			res := r.Check(auditCtx)
			if e, ok := eff[r.ID()]; ok {
				res.Severity = e.Severity
			}
			all = append(all, phaseResult{phase: rules.Phase(r.Category()), result: res})
		}
	}

	results := make([]*rules.RuleResult, 0, len(all))
	for _, e := range all {
		results = append(results, e.result)
	}
	blocked := renderRulesRunOutput(cmd.OutOrStderr(), results)

	// Write the JSON report when --report is provided.
	if rulesRunReport != "" {
		phaseOf := make(map[string]rules.Phase, len(all))
		for _, e := range all {
			phaseOf[e.result.RuleID] = e.phase
		}
		if err := writeAuditReport(rulesRunReport, phaseOf, results); err != nil {
			return fmt.Errorf("failed to write audit report: %w", err)
		}
	}

	if blocked && !rulesRunAudit {
		os.Exit(exitBlocked)
	}
	return nil
}

// JSON report schema.
type auditEntry struct {
	Phase    string   `json:"phase"`
	RuleID   string   `json:"rule_id"`
	Status   string   `json:"status"`
	Severity string   `json:"severity"`
	Message  string   `json:"message,omitempty"`
	Evidence []string `json:"evidence,omitempty"`
}

type auditReport struct {
	Version string       `json:"version"`
	Entries []auditEntry `json:"entries"`
	Summary struct {
		Total    int `json:"total"`
		OK       int `json:"ok"`
		Violated int `json:"violated"`
		Skipped  int `json:"skipped"`
		Error    int `json:"error"`
	} `json:"summary"`
}

// rulesReportFileMode is the audit report file mode (observability artifact, read-only).
const rulesReportFileMode os.FileMode = 0o644

// writeAuditReport writes the JSON report file using the results and phase mapping.
func writeAuditReport(path string, phaseOf map[string]rules.Phase, results []*rules.RuleResult) error {
	rep := auditReport{Version: "1"}
	for _, r := range results {
		e := auditEntry{
			Phase:    string(phaseOf[r.RuleID]),
			RuleID:   r.RuleID,
			Status:   statusString(r.Status),
			Severity: r.Severity.String(),
			Message:  r.Message,
			Evidence: r.Evidence,
		}
		rep.Entries = append(rep.Entries, e)
		rep.Summary.Total++
		switch r.Status {
		case rules.StatusOK:
			rep.Summary.OK++
		case rules.StatusViolated:
			rep.Summary.Violated++
		case rules.StatusSkipped:
			rep.Summary.Skipped++
		case rules.StatusError:
			rep.Summary.Error++
		}
	}
	data, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, rulesReportFileMode)
}

// statusString converts a RuleStatus to its JSON report string.
func statusString(s rules.RuleStatus) string {
	switch s {
	case rules.StatusOK:
		return "ok"
	case rules.StatusViolated:
		return "violated"
	case rules.StatusSkipped:
		return "skipped"
	case rules.StatusError:
		return "error"
	}
	return "unknown"
}

// parallelismForPhase determines the parallelism for the precommit phase.
//
// The Rules in the precommit phase invoke independent external commands with
// no shared state, so concurrent execution via goroutines is safe. Other
// phases (e.g. commit-msg) have only a few Rules and benefit little from
// parallelism, so they remain sequential to keep stdin/msg-file read order
// straightforward.
//
// HSTL_PRECOMMIT_PARALLELISM:
//   - empty / unset: fall back to runtime.NumCPU()
//   - positive int N: use N (set to 1 for sequential)
//   - parse failure: fall back
func parallelismForPhase(phase rules.Phase) int {
	if phase != rules.PhasePreCommit {
		return sequentialParallelism
	}
	if v := envalias.Lookup("PRECOMMIT_PARALLELISM"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 {
			return n
		}
	}
	return runtime.NumCPU()
}

// executeRules runs the given Rule IDs at the requested parallelism and
// returns a result slice.
//
// parallelism <= 1 runs sequentially (equivalent to the original
// append-based logic). In parallel mode, results are written to fixed,
// index-based slots to avoid races; nil entries (rules whose Get failed)
// are filtered out at the end. The caller does not depend on slice order
// because renderRulesRunOutput sorts by RuleID.
func executeRules(ctx *rules.RuleContext, ids []string, eff map[string]*rules.EffectiveRule, parallelism int) []*rules.RuleResult {
	if parallelism <= sequentialParallelism {
		out := make([]*rules.RuleResult, 0, len(ids))
		for _, id := range ids {
			rRaw, ok := app.RulesGet(id)
			if !ok {
				continue
			}
			r, _ := rRaw.(rules.Rule)
			res := r.Check(ctx)
			if e, ok := eff[id]; ok {
				res.Severity = e.Severity
			}
			// gate-log record (sequential execution path).
			recordRuleJudgement(id, res)
			out = append(out, res)
		}
		return out
	}

	slots := make([]*rules.RuleResult, len(ids))
	var wg sync.WaitGroup
	sem := make(chan struct{}, parallelism)
	for i, id := range ids {
		rRaw, ok := app.RulesGet(id)
		if !ok {
			continue
		}
		r, _ := rRaw.(rules.Rule)
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, rule rules.Rule, ruleID string) {
			defer wg.Done()
			defer func() { <-sem }()
			res := rule.Check(ctx)
			if e, ok := eff[ruleID]; ok {
				res.Severity = e.Severity
			}
			// gate-log record (parallel execution path).
			recordRuleJudgement(ruleID, res)
			slots[idx] = res
		}(i, r, id)
	}
	wg.Wait()

	out := make([]*rules.RuleResult, 0, len(ids))
	for _, r := range slots {
		if r != nil {
			out = append(out, r)
		}
	}
	return out
}

// resolvePhases converts the flag string into a slice of phases to run.
// "all" iterates every supported phase.
func resolvePhases(s string) ([]rules.Phase, error) {
	switch s {
	case "commit-msg":
		return []rules.Phase{rules.PhaseCommitMsg}, nil
	case "precommit":
		return []rules.Phase{rules.PhasePreCommit}, nil
	case rulesPhaseAll:
		// Only includes the phases populateExtra currently supports.
		// task/sprint/bash phases are scheduled for the bridge Rule migration.
		return []rules.Phase{rules.PhaseCommitMsg, rules.PhasePreCommit}, nil
	case "":
		return nil, fmt.Errorf("--phase flag is required (commit-msg | precommit | all)")
	default:
		return nil, fmt.Errorf("unsupported phase: %q (commit-msg | precommit | all)", s)
	}
}

// populateExtra fills ctx.Extra according to the phase.
// This is the only place that performs external I/O.
// In audit mode, when the required input is missing, the upper loop receives
// a "skip signal" and continues execution (input-missing phases are excluded
// from observation). Here we keep things simple and return an error directly,
// and the caller checks rulesRunAudit and continues.
func populateExtra(ctx *rules.RuleContext, phase rules.Phase, msgFile string) error {
	switch phase {
	case rules.PhaseCommitMsg:
		if msgFile == "" {
			return fmt.Errorf("commit-msg phase requires --msg-file")
		}
		data, err := os.ReadFile(msgFile)
		if err != nil {
			return fmt.Errorf("failed to read commit message file: %w", err)
		}
		ctx.Extra[rules.ExtraKeyCommitMsg] = string(data)
	case rules.PhasePreCommit:
		diff, err := runGitDiffCached()
		if err != nil {
			return fmt.Errorf("failed to run git diff --cached: %w", err)
		}
		ctx.Extra[rules.ExtraKeyStagedDiff] = diff
	}
	return nil
}

// runGitDiffCached collects the staged diff as a single text blob.
// Kept as a variable so tests can override it; the call itself is one line,
// so we do not introduce a richer DI mechanism (YAGNI).
var runGitDiffCached = func() (string, error) {
	cmd := exec.Command("git", "diff", "--cached")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// rulesForPhase returns the Rule IDs to run for the given phase.
// PhasePreCommit also pulls every Registry rule with category=="precommit".
// Once bash hooks are slimmed down, the Rule Engine performs the actual
// enforcement, so a missing mapping silently drops the check entirely.
func rulesForPhase(phase rules.Phase) []string {
	switch phase {
	case rules.PhaseCommitMsg:
		return []string{"commit.message.korean", "commit.task_id.present"}
	case rules.PhasePreCommit:
		// Fixed entries (secret + legacy compatibility) plus the precommit category from the Registry.
		ids := []string{"commit.files.no_secrets"}
		allRules, _ := app.RulesList().([]rules.Rule)
		for _, r := range allRules {
			if r.Category() == "precommit" {
				ids = append(ids, r.ID())
			}
		}
		return ids
	}
	return nil
}

// renderRulesRunOutput writes the results to stderr in human-readable form
// and returns true when at least one block / hard-block violation is present.
func renderRulesRunOutput(w interface{ Write(p []byte) (int, error) }, results []*rules.RuleResult) bool {
	sort.Slice(results, func(i, j int) bool { return results[i].RuleID < results[j].RuleID })
	blocked := false
	for _, res := range results {
		line := ""
		switch res.Status {
		case rules.StatusOK:
			line = fmt.Sprintf("[OK] %-30s %s\n", res.RuleID, "OK")
		case rules.StatusSkipped:
			line = fmt.Sprintf("[-] %-30s %s\n", res.RuleID, "skipped")
		case rules.StatusViolated:
			sev := res.Severity.String()
			icon := "warn"
			if res.Severity == rules.SeverityBlock || res.Severity == rules.SeverityHardBlock {
				icon = "BLOCK"
				blocked = true
			} else if res.Severity == rules.SeverityOff {
				continue
			}
			line = fmt.Sprintf("[%s] %-30s [%s] %s\n", icon, res.RuleID, sev, res.Message)
			for _, ev := range res.Evidence {
				line += fmt.Sprintf("    - %s\n", ev)
			}
		case rules.StatusError:
			line = fmt.Sprintf("[!] %-30s error: %v\n", res.RuleID, res.Err)
		}
		_, _ = w.Write([]byte(line))
	}
	return blocked
}

func init() {
	rulesRunCmd.Flags().StringVar(&rulesRunPhase, "phase", "", "Phase to run (commit-msg | precommit | all)")
	rulesRunCmd.Flags().StringVar(&rulesRunMsgFile, "msg-file", "", "commit-msg phase only: path to the commit message file")
	rulesRunCmd.Flags().BoolVar(&rulesRunAudit, "audit", false, "Non-blocking audit mode - exit 0 even on violations")
	rulesRunCmd.Flags().StringVar(&rulesRunReport, "report", "", "Path to the JSON report file")
	rulesCmd.AddCommand(rulesRunCmd)
}
