// Package sqlite provides a SQLite-backed implementation of ports.GraphStore
// and ports.IDStore. It uses database/sql directly to avoid circular imports
// with pkg/db.
package sqlite

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"os"
	"path/filepath"
	"strings"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/dblock"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/ports"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/envalias"
)

// newSHA256 returns a hash instance for the audit hash chain.
func newSHA256() hash.Hash { return sha256.New() }

// Compile-time interface assertions.

var _ ports.GraphStore = (*SQLiteStore)(nil)
var _ ports.IDStore = (*SQLiteStore)(nil)
var _ ports.TaskStore = (*SQLiteStore)(nil)
var _ ports.SprintStore = (*SQLiteStore)(nil)
var _ ports.HarnessStore = (*SQLiteStore)(nil)
var _ ports.AuditLog = (*SQLiteStore)(nil)

// Constants (no magic numbers).

const (
	auditQueryDefaultLimit  = 100
	projectSecretByteLength = 32
	placeholderBodyMaxLen   = 300
	taskIDLikePattern       = "T%"
	taskIDFormat            = "T%03d"
	counterKeyTask          = "task"
)

// placeholderMarkers lists markers that indicate placeholder body content.
var placeholderMarkers = []string{
	"{The problem this Task",
	"{Requirement",
	"{Criterion",
	"{TODO",
	"{Placeholder",
	"{One-line summary",
}

// productionKeywords lists the keywords used to flag a sprint as production-related.
var productionKeywords = []string{"prod", "production", "release", "deploy"}

// Type and constructor.

// SQLiteStore implements GraphStore and IDStore over a SQLite DB.
type SQLiteStore struct {
	db *sql.DB
}

// New returns a SQLiteStore that wraps the provided *sql.DB.
func New(database *sql.DB) *SQLiteStore {
	return &SQLiteStore{db: database}
}

// Audit.

// AppendAuditEvent appends an audit event to the audit_events table.
// When event.Timestamp is empty, the DB DEFAULT (datetime('now')) is used.
// When set, that timestamp is preserved (supports importing historical
// events).
//
// Hash-chain computation:
//  1. begin a transaction
//  2. INSERT first
//  3. build the canonical event from LastInsertId + the freshly recorded
//     timestamp
//  4. query the previous record's event_hash (prev_hash)
//  5. compute SHA256 + UPDATE
func (s *SQLiteStore) AppendAuditEvent(event ports.AuditEvent) error {
	// Avoid an import cycle with pkg/audit: implement the minimal hash
	// logic locally. Equivalent to pkg/audit.CanonicalEventJSON, kept
	// independent so the adapter can stand alone.
	detailsJSON, err := json.Marshal(event.Details)
	if err != nil {
		return fmt.Errorf("audit details marshal: %w", err)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("AppendAuditEvent begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var res sql.Result
	if event.Timestamp != "" {
		res, err = tx.Exec(
			`INSERT INTO audit_events (event_type, entity_type, entity_id, actor, details, session_id, timestamp)
			 VALUES (?, ?, ?, ?, ?, ?, ?)`,
			event.EventType, event.EntityType, event.EntityID, event.ActorID, string(detailsJSON), event.SessionID, event.Timestamp,
		)
	} else {
		res, err = tx.Exec(
			`INSERT INTO audit_events (event_type, entity_type, entity_id, actor, details, session_id)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			event.EventType, event.EntityType, event.EntityID, event.ActorID, string(detailsJSON), event.SessionID,
		)
	}
	if err != nil {
		return fmt.Errorf("AppendAuditEvent INSERT: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("AppendAuditEvent LastInsertId: %w", err)
	}

	// Query the timestamp of the just-inserted record (relevant when DEFAULT applied).
	var insertedTS string
	if err = tx.QueryRow(`SELECT timestamp FROM audit_events WHERE id = ?`, id).Scan(&insertedTS); err != nil {
		return fmt.Errorf("AppendAuditEvent select timestamp: %w", err)
	}

	// Fetch the previous record's event_hash (the immediately prior row in this DB). NULL for the very first record.
	var prevHash sql.NullString
	if err = tx.QueryRow(`SELECT event_hash FROM audit_events WHERE id < ? AND event_hash IS NOT NULL ORDER BY id DESC LIMIT 1`, id).Scan(&prevHash); err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("AppendAuditEvent select prev_hash: %w", err)
	}

	// Compute event hash — canonical: id|timestamp|event_type|entity_type|entity_id|actor|session|details
	canonical := auditCanonical(id, insertedTS, event, string(detailsJSON))
	eventHash := sha256hex(prevHash.String, canonical)

	if _, err = tx.Exec(`UPDATE audit_events SET prev_hash = ?, event_hash = ? WHERE id = ?`,
		nilIfEmpty(prevHash.String), eventHash, id); err != nil {
		return fmt.Errorf("AppendAuditEvent UPDATE hash: %w", err)
	}

	return tx.Commit()
}

// auditCanonical returns the deterministic serialisation of an
// audit_events row. Equivalent to pkg/audit.CanonicalEventJSON; kept
// here so the adapter is independent.
// Key order: id, timestamp, event_type, entity_type, entity_id, actor, session_id, details.
func auditCanonical(id int64, ts string, e ports.AuditEvent, detailsJSON string) string {
	return fmt.Sprintf(`{"id":%d,"timestamp":%q,"event_type":%q,"entity_type":%q,"entity_id":%q,"actor":%q,"session_id":%q,"details":%s}`,
		id, ts, e.EventType, e.EntityType, e.EntityID, e.ActorID, e.SessionID, detailsJSON)
}

// sha256hex returns the SHA256 hex string of prev_hash concatenated with canonical.
func sha256hex(prev, canonical string) string {
	h := newSHA256()
	if prev != "" {
		h.Write([]byte(prev))
	}
	h.Write([]byte(canonical))
	return hex.EncodeToString(h.Sum(nil))
}

func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// QueryAuditEvents queries audit events using the supplied filter.
func (s *SQLiteStore) QueryAuditEvents(filter ports.AuditQueryFilter) ([]ports.AuditEvent, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = auditQueryDefaultLimit
	}

	// id is included in the SELECT list so it can be mapped to ports.AuditEvent.ID.
	query := `SELECT id, event_type, entity_type, entity_id, actor, COALESCE(details,''), COALESCE(session_id,''), COALESCE(timestamp,'') FROM audit_events WHERE 1=1`
	args := []any{}

	if filter.EntityType != "" {
		query += " AND entity_type=?"
		args = append(args, filter.EntityType)
	}
	if filter.EntityID != "" {
		query += " AND entity_id=?"
		args = append(args, filter.EntityID)
	}
	if filter.EventType != "" {
		query += " AND event_type=?"
		args = append(args, filter.EventType)
	}
	if filter.Actor != "" {
		query += " AND actor=?"
		args = append(args, filter.Actor)
	}
	if filter.Since != "" {
		query += " AND timestamp>=?"
		args = append(args, filter.Since)
	}
	if filter.Until != "" {
		query += " AND timestamp<=?"
		args = append(args, filter.Until)
	}
	query += " ORDER BY timestamp DESC LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("QueryAuditEvents: %w", err)
	}
	defer rows.Close()

	var events []ports.AuditEvent
	for rows.Next() {
		var ev ports.AuditEvent
		var detailsStr string
		if err := rows.Scan(&ev.ID, &ev.EventType, &ev.EntityType, &ev.EntityID, &ev.ActorID, &detailsStr, &ev.SessionID, &ev.Timestamp); err != nil {
			return nil, fmt.Errorf("QueryAuditEvents scan: %w", err)
		}
		var details any
		if jsonErr := json.Unmarshal([]byte(detailsStr), &details); jsonErr == nil {
			ev.Details = details
		} else {
			ev.Details = detailsStr
		}
		events = append(events, ev)
	}
	return events, rows.Err()
}

// SummarizeAuditEvents returns per-event-type counts for the given time range.
func (s *SQLiteStore) SummarizeAuditEvents(since, until string) (map[string]int, error) {
	query := `SELECT event_type, COUNT(*) as cnt FROM audit_events WHERE 1=1`
	args := []any{}
	if since != "" {
		query += " AND timestamp>=?"
		args = append(args, since)
	}
	if until != "" {
		query += " AND timestamp<=?"
		args = append(args, until)
	}
	query += " GROUP BY event_type"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("SummarizeAuditEvents: %w", err)
	}
	defer rows.Close()

	result := map[string]int{}
	for rows.Next() {
		var eventType string
		var cnt int
		if err := rows.Scan(&eventType, &cnt); err != nil {
			return nil, fmt.Errorf("SummarizeAuditEvents scan: %w", err)
		}
		result[eventType] = cnt
	}
	return result, rows.Err()
}

// Sprint.

// SaveSprint stores or upserts a sprint.
func (s *SQLiteStore) SaveSprint(sprint *ports.SprintRecord) error {
	_, err := s.db.Exec(
		`INSERT INTO sprints (sprint_id,title,status,folder_path,started_at,completed_at,goal,created_at,updated_at)
		 VALUES (?,?,?,?,?,?,?,?,?)
		 ON CONFLICT(sprint_id) DO UPDATE SET
		   title=excluded.title,
		   status=excluded.status,
		   folder_path=excluded.folder_path,
		   started_at=excluded.started_at,
		   completed_at=excluded.completed_at,
		   goal=excluded.goal,
		   updated_at=excluded.updated_at`,
		sprint.SprintID, sprint.Title, sprint.Status, sprint.FolderPath,
		sprint.StartedAt, sprint.CompletedAt, sprint.Goal,
		sprint.CreatedAt, sprint.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("SaveSprint: %w", err)
	}
	return nil
}

// UpdateSprintStatus updates the sprint status and optional fields.
func (s *SQLiteStore) UpdateSprintStatus(sprintID, status string, opts *ports.SprintUpdateOpts) error {
	var folderPath, startedAt, completedAt *string
	if opts != nil {
		if opts.FolderPath != "" {
			folderPath = &opts.FolderPath
		}
		if opts.StartedAt != "" {
			startedAt = &opts.StartedAt
		}
		if opts.CompletedAt != "" {
			completedAt = &opts.CompletedAt
		}
	}
	_, err := s.db.Exec(
		`UPDATE sprints SET
		   status=?,
		   folder_path=COALESCE(?,folder_path),
		   started_at=COALESCE(?,started_at),
		   completed_at=COALESCE(?,completed_at),
		   updated_at=datetime('now')
		 WHERE sprint_id=?`,
		status, folderPath, startedAt, completedAt, sprintID,
	)
	if err != nil {
		return fmt.Errorf("UpdateSprintStatus: %w", err)
	}
	return nil
}

// GetSprint fetches the sprint record for the given sprint ID.
func (s *SQLiteStore) GetSprint(sprintID string) (*ports.SprintRecord, error) {
	row := s.db.QueryRow(
		`SELECT sprint_id,title,status,COALESCE(folder_path,''),
		        COALESCE(started_at,''),COALESCE(completed_at,''),COALESCE(goal,''),
		        created_at,updated_at
		 FROM sprints WHERE sprint_id=?`,
		sprintID,
	)
	var sp ports.SprintRecord
	if err := row.Scan(
		&sp.SprintID, &sp.Title, &sp.Status, &sp.FolderPath,
		&sp.StartedAt, &sp.CompletedAt, &sp.Goal,
		&sp.CreatedAt, &sp.UpdatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("sprint not found: %s", sprintID)
		}
		return nil, fmt.Errorf("GetSprint: %w", err)
	}
	return &sp, nil
}

// ListSprints lists sprints filtered by status. An empty status returns every sprint.
func (s *SQLiteStore) ListSprints(status string) ([]ports.SprintRecord, error) {
	query := `SELECT sprint_id,title,status,COALESCE(folder_path,''),COALESCE(started_at,''),COALESCE(completed_at,''),COALESCE(goal,''),created_at,updated_at FROM sprints`
	args := []any{}
	if status != "" {
		query += " WHERE status=?"
		args = append(args, status)
	}
	query += " ORDER BY sprint_id"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("ListSprints: %w", err)
	}
	defer rows.Close()

	var sprints []ports.SprintRecord
	for rows.Next() {
		var sp ports.SprintRecord
		if err := rows.Scan(
			&sp.SprintID, &sp.Title, &sp.Status, &sp.FolderPath,
			&sp.StartedAt, &sp.CompletedAt, &sp.Goal,
			&sp.CreatedAt, &sp.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("ListSprints scan: %w", err)
		}
		sprints = append(sprints, sp)
	}
	return sprints, rows.Err()
}

// AggregateSprintProgress returns per-status counts and burndown data for the sprint.
func (s *SQLiteStore) AggregateSprintProgress(sprintID string) (*ports.ProgressResult, error) {
	rows, err := s.db.Query(
		`SELECT status, COUNT(*) FROM tasks WHERE sprint=? GROUP BY status`,
		sprintID,
	)
	if err != nil {
		return nil, fmt.Errorf("AggregateSprintProgress: %w", err)
	}
	defer rows.Close()

	counts := map[string]int{"done": 0, "in-progress": 0, "todo": 0}
	for rows.Next() {
		var st string
		var cnt int
		if err := rows.Scan(&st, &cnt); err != nil {
			return nil, fmt.Errorf("AggregateSprintProgress scan: %w", err)
		}
		if st == "absorbed" {
			counts["done"] += cnt
		} else if _, ok := counts[st]; ok {
			counts[st] = cnt
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("AggregateSprintProgress rows: %w", err)
	}

	// Burndown data.
	bdRows, err := s.db.Query(
		`SELECT DATE(updated_at) as date, COUNT(*) as cnt
		   FROM tasks
		  WHERE sprint = ? AND status = 'done'
		  GROUP BY DATE(updated_at)
		  ORDER BY date`,
		sprintID,
	)
	if err != nil {
		return nil, fmt.Errorf("AggregateSprintProgress burndown: %w", err)
	}
	defer bdRows.Close()

	var burndown []ports.BurndownEntry
	for bdRows.Next() {
		var entry ports.BurndownEntry
		if err := bdRows.Scan(&entry.Date, &entry.Done); err != nil {
			continue
		}
		burndown = append(burndown, entry)
	}
	if burndown == nil {
		burndown = []ports.BurndownEntry{}
	}

	total := counts["done"] + counts["in-progress"] + counts["todo"]
	done := counts["done"]
	percent := 0
	if total > 0 {
		percent = int(float64(done) / float64(total) * 100)
	}

	return &ports.ProgressResult{
		SprintID:   sprintID,
		Total:      total,
		Done:       done,
		InProgress: counts["in-progress"],
		Todo:       counts["todo"],
		Percent:    percent,
		Progress:   percent,
		Burndown:   burndown,
	}, nil
}

// GetSprintTaskFilePaths returns the file paths of tasks belonging to the sprint.
func (s *SQLiteStore) GetSprintTaskFilePaths(sprintID string) ([]string, error) {
	rows, err := s.db.Query(
		`SELECT file_path FROM tasks WHERE sprint=? AND file_path != ''`,
		sprintID,
	)
	if err != nil {
		return nil, fmt.Errorf("GetSprintTaskFilePaths: %w", err)
	}
	defer rows.Close()

	var paths []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, fmt.Errorf("GetSprintTaskFilePaths scan: %w", err)
		}
		paths = append(paths, p)
	}
	return paths, rows.Err()
}

// UpdateTaskPathsForSprint rewrites the file paths of all tasks in a
// sprint in one shot.
//
// Earlier the WHERE clause `file_path LIKE 'active/sprint-15/%'` did not
// match the real value `works/sprints/active/sprint-15/...`, so every
// call updated zero rows. The fix drops the LIKE and filters on
// sprint=? only. REPLACE only triggers when the substring matches, so
// rows that do not match are left untouched (safe no-op).
func (s *SQLiteStore) UpdateTaskPathsForSprint(sprintID, fromLocation, toLocation string) error {
	// dblock.Exec auto-retries when a multi-worktree race is detected.
	_, err := dblock.Exec(s.db,
		`UPDATE tasks SET file_path=REPLACE(file_path,?,?), updated_at=datetime('now')
		 WHERE sprint=?`,
		fromLocation, toLocation, sprintID,
	)
	if err != nil {
		return fmt.Errorf("UpdateTaskPathsForSprint: %w", err)
	}
	return nil
}

// UpdateSprintMetadata updates only the non-empty fields.
func (s *SQLiteStore) UpdateSprintMetadata(sprintID, title, folderPath, status string) error {
	// dblock.Exec absorbs races during sprint complete / backlog sync.
	_, err := dblock.Exec(s.db,
		`UPDATE sprints SET
		   title=COALESCE(NULLIF(?,''),title),
		   folder_path=COALESCE(NULLIF(?,''),folder_path),
		   status=COALESCE(NULLIF(?,''),status),
		   updated_at=datetime('now')
		 WHERE sprint_id=?`,
		title, folderPath, status, sprintID,
	)
	if err != nil {
		return fmt.Errorf("UpdateSprintMetadata: %w", err)
	}
	return nil
}

// InsertSprintIfMissing inserts the sprint only when it is absent.
func (s *SQLiteStore) InsertSprintIfMissing(sprint *ports.SprintRecord) error {
	_, err := s.db.Exec(
		`INSERT OR IGNORE INTO sprints
		   (sprint_id,title,status,folder_path,started_at,completed_at,goal,created_at,updated_at)
		 VALUES (?,?,?,?,?,?,?,?,?)`,
		sprint.SprintID, sprint.Title, sprint.Status, sprint.FolderPath,
		sprint.StartedAt, sprint.CompletedAt, sprint.Goal,
		sprint.CreatedAt, sprint.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("InsertSprintIfMissing: %w", err)
	}
	return nil
}

// GetAllSprintsSync returns ID and status for every sprint, used during sync.
func (s *SQLiteStore) GetAllSprintsSync() ([]ports.SprintSyncRecord, error) {
	rows, err := s.db.Query(`SELECT sprint_id, status FROM sprints`)
	if err != nil {
		return nil, fmt.Errorf("GetAllSprintsSync: %w", err)
	}
	defer rows.Close()

	var records []ports.SprintSyncRecord
	for rows.Next() {
		var r ports.SprintSyncRecord
		if err := rows.Scan(&r.SprintID, &r.Status); err != nil {
			return nil, fmt.Errorf("GetAllSprintsSync scan: %w", err)
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

// Task.

const priorityOrderExpr = `CASE priority WHEN 'p0' THEN 0 WHEN 'p1' THEN 1 WHEN 'p2' THEN 2 WHEN 'p3' THEN 3 ELSE 9 END`

// CreateTask inserts the task into the tasks table.
// When sprint is the empty string it is stored as NULL (backlog normalisation).
// work_ticket is recorded at INSERT time too (empty string → NULL).
func (s *SQLiteStore) CreateTask(task *ports.TaskRecord) error {
	var sprintVal any
	if task.Sprint != "" {
		sprintVal = task.Sprint
	}
	var ticketVal any
	if task.WorkTicket != "" {
		ticketVal = task.WorkTicket
	}
	_, err := s.db.Exec(
		`INSERT INTO tasks
		   (task_id,title,type,sprint,status,priority,estimate,file_path,depends_on,work_ticket,created_at,updated_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,datetime('now'))`,
		task.TaskID, task.Title, task.Type, sprintVal, task.Status,
		task.Priority, task.Estimate, task.FilePath, task.DependsOn,
		ticketVal, task.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("CreateTask: %w", err)
	}
	return nil
}

// GetTaskFilePath returns the file path of the task.
func (s *SQLiteStore) GetTaskFilePath(taskID string) (string, error) {
	var filePath string
	err := s.db.QueryRow(`SELECT file_path FROM tasks WHERE task_id=?`, taskID).Scan(&filePath)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("task not found: %s", taskID)
		}
		return "", fmt.Errorf("GetTaskFilePath: %w", err)
	}
	return filePath, nil
}

// GetTaskStatus returns the task status.
func (s *SQLiteStore) GetTaskStatus(taskID string) (string, error) {
	var status string
	err := s.db.QueryRow(`SELECT status FROM tasks WHERE task_id=?`, taskID).Scan(&status)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("task not found: %s", taskID)
		}
		return "", fmt.Errorf("GetTaskStatus: %w", err)
	}
	return status, nil
}

// GetTaskType returns the task type.
func (s *SQLiteStore) GetTaskType(taskID string) (string, error) {
	var taskType string
	err := s.db.QueryRow(`SELECT type FROM tasks WHERE task_id=?`, taskID).Scan(&taskType)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("task not found: %s", taskID)
		}
		return "", fmt.Errorf("GetTaskType: %w", err)
	}
	return taskType, nil
}

// GetTaskTypeAndPath returns the task type and file path together.
func (s *SQLiteStore) GetTaskTypeAndPath(taskID string) (taskType, filePath string, err error) {
	row := s.db.QueryRow(`SELECT type, file_path FROM tasks WHERE task_id=?`, taskID)
	if scanErr := row.Scan(&taskType, &filePath); scanErr != nil {
		if scanErr == sql.ErrNoRows {
			return "", "", fmt.Errorf("task not found: %s", taskID)
		}
		return "", "", fmt.Errorf("GetTaskTypeAndPath: %w", scanErr)
	}
	return taskType, filePath, nil
}

// GetTaskDetails returns the principal fields of the task.
func (s *SQLiteStore) GetTaskDetails(taskID string) (*ports.TaskDetails, error) {
	row := s.db.QueryRow(
		`SELECT task_id, title, type, status, COALESCE(priority,'p2'), COALESCE(estimate,'M'),
		        COALESCE(sprint,''), COALESCE(file_path,''), COALESCE(depends_on,'[]'),
		        created_at, updated_at
		   FROM tasks WHERE task_id=?`,
		taskID,
	)
	var d ports.TaskDetails
	if err := row.Scan(
		&d.TaskID, &d.Title, &d.Type, &d.Status, &d.Priority,
		&d.Estimate, &d.Sprint, &d.FilePath, &d.DependsOn,
		&d.CreatedAt, &d.UpdatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("task not found: %s", taskID)
		}
		return nil, fmt.Errorf("GetTaskDetails: %w", err)
	}
	return &d, nil
}

// GetTaskFull returns every field of the task.
func (s *SQLiteStore) GetTaskFull(taskID string) (*ports.TaskGetResult, error) {
	row := s.db.QueryRow(
		`SELECT title,type,COALESCE(sprint,''),status,priority,estimate,file_path,
		        COALESCE(depends_on,''),created_at,updated_at
		 FROM tasks WHERE task_id=?`,
		taskID,
	)
	var r ports.TaskGetResult
	r.TaskID = taskID
	if err := row.Scan(
		&r.Title, &r.Type, &r.Sprint, &r.Status, &r.Priority,
		&r.Estimate, &r.FilePath, &r.DependsOn,
		&r.CreatedAt, &r.UpdatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("task not found: %s", taskID)
		}
		return nil, fmt.Errorf("GetTaskFull: %w", err)
	}
	return &r, nil
}

// UpdateTaskFilePath updates the task's file path.
func (s *SQLiteStore) UpdateTaskFilePath(taskID, newPath string) error {
	_, err := s.db.Exec(
		`UPDATE tasks SET file_path=?, updated_at=datetime('now') WHERE task_id=?`,
		newPath, taskID,
	)
	if err != nil {
		return fmt.Errorf("UpdateTaskFilePath: %w", err)
	}
	return nil
}

// UpdateTaskStatus updates the task status.
func (s *SQLiteStore) UpdateTaskStatus(taskID, status string) error {
	_, err := s.db.Exec(
		`UPDATE tasks SET status=?, updated_at=datetime('now') WHERE task_id=?`,
		status, taskID,
	)
	if err != nil {
		return fmt.Errorf("UpdateTaskStatus: %w", err)
	}
	return nil
}

// UpdateTaskSprint moves the task to a different sprint.
func (s *SQLiteStore) UpdateTaskSprint(taskID string, sprint *string) error {
	_, err := s.db.Exec(
		`UPDATE tasks SET sprint=?, updated_at=datetime('now') WHERE task_id=?`,
		sprint, taskID,
	)
	if err != nil {
		return fmt.Errorf("UpdateTaskSprint: %w", err)
	}
	return nil
}

// TransitionTaskStatus transitions the task only when the current status matches the expected one.
func (s *SQLiteStore) TransitionTaskStatus(taskID, expectedStatus, newStatus string) error {
	res, err := s.db.Exec(
		`UPDATE tasks SET status=?, updated_at=datetime('now') WHERE task_id=? AND status=?`,
		newStatus, taskID, expectedStatus,
	)
	if err != nil {
		return fmt.Errorf("TransitionTaskStatus: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("TransitionTaskStatus rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("cannot transition status: task=%s expected=%s", taskID, expectedStatus)
	}
	return nil
}

// ResetTaskHarness clears every harness item belonging to the task.
func (s *SQLiteStore) ResetTaskHarness(taskID string) error {
	_, err := s.db.Exec(
		`DELETE FROM harness_items WHERE entity_type='task' AND entity_id=?`,
		taskID,
	)
	if err != nil {
		return fmt.Errorf("ResetTaskHarness: %w", err)
	}
	return nil
}

// ListTasks lists tasks matching the supplied filters.
func (s *SQLiteStore) ListTasks(sprintFilter, statusFilter *string) (*ports.TaskListResult, error) {
	query := `SELECT task_id, title, type, COALESCE(sprint,''), status,
	                 COALESCE(priority,'p2'), COALESCE(estimate,'M'),
	                 COALESCE(file_path,''), COALESCE(depends_on,'[]'),
	                 created_at, updated_at
	            FROM tasks WHERE 1=1`
	args := []any{}
	if sprintFilter != nil {
		query += " AND sprint=?"
		args = append(args, *sprintFilter)
	}
	if statusFilter != nil {
		query += " AND status=?"
		args = append(args, *statusFilter)
	}
	query += " ORDER BY " + priorityOrderExpr + ", created_at"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("ListTasks: %w", err)
	}
	defer rows.Close()

	var tasks []ports.TaskDetails
	for rows.Next() {
		var d ports.TaskDetails
		if err := rows.Scan(
			&d.TaskID, &d.Title, &d.Type, &d.Sprint, &d.Status,
			&d.Priority, &d.Estimate, &d.FilePath, &d.DependsOn,
			&d.CreatedAt, &d.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("ListTasks scan: %w", err)
		}
		tasks = append(tasks, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &ports.TaskListResult{
		Tasks:                 tasks,
		OutdatedTemplateCount: 0,
	}, nil
}

// ListTasksFiltered lists tasks using the extended filter set.
// Semantic SSOT: docs/08-references/standards/cli-filter-schema.md.
func (s *SQLiteStore) ListTasksFiltered(filter ports.TaskListFilter) (*ports.TaskListResult, error) {
	query := `SELECT task_id, title, type, COALESCE(sprint,''), status,
	                 COALESCE(priority,'p2'), COALESCE(estimate,'M'),
	                 COALESCE(file_path,''), COALESCE(depends_on,'[]'),
	                 created_at, updated_at
	            FROM tasks WHERE 1=1`
	args := []any{}
	if filter.Sprint != nil {
		query += " AND sprint=?"
		args = append(args, *filter.Sprint)
	}
	if filter.Status != nil {
		query += " AND status=?"
		args = append(args, *filter.Status)
	}
	if filter.Priority != nil {
		query += " AND priority=?"
		args = append(args, *filter.Priority)
	}
	if filter.Type != nil {
		query += " AND type=?"
		args = append(args, *filter.Type)
	}
	if filter.Since != nil {
		query += " AND COALESCE(updated_at, created_at) >= ?"
		args = append(args, *filter.Since)
	}
	if filter.Until != nil {
		query += " AND COALESCE(updated_at, created_at) <= ?"
		args = append(args, *filter.Until)
	}

	// --last forces time-desc + truncation. --limit applies the default
	// ordering plus truncation. When both are set, --last wins (see
	// docs/08-references/standards/cli-filter-schema.md §2.2).
	if filter.Last > 0 {
		query += " ORDER BY COALESCE(updated_at, created_at) DESC LIMIT ?"
		args = append(args, filter.Last)
	} else {
		query += " ORDER BY " + priorityOrderExpr + ", created_at"
		if filter.Limit > 0 {
			query += " LIMIT ?"
			args = append(args, filter.Limit)
		}
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("ListTasksFiltered: %w", err)
	}
	defer rows.Close()

	var tasks []ports.TaskDetails
	for rows.Next() {
		var d ports.TaskDetails
		if err := rows.Scan(
			&d.TaskID, &d.Title, &d.Type, &d.Sprint, &d.Status,
			&d.Priority, &d.Estimate, &d.FilePath, &d.DependsOn,
			&d.CreatedAt, &d.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("ListTasksFiltered scan: %w", err)
		}
		tasks = append(tasks, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &ports.TaskListResult{
		Tasks:                 tasks,
		OutdatedTemplateCount: 0,
	}, nil
}

// NextTask returns the next task to work on, ordered by priority.
func (s *SQLiteStore) NextTask() (*ports.TaskNextResult, error) {
	// Look up the active sprint.
	var activeSprint string
	err := s.db.QueryRow(
		`SELECT sprint_id FROM sprints WHERE status='active' LIMIT 1`,
	).Scan(&activeSprint)

	var query string
	var args []any
	if err == nil {
		// Active sprint exists.
		query = `SELECT task_id,title FROM tasks WHERE status='todo' AND sprint=? ORDER BY ` + priorityOrderExpr + ` LIMIT 1`
		args = []any{activeSprint}
	} else {
		// No active sprint.
		query = `SELECT task_id,title FROM tasks WHERE status='todo' ORDER BY ` + priorityOrderExpr + ` LIMIT 1`
	}

	var result ports.TaskNextResult
	if scanErr := s.db.QueryRow(query, args...).Scan(&result.TaskID, &result.Title); scanErr != nil {
		if scanErr == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("NextTask: %w", scanErr)
	}
	return &result, nil
}

// HasPlaceholderBody reports whether the task file's body is in a placeholder state.
func (s *SQLiteStore) HasPlaceholderBody(taskID string) (bool, error) {
	filePath, err := s.GetTaskFilePath(taskID)
	if err != nil {
		return false, err
	}

	projectRoot := envalias.Lookup("PROJECT_ROOT")
	if projectRoot == "" {
		var wdErr error
		projectRoot, wdErr = os.Getwd()
		if wdErr != nil {
			return false, fmt.Errorf("HasPlaceholderBody getwd: %w", wdErr)
		}
	}

	data, err := os.ReadFile(filepath.Join(projectRoot, filePath))
	if err != nil {
		return false, fmt.Errorf("HasPlaceholderBody read file: %w", err)
	}

	body := string(data)
	if len(body) < placeholderBodyMaxLen {
		return true, nil
	}
	for _, marker := range placeholderMarkers {
		if strings.Contains(body, marker) {
			return true, nil
		}
	}
	return false, nil
}

// FindPlaceholderTasks returns task IDs whose bodies remain in a placeholder state.
func (s *SQLiteStore) FindPlaceholderTasks(sprintFilter string) ([]string, error) {
	query := `SELECT task_id FROM tasks`
	args := []any{}
	if sprintFilter != "" {
		query += " WHERE sprint=?"
		args = append(args, sprintFilter)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("FindPlaceholderTasks: %w", err)
	}
	defer rows.Close()

	var taskIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("FindPlaceholderTasks scan: %w", err)
		}
		taskIDs = append(taskIDs, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var placeholders []string
	for _, id := range taskIDs {
		isPlaceholder, err := s.HasPlaceholderBody(id)
		if err != nil {
			continue // skip when the file read fails
		}
		if isPlaceholder {
			placeholders = append(placeholders, id)
		}
	}
	return placeholders, nil
}

// GetAllTasksSync returns the basic fields for every task, used during sync.
func (s *SQLiteStore) GetAllTasksSync() ([]ports.TaskSyncRecord, error) {
	rows, err := s.db.Query(`SELECT task_id, status, COALESCE(sprint,''), COALESCE(file_path,'') FROM tasks`)
	if err != nil {
		return nil, fmt.Errorf("GetAllTasksSync: %w", err)
	}
	defer rows.Close()

	var records []ports.TaskSyncRecord
	for rows.Next() {
		var r ports.TaskSyncRecord
		if err := rows.Scan(&r.TaskID, &r.Status, &r.Sprint, &r.FilePath); err != nil {
			return nil, fmt.Errorf("GetAllTasksSync scan: %w", err)
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

// InsertTaskAndSyncCounter inserts the task and synchronises the counter in a single transaction.
func (s *SQLiteStore) InsertTaskAndSyncCounter(task *ports.TaskRecord, maxID int) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("InsertTaskAndSyncCounter begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	_, err = tx.Exec(
		`INSERT OR IGNORE INTO tasks
		   (task_id,title,type,sprint,status,priority,estimate,file_path,depends_on,created_at,updated_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?,datetime('now'))`,
		task.TaskID, task.Title, task.Type, task.Sprint, task.Status,
		task.Priority, task.Estimate, task.FilePath, task.DependsOn,
		task.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("InsertTaskAndSyncCounter insert task: %w", err)
	}

	_, err = tx.Exec(
		`INSERT INTO id_counters (counter_key,current_value,updated_at) VALUES (?,?,datetime('now'))
		 ON CONFLICT(counter_key) DO UPDATE SET
		   current_value=MAX(current_value,excluded.current_value),
		   updated_at=datetime('now')`,
		counterKeyTask, maxID,
	)
	if err != nil {
		return fmt.Errorf("InsertTaskAndSyncCounter sync counter: %w", err)
	}

	return tx.Commit()
}

// GetBacklogTasks returns tasks that are not yet assigned to any sprint.
func (s *SQLiteStore) GetBacklogTasks(includeInProgress bool) ([]ports.TaskInfo, error) {
	query := `SELECT task_id,title,type,priority,estimate,COALESCE(sprint,''),status FROM tasks
	          WHERE (sprint IS NULL OR sprint='') AND status='todo'`
	if includeInProgress {
		query += ` OR status='in-progress'`
	}

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("GetBacklogTasks: %w", err)
	}
	defer rows.Close()

	var tasks []ports.TaskInfo
	for rows.Next() {
		var t ports.TaskInfo
		if err := rows.Scan(
			&t.TaskID, &t.Title, &t.Type, &t.Priority, &t.Estimate, &t.Sprint, &t.Status,
		); err != nil {
			return nil, fmt.Errorf("GetBacklogTasks scan: %w", err)
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

// Harness.

// CountHarnessItems returns the harness-item count attached to the entity.
func (s *SQLiteStore) CountHarnessItems(entityType, entityID string) (int, error) {
	var count int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM harness_items WHERE entity_type=? AND entity_id=?`,
		entityType, entityID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("CountHarnessItems: %w", err)
	}
	return count, nil
}

// EnsureHarnessItems inserts harness items that do not yet exist.
func (s *SQLiteStore) EnsureHarnessItems(entityType, entityID string, items []ports.HarnessItemTemplate) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("EnsureHarnessItems begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	for _, item := range items {
		reqInt := 0
		if item.Required {
			reqInt = 1
		}
		_, err := tx.Exec(
			`INSERT OR IGNORE INTO harness_items
			   (entity_type,entity_id,item_id,item_name,required,done)
			 VALUES (?,?,?,?,?,0)`,
			entityType, entityID, item.ID, item.Description, reqInt,
		)
		if err != nil {
			return fmt.Errorf("EnsureHarnessItems insert %s: %w", item.ID, err)
		}
	}
	return tx.Commit()
}

// GetHarnessItems returns the harness items attached to the entity.
func (s *SQLiteStore) GetHarnessItems(entityType, entityID string) ([]ports.HarnessItem, error) {
	rows, err := s.db.Query(
		`SELECT item_id,item_name,required,done,
		        COALESCE(evidence,''),COALESCE(checked_at,''),COALESCE(checked_by,'')
		 FROM harness_items WHERE entity_type=? AND entity_id=? ORDER BY id`,
		entityType, entityID,
	)
	if err != nil {
		return nil, fmt.Errorf("GetHarnessItems: %w", err)
	}
	defer rows.Close()

	var items []ports.HarnessItem
	for rows.Next() {
		var item ports.HarnessItem
		var required int
		var done int
		if err := rows.Scan(
			&item.ID, &item.Description, &required, &done,
			&item.Evidence, &item.CheckedAt, &item.CheckedBy,
		); err != nil {
			return nil, fmt.Errorf("GetHarnessItems scan: %w", err)
		}
		item.Required = required != 0
		item.Done = done != 0
		items = append(items, item)
	}
	return items, rows.Err()
}

// GetHarnessTemplate returns the harness configuration for the given template key.
func (s *SQLiteStore) GetHarnessTemplate(templateKey string) ([]map[string]any, error) {
	var configStr string
	err := s.db.QueryRow(
		`SELECT config FROM harness_templates WHERE template_key=?`,
		templateKey,
	).Scan(&configStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("harness_template not found: %s", templateKey)
		}
		return nil, fmt.Errorf("GetHarnessTemplate: %w", err)
	}

	var result []map[string]any
	if err := json.Unmarshal([]byte(configStr), &result); err != nil {
		return nil, fmt.Errorf("GetHarnessTemplate unmarshal: %w", err)
	}
	return result, nil
}

// GetHarnessItemID returns the database-internal ID of a harness item.
func (s *SQLiteStore) GetHarnessItemID(entityType, entityID, itemID string) (int64, error) {
	var id int64
	err := s.db.QueryRow(
		`SELECT id FROM harness_items WHERE entity_type=? AND entity_id=? AND item_id=?`,
		entityType, entityID, itemID,
	).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("harness_item not found: %s/%s/%s", entityType, entityID, itemID)
		}
		return 0, fmt.Errorf("GetHarnessItemID: %w", err)
	}
	return id, nil
}

// CheckHarnessItem marks a harness item as completed.
func (s *SQLiteStore) CheckHarnessItem(entityType, entityID, itemID, evidence, actor string) error {
	_, err := s.db.Exec(
		`UPDATE harness_items SET done=1, evidence=?, checked_at=datetime('now'), checked_by=?
		 WHERE entity_type=? AND entity_id=? AND item_id=?`,
		evidence, actor, entityType, entityID, itemID,
	)
	if err != nil {
		return fmt.Errorf("CheckHarnessItem: %w", err)
	}
	return nil
}

// CheckSprintProduction reports whether the sprint relates to a production deployment.
func (s *SQLiteStore) CheckSprintProduction(sprintID string) (bool, error) {
	var goal, title string
	err := s.db.QueryRow(
		`SELECT COALESCE(goal,''), title FROM sprints WHERE sprint_id=?`,
		sprintID,
	).Scan(&goal, &title)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, fmt.Errorf("sprint not found: %s", sprintID)
		}
		return false, fmt.Errorf("CheckSprintProduction: %w", err)
	}

	combined := strings.ToLower(goal) + " " + strings.ToLower(title)
	for _, keyword := range productionKeywords {
		if strings.Contains(combined, keyword) {
			return true, nil
		}
	}
	return false, nil
}

// EnsureProjectSecret returns the project HMAC secret, generating one when absent.
func (s *SQLiteStore) EnsureProjectSecret() ([]byte, error) {
	var hexSecret string
	err := s.db.QueryRow(
		`SELECT value FROM project_meta WHERE key='task_hmac_secret'`,
	).Scan(&hexSecret)

	if err == sql.ErrNoRows {
		// Generate a new secret.
		raw := make([]byte, projectSecretByteLength)
		if _, genErr := rand.Read(raw); genErr != nil {
			return nil, fmt.Errorf("EnsureProjectSecret generate: %w", genErr)
		}
		hexSecret = hex.EncodeToString(raw)
		_, insertErr := s.db.Exec(
			`INSERT OR IGNORE INTO project_meta (key, value) VALUES ('task_hmac_secret', ?)`,
			hexSecret,
		)
		if insertErr != nil {
			return nil, fmt.Errorf("EnsureProjectSecret insert: %w", insertErr)
		}
		// Re-query (guards against a race).
		if reErr := s.db.QueryRow(
			`SELECT value FROM project_meta WHERE key='task_hmac_secret'`,
		).Scan(&hexSecret); reErr != nil {
			return nil, fmt.Errorf("EnsureProjectSecret re-query: %w", reErr)
		}
	} else if err != nil {
		return nil, fmt.Errorf("EnsureProjectSecret query: %w", err)
	}

	decoded, err := hex.DecodeString(hexSecret)
	if err != nil {
		return nil, fmt.Errorf("EnsureProjectSecret decode: %w", err)
	}
	return decoded, nil
}

// RotateProjectSecret replaces task_hmac_secret in project_meta with newHexSecret.
func (s *SQLiteStore) RotateProjectSecret(newHexSecret string) error {
	_, err := s.db.Exec(
		`UPDATE project_meta SET value = ?, updated_at = datetime('now') WHERE key = 'task_hmac_secret'`,
		newHexSecret,
	)
	if err != nil {
		return fmt.Errorf("RotateProjectSecret: %w", err)
	}
	return nil
}

// IDStore.

// HealAndGetNextTaskID reconciles the counter with the actual DB state and
// then returns the next task ID.
func (s *SQLiteStore) HealAndGetNextTaskID() (string, error) {
	// 1. Look up the current maximum ID (NULL when the table is empty).
	var maxNumPtr *int64
	err := s.db.QueryRow(
		`SELECT MAX(CAST(SUBSTR(task_id,2) AS INTEGER)) FROM tasks WHERE task_id LIKE ?`,
		taskIDLikePattern,
	).Scan(&maxNumPtr)
	if err != nil && err != sql.ErrNoRows {
		return "", fmt.Errorf("HealAndGetNextTaskID max query: %w", err)
	}
	var maxNum int64
	if maxNumPtr != nil {
		maxNum = *maxNumPtr
	}

	// 2. Sync the counter.
	_, err = s.db.Exec(
		`INSERT INTO id_counters (counter_key,current_value,updated_at) VALUES (?,?,datetime('now'))
		 ON CONFLICT(counter_key) DO UPDATE SET
		   current_value=MAX(current_value,excluded.current_value),
		   updated_at=datetime('now')`,
		counterKeyTask, maxNum,
	)
	if err != nil {
		return "", fmt.Errorf("HealAndGetNextTaskID sync counter: %w", err)
	}

	// 3. Get the next counter value.
	seq, err := nextCounter(s.db, counterKeyTask)
	if err != nil {
		return "", fmt.Errorf("HealAndGetNextTaskID next counter: %w", err)
	}
	return fmt.Sprintf(taskIDFormat, seq), nil
}

// BumpTaskCounter raises the task counter to at least minValue.
// No-op when the counter is already >= minValue. Used to reflect the
// SPRINT.md / BACKLOG.md placeholder-scan result.
func (s *SQLiteStore) BumpTaskCounter(minValue int64) error {
	if minValue <= 0 {
		return nil
	}
	_, err := s.db.Exec(
		`INSERT INTO id_counters (counter_key,current_value,updated_at) VALUES (?,?,datetime('now'))
		 ON CONFLICT(counter_key) DO UPDATE SET
		   current_value=MAX(current_value,excluded.current_value),
		   updated_at=datetime('now')`,
		counterKeyTask, minValue,
	)
	if err != nil {
		return fmt.Errorf("BumpTaskCounter: %w", err)
	}
	return nil
}

// NextCounter returns the next sequence number for a given counter key.
func (s *SQLiteStore) NextCounter(counterKey string) (int64, error) {
	return nextCounter(s.db, counterKey)
}

// nextCounter is the internal helper that atomically increments the
// id_counters row and returns the new value.
func nextCounter(db *sql.DB, counterKey string) (int64, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, fmt.Errorf("nextCounter begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	_, err = tx.Exec(
		`INSERT OR IGNORE INTO id_counters (counter_key,current_value,updated_at) VALUES (?,0,datetime('now'))`,
		counterKey,
	)
	if err != nil {
		return 0, fmt.Errorf("nextCounter insert: %w", err)
	}

	_, err = tx.Exec(
		`UPDATE id_counters SET current_value=current_value+1, updated_at=datetime('now') WHERE counter_key=?`,
		counterKey,
	)
	if err != nil {
		return 0, fmt.Errorf("nextCounter update: %w", err)
	}

	var seq int64
	if err := tx.QueryRow(
		`SELECT current_value FROM id_counters WHERE counter_key=?`,
		counterKey,
	).Scan(&seq); err != nil {
		return 0, fmt.Errorf("nextCounter select: %w", err)
	}

	return seq, tx.Commit()
}

// Work Ticket / Context Acknowledgment.

// GetTaskWorkTicket returns the tasks.work_ticket value (empty when absent).
func (s *SQLiteStore) GetTaskWorkTicket(taskID string) (string, error) {
	var ticket sql.NullString
	err := s.db.QueryRow(
		"SELECT work_ticket FROM tasks WHERE task_id = ?", taskID,
	).Scan(&ticket)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("GetTaskWorkTicket: %w", err)
	}
	if !ticket.Valid {
		return "", nil
	}
	return ticket.String, nil
}

// SetTaskWorkTicket updates the tasks.work_ticket column.
func (s *SQLiteStore) SetTaskWorkTicket(taskID, ticket string) error {
	_, err := s.db.Exec(
		"UPDATE tasks SET work_ticket = ?, updated_at = datetime('now') WHERE task_id = ?",
		ticket, taskID,
	)
	if err != nil {
		return fmt.Errorf("SetTaskWorkTicket: %w", err)
	}
	return nil
}

// InsertContextAck records a single ack into context_acknowledgments (idempotent).
func (s *SQLiteStore) InsertContextAck(ticket, hash string, size int) error {
	_, err := s.db.Exec(
		`INSERT OR IGNORE INTO context_acknowledgments (ticket, hash, content_size)
		 VALUES (?, ?, ?)`,
		ticket, hash, size,
	)
	if err != nil {
		return fmt.Errorf("InsertContextAck: %w", err)
	}
	return nil
}

// HasContextAck reports whether the given ticket + hash pair has been recorded.
func (s *SQLiteStore) HasContextAck(ticket, hash string) (bool, error) {
	var cnt int
	err := s.db.QueryRow(
		"SELECT COUNT(*) FROM context_acknowledgments WHERE ticket = ? AND hash = ?",
		ticket, hash,
	).Scan(&cnt)
	if err != nil {
		return false, fmt.Errorf("HasContextAck: %w", err)
	}
	return cnt > 0, nil
}

// HasContextAckFresh applies a TTL: when ttlMinutes > 0, only acks
// created within the last ttlMinutes minutes are considered valid.
// ttlMinutes <= 0 behaves the same as HasContextAck.
// SQLite datetime: rows created after datetime('now', '-N minutes').
func (s *SQLiteStore) HasContextAckFresh(ticket, hash string, ttlMinutes int) (bool, error) {
	if ttlMinutes <= 0 {
		return s.HasContextAck(ticket, hash)
	}
	var cnt int
	// created_at uses the 'YYYY-MM-DD HH:MM:SS' format (tables.sql DEFAULT datetime('now')).
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM context_acknowledgments
		  WHERE ticket = ? AND hash = ?
		    AND created_at >= datetime('now', ?)`,
		ticket, hash, fmt.Sprintf("-%d minutes", ttlMinutes),
	).Scan(&cnt)
	if err != nil {
		return false, fmt.Errorf("HasContextAckFresh: %w", err)
	}
	return cnt > 0, nil
}

// Task Extended.

// GetTaskReopenInfo returns the composite Task fields needed by the Reopen operation.
func (s *SQLiteStore) GetTaskReopenInfo(taskID string) (*ports.TaskReopenInfo, error) {
	var r ports.TaskReopenInfo
	err := s.db.QueryRow(
		`SELECT task_id, status, file_path, COALESCE(sprint,''), title, type, estimate,
		        COALESCE(priority,'p2'), COALESCE(depends_on,'[]')
		   FROM tasks WHERE task_id = ?`,
		taskID,
	).Scan(
		&r.TaskID, &r.Status, &r.FilePath, &r.SprintID, &r.Title, &r.Type,
		&r.Estimate, &r.Priority, &r.DependsOnJSON,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("task not found: %s", taskID)
		}
		return nil, fmt.Errorf("GetTaskReopenInfo: %w", err)
	}
	return &r, nil
}

// UpdateTaskStatusDirect updates the task status without an expected-status check.
func (s *SQLiteStore) UpdateTaskStatusDirect(taskID, status string) error {
	_, err := s.db.Exec(
		"UPDATE tasks SET status = ?, updated_at = datetime('now') WHERE task_id = ?",
		status, taskID,
	)
	if err != nil {
		return fmt.Errorf("UpdateTaskStatusDirect: %w", err)
	}
	return nil
}

// TransitionTaskAtomic is the atomic status-transition entry point.
// Order of operations: BEGIN → SELECT status → UPDATE status → fileOp()
// → (when a new path is returned) UPDATE file_path → COMMIT. Any
// intermediate failure ROLLBACKs the whole transaction.
// The fileOp implementer is responsible for filesystem-side rollback
// (frontmatter / file move undo). When fileOp returns an error, the tx
// is rolled back so the DB side is restored automatically.
func (s *SQLiteStore) TransitionTaskAtomic(
	taskID, expectedStatus, newStatus string,
	fileOp func() (newFilePath string, err error),
) error {
	if fileOp == nil {
		return fmt.Errorf("TransitionTaskAtomic: fileOp callback is nil")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("TransitionTaskAtomic begin: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	var currentStatus string
	if err := tx.QueryRow(
		"SELECT status FROM tasks WHERE task_id = ?", taskID,
	).Scan(&currentStatus); err != nil {
		return fmt.Errorf("TransitionTaskAtomic select: %w", err)
	}
	if currentStatus != expectedStatus {
		return fmt.Errorf("cannot transition status: task=%s current=%s expected=%s",
			taskID, currentStatus, expectedStatus)
	}

	if _, err := tx.Exec(
		"UPDATE tasks SET status = ?, updated_at = datetime('now') WHERE task_id = ?",
		newStatus, taskID,
	); err != nil {
		return fmt.Errorf("TransitionTaskAtomic update status: %w", err)
	}

	newFilePath, fileErr := fileOp()
	if fileErr != nil {
		return fmt.Errorf("TransitionTaskAtomic fileOp: %w", fileErr)
	}

	if newFilePath != "" {
		if _, err := tx.Exec(
			"UPDATE tasks SET file_path = ?, updated_at = datetime('now') WHERE task_id = ?",
			newFilePath, taskID,
		); err != nil {
			return fmt.Errorf("TransitionTaskAtomic update file_path: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("TransitionTaskAtomic commit: %w", err)
	}
	committed = true
	return nil
}

// DeleteHarnessItemsForEntity deletes every harness_items row for the entity.
func (s *SQLiteStore) DeleteHarnessItemsForEntity(entityType, entityID string) error {
	_, err := s.db.Exec(
		"DELETE FROM harness_items WHERE entity_type = ? AND entity_id = ?",
		entityType, entityID,
	)
	if err != nil {
		return fmt.Errorf("DeleteHarnessItemsForEntity: %w", err)
	}
	return nil
}

// UpsertTaskInsert performs INSERT OR IGNORE on tasks and synchronises the counter.
func (s *SQLiteStore) UpsertTaskInsert(task *ports.TaskRecord, taskNum int) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("UpsertTaskInsert begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	var sprintVal interface{}
	if task.Sprint != "" {
		sprintVal = task.Sprint
	}
	// work_ticket is restored too (backlog sync db_missing_task repair path).
	var ticketVal interface{}
	if task.WorkTicket != "" {
		ticketVal = task.WorkTicket
	}

	_, err = tx.Exec(
		`INSERT OR IGNORE INTO tasks
		   (task_id,title,type,sprint,status,priority,estimate,file_path,depends_on,work_ticket,created_at,updated_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,datetime('now'))`,
		task.TaskID, task.Title, task.Type, sprintVal, task.Status,
		task.Priority, task.Estimate, task.FilePath, task.DependsOn,
		ticketVal, task.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("UpsertTaskInsert insert task: %w", err)
	}

	if taskNum > 0 {
		_, err = tx.Exec(
			`INSERT INTO id_counters (counter_key, current_value, updated_at)
			 VALUES ('task', ?, datetime('now'))
			 ON CONFLICT(counter_key) DO UPDATE
			   SET current_value = MAX(current_value, excluded.current_value),
			       updated_at    = datetime('now')`,
			taskNum,
		)
		if err != nil {
			return fmt.Errorf("UpsertTaskInsert sync counter: %w", err)
		}
	}

	return tx.Commit()
}

// UpsertSprintInsert performs INSERT OR REPLACE on sprints.
func (s *SQLiteStore) UpsertSprintInsert(sprint *ports.SprintRecord) error {
	var startedAt, completedAt, goal interface{}
	if sprint.StartedAt != "" {
		startedAt = sprint.StartedAt
	}
	if sprint.CompletedAt != "" {
		completedAt = sprint.CompletedAt
	}
	if sprint.Goal != "" {
		goal = sprint.Goal
	}
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO sprints
		   (sprint_id, title, status, folder_path,
		    started_at, completed_at, goal,
		    created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))`,
		sprint.SprintID, sprint.Title, sprint.Status, sprint.FolderPath,
		startedAt, completedAt, goal,
	)
	if err != nil {
		return fmt.Errorf("UpsertSprintInsert: %w", err)
	}
	return nil
}

// SetCounterMax upserts the counter to MAX(current, val).
func (s *SQLiteStore) SetCounterMax(counterKey string, val int64) error {
	_, err := s.db.Exec(
		`INSERT INTO id_counters (counter_key, current_value, updated_at)
		 VALUES (?, ?, datetime('now'))
		 ON CONFLICT(counter_key) DO UPDATE
		   SET current_value = MAX(current_value, excluded.current_value),
		       updated_at    = datetime('now')`,
		counterKey, val,
	)
	if err != nil {
		return fmt.Errorf("SetCounterMax: %w", err)
	}
	return nil
}

// GetNonArchivedTasks returns task_id and status for tasks that are not archived/superseded.
func (s *SQLiteStore) GetNonArchivedTasks() ([]ports.TaskSyncRecord, error) {
	rows, err := s.db.Query(
		"SELECT task_id, status FROM tasks WHERE status NOT IN ('archived', 'superseded')",
	)
	if err != nil {
		return nil, fmt.Errorf("GetNonArchivedTasks: %w", err)
	}
	defer rows.Close()

	var records []ports.TaskSyncRecord
	for rows.Next() {
		var r ports.TaskSyncRecord
		if err := rows.Scan(&r.TaskID, &r.Status); err != nil {
			return nil, fmt.Errorf("GetNonArchivedTasks scan: %w", err)
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

// ArchiveTasks marks the supplied task IDs as archived.
func (s *SQLiteStore) ArchiveTasks(taskIDs []string) error {
	if len(taskIDs) == 0 {
		return nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("ArchiveTasks begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	for _, tid := range taskIDs {
		_, err := tx.Exec(
			"UPDATE tasks SET status = 'archived', updated_at = datetime('now') WHERE task_id = ?",
			tid,
		)
		if err != nil {
			return fmt.Errorf("ArchiveTasks update %s: %w", tid, err)
		}
	}
	return tx.Commit()
}

// GetAllSprintsForSync returns the full sprint payload used during sync.
func (s *SQLiteStore) GetAllSprintsForSync() ([]ports.SprintFullSyncRecord, error) {
	rows, err := s.db.Query(
		`SELECT sprint_id, title, status,
		        COALESCE(started_at,''), COALESCE(completed_at,''),
		        COALESCE(folder_path,'')
		   FROM sprints`,
	)
	if err != nil {
		return nil, fmt.Errorf("GetAllSprintsForSync: %w", err)
	}
	defer rows.Close()

	var records []ports.SprintFullSyncRecord
	for rows.Next() {
		var r ports.SprintFullSyncRecord
		if err := rows.Scan(
			&r.SprintID, &r.Title, &r.Status,
			&r.StartedAt, &r.CompletedAt, &r.FolderPath,
		); err != nil {
			return nil, fmt.Errorf("GetAllSprintsForSync scan: %w", err)
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

// GetSprintTaskCounts returns the per-sprint Task count.
func (s *SQLiteStore) GetSprintTaskCounts() (map[string]int, error) {
	rows, err := s.db.Query(
		"SELECT sprint, COUNT(*) as cnt FROM tasks WHERE sprint IS NOT NULL AND sprint != '' GROUP BY sprint",
	)
	if err != nil {
		return nil, fmt.Errorf("GetSprintTaskCounts: %w", err)
	}
	defer rows.Close()

	counts := map[string]int{}
	for rows.Next() {
		var sprint string
		var cnt int
		if err := rows.Scan(&sprint, &cnt); err != nil {
			return nil, fmt.Errorf("GetSprintTaskCounts scan: %w", err)
		}
		counts[sprint] = cnt
	}
	return counts, rows.Err()
}

// UpdateTaskDirect updates the named fields unconditionally.
//
// Every UPDATE is wrapped with dblock.Exec so that task assign /
// sprint complete / reconcile and similar paths auto-retry on
// multi-worktree races.
func (s *SQLiteStore) UpdateTaskDirect(taskID string, fields ports.TaskUpdateFields) error {
	if fields.Status != nil {
		if _, err := dblock.Exec(s.db,
			"UPDATE tasks SET status = ?, updated_at = datetime('now') WHERE task_id = ?",
			*fields.Status, taskID,
		); err != nil {
			return fmt.Errorf("UpdateTaskDirect status: %w", err)
		}
	}
	if fields.Sprint != nil {
		var sprintVal interface{}
		if *fields.Sprint != "" {
			sprintVal = *fields.Sprint
		}
		if _, err := dblock.Exec(s.db,
			"UPDATE tasks SET sprint = ?, updated_at = datetime('now') WHERE task_id = ?",
			sprintVal, taskID,
		); err != nil {
			return fmt.Errorf("UpdateTaskDirect sprint: %w", err)
		}
	}
	if fields.FilePath != nil {
		if _, err := dblock.Exec(s.db,
			"UPDATE tasks SET file_path = ?, updated_at = datetime('now') WHERE task_id = ?",
			*fields.FilePath, taskID,
		); err != nil {
			return fmt.Errorf("UpdateTaskDirect file_path: %w", err)
		}
	}
	if fields.Title != nil {
		if _, err := s.db.Exec(
			"UPDATE tasks SET title = ?, updated_at = datetime('now') WHERE task_id = ?",
			*fields.Title, taskID,
		); err != nil {
			return fmt.Errorf("UpdateTaskDirect title: %w", err)
		}
	}
	if fields.Type != nil {
		if _, err := s.db.Exec(
			"UPDATE tasks SET type = ?, updated_at = datetime('now') WHERE task_id = ?",
			*fields.Type, taskID,
		); err != nil {
			return fmt.Errorf("UpdateTaskDirect type: %w", err)
		}
	}
	if fields.Priority != nil {
		if _, err := s.db.Exec(
			"UPDATE tasks SET priority = ?, updated_at = datetime('now') WHERE task_id = ?",
			*fields.Priority, taskID,
		); err != nil {
			return fmt.Errorf("UpdateTaskDirect priority: %w", err)
		}
	}
	if fields.Estimate != nil {
		if _, err := s.db.Exec(
			"UPDATE tasks SET estimate = ?, updated_at = datetime('now') WHERE task_id = ?",
			*fields.Estimate, taskID,
		); err != nil {
			return fmt.Errorf("UpdateTaskDirect estimate: %w", err)
		}
	}
	if fields.DependsOn != nil {
		if _, err := s.db.Exec(
			"UPDATE tasks SET depends_on = ?, updated_at = datetime('now') WHERE task_id = ?",
			*fields.DependsOn, taskID,
		); err != nil {
			return fmt.Errorf("UpdateTaskDirect depends_on: %w", err)
		}
	}
	return nil
}

// DeleteTask removes the task row from the tasks table.
func (s *SQLiteStore) DeleteTask(taskID string) error {
	if _, err := s.db.Exec("DELETE FROM tasks WHERE task_id = ?", taskID); err != nil {
		return fmt.Errorf("DeleteTask: %w", err)
	}
	return nil
}

// GetDoneTasksWithPaths returns ID + file_path for tasks in the done state.
func (s *SQLiteStore) GetDoneTasksWithPaths() ([]ports.TaskSyncRecord, error) {
	rows, err := s.db.Query(
		"SELECT task_id, COALESCE(file_path,''), COALESCE(sprint,'') FROM tasks WHERE status = 'done' ORDER BY task_id",
	)
	if err != nil {
		return nil, fmt.Errorf("GetDoneTasksWithPaths: %w", err)
	}
	defer rows.Close()

	var result []ports.TaskSyncRecord
	for rows.Next() {
		var r ports.TaskSyncRecord
		r.Status = "done"
		if err := rows.Scan(&r.TaskID, &r.FilePath, &r.Sprint); err != nil {
			return nil, fmt.Errorf("GetDoneTasksWithPaths scan: %w", err)
		}
		result = append(result, r)
	}
	return result, rows.Err()
}
