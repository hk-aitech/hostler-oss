-- hostler-oss SQLite schema.
-- WAL mode + IF NOT EXISTS for idempotent execution.

-- ---------------------------------------------------------------------------
-- ID counters (atomic allocation)
-- counter_key example: 'task'.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS id_counters (
    counter_key   TEXT    PRIMARY KEY,
    current_value INTEGER NOT NULL DEFAULT 0,
    updated_at    TEXT    NOT NULL DEFAULT (datetime('now'))
);

-- ---------------------------------------------------------------------------
-- Task cache (files are the SSOT; the DB is for fast queries / aggregation).
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS tasks (
    task_id      TEXT PRIMARY KEY,          -- 'T001'
    title        TEXT NOT NULL,
    type         TEXT NOT NULL,             -- feature/bugfix/refactor/infra/docs/test/chore
    sprint       TEXT,                      -- 'sprint-01' or NULL (backlog)
    status       TEXT NOT NULL,             -- todo/in-progress/done
    priority     TEXT,                      -- p0/p1/p2/p3
    estimate     TEXT,                      -- XS/S/M/L/XL
    file_path    TEXT NOT NULL,
    depends_on   TEXT,                      -- JSON array: ["T000", "T002"]
    work_ticket  TEXT,                      -- 'WT-TNN-...' or NULL
    created_at   TEXT NOT NULL,
    updated_at   TEXT NOT NULL
);

-- ---------------------------------------------------------------------------
-- Sprint cache.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS sprints (
    sprint_id    TEXT PRIMARY KEY,          -- 'sprint-01'
    title        TEXT NOT NULL,
    status       TEXT NOT NULL,             -- backlog/active/completed
    folder_path  TEXT NOT NULL,
    started_at   TEXT,
    completed_at TEXT,
    goal         TEXT,
    created_at   TEXT NOT NULL,
    updated_at   TEXT NOT NULL
);

-- ---------------------------------------------------------------------------
-- Harness Gate checklist (shared by Task + Sprint).
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS harness_items (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    entity_type TEXT    NOT NULL,           -- 'task' or 'sprint'
    entity_id   TEXT    NOT NULL,           -- 'T001' or 'sprint-01'
    item_id     TEXT    NOT NULL,           -- 'build_passed', 'phase5_retro', ...
    item_name   TEXT    NOT NULL,
    required    INTEGER NOT NULL DEFAULT 1, -- 1=required, 0=optional
    done        INTEGER NOT NULL DEFAULT 0, -- 0=open, 1=done
    evidence    TEXT,                       -- commit SHA / report ID / job ID / etc.
    checked_at  TEXT,
    checked_by  TEXT,                       -- 'claude' or 'human:<name>'
    UNIQUE(entity_type, entity_id, item_id)
);

-- ---------------------------------------------------------------------------
-- Harness templates (per Task type + Sprint 10-Phase).
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS harness_templates (
    template_key TEXT PRIMARY KEY,          -- 'task:feature', 'sprint:default', ...
    config       TEXT NOT NULL              -- JSON: [{item_id, item_name, required, ...}]
);

-- ---------------------------------------------------------------------------
-- Worktree management.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS worktrees (
    sprint_id          TEXT PRIMARY KEY,    -- 'sprint-01'
    path               TEXT NOT NULL,
    branch             TEXT NOT NULL,
    plugin_symlink_path TEXT,
    created_at         TEXT NOT NULL,
    removed_at         TEXT                 -- NULL = active, TEXT = removed-at timestamp
);

-- ---------------------------------------------------------------------------
-- Audit log (every state change).
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS audit_events (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp   TEXT    NOT NULL DEFAULT (datetime('now')),
    event_type  TEXT    NOT NULL,           -- task.created, task.transitioned, sprint.completed, harness.blocked, ...
    entity_type TEXT    NOT NULL,           -- 'task', 'sprint', 'worktree'
    entity_id   TEXT    NOT NULL,           -- 'T001', 'sprint-01'
    actor       TEXT    NOT NULL,           -- 'claude' or 'human:<name>'
    details     TEXT,                       -- JSON (before/after values, reasoning, etc.)
    session_id  TEXT,                       -- AI session identifier (optional)
    -- Audit Events Hash Chain — tamper evidence.
    prev_hash   TEXT,
    event_hash  TEXT
);

-- ---------------------------------------------------------------------------
-- Subagent context-acknowledgement records.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS context_acknowledgments (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    ticket       TEXT    NOT NULL,
    hash         TEXT    NOT NULL,
    content_size INTEGER NOT NULL,
    created_at   TEXT    NOT NULL DEFAULT (datetime('now')),
    UNIQUE(ticket, hash)
);

-- ---------------------------------------------------------------------------
-- Project-level key/value metadata (project_key, hmac secrets, etc.).
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS project_meta (
    key        TEXT    PRIMARY KEY,
    value      TEXT    NOT NULL,
    created_at TEXT    NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT    NOT NULL DEFAULT (datetime('now'))
);

-- ---------------------------------------------------------------------------
-- Indexes.
-- ---------------------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_ctx_ack_ticket ON context_acknowledgments(ticket);
CREATE INDEX IF NOT EXISTS idx_tasks_sprint  ON tasks(sprint);
CREATE INDEX IF NOT EXISTS idx_tasks_status  ON tasks(status);
CREATE INDEX IF NOT EXISTS idx_harness_entity ON harness_items(entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_audit_entity  ON audit_events(entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_audit_time    ON audit_events(timestamp);
