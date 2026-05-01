// Package app unit tests covering injected and missing LayoutResolver.
package app

import (
	"errors"
	"testing"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/ports"
)

// mockLayout is a test LayoutResolver. Each method returns a fixed
// value so accessors verify that the package delegates through the
// port.
type mockLayout struct {
	projectRoot  string
	mainRepoRoot string
	repoRelBase  string
	sprintDirMap map[string]string
}

// mockIDAlloc is a test IdentifierAllocator.
type mockIDAlloc struct {
	taskID     string
	counterVal int64
}

func (m *mockIDAlloc) NextTaskID() (string, error)             { return m.taskID, nil }
func (m *mockIDAlloc) NextCounter(key string) (int64, error)   { return m.counterVal, nil }

var _ ports.IdentifierAllocator = (*mockIDAlloc)(nil)

// mockFMCodec is a test FrontmatterCodec.
type mockFMCodec struct {
	decoded   map[string]any
	encoded   []byte
	updateRet []byte
}

func (m *mockFMCodec) Decode(content []byte) (map[string]any, string, error) {
	return m.decoded, string(content), nil
}
func (m *mockFMCodec) Encode(fm map[string]any, body string) ([]byte, error) {
	return m.encoded, nil
}
func (m *mockFMCodec) UpdateField(content []byte, f string, v any) ([]byte, error) {
	return m.updateRet, nil
}

var _ ports.FrontmatterCodec = (*mockFMCodec)(nil)

// mockFMFile is a test FrontmatterFileOps.
type mockFMFile struct {
	readRet   map[string]any
	updateErr error
}

func (m *mockFMFile) ReadTask(p string) (map[string]any, error)   { return m.readRet, nil }
func (m *mockFMFile) UpdateTask(p string, f map[string]any) error { return m.updateErr }

var _ FrontmatterFileOps = (*mockFMFile)(nil)

// mockKB is a test KBStore.
type mockKB struct {
	createRet *ports.KBCardRecord
	listRet   []ports.KBCardRecord
	rebuild   *ports.KBRebuildResult
}

func (m *mockKB) CreateCard(input ports.KBCardInput) (*ports.KBCardRecord, error) {
	return m.createRet, nil
}
func (m *mockKB) ListCards(f ports.KBListFilter) ([]ports.KBCardRecord, error) {
	return m.listRet, nil
}
func (m *mockKB) RebuildIndex() (*ports.KBRebuildResult, error) { return m.rebuild, nil }

var _ ports.KBStore = (*mockKB)(nil)

// mockAudit is a test AuditLog.
type mockAudit struct {
	appended []ports.AuditEvent
	queryRet []ports.AuditEvent
	summary  map[string]int
}

func (m *mockAudit) AppendAuditEvent(ev ports.AuditEvent) error {
	m.appended = append(m.appended, ev)
	return nil
}
func (m *mockAudit) QueryAuditEvents(f ports.AuditQueryFilter) ([]ports.AuditEvent, error) {
	return m.queryRet, nil
}
func (m *mockAudit) SummarizeAuditEvents(s, u string) (map[string]int, error) {
	return m.summary, nil
}

var _ ports.AuditLog = (*mockAudit)(nil)

// mockAuditFile is a test AuditFileOps.
type mockAuditFile struct {
	restoreRet any
	restoreErr error
}

func (m *mockAuditFile) Restore(dryRun bool) (any, error) { return m.restoreRet, m.restoreErr }

var _ AuditFileOps = (*mockAuditFile)(nil)

// mockHarness is a test HarnessService.
type mockHarness struct {
	getRet      any
	checkRet    any
	checkAllRet any
	autoRet     any
}

func (m *mockHarness) Get(t, id string) (any, error)                    { return m.getRet, nil }
func (m *mockHarness) Check(t, id, item, ev, actor string) (any, error) { return m.checkRet, nil }
func (m *mockHarness) CheckAll(t, id string) (any, error)               { return m.checkAllRet, nil }
func (m *mockHarness) AutoCheckCriteria(tid string) (any, error)        { return m.autoRet, nil }

var _ HarnessService = (*mockHarness)(nil)

// mockSprint is a test SprintService. Most methods return defaults — tests override as needed.
type mockSprint struct {
	listRet   any
	getRet    any
	createRet any
}

func (m *mockSprint) List(status string) (any, error)                         { return m.listRet, nil }
func (m *mockSprint) Get(id string) (any, error)                              { return m.getRet, nil }
func (m *mockSprint) Create(id, t, g string, ts []any) (any, error)           { return m.createRet, nil }
func (m *mockSprint) Start(id string) (any, error)                            { return nil, nil }
func (m *mockSprint) Complete(id string) (any, error)                         { return nil, nil }
func (m *mockSprint) Discard(id, reason string, rt, dbOnly bool) (any, error) { return nil, nil }
func (m *mockSprint) UpdateFields(id string, a, b, c, d *string) error        { return nil }
func (m *mockSprint) AggregateProgress(id string) (any, error)                { return nil, nil }
func (m *mockSprint) CheckCreateGuard(id, t string) error                     { return nil }
func (m *mockSprint) AppendAutoVerifyTask(id string) (any, error)             { return nil, nil }
func (m *mockSprint) FindSprintOnDisk(id string) (any, error)                 { return nil, nil }
func (m *mockSprint) Reconcile(id string, dry bool) (any, error)              { return nil, nil }
func (m *mockSprint) UpdateMDField(p, f, v string) error                      { return nil }

var _ SprintService = (*mockSprint)(nil)

// mockTask is a test TaskService — mostly pass-through.
type mockTask struct {
	listRet any
	getRet  any
	preset  any
}

func (m *mockTask) Create(title, taskType, sprint, priority, estimate, summary string, dependsOn []string) (any, error) {
	return nil, nil
}
func (m *mockTask) Get(id string) (any, error)                         { return m.getRet, nil }
func (m *mockTask) List(s, st *string) (any, error)                    { return m.listRet, nil }
func (m *mockTask) ListFiltered(_ any) (any, error)                    { return m.listRet, nil }
func (m *mockTask) Next() (any, error)                                 { return nil, nil }
func (m *mockTask) Start(id string) (any, error)                       { return nil, nil }
func (m *mockTask) Complete(id string, s bool) (any, error)            { return nil, nil }
func (m *mockTask) Reopen(id, r, s string) (any, error)                { return nil, nil }
func (m *mockTask) Update(req any) (any, error)                        { return nil, nil }
func (m *mockTask) DeleteWithOptions(id, r string, o any) (any, error) { return nil, nil }
func (m *mockTask) AssignSprint(ids []string, s string) (any, error)   { return nil, nil }
func (m *mockTask) UnassignSprint(ids []string) (any, error)           { return nil, nil }
func (m *mockTask) Checkpoint(id, r string) (any, error)               { return nil, nil }
func (m *mockTask) ResolvePreset(name string) any                      { return m.preset }
func (m *mockTask) HeuristicCheck(s string, c any) ([]string, bool)    { return nil, true }
func (m *mockTask) HasPlaceholderBody(p string) bool                   { return false }
func (m *mockTask) GetGitChangedFiles() []string                       { return nil }

var _ TaskService = (*mockTask)(nil)

func (m *mockLayout) ProjectRoot() string            { return m.projectRoot }
func (m *mockLayout) MainRepoRoot() string           { return m.mainRepoRoot }
func (m *mockLayout) RepoRelative(abs string) string { return m.repoRelBase + abs }
func (m *mockLayout) SprintDir(id string) (string, string, error) {
	if p, ok := m.sprintDirMap[id]; ok {
		return p, "active", nil
	}
	return "", "", errors.New("not found")
}

// UpdateCurrentFocus is a no-op in tests. Promote to a struct field
// counter when call-count verification becomes necessary.
func (m *mockLayout) UpdateCurrentFocus(activeSprintID, activeSprintTitle, status string) error {
	return nil
}

var _ ports.LayoutResolver = (*mockLayout)(nil)

func TestLayout_AccessorsDelegateAfterInjection(t *testing.T) {
	mock := &mockLayout{
		projectRoot:  "/proj",
		mainRepoRoot: "/main",
		repoRelBase:  "rel:",
		sprintDirMap: map[string]string{"sprint-1": "/proj/works/sprints/active/sprint-1"},
	}
	Init(&Services{Layout: mock})
	t.Cleanup(func() { current = nil })

	if got := ProjectRoot(); got != "/proj" {
		t.Errorf("ProjectRoot=%q, want /proj", got)
	}
	if got := MainRepoRoot(); got != "/main" {
		t.Errorf("MainRepoRoot=%q, want /main", got)
	}
	if got := RepoRelative("/x"); got != "rel:/x" {
		t.Errorf("RepoRelative=%q, want rel:/x", got)
	}
	p, loc, err := SprintDir("sprint-1")
	if err != nil || p != "/proj/works/sprints/active/sprint-1" || loc != "active" {
		t.Errorf("SprintDir=(%q,%q,%v)", p, loc, err)
	}
}

func TestLayout_NotInitialized_Panic(t *testing.T) {
	current = nil
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when Init was not called")
		}
	}()
	_ = ProjectRoot()
}

func TestLayout_LayoutFieldNil_Panic(t *testing.T) {
	Init(&Services{Layout: nil})
	t.Cleanup(func() { current = nil })
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when Layout is nil")
		}
	}()
	_ = ProjectRoot()
}

func TestIDAlloc_AccessorsDelegateAfterInjection(t *testing.T) {
	Init(&Services{
		Layout:  &mockLayout{projectRoot: "/p"},
		IDAlloc: &mockIDAlloc{taskID: "T999", counterVal: 42},
	})
	t.Cleanup(func() { current = nil })

	if id, err := NextTaskID(); err != nil || id != "T999" {
		t.Errorf("NextTaskID=(%q,%v), want (T999,nil)", id, err)
	}
	if n, err := NextCounter("any"); err != nil || n != 42 {
		t.Errorf("NextCounter=(%d,%v), want (42,nil)", n, err)
	}
}

func TestIDAlloc_Nil_Panic(t *testing.T) {
	Init(&Services{Layout: &mockLayout{}, IDAlloc: nil})
	t.Cleanup(func() { current = nil })
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when IDAlloc is nil")
		}
	}()
	_, _ = NextTaskID()
}

func TestFMCodec_AccessorsDelegateAfterInjection(t *testing.T) {
	Init(&Services{
		Layout:  &mockLayout{},
		IDAlloc: &mockIDAlloc{},
		FMCodec: &mockFMCodec{
			decoded:   map[string]any{"id": "T1"},
			encoded:   []byte("ENC"),
			updateRet: []byte("UPD"),
		},
		FMFile: &mockFMFile{readRet: map[string]any{"status": "done"}},
	})
	t.Cleanup(func() { current = nil })

	fm, _, err := DecodeFrontmatter([]byte("body"))
	if err != nil || fm["id"] != "T1" {
		t.Errorf("DecodeFrontmatter=(%v,%v), want id=T1", fm, err)
	}
	out, _ := EncodeFrontmatter(fm, "body")
	if string(out) != "ENC" {
		t.Errorf("EncodeFrontmatter=%q, want ENC", out)
	}
	upd, _ := UpdateFrontmatterField([]byte("x"), "k", "v")
	if string(upd) != "UPD" {
		t.Errorf("UpdateFrontmatterField=%q, want UPD", upd)
	}
	r, _ := ReadTaskFrontmatter("/any/path")
	if r["status"] != "done" {
		t.Errorf("ReadTaskFrontmatter=%v, want status=done", r)
	}
	if err := UpdateTaskFrontmatter("/p", map[string]any{"k": "v"}); err != nil {
		t.Errorf("UpdateTaskFrontmatter err=%v", err)
	}
}

func TestFMCodec_Nil_Panic(t *testing.T) {
	Init(&Services{Layout: &mockLayout{}, IDAlloc: &mockIDAlloc{}, FMCodec: nil})
	t.Cleanup(func() { current = nil })
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when FMCodec is nil")
		}
	}()
	_, _, _ = DecodeFrontmatter([]byte("x"))
}

func TestFMFile_Nil_Panic(t *testing.T) {
	Init(&Services{Layout: &mockLayout{}, IDAlloc: &mockIDAlloc{}, FMCodec: &mockFMCodec{}, FMFile: nil})
	t.Cleanup(func() { current = nil })
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when FMFile is nil")
		}
	}()
	_, _ = ReadTaskFrontmatter("/p")
}

func TestKB_AccessorsDelegateAfterInjection(t *testing.T) {
	wantRec := &ports.KBCardRecord{CardID: "M042", Title: "new"}
	wantList := []ports.KBCardRecord{{CardID: "M001"}, {CardID: "M002"}}
	wantRebuild := &ports.KBRebuildResult{TotalCards: 7, FilesScanned: 3}
	Init(&Services{
		Layout:  &mockLayout{},
		IDAlloc: &mockIDAlloc{},
		FMCodec: &mockFMCodec{},
		FMFile:  &mockFMFile{},
		KB:      &mockKB{createRet: wantRec, listRet: wantList, rebuild: wantRebuild},
	})
	t.Cleanup(func() { current = nil })

	rec, err := KBCreate(ports.KBCardInput{Category: "mistakes", Title: "x"})
	if err != nil || rec.CardID != "M042" {
		t.Errorf("KBCreate=(%v,%v), want CardID=M042", rec, err)
	}
	list, err := KBList(ports.KBListFilter{})
	if err != nil || len(list) != 2 {
		t.Errorf("KBList=(%v,%v), want len=2", list, err)
	}
	r, err := KBRebuild()
	if err != nil || r.TotalCards != 7 {
		t.Errorf("KBRebuild=(%v,%v), want TotalCards=7", r, err)
	}
}

func TestKB_Nil_Panic(t *testing.T) {
	Init(&Services{Layout: &mockLayout{}, IDAlloc: &mockIDAlloc{}, FMCodec: &mockFMCodec{}, FMFile: &mockFMFile{}, KB: nil})
	t.Cleanup(func() { current = nil })
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when KB is nil")
		}
	}()
	_, _ = KBCreate(ports.KBCardInput{})
}

func TestAudit_AccessorsDelegateAfterInjection(t *testing.T) {
	mAudit := &mockAudit{
		queryRet: []ports.AuditEvent{{ID: 1, EventType: "task.created"}},
		summary:  map[string]int{"task.created": 3},
	}
	mFile := &mockAuditFile{restoreRet: "restored-10"}
	Init(&Services{
		Layout:    &mockLayout{},
		IDAlloc:   &mockIDAlloc{},
		FMCodec:   &mockFMCodec{},
		FMFile:    &mockFMFile{},
		KB:        &mockKB{},
		Audit:     mAudit,
		AuditFile: mFile,
	})
	t.Cleanup(func() { current = nil })

	if err := AuditAppend("task.created", "task", "T1", "", nil, ""); err != nil {
		t.Errorf("AuditAppend err=%v", err)
	}
	if len(mAudit.appended) != 1 || mAudit.appended[0].ActorID != "claude" {
		t.Errorf("default actor 'claude' not applied: got %+v", mAudit.appended)
	}

	ev, err := AuditQuery(ports.AuditQueryFilter{EntityType: "task"})
	if err != nil || len(ev) != 1 {
		t.Errorf("AuditQuery=(%v,%v)", ev, err)
	}
	s, err := AuditSummarize("2026-01", "2026-12")
	if err != nil || s["task.created"] != 3 {
		t.Errorf("AuditSummarize=(%v,%v)", s, err)
	}
	r, err := AuditRestore(true)
	if err != nil || r != "restored-10" {
		t.Errorf("AuditRestore=(%v,%v)", r, err)
	}
}

func TestAudit_Nil_Panic(t *testing.T) {
	Init(&Services{Layout: &mockLayout{}, IDAlloc: &mockIDAlloc{}, FMCodec: &mockFMCodec{}, FMFile: &mockFMFile{}, KB: &mockKB{}, Audit: nil})
	t.Cleanup(func() { current = nil })
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when Audit is nil")
		}
	}()
	_ = AuditAppend("x", "y", "z", "", nil, "")
}

func TestUseAudit_PartialInjection(t *testing.T) {
	Init(&Services{Layout: &mockLayout{}, IDAlloc: &mockIDAlloc{}, FMCodec: &mockFMCodec{}, FMFile: &mockFMFile{}, KB: &mockKB{}})
	t.Cleanup(func() { current = nil })
	// Audit must be nil right after Init.
	if current.Audit != nil {
		t.Error("Audit must be nil immediately after Init")
	}
	// UseAudit injects later.
	UseAudit(&mockAudit{})
	if current.Audit == nil {
		t.Error("Audit must be set after UseAudit")
	}
}

func TestHarness_AccessorsDelegateAfterInjection(t *testing.T) {
	m := &mockHarness{
		getRet:      "state-1",
		checkRet:    "check-ok",
		checkAllRet: "all-ok",
		autoRet:     "auto-7",
	}
	Init(&Services{
		Layout: &mockLayout{}, IDAlloc: &mockIDAlloc{}, FMCodec: &mockFMCodec{},
		FMFile: &mockFMFile{}, KB: &mockKB{}, Audit: &mockAudit{},
		AuditFile: &mockAuditFile{}, Harness: m,
	})
	t.Cleanup(func() { current = nil })

	if r, _ := HarnessGet("task", "T1"); r != "state-1" {
		t.Errorf("HarnessGet=%v want state-1", r)
	}
	if r, _ := HarnessCheck("task", "T1", "i", "e", "cli"); r != "check-ok" {
		t.Errorf("HarnessCheck=%v want check-ok", r)
	}
	if r, _ := HarnessCheckAll("task", "T1"); r != "all-ok" {
		t.Errorf("HarnessCheckAll=%v want all-ok", r)
	}
	if r, _ := HarnessAutoCheckCriteria("T1"); r != "auto-7" {
		t.Errorf("HarnessAutoCheckCriteria=%v want auto-7", r)
	}
}

func TestHarness_Nil_Panic(t *testing.T) {
	Init(&Services{Layout: &mockLayout{}, IDAlloc: &mockIDAlloc{}, FMCodec: &mockFMCodec{}, FMFile: &mockFMFile{}, KB: &mockKB{}, Audit: &mockAudit{}, AuditFile: &mockAuditFile{}, Harness: nil})
	t.Cleanup(func() { current = nil })
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when Harness is nil")
		}
	}()
	_, _ = HarnessGet("task", "T1")
}

func TestSprint_AccessorsDelegateAfterInjection(t *testing.T) {
	m := &mockSprint{
		listRet:   "sprints-3",
		getRet:    "rec-1",
		createRet: "created-X",
	}
	Init(&Services{
		Layout: &mockLayout{}, IDAlloc: &mockIDAlloc{}, FMCodec: &mockFMCodec{},
		FMFile: &mockFMFile{}, KB: &mockKB{}, Audit: &mockAudit{},
		AuditFile: &mockAuditFile{}, Harness: &mockHarness{}, Sprint: m,
	})
	t.Cleanup(func() { current = nil })

	if r, _ := SprintList("active"); r != "sprints-3" {
		t.Errorf("SprintList=%v", r)
	}
	if r, _ := SprintGet("sprint-1"); r != "rec-1" {
		t.Errorf("SprintGet=%v", r)
	}
	if r, _ := SprintCreate("sprint-1", "t", "g", nil); r != "created-X" {
		t.Errorf("SprintCreate=%v", r)
	}
}

func TestSprint_Nil_Panic(t *testing.T) {
	Init(&Services{Layout: &mockLayout{}, IDAlloc: &mockIDAlloc{}, FMCodec: &mockFMCodec{}, FMFile: &mockFMFile{}, KB: &mockKB{}, Audit: &mockAudit{}, AuditFile: &mockAuditFile{}, Harness: &mockHarness{}, Sprint: nil})
	t.Cleanup(func() { current = nil })
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when Sprint is nil")
		}
	}()
	_, _ = SprintList("active")
}

func TestTask_AccessorsDelegateAfterInjection(t *testing.T) {
	m := &mockTask{listRet: "tasks-N", getRet: "task-rec", preset: "preset-moderate"}
	Init(&Services{
		Layout: &mockLayout{}, IDAlloc: &mockIDAlloc{}, FMCodec: &mockFMCodec{},
		FMFile: &mockFMFile{}, KB: &mockKB{}, Audit: &mockAudit{},
		AuditFile: &mockAuditFile{}, Harness: &mockHarness{}, Sprint: &mockSprint{},
		Task: m,
	})
	t.Cleanup(func() { current = nil })

	if r, _ := TaskList(nil, nil); r != "tasks-N" {
		t.Errorf("TaskList=%v", r)
	}
	if r, _ := TaskGet("T1"); r != "task-rec" {
		t.Errorf("TaskGet=%v", r)
	}
	if p := TaskResolvePreset("x"); p != "preset-moderate" {
		t.Errorf("TaskResolvePreset=%v", p)
	}
}

func TestTask_Nil_Panic(t *testing.T) {
	Init(&Services{Layout: &mockLayout{}, IDAlloc: &mockIDAlloc{}, FMCodec: &mockFMCodec{}, FMFile: &mockFMFile{}, KB: &mockKB{}, Audit: &mockAudit{}, AuditFile: &mockAuditFile{}, Harness: &mockHarness{}, Sprint: &mockSprint{}, Task: nil})
	t.Cleanup(func() { current = nil })
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when Task is nil")
		}
	}()
	_, _ = TaskList(nil, nil)
}

func TestInit_RecallOverrides(t *testing.T) {
	Init(&Services{Layout: &mockLayout{projectRoot: "/first"}})
	Init(&Services{Layout: &mockLayout{projectRoot: "/second"}})
	t.Cleanup(func() { current = nil })
	if got := ProjectRoot(); got != "/second" {
		t.Errorf("Init re-call did not override: got=%q, want=/second", got)
	}
}
