package db_test

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/brand"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/testutil"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/db"
	_ "modernc.org/sqlite"
)

// TestInitDB_CreateAndSchema verifies that the DB file is created and
// the required schema tables exist.
func TestInitDB_CreateAndSchema(t *testing.T) {
	// Configure a temporary path for the test.
	tmpDir := t.TempDir()
	t.Setenv("HSTL_PROJECT", "test-"+filepath.Base(tmpDir))

	// Compute the DB path before InitDB runs.
	dbPath := db.GetDBPath()
	t.Logf("test DB path: %s", dbPath)

	if err := db.InitDB(); err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	defer db.Close()

	// DB file must exist.
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Errorf("DB file was not created: %s", dbPath)
	}

	// Required tables must exist.
	sqlDB := db.GetDB()
	requiredTables := []string{
		"id_counters",
		"tasks",
		"sprints",
		"harness_items",
		"harness_templates",
		"audit_events",
	}
	for _, tbl := range requiredTables {
		var name string
		err := sqlDB.QueryRow(
			"SELECT name FROM sqlite_master WHERE type='table' AND name=?", tbl,
		).Scan(&name)
		if err != nil {
			t.Errorf("table '%s' missing: %v", tbl, err)
		}
	}
}

// TestInitDB_WALMode verifies that WAL journal mode is enabled.
func TestInitDB_WALMode(t *testing.T) {
	t.Setenv("HSTL_PROJECT", "test-wal-mode")

	if err := db.InitDB(); err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	defer db.Close()

	var journalMode string
	err := db.GetDB().QueryRow("PRAGMA journal_mode").Scan(&journalMode)
	if err != nil {
		t.Fatalf("failed to query PRAGMA journal_mode: %v", err)
	}
	if journalMode != "wal" {
		t.Errorf("not WAL mode: %s", journalMode)
	}
}

// TestInitDB_Idempotent verifies that calling InitDB twice does not
// produce an error.
func TestInitDB_Idempotent(t *testing.T) {
	t.Setenv("HSTL_PROJECT", "test-idempotent")

	if err := db.InitDB(); err != nil {
		t.Fatalf("first InitDB failed: %v", err)
	}
	db.Close()

	if err := db.InitDB(); err != nil {
		t.Fatalf("second InitDB failed: %v", err)
	}
	defer db.Close()
}

// TestLoadHarnessDefaults_Loaded verifies that harness_defaults.json is
// loaded into harness_templates.
func TestLoadHarnessDefaults_Loaded(t *testing.T) {
	t.Setenv("HSTL_PROJECT", "test-harness-defaults")

	if err := db.InitDB(); err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	defer db.Close()

	// task:feature template must exist.
	var config string
	err := db.GetDB().QueryRow(
		"SELECT config FROM harness_templates WHERE template_key=?", "task:feature",
	).Scan(&config)
	if err == sql.ErrNoRows {
		t.Error("task:feature template missing")
	} else if err != nil {
		t.Errorf("harness_templates query failed: %v", err)
	}

	// sprint:default template must also exist.
	err = db.GetDB().QueryRow(
		"SELECT config FROM harness_templates WHERE template_key=?", "sprint:default",
	).Scan(&config)
	if err == sql.ErrNoRows {
		t.Error("sprint:default template missing")
	} else if err != nil {
		t.Errorf("harness_templates query failed: %v", err)
	}
}

// TestDetectProjectKey_EnvVar verifies the HSTL_PROJECT environment
// variable wins.
func TestDetectProjectKey_EnvVar(t *testing.T) {
	t.Setenv("HSTL_PROJECT", "my-test-project")
	key := db.DetectProjectKey()
	if key != "my-test-project" {
		t.Errorf("env var key wrong: %s", key)
	}
}

// TestDetectProjectKey_Fallback verifies the fallback behaviour when
// the env var is unset.
//
// If a project-config.yaml exists above cwd, YAML is the first
// resort, so this test moves into a fresh tmpdir to verify the 8-char
// hash fallback.
func TestDetectProjectKey_Fallback(t *testing.T) {
	os.Unsetenv("HSTL_PROJECT")
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	tmp := t.TempDir()
	_ = os.Chdir(tmp)

	key := db.DetectProjectKey()
	if len(key) != 8 {
		t.Errorf("fallback key length wrong (expected 8): %d, key=%s", len(key), key)
	}
}

// TestDetectProjectKey_YAMLPriority verifies that when HSTL_PROJECT is
// unset and project-config.yaml provides project.key, the YAML value
// wins over the git-remote hash (drift prevention).
func TestDetectProjectKey_YAMLPriority(t *testing.T) {
	ws := testutil.NewIsolatedWorkspace(t,
		testutil.WithProjectConfig(`version: "2.0.0"
project:
    key: my-stable-project
`))

	// Unset HSTL_PROJECT and chdir into the workspace.
	os.Unsetenv("HSTL_PROJECT")
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	if err := os.Chdir(ws.Root); err != nil {
		t.Fatal(err)
	}

	key := db.DetectProjectKey()
	if key != "my-stable-project" {
		t.Errorf("yaml project.key not preferred: got %q, want my-stable-project", key)
	}
}

// TestDetectProjectKey_EnvOverridesYAML verifies that HSTL_PROJECT
// still wins over a YAML key (test-isolation scenario).
func TestDetectProjectKey_EnvOverridesYAML(t *testing.T) {
	ws := testutil.NewIsolatedWorkspace(t,
		testutil.WithProjectConfig("project:\n  key: yaml-key\n"))

	t.Setenv("HSTL_PROJECT", "env-key")
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	_ = os.Chdir(ws.Root)

	if key := db.DetectProjectKey(); key != "env-key" {
		t.Errorf("env > yaml priority broken: %s", key)
	}
}

// TestDetectProjectKey_YAMLSubdirectory verifies that project.key is
// discovered when cwd is a sub-directory of the directory holding the
// project-config directory.
func TestDetectProjectKey_YAMLSubdirectory(t *testing.T) {
	ws := testutil.NewIsolatedWorkspace(t,
		testutil.WithProjectConfig("project:\n  key: parent-project\n"))

	// Create a nested directory and chdir into it.
	subdir := filepath.Join(ws.Root, "src", "nested", "deep")
	_ = os.MkdirAll(subdir, 0o755)

	os.Unsetenv("HSTL_PROJECT")
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	_ = os.Chdir(subdir)

	if key := db.DetectProjectKey(); key != "parent-project" {
		t.Errorf("upward-directory search failed: %s", key)
	}
}

// TestDetectProjectKey_AutoPinYAML verifies that a fallback-computed
// key is auto-pinned to the canonical `.hstl/project-config.yaml` so
// it stays stable across binary upgrades.
//
// The canonical brand directory is brand.ProjectDirName, with the
// brand.LegacyProjectDirNames entries searched as fallbacks. New files
// land at the canonical path.
func TestDetectProjectKey_AutoPinYAML(t *testing.T) {
	os.Unsetenv("HSTL_PROJECT")
	tmp := t.TempDir()
	t.Setenv("HSTL_PROJECT_ROOT", tmp)
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	_ = os.Chdir(tmp)

	// 1. First run — no YAML → fallback compute + pin under the
	// canonical path.
	firstKey := db.DetectProjectKey()
	if len(firstKey) != 8 {
		t.Fatalf("fallback key length wrong: %d, key=%s", len(firstKey), firstKey)
	}

	// Pinned YAML must land under the canonical hidden config directory.
	yamlPath := filepath.Join(tmp, brand.ProjectDirName, "project-config.yaml")
	data, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatalf("YAML pinning failed — file missing under canonical %s/: %v", brand.ProjectDirName, err)
	}
	if !strings.Contains(string(data), firstKey) {
		t.Errorf("YAML does not contain the computed key: data=%q key=%q", data, firstKey)
	}

	// 2. Second run — YAML now seeds the result, returning the same key.
	secondKey := db.DetectProjectKey()
	if secondKey != firstKey {
		t.Errorf("second-call key drifted: first=%q second=%q", firstKey, secondKey)
	}
}

// TestPinProjectKey_NoOverwriteExisting verifies that DetectProjectKey does
// not overwrite an existing project-config.yaml in the canonical directory.
func TestPinProjectKey_NoOverwriteExisting(t *testing.T) {
	os.Unsetenv("HSTL_PROJECT")
	tmp := t.TempDir()
	t.Setenv("HSTL_PROJECT_ROOT", tmp)
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	_ = os.Chdir(tmp)

	configDir := filepath.Join(tmp, brand.ProjectDirName)
	_ = os.MkdirAll(configDir, 0o755)
	existing := "# user edited\nproject:\n  key: my-custom-key\nfoo: bar\n"
	yamlPath := filepath.Join(configDir, "project-config.yaml")
	_ = os.WriteFile(yamlPath, []byte(existing), 0o644)

	_ = db.DetectProjectKey()

	data, _ := os.ReadFile(yamlPath)
	if string(data) != existing {
		t.Errorf("existing YAML was overwritten:\n%s", data)
	}
}
