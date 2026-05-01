// Package gatejudgement — Store factory.
//
// Picks between JSONLStore and SQLiteStore based on an env var.
// - HSTL_GATE_JUDGEMENT_STORE=jsonl  (default, preserves the existing behaviour)
// - HSTL_GATE_JUDGEMENT_STORE=sqlite (switch to SQLite)
//
// This factory is the single entry point for GlobalStore() initialisation.
// NewJSONLStore and NewSQLiteStore are still exported for direct use in
// tests and DI.
package gatejudgement

import (
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/ports"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/db"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/envalias"
)

// StoreEnvVar — factory selector env var.
const StoreEnvVar = "HSTL_GATE_JUDGEMENT_STORE"

// StoreKind enumeration values.
const (
	StoreKindJSONL  = "jsonl"
	StoreKindSQLite = "sqlite"
)

// NewStore is the env-driven store factory. JSONL by default.
// projectRoot is used to resolve the JSONL path (SQLite uses the pkg/db global instance).
func NewStore(projectRoot string) ports.GateJudgementLog {
	kind := envalias.Lookup("GATE_JUDGEMENT_STORE")
	if kind == StoreKindSQLite {
		return newSQLiteStoreOrFallback(projectRoot)
	}
	return NewJSONLStore(projectRoot)
}

// newSQLiteStoreOrFallback falls back to JSONL when SQLite construction fails.
// Gate logs are an observability-only path, so failures here must not
// block the actual validation path.
func newSQLiteStoreOrFallback(projectRoot string) ports.GateJudgementLog {
	dbConn := db.GetDB()
	if dbConn == nil {
		// pkg/db not initialised (e.g. standalone CLI path) -> JSONL fallback
		return NewJSONLStore(projectRoot)
	}
	s, err := NewSQLiteStore(dbConn)
	if err != nil {
		return NewJSONLStore(projectRoot)
	}
	return s
}
