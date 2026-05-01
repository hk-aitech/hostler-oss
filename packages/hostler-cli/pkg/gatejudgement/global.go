// Package gatejudgement — global helper.
//
// Provides a package-level singleton plus sampling so the four call paths
// (harness/task/sprint/rules) can record without each injecting a store.
// Tests can isolate the JSONL path via InitForTest.
package gatejudgement

import (
	"math/rand"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/ports"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/envalias"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/fileutil"
)

// Global singleton — one per process.
var (
	globalStore     ports.GateJudgementLog
	globalStoreOnce sync.Once
	// disabled — for tests and opt-out.
	disabledFlag atomic.Bool
)

// Constants / env vars.
const (
	// PassSampleRateEnvVar — sampling ratio for PASS verdict records (0.0~1.0). Default 0.1.
	// WARN / BLOCK / HARD_BLOCK are not sampled (always recorded).
	PassSampleRateEnvVar = "HSTL_GATE_JUDGEMENT_PASS_SAMPLE_RATE"

	// DefaultPassSampleRate — default 10% PASS sampling.
	DefaultPassSampleRate = 0.1

	// DisabledEnvVar — "off" turns Record into a no-op before invocation.
	DisabledEnvVar = "HSTL_GATE_JUDGEMENT_LOG"
)

// GlobalStore returns the global store. Lazy-initialises a projectRoot-based
// JSONL adapter when missing.
// Discovers the path via fileutil.GetProjectRoot() (HSTL_PROJECT_ROOT ->
// git toplevel -> cwd). A plain cwd fallback would create a fresh
// state directory from subdirectories and fragment the data, so the
// lookup is unified through the git toplevel.
func GlobalStore() ports.GateJudgementLog {
	globalStoreOnce.Do(func() {
		globalStore = NewJSONLStore(fileutil.GetProjectRoot())
	})
	return globalStore
}

// SetGlobalStore swaps the store (for tests / DI).
func SetGlobalStore(s ports.GateJudgementLog) {
	globalStore = s
	globalStoreOnce.Do(func() {}) // consume Once to block lazy init
}

// ResetGlobalStore resets the singleton (used by tests).
func ResetGlobalStore() {
	globalStore = nil
	globalStoreOnce = sync.Once{}
	disabledFlag.Store(false)
}

// RecordJudgement records a judgement in the global store. Handles
// sampling, disabled, and errors silently.
// Callers do not need to worry about error propagation (observability
// logger principle — failures here must not block the actual validation
// path).
//
// PASS is sampled by HSTL_GATE_JUDGEMENT_PASS_SAMPLE_RATE; WARN+ is recorded in full.
func RecordJudgement(r ports.JudgementRecord) {
	if isDisabled() {
		return
	}
	if !shouldRecord(r.Verdict) {
		return
	}
	store := GlobalStore()
	if store == nil {
		return
	}
	_ = store.Record(r) // silent — observability log failures must not block the caller
}

// isDisabled checks env-based opt-out + atomic flag.
func isDisabled() bool {
	if disabledFlag.Load() {
		return true
	}
	return envalias.Lookup("GATE_JUDGEMENT_LOG") == "off"
}

// SetDisabled disables recording (test or operational use).
func SetDisabled(v bool) {
	disabledFlag.Store(v)
}

// shouldRecord decides whether to record based on the verdict.
// WARN / BLOCK / HARD_BLOCK always; PASS is sampled; SKIP has no state -> skip.
func shouldRecord(v ports.Verdict) bool {
	switch v {
	case ports.VerdictWarn, ports.VerdictBlock, ports.VerdictHardBlock:
		return true
	case ports.VerdictPass:
		return rand.Float64() < passSampleRate()
	case ports.VerdictSkip:
		return false
	}
	return true
}

// passSampleRate parses the env value or returns the default.
func passSampleRate() float64 {
	if v := envalias.Lookup("GATE_JUDGEMENT_PASS_SAMPLE_RATE"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f >= 0 && f <= 1 {
			return f
		}
	}
	return DefaultPassSampleRate
}
