package harness

import (
	"os"
	"testing"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/gatejudgement"
)

// TestMain — T201: prevent pollution of the gatejudgement subdirectory.
// Blocks pkg/harness tests from creating .hostler/gate-judgement.jsonl in the
// pkg cwd via RecordJudgement. Tests that exercise the store itself use a
// separate SetGlobalStore (calling store.Record directly), so they are
// unaffected.
func TestMain(m *testing.M) {
	gatejudgement.SetDisabled(true)
	os.Exit(m.Run())
}
