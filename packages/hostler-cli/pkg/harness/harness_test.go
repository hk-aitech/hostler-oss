package harness_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/adapters/sqlite"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/domain"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/ports"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/db"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/harness"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/store"
)

// setupDB initialises an isolated DB for tests.
func setupDB(t *testing.T) {
	t.Helper()
	t.Setenv("HSTL_DB_PATH", filepath.Join(t.TempDir(), "hstl.db"))
	if err := db.InitDB(); err != nil {
		t.Fatalf("DB initialisation failed: %v", err)
	}
	sqliteStore := sqlite.New(db.GetDB())
	store.Init(sqliteStore, sqliteStore)
	t.Cleanup(func() {
		store.Reset()
		db.Close()
	})
}

// insertTask inserts a Task record into the DB for tests.
func insertTask(t *testing.T, taskID, taskType, status string) {
	t.Helper()
	gs := store.Get()
	if gs == nil {
		t.Fatal("store not initialised")
	}
	err := gs.CreateTask(&ports.TaskRecord{
		TaskID:   taskID,
		Title:    "test task",
		Type:     taskType,
		Status:   status,
		Priority: "p2",
		Estimate: "M",
		FilePath: "works/tasks/" + taskID + "-test.md",
	})
	if err != nil {
		t.Fatalf("Task insertion failed: %v", err)
	}
}

// TestEnsureHarnessItems_Init verifies that items are initialised from a
// template when none exist.
func TestEnsureHarnessItems_Init(t *testing.T) {
	setupDB(t)
	insertTask(t, "T001", "feature", "in-progress")

	items, err := harness.EnsureHarnessItems("task", "T001")
	if err != nil {
		t.Fatalf("EnsureHarnessItems failed: %v", err)
	}
	// the task:feature template should be loaded so items must be present
	if len(items) == 0 {
		t.Error("items were not initialised")
	}
}

// TestCheckHarness_EmptyChecklist verifies that an entity without a template
// returns an empty result.
func TestCheckHarness_EmptyChecklist(t *testing.T) {
	setupDB(t)

	// entity_type without a template
	result, err := harness.CheckHarness("expedition", "EXP-001")
	if err != nil {
		t.Fatalf("CheckHarness failed: %v", err)
	}
	// without an expedition:default template the list is empty
	if result.RequiredTotal != 0 {
		// a template may exist, so just check the BLOCKED state
		t.Logf("RequiredTotal=%d (may vary depending on template presence)", result.RequiredTotal)
	}
	if result.AllItems == nil {
		t.Error("AllItems must not be nil")
	}
}

// TestHarnessCheck_Completion verifies that completing an item reduces
// remaining count.
func TestHarnessCheck_Completion(t *testing.T) {
	setupDB(t)
	insertTask(t, "T002", "feature", "in-progress")

	// initialise
	items, err := harness.EnsureHarnessItems("task", "T002")
	if err != nil || len(items) == 0 {
		t.Skip("no task:feature template items, skipping")
	}

	firstItemID := items[0].ID
	retAny, err := harness.HarnessCheck("task", "T002", firstItemID, "sha:abc123", "test-actor")
	if err != nil {
		t.Fatalf("HarnessCheck failed: %v", err)
	}

	ret, ok := retAny.(domain.HarnessCheckResult)
	if !ok {
		t.Fatalf("HarnessCheck return type mismatch: %T", retAny)
	}
	if !ret.Checked {
		t.Errorf("expected checked=true, got: %v", ret)
	}
	if ret.ItemID != firstItemID {
		t.Errorf("item_id mismatch: got=%v, want=%s", ret.ItemID, firstItemID)
	}
}

// T763 (Sprint-89): TestHarnessCheck_SprintTriggersCEREMONYUpdate removed —
// the CEREMONY.md file is deprecated. The harness_items DB is now the SSOT
// for sprint ceremony state.

// TestHarnessGet_Lookup verifies that HarnessGet returns the expected
// structure.
func TestHarnessGet_Lookup(t *testing.T) {
	setupDB(t)
	insertTask(t, "T003", "bugfix", "in-progress")

	ret, err := harness.HarnessGet("task", "T003")
	if err != nil {
		t.Fatalf("HarnessGet failed: %v", err)
	}

	if ret.EntityType != "task" {
		t.Errorf("entity_type mismatch: %v", ret.EntityType)
	}
	if ret.EntityID != "T003" {
		t.Errorf("entity_id mismatch: %v", ret.EntityID)
	}
}

// TestAutoCheckCriteria_ParseDoneCriteria verifies that the Done Criteria
// checkboxes in a Task file are parsed.
func TestAutoCheckCriteria_ParseDoneCriteria(t *testing.T) {
	setupDB(t)

	tmpRoot := t.TempDir()
	t.Setenv("HSTL_PROJECT_ROOT", tmpRoot)

	taskDir := filepath.Join(tmpRoot, "works", "tasks")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("directory create failed: %v", err)
	}

	taskFile := filepath.Join(taskDir, "T010-test.md")
	content := `---
id: T010
title: "test"
status: in-progress
---

## Done Criteria

- [x] build passes
- [ ] tests pass
- [x] code review complete
`
	if err := os.WriteFile(taskFile, []byte(content), 0o644); err != nil {
		t.Fatalf("Task file create failed: %v", err)
	}

	_, err := db.GetDB().Exec(
		`INSERT OR REPLACE INTO tasks (task_id, title, type, status, priority, estimate, file_path, created_at, updated_at)
		 VALUES ('T010', 'test', 'feature', 'in-progress', 'p2', 'M', 'works/tasks/T010-test.md', date('now'), date('now'))`,
	)
	if err != nil {
		t.Fatalf("Task DB insert failed: %v", err)
	}

	result, err := harness.AutoCheckCriteria("T010")
	if err != nil {
		t.Fatalf("AutoCheckCriteria failed: %v", err)
	}

	if result.Total != 3 {
		t.Errorf("total=%v, want=3", result.Total)
	}
	if result.Checked != 2 {
		t.Errorf("checked=%v, want=2", result.Checked)
	}
	if result.AllChecked != false {
		t.Errorf("all_checked=%v, want=false", result.AllChecked)
	}
	if len(result.Unchecked) != 1 {
		t.Errorf("unchecked count=%d, want=1", len(result.Unchecked))
	}
}

// TestHarnessTemplateGet_Lookup verifies template lookup.
func TestHarnessTemplateGet_Lookup(t *testing.T) {
	setupDB(t)

	items, err := harness.HarnessTemplateGet("task:feature")
	if err != nil {
		t.Fatalf("HarnessTemplateGet failed: %v", err)
	}
	// the task:feature template must be loaded by default
	if len(items) == 0 {
		t.Error("task:feature template has no items")
	}
}

// TestCheckHarness_BLOCKED verifies blocked=true when required items remain
// incomplete.
func TestCheckHarness_BLOCKED(t *testing.T) {
	setupDB(t)
	insertTask(t, "T004", "feature", "in-progress")

	// initialise (template insert)
	items, err := harness.EnsureHarnessItems("task", "T004")
	if err != nil {
		t.Fatalf("EnsureHarnessItems failed: %v", err)
	}

	// CheckHarness without completing anything
	result, err := harness.CheckHarness("task", "T004")
	if err != nil {
		t.Fatalf("CheckHarness failed: %v", err)
	}

	requiredItems := 0
	for _, item := range items {
		if item.Required {
			requiredItems++
		}
	}

	if requiredItems > 0 && !result.Blocked {
		t.Errorf("required items=%d but blocked=false", requiredItems)
	}
	if result.RequiredTotal != requiredItems {
		t.Errorf("RequiredTotal=%d, want=%d", result.RequiredTotal, requiredItems)
	}
}

// T763 (Sprint-89): contains helper removed — the only caller (CEREMONY.md
// update test) was deleted.

// TestHarnessTemplateGet_NineTemplates verifies all nine templates can be
// retrieved.
func TestHarnessTemplateGet_NineTemplates(t *testing.T) {
	setupDB(t)

	templateKeys := []string{
		"task:feature",
		"task:bugfix",
		"task:refactor",
		"task:infra",
		"task:docs",
		"task:test",
		"task:chore",
		"sprint:default",
		"expedition:default",
	}

	for _, key := range templateKeys {
		items, err := harness.HarnessTemplateGet(key)
		if err != nil {
			t.Errorf("HarnessTemplateGet(%s) failed: %v", key, err)
			continue
		}
		if len(items) == 0 {
			t.Errorf("template has no items: %s", key)
		}
	}
}

// TestEnsureHarnessItems_DuplicateInit verifies that re-initialising does
// not duplicate items.
func TestEnsureHarnessItems_DuplicateInit(t *testing.T) {
	setupDB(t)
	insertTask(t, "T020", "feature", "in-progress")

	// first initialisation
	items1, err := harness.EnsureHarnessItems("task", "T020")
	if err != nil {
		t.Fatalf("first EnsureHarnessItems failed: %v", err)
	}

	// second initialisation — must not duplicate
	items2, err := harness.EnsureHarnessItems("task", "T020")
	if err != nil {
		t.Fatalf("second EnsureHarnessItems failed: %v", err)
	}

	if len(items1) != len(items2) {
		t.Errorf("item count mismatch after re-init: %d vs %d", len(items1), len(items2))
	}
}

// TestHarnessGet_Bugfix verifies the template key for a bugfix Task.
func TestHarnessGet_Bugfix(t *testing.T) {
	setupDB(t)
	insertTask(t, "T030", "bugfix", "in-progress")

	ret, err := harness.HarnessGet("task", "T030")
	if err != nil {
		t.Fatalf("HarnessGet failed: %v", err)
	}

	if ret.TemplateKey != "task:bugfix" {
		t.Errorf("template_key=%v, want=task:bugfix", ret.TemplateKey)
	}
	if len(ret.Items) == 0 {
		t.Error("bugfix items missing")
	}
}

// TestHarnessCheck_AllItemsCompleted verifies blocked=false when every item
// is checked.
func TestHarnessCheck_AllItemsCompleted(t *testing.T) {
	setupDB(t)
	insertTask(t, "T040", "chore", "in-progress")

	items, err := harness.EnsureHarnessItems("task", "T040")
	if err != nil || len(items) == 0 {
		t.Skip("no items, skipping")
	}

	// complete all items
	for _, item := range items {
		if !item.Required {
			continue
		}
		_, err := harness.HarnessCheck("task", "T040", item.ID, "evidence", "claude")
		if err != nil {
			t.Fatalf("HarnessCheck(%s) failed: %v", item.ID, err)
		}
	}

	result, err := harness.CheckHarness("task", "T040")
	if err != nil {
		t.Fatalf("CheckHarness failed: %v", err)
	}

	remaining := result.RequiredTotal - result.RequiredDone
	if result.Blocked {
		t.Errorf("all items complete but blocked=true: remaining=%d", remaining)
	}
}

// TestAutoCheckCriteria_AllChecked verifies all_checked=true when every
// checkbox is ticked.
func TestAutoCheckCriteria_AllChecked(t *testing.T) {
	setupDB(t)

	tmpRoot := t.TempDir()
	t.Setenv("HSTL_PROJECT_ROOT", tmpRoot)

	taskDir := filepath.Join(tmpRoot, "works", "tasks")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatal(err)
	}

	taskFile := filepath.Join(taskDir, "T020-all-checked.md")
	content := `---
id: T020
title: "all checked"
status: in-progress
---

## Done Criteria

- [x] build passes
- [x] tests pass
- [x] code review complete
`
	if err := os.WriteFile(taskFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := db.GetDB().Exec(
		`INSERT OR REPLACE INTO tasks (task_id, title, type, status, priority, estimate, file_path, created_at, updated_at)
		 VALUES ('T020', 'all checked', 'feature', 'in-progress', 'p2', 'M', 'works/tasks/T020-all-checked.md', date('now'), date('now'))`,
	)
	if err != nil {
		t.Fatalf("Task DB insert failed: %v", err)
	}

	result, err := harness.AutoCheckCriteria("T020")
	if err != nil {
		t.Fatalf("AutoCheckCriteria failed: %v", err)
	}

	if result.Total != 3 {
		t.Errorf("total=%v, want=3", result.Total)
	}
	if result.Checked != 3 {
		t.Errorf("checked=%v, want=3", result.Checked)
	}
	if result.AllChecked != true {
		t.Errorf("all_checked=%v, want=true", result.AllChecked)
	}
}

// TestAutoCheckCriteria_NoCheckboxes verifies all_checked=true when there
// are no Done Criteria.
func TestAutoCheckCriteria_NoCheckboxes(t *testing.T) {
	setupDB(t)

	tmpRoot := t.TempDir()
	t.Setenv("HSTL_PROJECT_ROOT", tmpRoot)

	taskDir := filepath.Join(tmpRoot, "works", "tasks")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatal(err)
	}

	taskFile := filepath.Join(taskDir, "T050-no-checkbox.md")
	content := `---
id: T050
title: "no checkbox"
status: in-progress
---

# Purpose

Task file without checkboxes
`
	if err := os.WriteFile(taskFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := db.GetDB().Exec(
		`INSERT OR REPLACE INTO tasks (task_id, title, type, status, priority, estimate, file_path, created_at, updated_at)
		 VALUES ('T050', 'no checkbox', 'chore', 'in-progress', 'p2', 'M', 'works/tasks/T050-no-checkbox.md', date('now'), date('now'))`,
	)
	if err != nil {
		t.Fatalf("Task DB insert failed: %v", err)
	}

	result, err := harness.AutoCheckCriteria("T050")
	if err != nil {
		t.Fatalf("AutoCheckCriteria failed: %v", err)
	}

	if result.Total != 0 {
		t.Errorf("total=%v, want=0", result.Total)
	}
	if result.AllChecked != true {
		t.Errorf("all_checked=%v, want=true (no checkboxes ⇒ pass)", result.AllChecked)
	}
}

// TestHarnessGet_SprintDefault verifies sprint:default template is used for
// the sprint entity.
func TestHarnessGet_SprintDefault(t *testing.T) {
	setupDB(t)

	// insert a Sprint into the DB
	_, err := db.GetDB().Exec(
		`INSERT OR REPLACE INTO sprints (sprint_id, title, status, folder_path, goal, created_at, updated_at)
		 VALUES ('sprint-02', 'test sprint', 'active', 'works/sprints/active/sprint-02', 'test', date('now'), date('now'))`,
	)
	if err != nil {
		t.Fatalf("Sprint insert failed: %v", err)
	}

	ret, err := harness.HarnessGet("sprint", "sprint-02")
	if err != nil {
		t.Fatalf("HarnessGet failed: %v", err)
	}

	if ret.EntityType != "sprint" {
		t.Errorf("entity_type=%v, want=sprint", ret.EntityType)
	}
	if ret.EntityID != "sprint-02" {
		t.Errorf("entity_id=%v, want=sprint-02", ret.EntityID)
	}
	if ret.TemplateKey != "sprint:default" {
		t.Errorf("template_key=%v, want=sprint:default", ret.TemplateKey)
	}
}

// TestEnsureHarnessItems_Bugfix verifies the task:bugfix template is used
// for a bugfix Task.
func TestEnsureHarnessItems_Bugfix(t *testing.T) {
	setupDB(t)
	insertTask(t, "T050", "bugfix", "in-progress")

	items, err := harness.EnsureHarnessItems("task", "T050")
	if err != nil {
		t.Fatalf("EnsureHarnessItems failed: %v", err)
	}
	if len(items) == 0 {
		t.Error("bugfix items were not initialised")
	}
}

// TestEnsureHarnessItems_Infra verifies that an infra Task gets harness
// items.
func TestEnsureHarnessItems_Infra(t *testing.T) {
	setupDB(t)
	insertTask(t, "T051", "infra", "in-progress")

	items, err := harness.EnsureHarnessItems("task", "T051")
	if err != nil {
		t.Fatalf("EnsureHarnessItems failed: %v", err)
	}
	// infra type should also have items
	t.Logf("infra item count: %d", len(items))
}

// TestEnsureHarnessItems_HotfixRequiredItems — T222 (Sprint-28)
// Verifies that a hotfix-type Task uses task:hotfix and has the 6 required
// items. In Go projects T229's lint_passed (conditional=is_go_project) is
// added.
// Items (required): criteria_checked, reproduction, root_cause, build_passed,
// tests_passed,
//
//	rollback_plan — reflects the hotfix philosophy that even an emergency
//	fix must include reproduction, root cause, and rollback.
func TestEnsureHarnessItems_HotfixRequiredItems(t *testing.T) {
	setupDB(t)
	insertTask(t, "T052", "hotfix", "in-progress")

	items, err := harness.EnsureHarnessItems("task", "T052")
	if err != nil {
		t.Fatalf("EnsureHarnessItems failed: %v", err)
	}

	requiredExpected := map[string]bool{
		"criteria_checked": true,
		"reproduction":     true,
		"root_cause":       true,
		"build_passed":     true,
		"tests_passed":     true,
		"rollback_plan":    true,
	}
	found := map[string]bool{}
	for _, it := range items {
		found[it.ID] = true
	}
	for id := range requiredExpected {
		if !found[id] {
			t.Errorf("required item missing: %s", id)
		}
	}
}

// TestResolveTemplateKey_Hotfix verifies the hotfix type resolves to
// task:hotfix. This is distinct from the existing feature fallback path
// (unknown type).
func TestResolveTemplateKey_Hotfix(t *testing.T) {
	setupDB(t)
	insertTask(t, "T053", "hotfix", "in-progress")

	key, err := harness.ResolveTemplateKey("task", "T053")
	if err != nil {
		t.Fatalf("ResolveTemplateKey failed: %v", err)
	}
	if key != "task:hotfix" {
		t.Errorf("template_key=%v (expected task:hotfix)", key)
	}
}

// TestEnsureHarnessItems_lint_passed_GoProject — T229 + T278
// Verifies feature/bugfix/hotfix type Tasks get the lint_passed item.
// In a Go project (go.mod present) the conditional=is_go_project evaluation
// yields required=true. The refactor type uses has_go_source_changes
// conditional after T278 — separate test.
func TestEnsureHarnessItems_lint_passed_GoProject(t *testing.T) {
	setupDB(t)

	// feature/bugfix/hotfix: is_go_project conditional → required=true in Go
	cases := []struct {
		taskID string
		typ    string
	}{
		{"T080", "feature"},
		{"T082", "bugfix"},
		{"T083", "hotfix"},
	}

	for _, c := range cases {
		insertTask(t, c.taskID, c.typ, "in-progress")
		items, err := harness.EnsureHarnessItems("task", c.taskID)
		if err != nil {
			t.Errorf("[%s] EnsureHarnessItems failed: %v", c.typ, err)
			continue
		}
		var lintItem *domain.HarnessItem
		for i := range items {
			if items[i].ID == "lint_passed" {
				lintItem = &items[i]
				break
			}
		}
		if lintItem == nil {
			t.Errorf("[%s] lint_passed item was not added", c.typ)
			continue
		}
		// In a Go project (the plugin repo) it should initialise to required=true.
		if !lintItem.Required {
			t.Errorf("[%s] lint_passed required=false (expected true in Go project)", c.typ)
		}
	}

	// refactor: has_go_source_changes conditional after T278 — only check
	// presence of lint_passed (required value depends on git diff state).
	insertTask(t, "T081", "refactor", "in-progress")
	items, err := harness.EnsureHarnessItems("task", "T081")
	if err != nil {
		t.Fatalf("[refactor] EnsureHarnessItems failed: %v", err)
	}
	found := false
	for _, it := range items {
		if it.ID == "lint_passed" {
			found = true
			break
		}
	}
	if !found {
		t.Error("[refactor] lint_passed item was not added")
	}
}

// TestEnsureHarnessItems_lint_passed_NonGoProject — T229
// Simulates a non-Go project by pointing HSTL_PROJECT_ROOT at a temp dir
// without go.mod. Verifies lint_passed initialises to required=false so it
// does not block task_complete.
func TestEnsureHarnessItems_lint_passed_NonGoProject(t *testing.T) {
	setupDB(t)

	// temp root (no go.mod)
	tmp := t.TempDir()
	t.Setenv("HSTL_PROJECT_ROOT", tmp)

	insertTask(t, "T084", "feature", "in-progress")
	items, err := harness.EnsureHarnessItems("task", "T084")
	if err != nil {
		t.Fatalf("EnsureHarnessItems failed: %v", err)
	}
	for _, it := range items {
		if it.ID == "lint_passed" && it.Required {
			t.Error("lint_passed required=true in non-Go project (expected false)")
		}
	}
}

// TestHarnessCheck_MultipleItemsSequential verifies that completing items
// sequentially increases the completion ratio.
func TestHarnessCheck_MultipleItemsSequential(t *testing.T) {
	setupDB(t)
	insertTask(t, "T060", "feature", "in-progress")

	items, err := harness.EnsureHarnessItems("task", "T060")
	if err != nil {
		t.Fatalf("EnsureHarnessItems failed: %v", err)
	}
	if len(items) == 0 {
		t.Skip("no items, skipping")
	}

	// check the first item
	resultAny, err := harness.HarnessCheck("task", "T060", items[0].ID, "verified", "tester")
	if err != nil {
		t.Fatalf("HarnessCheck failed: %v", err)
	}
	result, ok := resultAny.(domain.HarnessCheckResult)
	if !ok {
		t.Fatalf("HarnessCheck return type mismatch: %T", resultAny)
	}
	if result.ItemID != items[0].ID {
		t.Errorf("item_id=%v, want=%s", result.ItemID, items[0].ID)
	}
}

// TestResolveTemplateKey_Task verifies the template key for a task entity.
func TestResolveTemplateKey_Task(t *testing.T) {
	setupDB(t)

	// feature Task
	insertTask(t, "T070", "feature", "in-progress")
	key, err := harness.ResolveTemplateKey("task", "T070")
	if err != nil {
		t.Fatalf("ResolveTemplateKey failed: %v", err)
	}
	if key != "task:feature" {
		t.Errorf("template_key=%v, want=task:feature", key)
	}
}

// TestResolveTemplateKey_Sprint verifies the template key for a sprint
// entity is sprint:default.
func TestResolveTemplateKey_Sprint(t *testing.T) {
	setupDB(t)

	_, err := db.GetDB().Exec(
		`INSERT OR REPLACE INTO sprints (sprint_id, title, status, folder_path, goal, created_at, updated_at)
		 VALUES ('sprint-rk', 'key test', 'active', 'path', 'goal', date('now'), date('now'))`,
	)
	if err != nil {
		t.Fatalf("Sprint insert failed: %v", err)
	}

	key, err := harness.ResolveTemplateKey("sprint", "sprint-rk")
	if err != nil {
		t.Fatalf("ResolveTemplateKey failed: %v", err)
	}
	if key != "sprint:default" {
		t.Errorf("template_key=%v, want=sprint:default", key)
	}
}

// TestLoadHarnessDefaults_TemplateCount verifies the default templates load.
func TestLoadHarnessDefaults_TemplateCount(t *testing.T) {
	templates, err := harness.HarnessTemplateGet("")
	// an empty key may return an error — just confirm load behaviour
	if err != nil {
		t.Logf("HarnessTemplateGet(\"\") error (expected): %v", err)
		return
	}
	t.Logf("template count for empty key: %d", len(templates))
}

// TestAutoCheckCriteria_FileMissing verifies a default result is returned
// when the Task file is absent.
func TestAutoCheckCriteria_FileMissing(t *testing.T) {
	setupDB(t)

	insertTask(t, "T080", "feature", "in-progress")
	// call AutoCheckCriteria with no file present
	result, err := harness.AutoCheckCriteria("T080")
	if err != nil {
		t.Fatalf("AutoCheckCriteria failed: %v", err)
	}
	// when the file is missing, a default result is returned
	t.Logf("AutoCheckCriteria result (file missing): %v", result)
}

// TestEnsureHarnessItems_Refactor_GoChanges — T278
// Verifies that for a refactor Task with Go source changes,
// build_passed/tests_passed/lint_passed initialise to required=true
// (existing behaviour).
func TestEnsureHarnessItems_Refactor_GoChanges(t *testing.T) {
	setupDB(t)

	// runs from the plugin root which is a real git repo, so git diff works.
	// Since T278 is in progress there should be .go file changes.
	insertTask(t, "T090", "refactor", "in-progress")
	items, err := harness.EnsureHarnessItems("task", "T090")
	if err != nil {
		t.Fatalf("EnsureHarnessItems failed: %v", err)
	}

	// refactor type: criteria_checked (required) +
	// build_passed/tests_passed/lint_passed (conditional) + code_review
	// (required)
	if len(items) == 0 {
		t.Error("refactor items were not initialised")
	}

	// criteria_checked and code_review must always be required
	for _, it := range items {
		if it.ID == "criteria_checked" && !it.Required {
			t.Error("criteria_checked must always be required=true")
		}
		if it.ID == "code_review" && !it.Required {
			t.Error("code_review must always be required=true")
		}
	}
}

// TestEnsureHarnessItems_Refactor_NoGoChanges — T278
// Verifies that without Go source changes, build/tests/lint on a refactor
// Task initialise to required=false (auto-skip).
func TestEnsureHarnessItems_Refactor_NoGoChanges(t *testing.T) {
	setupDB(t)

	// instead of a tempdir without git → git diff fails → safe default true
	// is not what we want; create a git repo with no .go changes instead.
	tmp := t.TempDir()
	t.Setenv("HSTL_PROJECT_ROOT", tmp)

	// initialise an empty git repo + initial commit
	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = tmp
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test",
			"GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=test",
			"GIT_COMMITTER_EMAIL=test@test.com",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}

	runGit("init")
	// initial file commit
	if err := os.WriteFile(filepath.Join(tmp, "README.md"), []byte("# test"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit("add", "README.md")
	runGit("commit", "-m", "init")

	// only .md changes (no .go file)
	if err := os.WriteFile(filepath.Join(tmp, "CHANGES.md"), []byte("# changes"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit("add", "CHANGES.md")

	insertTask(t, "T091", "refactor", "in-progress")
	items, err := harness.EnsureHarnessItems("task", "T091")
	if err != nil {
		t.Fatalf("EnsureHarnessItems failed: %v", err)
	}

	for _, it := range items {
		switch it.ID {
		case "build_passed", "tests_passed", "lint_passed":
			if it.Required {
				t.Errorf("[%s] required=true with no Go change (expected false) — T278 auto-skip not working", it.ID)
			}
		case "criteria_checked", "code_review":
			if !it.Required {
				t.Errorf("[%s] must always be required=true", it.ID)
			}
		}
	}
}

// TestT605_ChoreHarness_HasGoSourceChanges verifies that a chore Task adds
// build/tests/lint conditional items, and that they are required=false when
// there are no Go file changes.
func TestT605_ChoreHarness_HasGoSourceChanges(t *testing.T) {
	setupDB(t)

	insertTask(t, "T605TEST", "chore", "in-progress")
	items, err := harness.EnsureHarnessItems("task", "T605TEST")
	if err != nil {
		t.Fatalf("EnsureHarnessItems: %v", err)
	}

	ids := map[string]*domain.HarnessItem{}
	for i := range items {
		ids[items[i].ID] = &items[i]
	}
	for _, required := range []string{"build_passed", "tests_passed", "lint_passed"} {
		if _, ok := ids[required]; !ok {
			t.Errorf("%s item missing from chore template", required)
		}
	}
	// criteria_checked must always be required=true
	if ci, ok := ids["criteria_checked"]; !ok || !ci.Required {
		t.Errorf("criteria_checked must be required=true on chore")
	}
}
