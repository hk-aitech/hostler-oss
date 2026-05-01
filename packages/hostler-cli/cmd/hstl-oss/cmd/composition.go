// Package cmd - composition root.
//
// ADR-001 Gate B2 goal: "zero SQLite calls outside adapters."
// hstl is a cobra-based CLI where each command owns its own RunE. Moving
// initialization fully into main.go would force DB initialization costs onto
// commands that do not need a DB (version/manifest/help). The composition
// root therefore stays in the cmd package but is split out from domain files
// (task.go) into its own file.
//
// Restated Gate B2 metric:
//   - zero `db.GetDB()` / `sql.` calls outside `pkg/*` (non-composition code).
//   - composition.go is the intentional exception - adapter wiring only.
//   - grep check: zero hits outside adapters and outside composition.go.
//
// In a future Phase D (CLI conversion) this can move to internal/app/
// service wiring.
package cmd

import (
	"os"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/adapters/sqlite"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/app"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/apperr"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/audit"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/backlog"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/ceremony"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/config"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/db"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/fileutil"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/gatejudgement"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/harness"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/hooks"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/id"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/kb"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/rules"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/sprint"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/store"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/task"
)

// Inject app.Services at process startup so cmd/hstl-oss/cmd/*.go can
// reference internal/app only, gradually meeting Gate D2. Initial injection
// covers a single LayoutResolver port; subsequent tasks add
// IdentifierAllocator / FrontmatterCodec / KBStore here.
func init() {
	app.Init(&app.Services{
		Layout:    fileutil.NewFSLayout(),
		IDAlloc:   id.NewIDAllocator(),
		FMCodec:   fileutil.NewYAMLCodec(),
		FMFile:    fileutil.NewFSFrontmatterFile(),
		KB:        kb.NewKBFSStore(),
		AuditFile: audit.NewFSRestoreAdapter(),
		Harness:   harness.NewServiceAdapter(),
		Sprint:    sprint.NewServiceAdapter(),
		Task:      task.NewServiceAdapter(),
		Backlog:   backlog.NewServiceAdapter(),
		GateLog:   gatejudgement.NewServiceAdapter(),
		Ceremony:  ceremony.NewServiceAdapter(),
		Config:    config.NewServiceAdapter(),
		Rules:     rules.NewServiceAdapter(),
		// Audit is injected later in initDB() through sqliteStore.
	})
}

// initDB opens the SQLite DB connection and injects the ports.GraphStore +
// ports.IDStore adapters into the store package. The caller (RunE) is
// responsible for error handling.
//
// This function is the composition root - the only place where adapters are
// wired by concrete type. Calling `sqlite.New(db.GetDB())` from anywhere
// else violates Gate B2.
func initDB() error {
	if err := db.InitDB(); err != nil {
		return err
	}
	sqliteStore := sqlite.New(db.GetDB())
	store.Init(sqliteStore, sqliteStore)
	// sqliteStore also implements ports.AuditLog, so we additionally inject
	// the same instance into app.Services.Audit.
	app.UseAudit(sqliteStore)
	return nil
}

// mustInitDB is a shared helper for RunE: when DB initialization fails it
// emits Out.Error and exits with os.Exit(exitError).
//
// After a successful DB init it also checks whether the pre-commit hook is
// installed and emits a one-shot stderr warning per session (disable with
// HSTL_HOOK_WARNING=off). Prevents the silent-failure scenario where the
// HMAC gate is disabled when an external project has hstl installed but no
// git hooks (observed downstream on 2026-04-17).
func mustInitDB() {
	if err := initDB(); err != nil {
		// Use apperr.CategoryDBError.String() instead of a raw DB_INIT_ERROR literal.
		Out.Error("DB initialization failed: "+err.Error(), apperr.CategoryDBError.String(), "")
		os.Exit(exitError)
	}
	hooks.CheckAndWarnOnce("")
}
