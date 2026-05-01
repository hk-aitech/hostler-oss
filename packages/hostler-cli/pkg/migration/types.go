// Package migration return-type declarations.
// Provides compile-time type safety in place of map[string]any.
package migration

// BootstrapResponse is the return type of BuildBootstrapResponse.
type BootstrapResponse struct {
	Status          string          `json:"status"`
	Reason          string          `json:"reason"`
	RegistryPath    string          `json:"registry_path"`
	LastID          int             `json:"last_id"`
	TotalCount      int             `json:"total_count"`
	Message         string          `json:"message"`
	SuggestedAction BootstrapAction `json:"suggested_action"`
}

// BootstrapAction describes the recommended bootstrap follow-up.
type BootstrapAction struct {
	Tool string              `json:"tool"`
	Args BootstrapActionArgs `json:"args"`
}

// BootstrapActionArgs lists the project_migrate call arguments.
type BootstrapActionArgs struct {
	SourceRepo string `json:"source_repo"`
	DryRun     bool   `json:"dry_run"`
}

// MigrateResult is the return type of MigrateProject.
type MigrateResult struct {
	DryRun   bool               `json:"dry_run"`
	Source   string             `json:"source"`
	Tasks    MigrateTaskStats   `json:"tasks"`
	Sprints  MigrateSprintStats `json:"sprints"`
	Counters MigrateCounters    `json:"counters"`
	Warnings []string           `json:"warnings"`
	Errors   []string           `json:"errors"`
}

// MigrateTaskStats is the Task migration statistics block.
type MigrateTaskStats struct {
	Scanned        int            `json:"scanned"`
	RegisteredOnly int            `json:"registered_only"`
	Loaded         int            `json:"loaded"`
	Skipped        int            `json:"skipped"`
	SkippedReasons map[string]int `json:"skipped_reasons"`
	TotalToLoad    int            `json:"total_to_load"`
	Orphaned       int            `json:"orphaned,omitempty"`
}

// MigrateSprintStats is the Sprint migration statistics block.
type MigrateSprintStats struct {
	Scanned    int `json:"scanned"`
	Active     int `json:"active"`
	Backlog    int `json:"backlog"`
	Completed  int `json:"completed"`
	Superseded int `json:"superseded"`
	Loaded     int `json:"loaded"`
}

// MigrateCounters is the ID-counter statistics block.
type MigrateCounters struct {
	Task int `json:"task"`
}
