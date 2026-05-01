// trac: WRK-QR003
package cmd

import (
	"fmt"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/brand"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/app"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/domain"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/ports"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/db"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/projectinfo"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/store"
)

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

// briefRecentSprintLimit is the number of recently completed Sprints to
// display.
const briefRecentSprintLimit = 3

// briefRecentP2TaskLimit is the number of recently completed
// p0/p1/p2-priority Tasks to display (T540).
const briefRecentP2TaskLimit = 10

// ---------------------------------------------------------------------------
// brief command
// ---------------------------------------------------------------------------

var briefCmd = &cobra.Command{
	Use:   "brief",
	Short: "User-facing project briefing (collected live, default verbose)",
	Long: `Collects the current project state in real time and returns a user-friendly text.
A rich briefing including Git state, Sprint progress, Task status, issues, and next actions.

T529 (Sprint 47 hotfix): the original definition of brief is "user-friendly rich
briefing", so verbose is now forced as the default (RecentCommits / Branches /
BranchGaps / BacklogTodoTasks included). For backward compatibility a compact
form is still available via --verbose=false (or -v=false).
With --output json, the data is emitted as a structured payload.`,
	Example: brand.Examplef(
		"brief",
		"brief -o json | jq '.active_sprints'",
		"brief --verbose=false  # compact (pre-T529 default)",
	),
	RunE: func(cmd *cobra.Command, args []string) error {
		mustInitDB()
		defer db.Close()

		// T529 (Sprint 47 hotfix): brief default flipped to verbose.
		// When the user has not explicitly passed --verbose, force
		// verbose=true. Resolves the meaning mismatch between the
		// original brief definition ("user-friendly rich briefing") and
		// the previous compact default. Backward compat: --verbose=false.
		if !cmd.Flags().Changed("verbose") {
			verbose = true
		}

		b := collectBrief()

		if outputFormat == "json" {
			Out.Print(b)
			return nil
		}

		Out.Print(formatBriefText(b))
		return nil
	},
}

// ---------------------------------------------------------------------------
// Data structures
// ---------------------------------------------------------------------------

type briefData struct {
	Project         briefProject        `json:"project"`
	Git             briefGit            `json:"git"`
	ActiveSprints   []briefSprint       `json:"active_sprints"`
	RecentCompleted []briefCompletedSpr `json:"recent_completed"`
	TaskSummary     briefTaskSummary    `json:"task_summary"`
	CurrentTask     *briefCurrentTask   `json:"current_task"`
	NextTask        *briefNextTask      `json:"next_task"`
	// RecentP2Tasks lists the 10 most recently completed p0/p1/p2-priority
	// Tasks (T540). Lets the user see major recent work at a glance.
	RecentP2Tasks []briefCompletedTask `json:"recent_p2_tasks,omitempty"`
	Issues        []briefIssue         `json:"issues"`

	// BacklogTodoTasks lists the top-N todo Tasks waiting in backlog
	// (T446, Sprint-38). Sorted by priority/estimate. Lets a single brief
	// invocation surface candidate next tasks.
	BacklogTodoTasks []briefBacklogTodo `json:"backlog_todo_tasks,omitempty"`

	// ContextModel exposes Project Context 12-Facet × 3-Layer Model
	// (ADR-047) metadata.
	// T565 (Sprint-53, C11 discoverability integration) — surfaces facet
	// model existence to the user via brief.
	ContextModel *briefContextModel `json:"context_model,omitempty"`
}

// briefContextModel — Project Context 12-Facet × 3-Layer Model metadata
// (T565).
//
// One added line in the brief response — lets a new user / AI seeing
// only `hstl brief` know that the facet model exists and where to learn
// about its concepts and usage.
//
// T568 (Sprint 53 hotfix): the previous ConceptSkill field pointed at a
// separate "context-engineering" skill; this hotfix demotes it to a
// reference under the parent skill (project-management). The field name
// is now ConceptReference and the value points to that references path.
type briefContextModel struct {
	Model            string `json:"model"`             // "12-Facet × 3-Layer (ADR-047)"
	UsageGuide       string `json:"usage_guide"`       // "docs/04-guides/context-15-facet-guide.md"
	ConceptReference string `json:"concept_reference"` // "project-management/references/context-engineering.md"
	UsageCommand     string `json:"usage_command"`     // "hstl context --facets/--layer/--all"
}

// briefBacklogTodo summarizes a backlog-waiting Task (T446).
type briefBacklogTodo struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Type     string `json:"type"`
	Priority string `json:"priority"`
	Estimate string `json:"estimate"`
}

// briefCompletedTask summarizes one recently completed Task (T540).
type briefCompletedTask struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Type        string `json:"type"`
	Priority    string `json:"priority"`
	Sprint      string `json:"sprint,omitempty"`
	CompletedAt string `json:"completed_at"`
}

type briefProject struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type briefGit struct {
	Branch      string `json:"branch"`
	Uncommitted int    `json:"uncommitted"`
	Ahead       int    `json:"ahead"`
	Behind      int    `json:"behind"`

	// T446 (Sprint-38): extended fields — populated only in verbose mode.
	// Allows a single brief invocation to assess branch merge state /
	// recent deployment activity.
	RecentCommits []briefCommit    `json:"recent_commits,omitempty"`
	Branches      *briefBranches   `json:"branches,omitempty"`
	BranchGaps    *briefBranchGaps `json:"branch_gaps,omitempty"`
}

// briefCommit — recent commit summary (T446).
type briefCommit struct {
	SHA     string `json:"sha"`
	Message string `json:"message"`
	Date    string `json:"date"`
}

// briefBranches — local/remote branch list + HEAD (T446).
type briefBranches struct {
	Local  []string `json:"local"`
	Remote []string `json:"remote"`
	HEAD   string   `json:"head"`
}

// briefBranchGaps — commit gaps between key branches (T446).
type briefBranchGaps struct {
	MainToDev     int `json:"main_to_dev"`     // main..dev commits (dev ahead of main)
	DevToCurrent  int `json:"dev_to_current"`  // dev..HEAD commits (current branch ahead of dev)
	CurrentToMain int `json:"current_to_main"` // HEAD..main commits (main ahead of current branch)
}

type briefSprint struct {
	ID      string      `json:"id"`
	Title   string      `json:"title"`
	Done    int         `json:"done"`
	Total   int         `json:"total"`
	Percent int         `json:"percent"`
	Tasks   []briefTask `json:"tasks,omitempty"`
	// Goal / Summary (T540): goal + 3+ line summary parsed from
	// SPRINT.md.
	Goal    string   `json:"goal,omitempty"`
	Summary []string `json:"summary,omitempty"`
}

type briefCompletedSpr struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	CompletedAt string `json:"completed_at"`
	Done        int    `json:"done"`
	Total       int    `json:"total"`
	// Goal / Summary (T540): goal + 3+ line summary parsed from
	// SPRINT.md.
	Goal    string   `json:"goal,omitempty"`
	Summary []string `json:"summary,omitempty"`
}

type briefTask struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Status   string `json:"status"`
	Priority string `json:"priority"`
}

type briefTaskSummary struct {
	Total      int `json:"total"`
	Done       int `json:"done"`
	InProgress int `json:"in_progress"`
	Todo       int `json:"todo"`
}

type briefCurrentTask struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Sprint string `json:"sprint,omitempty"`
}

type briefNextTask struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type briefIssue struct {
	Level   string `json:"level"`
	Message string `json:"message"`
}

// ---------------------------------------------------------------------------
// Collection logic
// ---------------------------------------------------------------------------

func collectBrief() *briefData {
	b := &briefData{}

	var wg sync.WaitGroup

	// goroutine 1: Git
	wg.Add(1)
	go func() {
		defer wg.Done()
		b.Project = collectBriefProject()
		b.Git = collectBriefGit()
	}()

	// goroutine 2: Sprint + Task (DB)
	wg.Add(1)
	go func() {
		defer wg.Done()
		b.ActiveSprints = collectBriefActiveSprints()
		b.RecentCompleted = collectBriefRecentCompleted()
		b.TaskSummary = collectBriefTaskSummary()
		b.CurrentTask = collectBriefCurrentTask()
		b.NextTask = collectBriefNextTask()
		b.RecentP2Tasks = collectBriefRecentP2Tasks()
		b.Issues = collectBriefIssues(b.ActiveSprints)
		// T446 (Sprint-38): include backlog todo list only in verbose
		// mode.
		if verbose {
			b.BacklogTodoTasks = collectBriefBacklogTodoTasks()
		}
	}()

	wg.Wait()

	// T565 (Sprint-53, C11 discoverability): static Project Context
	// 12-Facet × 3-Layer Model metadata (no external calls, zero cost).
	// Adding one field to the brief response surfaces the facet model
	// and its entry points to new users / AIs immediately.
	b.ContextModel = &briefContextModel{
		Model:            "12-Facet × 3-Layer (ADR-047)",
		UsageGuide:       "docs/04-guides/context-15-facet-guide.md",
		ConceptReference: "project-management/references/context-engineering.md",
		UsageCommand:     "hstl context --facets/--layer/--all",
	}
	return b
}

// collectBriefRecentP2Tasks returns up to briefRecentP2TaskLimit
// recently completed p0/p1/p2-priority Tasks (T540). Lets the user see
// major completed work at a glance.
func collectBriefRecentP2Tasks() []briefCompletedTask {
	gs := store.Get()
	if gs == nil {
		return nil
	}
	doneStatus := "done"
	listResult, err := gs.ListTasks(nil, &doneStatus)
	if err != nil || listResult == nil {
		return nil
	}

	// Filter to p0/p1/p2 + sort updated_at DESC, then take up to
	// briefRecentP2TaskLimit.
	allowedPriority := map[string]bool{"p0": true, "p1": true, "p2": true}
	var filtered []ports.TaskDetails
	for _, t := range listResult.Tasks {
		if allowedPriority[t.Priority] {
			filtered = append(filtered, t)
		}
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		if filtered[i].UpdatedAt != filtered[j].UpdatedAt {
			return filtered[i].UpdatedAt > filtered[j].UpdatedAt
		}
		return filtered[i].TaskID > filtered[j].TaskID
	})
	if len(filtered) > briefRecentP2TaskLimit {
		filtered = filtered[:briefRecentP2TaskLimit]
	}

	var result []briefCompletedTask
	for _, t := range filtered {
		result = append(result, briefCompletedTask{
			ID:          t.TaskID,
			Title:       t.Title,
			Type:        t.Type,
			Priority:    t.Priority,
			Sprint:      t.Sprint,
			CompletedAt: t.UpdatedAt,
		})
	}
	return result
}

func collectBriefProject() briefProject {
	name := contextGitOutput("rev-parse", "--show-toplevel")
	if name != "" {
		parts := strings.Split(name, string(os.PathSeparator))
		if len(parts) > 0 {
			name = parts[len(parts)-1]
		}
	} else {
		name = "unknown"
	}
	return briefProject{Name: name, Version: Version}
}

func collectBriefGit() briefGit {
	// T540: use the shared projectinfo.CollectGitState helper — same
	// source as context.
	g := projectinfo.CollectGitState()
	bg := briefGit{
		Branch:      g.Branch,
		Uncommitted: g.Uncommitted,
		Ahead:       g.Ahead,
		Behind:      g.Behind,
	}
	// T446 (Sprint-38): populate recent commits / branch topology / gaps
	// only in verbose mode to avoid payload bloat. The default brief
	// keeps the same field set as before.
	if verbose {
		bg.RecentCommits = collectBriefRecentCommits()
		bg.Branches = collectBriefBranches()
		bg.BranchGaps = collectBriefBranchGaps()
	}
	return bg
}

// collectBriefRecentCommits returns the latest 5 commits as
// sha/message/date (T446).
func collectBriefRecentCommits() []briefCommit {
	const limit = 5
	out := contextGitOutput("log", "--max-count", fmt.Sprintf("%d", limit),
		"--pretty=format:%h\x1f%s\x1f%cI")
	if out == "" {
		return nil
	}
	lines := strings.Split(out, "\n")
	result := make([]briefCommit, 0, len(lines))
	for _, line := range lines {
		parts := strings.SplitN(line, "\x1f", 3)
		if len(parts) < 3 {
			continue
		}
		result = append(result, briefCommit{
			SHA:     parts[0],
			Message: parts[1],
			Date:    parts[2],
		})
	}
	return result
}

// collectBriefBranches returns the local/remote branch list + HEAD
// (T446).
func collectBriefBranches() *briefBranches {
	b := &briefBranches{}
	b.HEAD = strings.TrimSpace(contextGitOutput("rev-parse", "--abbrev-ref", "HEAD"))
	if localOut := contextGitOutput("for-each-ref", "--format=%(refname:short)", "refs/heads"); localOut != "" {
		for _, l := range strings.Split(localOut, "\n") {
			if l = strings.TrimSpace(l); l != "" {
				b.Local = append(b.Local, l)
			}
		}
	}
	if remoteOut := contextGitOutput("for-each-ref", "--format=%(refname:short)", "refs/remotes"); remoteOut != "" {
		for _, l := range strings.Split(remoteOut, "\n") {
			if l = strings.TrimSpace(l); l != "" && !strings.HasSuffix(l, "/HEAD") {
				b.Remote = append(b.Remote, l)
			}
		}
	}
	return b
}

// collectBriefBranchGaps returns commit-count gaps between key branches
// (T446). Missing main/dev branches yield 0 gaps.
func collectBriefBranchGaps() *briefBranchGaps {
	bg := &briefBranchGaps{}
	countCommits := func(rangeSpec string) int {
		out := contextGitOutput("rev-list", "--count", rangeSpec)
		if out == "" {
			return 0
		}
		n, _ := strconv.Atoi(strings.TrimSpace(out))
		return n
	}
	head := strings.TrimSpace(contextGitOutput("rev-parse", "--abbrev-ref", "HEAD"))
	bg.MainToDev = countCommits("main..dev")
	if head != "" && head != "dev" {
		bg.DevToCurrent = countCommits("dev.." + head)
		bg.CurrentToMain = countCommits(head + "..main")
	}
	return bg
}

// collectBriefBacklogTodoTasks returns up to `limit` backlog (sprint=
// "") + status=todo Tasks sorted by priority/estimate (T446).
func collectBriefBacklogTodoTasks() []briefBacklogTodo {
	const limit = 10
	gs := store.Get()
	if gs == nil {
		return nil
	}
	todoStatus := "todo"
	backlogSprint := ""
	listResult, err := gs.ListTasks(&backlogSprint, &todoStatus)
	if err != nil || listResult == nil {
		return nil
	}
	// Sort by priority (p0 first) + estimate (XL first = "urgent
	// indicator") + task ID.
	priorityRank := map[string]int{"p0": 0, "p1": 1, "p2": 2, "p3": 3}
	estimateRank := map[string]int{"XL": 0, "L": 1, "M": 2, "S": 3, "XS": 4}
	sort.SliceStable(listResult.Tasks, func(i, j int) bool {
		pi := priorityRank[listResult.Tasks[i].Priority]
		pj := priorityRank[listResult.Tasks[j].Priority]
		if pi != pj {
			return pi < pj
		}
		ei := estimateRank[listResult.Tasks[i].Estimate]
		ej := estimateRank[listResult.Tasks[j].Estimate]
		if ei != ej {
			return ei < ej
		}
		return listResult.Tasks[i].TaskID < listResult.Tasks[j].TaskID
	})
	result := make([]briefBacklogTodo, 0, limit)
	for _, t := range listResult.Tasks {
		if len(result) >= limit {
			break
		}
		result = append(result, briefBacklogTodo{
			ID:       t.TaskID,
			Title:    t.Title,
			Type:     t.Type,
			Priority: t.Priority,
			Estimate: t.Estimate,
		})
	}
	return result
}

func collectBriefActiveSprints() []briefSprint {
	rawSprints, err := app.SprintList("active")
	if err != nil {
		return nil
	}
	sprints, _ := rawSprints.([]domain.SprintRecord)
	if len(sprints) == 0 {
		return nil
	}

	root := app.ProjectRoot()

	var result []briefSprint
	for _, s := range sprints {
		bs := briefSprint{ID: s.SprintID, Title: s.Title}

		rawProg, progErr := app.SprintAggregateProgress(s.SprintID)
		if prog, ok := rawProg.(*domain.ProgressResult); progErr == nil && ok && prog != nil {
			bs.Done = prog.Done
			bs.Total = prog.Total
			bs.Percent = prog.Percent
		}

		// T540: extract goal + summary from SPRINT.md.
		if s.FolderPath != "" {
			summary := projectinfo.LoadSprintSummary(filepath.Join(root, s.FolderPath))
			bs.Goal = summary.Goal
			bs.Summary = summary.Summary
		}

		// Tasks belonging to the Sprint.
		sprintPtr := s.SprintID
		raw, listErr := app.TaskList(&sprintPtr, nil)
		if listErr == nil {
			if lr, ok := raw.(*domain.ListResult); ok {
				for _, t := range lr.Tasks {
					bs.Tasks = append(bs.Tasks, briefTask{
						ID:       t.TaskID,
						Title:    t.Title,
						Status:   t.Status,
						Priority: t.Priority,
					})
				}
			}
		}

		result = append(result, bs)
	}
	return result
}

// sprintIDNumberRE extracts N from a sprint-N form (T414 sorting).
var sprintIDNumberRE = regexp.MustCompile(`(\d+)`)

// collectSprintCompletedAuditTimestamps returns the latest
// sprint.completed timestamp from the audit_events table for each
// Sprint (T414 — Python 4-tier sort port).
//
// On failure returns an empty map (caller proceeds to the next
// fallback).
func collectSprintCompletedAuditTimestamps() map[string]string {
	gs := store.Get()
	if gs == nil {
		return nil
	}
	events, err := gs.QueryAuditEvents(ports.AuditQueryFilter{
		EventType: "sprint.completed",
		Limit:     1000,
	})
	if err != nil {
		return nil
	}
	result := make(map[string]string)
	for _, e := range events {
		if e.EntityID != "" && e.Timestamp != "" {
			// MAX(timestamp) emulation: ORDER BY timestamp DESC, so
			// the first hit is the latest.
			if _, exists := result[e.EntityID]; !exists {
				result[e.EntityID] = e.Timestamp
			}
		}
	}
	return result
}

// collectBriefRecentCompleted returns the 3 most recently completed
// Sprints (T414 fix).
//
// Sort key:
// tuple (completed_at, audit_or_mtime, num) compared in descending order.
//   - completed_at: the DB completed_at column (date)
//   - audit_or_mtime: the latest audit_events.sprint.completed
//     timestamp, falling back to SPRINT.md mtime
//   - num: the integer extracted from sprint_id via regex (avoids the
//     sprint-9 vs sprint-10 string-sort bug)
func collectBriefRecentCompleted() []briefCompletedSpr {
	rawSprints, err := app.SprintList("completed")
	if err != nil {
		return nil
	}
	sprints, _ := rawSprints.([]domain.SprintRecord)
	if len(sprints) == 0 {
		return nil
	}

	auditTS := collectSprintCompletedAuditTimestamps()
	root := app.ProjectRoot()

	type sprintWithKey struct {
		rec          domain.SprintRecord
		completed    string
		auditOrMtime string
		num          int
	}
	items := make([]sprintWithKey, 0, len(sprints))
	for _, s := range sprints {
		item := sprintWithKey{rec: s, completed: s.CompletedAt, auditOrMtime: auditTS[s.SprintID]}
		if item.auditOrMtime == "" && s.FolderPath != "" {
			sprintMD := filepath.Join(root, s.FolderPath, "SPRINT.md")
			if info, err := os.Stat(sprintMD); err == nil {
				item.auditOrMtime = strconv.FormatInt(info.ModTime().Unix(), 10)
			}
		}
		// T474: when DB completed_at is empty, inject the audit/mtime
		// timestamp converted to a date string (YYYY-MM-DD) into the
		// primary key. Otherwise a recently completed Sprint missing
		// completed_at (e.g. due to T475 metadata loss) is sorted only
		// by the num tier, falling behind older high-number Sprints.
		if item.completed == "" && item.auditOrMtime != "" {
			if ts, err := strconv.ParseInt(item.auditOrMtime, 10, 64); err == nil {
				item.completed = time.Unix(ts, 0).UTC().Format("2006-01-02")
			}
		}
		if m := sprintIDNumberRE.FindStringSubmatch(s.SprintID); m != nil {
			if n, err := strconv.Atoi(m[1]); err == nil {
				item.num = n
			}
		}
		items = append(items, item)
	}

	sort.SliceStable(items, func(i, j int) bool {
		if items[i].completed != items[j].completed {
			return items[i].completed > items[j].completed
		}
		if items[i].auditOrMtime != items[j].auditOrMtime {
			return items[i].auditOrMtime > items[j].auditOrMtime
		}
		return items[i].num > items[j].num
	})

	if len(items) > briefRecentSprintLimit {
		items = items[:briefRecentSprintLimit]
	}

	var result []briefCompletedSpr
	for _, item := range items {
		cs := briefCompletedSpr{
			ID:          item.rec.SprintID,
			Title:       item.rec.Title,
			CompletedAt: item.rec.CompletedAt,
		}
		rawProg, _ := app.SprintAggregateProgress(item.rec.SprintID)
		if prog, ok := rawProg.(*domain.ProgressResult); ok && prog != nil {
			cs.Done = prog.Done
			cs.Total = prog.Total
		}
		// T540: inject SPRINT.md summary.
		if item.rec.FolderPath != "" {
			summary := projectinfo.LoadSprintSummary(filepath.Join(root, item.rec.FolderPath))
			cs.Goal = summary.Goal
			cs.Summary = summary.Summary
		}
		result = append(result, cs)
	}
	return result
}

func collectBriefTaskSummary() briefTaskSummary {
	summary := briefTaskSummary{}

	raw, err := app.TaskList(nil, nil)
	if err != nil {
		return summary
	}

	if lr, ok := raw.(*domain.ListResult); ok {
		summary.Total = lr.Count
		for _, t := range lr.Tasks {
			switch t.Status {
			case "done":
				summary.Done++
			case "in-progress":
				summary.InProgress++
			default:
				summary.Todo++
			}
		}
	}
	return summary
}

func collectBriefCurrentTask() *briefCurrentTask {
	gs := store.Get()
	if gs == nil {
		return nil
	}

	inProgress := "in-progress"
	result, err := gs.ListTasks(nil, &inProgress)
	if err != nil || result == nil || len(result.Tasks) == 0 {
		return nil
	}
	t := result.Tasks[0]
	ct := &briefCurrentTask{ID: t.TaskID, Title: t.Title}
	if t.Sprint != "" {
		ct.Sprint = t.Sprint
	}
	return ct
}

func collectBriefNextTask() *briefNextTask {
	result, err := app.TaskNext()
	if err != nil {
		return nil
	}

	if ts, ok := result.(*domain.TaskSummary); ok {
		return &briefNextTask{ID: ts.TaskID, Title: ts.Title}
	}
	return nil
}

func collectBriefIssues(activeSprints []briefSprint) []briefIssue {
	// T540: collect via projectinfo.CollectCoreIssues. Shared helper for
	// brief / context.
	ids := make([]string, 0, len(activeSprints))
	for _, s := range activeSprints {
		ids = append(ids, s.ID)
	}
	core := projectinfo.CollectCoreIssues(ids)
	out := make([]briefIssue, 0, len(core))
	for _, c := range core {
		out = append(out, briefIssue{Level: c.Level, Message: c.Message})
	}
	return out
}

// ---------------------------------------------------------------------------
// Text formatting
// ---------------------------------------------------------------------------

// briefDivider is the section separator line (T540 readability).
const briefDivider = "────────────────────────────────────────────────────────────"

func formatBriefText(b *briefData) string {
	var sb strings.Builder

	// Header.
	fmt.Fprintf(&sb, "%s (v%s)\n", b.Project.Name, b.Project.Version)
	sb.WriteString("\n")

	// Git.
	fmt.Fprintf(&sb, "Git: %s (uncommitted: %d, ahead: %d, behind: %d)\n",
		b.Git.Branch, b.Git.Uncommitted, b.Git.Ahead, b.Git.Behind)
	sb.WriteString("\n")

	// Active Sprints.
	sb.WriteString(briefDivider + "\n")
	if len(b.ActiveSprints) > 0 {
		for _, s := range b.ActiveSprints {
			fmt.Fprintf(&sb, "Sprint: %s [active] %d/%d done (%d%%)\n", s.ID, s.Done, s.Total, s.Percent)
			fmt.Fprintf(&sb, "  Title: %s\n", s.Title)
			if s.Goal != "" {
				fmt.Fprintf(&sb, "  Goal: %s\n", s.Goal)
			}
			if len(s.Summary) > 0 {
				sb.WriteString("  Summary:\n")
				for _, line := range s.Summary {
					fmt.Fprintf(&sb, "    %s\n", line)
				}
			}
			if len(s.Tasks) > 0 {
				sb.WriteString("  Tasks:\n")
				for _, t := range s.Tasks {
					icon := " "
					switch t.Status {
					case "done":
						icon = "x"
					case "in-progress":
						icon = ">"
					}
					fmt.Fprintf(&sb, "  [%s] %-6s %s\n", icon, t.ID, t.Title)
				}
			}
			sb.WriteString("\n")
		}
	} else {
		sb.WriteString("Sprint: none (no active sprint)\n\n")
	}

	// Recently completed Sprints (with SPRINT.md summary).
	if len(b.RecentCompleted) > 0 {
		sb.WriteString(briefDivider + "\n")
		sb.WriteString("Recently completed sprints:\n\n")
		for _, s := range b.RecentCompleted {
			fmt.Fprintf(&sb, "  %s  %s  %d/%d done\n", s.ID, s.CompletedAt, s.Done, s.Total)
			fmt.Fprintf(&sb, "    Title: %s\n", s.Title)
			if s.Goal != "" {
				fmt.Fprintf(&sb, "    Goal: %s\n", s.Goal)
			}
			if len(s.Summary) > 0 {
				for _, line := range s.Summary {
					fmt.Fprintf(&sb, "    · %s\n", line)
				}
			}
			sb.WriteString("\n")
		}
	}

	// 10 most recently completed P2+ Tasks.
	if len(b.RecentP2Tasks) > 0 {
		sb.WriteString(briefDivider + "\n")
		fmt.Fprintf(&sb, "Recently completed major Tasks (p0~p2, up to %d):\n", briefRecentP2TaskLimit)
		for _, t := range b.RecentP2Tasks {
			sprintPart := ""
			if t.Sprint != "" {
				sprintPart = " [" + t.Sprint + "]"
			}
			fmt.Fprintf(&sb, "  %s  %s  %-8s %s%s\n    %s\n",
				t.ID, t.CompletedAt, t.Priority, t.Type, sprintPart, t.Title)
		}
		sb.WriteString("\n")
	}

	// Task summary.
	sb.WriteString(briefDivider + "\n")
	fmt.Fprintf(&sb, "Task summary: %d done / %d total (in-progress: %d, todo: %d)\n",
		b.TaskSummary.Done, b.TaskSummary.Total, b.TaskSummary.InProgress, b.TaskSummary.Todo)
	sb.WriteString("\n")

	// Current Task + Next.
	if b.CurrentTask != nil {
		fmt.Fprintf(&sb, "Current: %s %s\n", b.CurrentTask.ID, b.CurrentTask.Title)
	}
	if b.NextTask != nil {
		fmt.Fprintf(&sb, "Next: %s %s\n", b.NextTask.ID, b.NextTask.Title)
	}
	if b.CurrentTask == nil && b.NextTask == nil {
		sb.WriteString("Current: none\n")
	}
	sb.WriteString("\n")

	// Issues.
	sb.WriteString(briefDivider + "\n")
	if len(b.Issues) > 0 {
		sb.WriteString("Issues:\n")
		for _, i := range b.Issues {
			fmt.Fprintf(&sb, "  [%s] %s\n", i.Level, i.Message)
		}
	} else {
		sb.WriteString("Issues: none\n")
	}

	return strings.TrimRight(sb.String(), "\n")
}

// ---------------------------------------------------------------------------
// init
// ---------------------------------------------------------------------------

func init() {
	rootCmd.AddCommand(briefCmd)
}
