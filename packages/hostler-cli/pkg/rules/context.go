package rules

// Phase is the lifecycle point at which a Rule executes.
type Phase string

const (
	PhaseTaskStart      Phase = "task:start"
	PhaseTaskComplete   Phase = "task:complete"
	PhaseSprintStart    Phase = "sprint:start"
	PhaseSprintComplete Phase = "sprint:complete"
	PhasePreCommit      Phase = "precommit"
	PhaseCommitMsg      Phase = "commit-msg" // git commit-msg hook
	PhaseBashExec       Phase = "bash:exec"
)

// RuleContext.Extra key constants.
// Rule implementations look up values by these keys before performing a type
// assertion.
const (
	// ExtraKeyCommitMsg holds the commit message body injected by the
	// commit-msg hook (string).
	ExtraKeyCommitMsg = "commit_msg"
	// ExtraKeyStagedDiff holds the full staged-diff text injected by the
	// pre-commit hook (string).
	ExtraKeyStagedDiff = "staged_diff"
)

// RuleContext is the execution context passed to Rule.Check.
// A Rule reads only the fields it needs from the context (no direct I/O).
type RuleContext struct {
	Phase        Phase         // execution phase
	TaskID       string        // Task ID (when applicable)
	SprintID     string        // Sprint ID (when applicable)
	SelfFilePath string        // path of the file being validated (used for self-exclusion)
	GitDiff      []string      // list of changed file paths
	ProjectRoot  string        // absolute path to the project root
	Config       *EngineConfig // resolved Config (after cascade)

	// Extra holds rule-specific extra data (avoids dynamic dispatch in
	// favour of type assertions).
	Extra map[string]any
}
