// trac: HAR-CM006
package task

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func setupReconcileDB(t *testing.T) *sql.DB {
	t.Helper()
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "test.db")
	d, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	_, err = d.Exec(`CREATE TABLE tasks (
		task_id TEXT PRIMARY KEY,
		title TEXT,
		type TEXT,
		sprint TEXT,
		status TEXT,
		priority TEXT,
		estimate TEXT,
		file_path TEXT NOT NULL,
		depends_on TEXT,
		created_at TEXT,
		updated_at TEXT,
		work_ticket TEXT
	)`)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func writeReconcileTaskFile(t *testing.T, dir, taskID, workTicket string) string {
	t.Helper()
	path := filepath.Join(dir, taskID+".md")
	body := "---\nid: " + taskID + "\n"
	if workTicket != "" {
		body += "work_ticket: " + workTicket + "\n"
	}
	body += "---\n\n# " + taskID + "\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestReconcileTickets_AllOk — DB and frontmatter agree.
func TestReconcileTickets_AllOk(t *testing.T) {
	d := setupReconcileDB(t)
	tmp := t.TempDir()
	path := writeReconcileTaskFile(t, tmp, "T100", "WT-T100-aaaa1111")

	_, err := d.Exec(`INSERT INTO tasks (task_id, title, type, status, file_path, work_ticket, created_at, updated_at)
		VALUES ('T100', 'test', 'feature', 'todo', ?, 'WT-T100-aaaa1111', '2026-04-28', '2026-04-28')`, path)
	if err != nil {
		t.Fatal(err)
	}

	result, err := ReconcileTickets(d, true, false)
	if err != nil {
		t.Fatal(err)
	}
	if result.OkCount != 1 || result.DriftCount != 0 {
		t.Errorf("expected ok=1 drift=0, got ok=%d drift=%d", result.OkCount, result.DriftCount)
	}
}

// TestReconcileTickets_MissingInDB — file has it but DB column is empty.
func TestReconcileTickets_MissingInDB(t *testing.T) {
	d := setupReconcileDB(t)
	tmp := t.TempDir()
	path := writeReconcileTaskFile(t, tmp, "T200", "WT-T200-bbbb2222")

	_, err := d.Exec(`INSERT INTO tasks (task_id, title, type, status, file_path, work_ticket, created_at, updated_at)
		VALUES ('T200', 'test', 'feature', 'todo', ?, '', '2026-04-28', '2026-04-28')`, path)
	if err != nil {
		t.Fatal(err)
	}

	result, err := ReconcileTickets(d, true, false)
	if err != nil {
		t.Fatal(err)
	}
	if result.DriftCount != 1 {
		t.Errorf("expected drift=1, got %d", result.DriftCount)
	}
	if result.Drifts[0].Kind != "missing_in_db" {
		t.Errorf("expected missing_in_db, got %q", result.Drifts[0].Kind)
	}
}

// TestReconcileTickets_Mismatch — both sides present with different values.
func TestReconcileTickets_Mismatch(t *testing.T) {
	d := setupReconcileDB(t)
	tmp := t.TempDir()
	path := writeReconcileTaskFile(t, tmp, "T300", "WT-T300-cccc3333")

	_, err := d.Exec(`INSERT INTO tasks (task_id, title, type, status, file_path, work_ticket, created_at, updated_at)
		VALUES ('T300', 'test', 'feature', 'todo', ?, 'WT-T300-OLDOLDOLD', '2026-04-28', '2026-04-28')`, path)
	if err != nil {
		t.Fatal(err)
	}

	result, err := ReconcileTickets(d, true, false)
	if err != nil {
		t.Fatal(err)
	}
	if result.Drifts[0].Kind != "mismatch" {
		t.Errorf("expected mismatch, got %q", result.Drifts[0].Kind)
	}
}

// TestReconcileTickets_FixApplied — --fix updates the DB.
func TestReconcileTickets_FixApplied(t *testing.T) {
	d := setupReconcileDB(t)
	tmp := t.TempDir()
	path := writeReconcileTaskFile(t, tmp, "T400", "WT-T400-dddd4444")

	_, err := d.Exec(`INSERT INTO tasks (task_id, title, type, status, file_path, work_ticket, created_at, updated_at)
		VALUES ('T400', 'test', 'feature', 'todo', ?, '', '2026-04-28', '2026-04-28')`, path)
	if err != nil {
		t.Fatal(err)
	}

	result, err := ReconcileTickets(d, false, true)
	if err != nil {
		t.Fatal(err)
	}
	if result.FixedCount != 1 {
		t.Errorf("expected fixed=1, got %d", result.FixedCount)
	}

	// confirm DB updated
	var updated string
	if err := d.QueryRow(`SELECT work_ticket FROM tasks WHERE task_id='T400'`).Scan(&updated); err != nil {
		t.Fatal(err)
	}
	if updated != "WT-T400-dddd4444" {
		t.Errorf("expected DB updated to file value, got %q", updated)
	}
}

// TestReconcileTickets_MissingInFile — DB has it but the file is absent.
func TestReconcileTickets_MissingInFile(t *testing.T) {
	d := setupReconcileDB(t)
	tmp := t.TempDir()
	missingPath := filepath.Join(tmp, "nope.md")

	_, err := d.Exec(`INSERT INTO tasks (task_id, title, type, status, file_path, work_ticket, created_at, updated_at)
		VALUES ('T500', 'test', 'feature', 'todo', ?, 'WT-T500-eeee5555', '2026-04-28', '2026-04-28')`, missingPath)
	if err != nil {
		t.Fatal(err)
	}

	result, err := ReconcileTickets(d, true, false)
	if err != nil {
		t.Fatal(err)
	}
	if result.Drifts[0].Kind != "missing_in_file" {
		t.Errorf("expected missing_in_file, got %q", result.Drifts[0].Kind)
	}
}

// TestExtractFrontmatterWorkTicket_Quoted — quoted yaml value.
func TestExtractFrontmatterWorkTicket_Quoted(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "T600.md")
	body := "---\nid: T600\nwork_ticket: \"WT-T600-ffff6666\"\n---\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := extractFrontmatterWorkTicket(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != "WT-T600-ffff6666" {
		t.Errorf("expected unquoted, got %q", got)
	}
}

// TestExtractFrontmatterWorkTicket_Missing — empty value when no
// frontmatter exists.
func TestExtractFrontmatterWorkTicket_Missing(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "T700.md")
	body := "# T700 plain markdown without frontmatter\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := extractFrontmatterWorkTicket(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

// TestReconcileResult_NoDriftReason — dry-run produces a clear result
// detail.
func TestReconcileResult_NoDriftReason(t *testing.T) {
	d := setupReconcileDB(t)
	result, err := ReconcileTickets(d, true, false)
	if err != nil {
		t.Fatal(err)
	}
	if result.TotalTasks != 0 || result.DriftCount != 0 {
		t.Errorf("empty DB should yield zero counts, got %+v", result)
	}
	if !result.DryRun {
		t.Errorf("DryRun should be true")
	}
	// when no drift, the Drifts slice is also nil/empty
	if len(result.Drifts) != 0 {
		t.Errorf("expected no drifts, got %d", len(result.Drifts))
	}
	_ = strings.Contains
}
