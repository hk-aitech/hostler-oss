package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/spf13/cobra"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/app"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/brand"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/domain"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/output"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/db"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/envalias"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/projectinfo"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/store"
)

// contextTicket holds the --ticket value for context-ack flow.
var contextTicket string

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

// contextRulesText is a short summary of the core rule keywords.
const contextRulesText = "use const, use jq, Command First, 1Task=1Commit"

// ---------------------------------------------------------------------------
// context command
// ---------------------------------------------------------------------------

var contextCmd = &cobra.Command{
	Use:   "context",
	Short: "Project context for AI/subagents (compact text)",
	Long: `Collects the current project state in real time and returns it as compact text.
The output fits in roughly 200 tokens and can be embedded directly in subagent prompts.
With --output json the same payload is emitted as structured data.`,
	Example: brand.Examplef(
		`context`,
		`context -o json | jq '.sprint'`,
	),
	RunE: func(cmd *cobra.Command, args []string) error {
		mustInitDB()
		defer db.Close()

		ctx := collectContext()

		// When --ticket is provided, compute the canonical JSON SHA256 and store an ack.
		var ackHash string
		if contextTicket != "" {
			h, size, err := computeContextHash(ctx)
			if err == nil {
				if gs := store.Get(); gs != nil {
					if ierr := gs.InsertContextAck(contextTicket, h, size); ierr == nil {
						ackHash = h
					}
				}
			}
		}

		if outputFormat == "json" {
			if ackHash != "" {
				Out.Print(map[string]any{
					"context": ctx,
					"ack": map[string]any{
						"ticket": contextTicket,
						"hash":   ackHash,
					},
				})
				return nil
			}
			Out.Print(ctx)
			return nil
		}

		// text mode: compact 5-line output
		Out.Print(formatContextText(ctx))
		if ackHash != "" {
			fmt.Fprintf(output.Stderr(), "[context-ack ticket=%s hash=sha256:%s]\n", contextTicket, ackHash)
		}
		return nil
	},
}

// computeContextHash hashes the canonical JSON form of the context with SHA256.
func computeContextHash(ctx *contextData) (string, int, error) {
	data, err := json.Marshal(ctx)
	if err != nil {
		return "", 0, err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), len(data), nil
}

// ---------------------------------------------------------------------------
// Data structures
// ---------------------------------------------------------------------------

// contextData is the collection result for the context command.
type contextData struct {
	Project  contextProject `json:"project"`
	Sprint   *contextSprint `json:"sprint"`
	Task     *contextTask   `json:"task"`
	NextTask *contextNext   `json:"next_task"`
	Rules    string         `json:"rules"`
	// Issues is a list of detailed messages; previous int callers should use len(Issues).
	Issues []contextIssue `json:"issues"`
}

// contextIssue represents a single warning in the session context.
// Matches the briefIssue format from brief for consistency.
type contextIssue struct {
	Level   string `json:"level"`
	Message string `json:"message"`
}

type contextProject struct {
	Name        string `json:"name"`
	Branch      string `json:"branch"`
	Uncommitted int    `json:"uncommitted"`
	Ahead       int    `json:"ahead"`
	Behind      int    `json:"behind"`
}

type contextSprint struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Done    int    `json:"done"`
	Total   int    `json:"total"`
	Percent int    `json:"percent"`
}

type contextTask struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
}

type contextNext struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// ---------------------------------------------------------------------------
// Collection logic
// ---------------------------------------------------------------------------

func collectContext() *contextData {
	ctx := &contextData{
		Rules: contextRulesText,
	}

	var wg sync.WaitGroup

	// goroutine 1: Git
	wg.Add(1)
	go func() {
		defer wg.Done()
		ctx.Project = collectContextGit()
	}()

	// goroutine 2: Sprint + Task (sequential DB calls)
	wg.Add(1)
	go func() {
		defer wg.Done()
		ctx.Sprint = collectContextSprint()
		ctx.Task = collectContextCurrentTask()
		ctx.NextTask = collectContextNextTask()
		ctx.Issues = collectContextIssues()
	}()

	wg.Wait()
	return ctx
}

func collectContextGit() contextProject {
	// Uses the shared projectinfo.CollectGitState helper - same source as brief.
	g := projectinfo.CollectGitState()
	return contextProject{
		Name:        contextProjectName(),
		Branch:      g.Branch,
		Uncommitted: g.Uncommitted,
		Ahead:       g.Ahead,
		Behind:      g.Behind,
	}
}

// contextProjectName determines the project name.
// Priority: HSTL_PROJECT_NAME env var -> project-config.yaml project.key -> git root folder name.
func contextProjectName() string {
	// 1. environment variable
	if name := envalias.Lookup("PROJECT_NAME"); name != "" {
		return name
	}

	// 2. project.key from project-config.yaml
	if name := app.ConfigGetProjectKey(); name != "" {
		return name
	}

	// 3. git root folder name
	root := contextGitOutput("rev-parse", "--show-toplevel")
	if root != "" {
		parts := strings.Split(root, string(os.PathSeparator))
		if len(parts) > 0 {
			return parts[len(parts)-1]
		}
	}
	return "unknown"
}

func contextGitOutput(args ...string) string {
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func collectContextSprint() *contextSprint {
	rawSprints, err := app.SprintList("active")
	if err != nil {
		return nil
	}
	sprints, _ := rawSprints.([]domain.SprintRecord)
	if len(sprints) == 0 {
		return nil
	}

	s := sprints[0]
	rawProg, err := app.SprintAggregateProgress(s.SprintID)
	if err != nil {
		return &contextSprint{ID: s.SprintID, Title: s.Title}
	}
	prog, _ := rawProg.(*domain.ProgressResult)
	if prog == nil {
		return &contextSprint{ID: s.SprintID, Title: s.Title}
	}

	return &contextSprint{
		ID:      s.SprintID,
		Title:   s.Title,
		Done:    prog.Done,
		Total:   prog.Total,
		Percent: prog.Percent,
	}
}

func collectContextCurrentTask() *contextTask {
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
	return &contextTask{ID: t.TaskID, Title: t.Title, Status: "in-progress"}
}

func collectContextNextTask() *contextNext {
	result, err := app.TaskNext()
	if err != nil {
		return nil
	}

	if ts, ok := result.(*domain.TaskSummary); ok {
		return &contextNext{ID: ts.TaskID, Title: ts.Title}
	}
	return nil
}

// collectContextIssues gathers warnings to surface at session start.
// Uses the shared projectinfo.CollectCoreIssues helper - same source as brief.
func collectContextIssues() []contextIssue {
	// Collect the list of active sprint IDs first.
	var activeIDs []string
	if rawSprints, err := app.SprintList("active"); err == nil {
		if sprints, ok := rawSprints.([]domain.SprintRecord); ok {
			for _, s := range sprints {
				activeIDs = append(activeIDs, s.SprintID)
			}
		}
	}
	core := projectinfo.CollectCoreIssues(activeIDs)
	out := make([]contextIssue, 0, len(core))
	for _, c := range core {
		out = append(out, contextIssue{Level: c.Level, Message: c.Message})
	}
	return out
}

// ---------------------------------------------------------------------------
// Text formatting
// ---------------------------------------------------------------------------

func formatContextText(ctx *contextData) string {
	var sb strings.Builder

	// [project]
	sb.WriteString(fmt.Sprintf("[project] %s branch:%s uncommitted:%d ahead:%d",
		ctx.Project.Name, ctx.Project.Branch, ctx.Project.Uncommitted, ctx.Project.Ahead))
	if ctx.Project.Behind > 0 {
		sb.WriteString(fmt.Sprintf(" behind:%d", ctx.Project.Behind))
	}
	sb.WriteString("\n")

	// [sprint]
	if ctx.Sprint != nil {
		sb.WriteString(fmt.Sprintf("[sprint] %s \"%s\" %d/%d done (%d%%)\n",
			ctx.Sprint.ID, ctx.Sprint.Title, ctx.Sprint.Done, ctx.Sprint.Total, ctx.Sprint.Percent))
	} else {
		sb.WriteString("[sprint] none\n")
	}

	// [task]
	if ctx.Task != nil {
		nextPart := "none"
		if ctx.NextTask != nil {
			nextPart = ctx.NextTask.ID
		}
		sb.WriteString(fmt.Sprintf("[task] %s \"%s\" (%s) | next: %s\n",
			ctx.Task.ID, ctx.Task.Title, ctx.Task.Status, nextPart))
	} else if ctx.NextTask != nil {
		sb.WriteString(fmt.Sprintf("[task] none | next: %s \"%s\"\n",
			ctx.NextTask.ID, ctx.NextTask.Title))
	} else {
		sb.WriteString("[task] none\n")
	}

	// [rules]
	sb.WriteString(fmt.Sprintf("[rules] %s\n", ctx.Rules))

	// [issues] - emit one line per detailed warning message.
	if len(ctx.Issues) > 0 {
		for _, iss := range ctx.Issues {
			sb.WriteString(fmt.Sprintf("[issues] [%s] %s\n", iss.Level, iss.Message))
		}
	} else {
		sb.WriteString("[issues] none\n")
	}

	return strings.TrimRight(sb.String(), "\n")
}

// ---------------------------------------------------------------------------
// init
// ---------------------------------------------------------------------------

func init() {
	contextCmd.Flags().StringVar(&contextTicket, "ticket", "", "work ticket — when set, hashes the context and stores an ack in the DB")
	rootCmd.AddCommand(contextCmd)
}
