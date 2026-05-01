// Package app is the service layer. CLI commands access domain ports
// only through this package; the composition root injects concrete
// adapters via `Init`.
//
// Design principles:
//   - cmd/hstl-oss/cmd/*.go does not import pkg/* (e.g. fileutil) directly;
//     it calls the package-level accessors of this package
//     (app.ProjectRoot, etc.).
//   - The composition root sets the default once
//     (cmd/hstl-oss/cmd/composition.go `init()`). Tests may re-call
//     Init to inject mock adapters.
//   - Calling an accessor before Init panics — surfaces mistakes early.
package app

import (
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/ports"
)

// Services is the bag of ports the composition root injects. Once the
// migration is complete it gathers Layout / IDAlloc / FMCodec / KBStore
// and the rest in one place.
type Services struct {
	// Layout computes works/ layout paths.
	Layout ports.LayoutResolver

	// IDAlloc issues sequential T### IDs.
	IDAlloc ports.IdentifierAllocator

	// FMCodec parses and serialises YAML frontmatter.
	FMCodec ports.FrontmatterCodec

	// FMFile is a file-based convenience wrapper around FrontmatterCodec.
	// Extends the port's pure in-memory operations with file read/write.
	// Currently backed by fileutil's Read/UpdateTaskFrontmatter — slated
	// for absorption into the adapter later.
	FMFile FrontmatterFileOps

	// KB exposes KB card CRUD + aggregation.
	KB ports.KBStore

	// Audit exposes audit-event append + query + summarisation.
	Audit ports.AuditLog

	// AuditFile is a file-based audit restore convenience wrapper that
	// sits outside the port. It encapsulates pkg/audit.Restore behind the
	// adapter pattern.
	AuditFile AuditFileOps

	// Harness exposes high-level Harness Gate operations.
	// Layered on top of the port's primitive methods
	// (EnsureHarnessItems / CheckHarnessItem / ...) by adding the
	// pkg/harness aggregations (HarnessGet / CheckHarness /
	// AutoCheckCriteria, ...). May fold into the port later.
	Harness HarnessService

	// Sprint exposes high-level Sprint lifecycle operations.
	// Pass-through wrapper over pkg/sprint's 16+ public APIs, mirroring
	// HarnessService. Returns are typed `any` and re-asserted to
	// concrete pkg/sprint types in cmd. The port + domain types may be
	// promoted later.
	Sprint SprintService

	// Task exposes high-level Task lifecycle operations.
	// Pass-through wrapper over pkg/task's 27+ public APIs, mirroring
	// SprintService. Returns are typed `any` and re-asserted in cmd.
	Task TaskService

	// Backlog exposes BACKLOG.md sync + regeneration.
	// Pass-through wrapper over pkg/backlog's Sync / RebuildMD.
	// Returns are typed `any` and re-asserted to *SyncResult /
	// *RebuildMDResult in cmd.
	Backlog BacklogService

	// GateLog exposes gate-judgement log observability.
	// Pass-through wrapper over pkg/gatejudgement's GlobalStore admin
	// operations + RecordJudgement sampling. ports.GateJudgementLog
	// covers Record/Query/Stats/MarkFP — this service additionally
	// exposes the sampled Record (RecordJudgement).
	GateLog GateLogService

	// Ceremony exposes Sprint/Task start/complete ceremony collection.
	// Pass-through wrapper over pkg/ceremony's four Collect* functions.
	// Returns are typed `any` and re-asserted to *SprintStartCeremony /
	// *SprintCompleteCeremony / *TaskStartCeremony /
	// *TaskCompleteCeremony in cmd.
	Ceremony CeremonyService

	// Config exposes project-config queries + problem detection +
	// auto-fix.
	// Pass-through wrapper over pkg/config's DetectConfigProblems /
	// ApplyConfigFixes / RenderDiff / CollectEffective / GetProjectKey.
	Config ConfigService

	// Rules exposes the high-level Rule Engine query/run service.
	// Pass-through wrapper over pkg/rules's List / Get / LoadConfig /
	// LoadConfigBytes / ResolveEffective. Returns are typed `any` and
	// re-asserted to concrete pkg/rules types in cmd (preserves
	// layering — KB A028). Distinct from ports.RuleRunner: the latter
	// exposes a domain-neutral RuleDescriptor whereas this service is a
	// cmd-call-convenience pass-through.
	Rules RuleRunnerService
}

// BacklogService describes the BACKLOG.md sync + regeneration use cases.
// Pass-through over pkg/backlog's two cmd entry points; opts carries
// the concrete RebuildMDOptions value as `any` (preserves layering).
type BacklogService interface {
	// Sync regenerates BACKLOG.md and verifies DB / file drift. Returns
	// *backlog.SyncResult.
	Sync(dryRun bool) (any, error)

	// SyncWithOptions adds option granularity. When
	// allowStatusRegression=true regression is permitted and an audit
	// event is recorded.
	SyncWithOptions(dryRun, allowStatusRegression bool) (any, error)

	// RebuildMD regenerates the BACKLOG.md file. Returns
	// *backlog.RebuildMDResult.
	RebuildMD(opts any) (any, error)
}

// CeremonyService describes the Sprint/Task ceremony collection use cases.
// Pass-through over pkg/ceremony's four Collect* functions; returns are
// `any` and re-asserted to the concrete ceremony types in cmd.
type CeremonyService interface {
	CollectSprintStart(sprintID string) any
	CollectSprintComplete(sprintID string) any
	CollectTaskStart(taskID string) any
	CollectTaskComplete(taskID string) any
}

// ConfigService describes the project-config use cases.
// Pass-through over pkg/config's five cmd entry points — DetectProblems
// / ApplyFixes / RenderDiff / CollectEffective / GetProjectKey. Returns
// are `any` and re-asserted to *ProblemReport / *FixResult /
// *EffectiveConfig in cmd.
type ConfigService interface {
	// DetectProblems collects the issues in the YAML config at yamlPath.
	// Returns *config.ProblemReport.
	DetectProblems(yamlPath string) (any, error)

	// ApplyFixes applies fixes derived from a *config.ProblemReport.
	// When dryRun=true no files are touched and only the plan is
	// returned. Returns *config.FixResult.
	ApplyFixes(report any, dryRun bool) (any, error)

	// RenderDiff returns the unified diff between original and migrated
	// YAML byte slices.
	RenderDiff(original, migrated []byte) string

	// CollectEffective merges global config + project config + env vars.
	// Returns *config.EffectiveConfig.
	CollectEffective() any

	// GetProjectKey returns the current project key
	// (project-config.yaml `project.key`).
	GetProjectKey() string
}

// RuleRunnerService describes the Rule Engine use cases.
// Pass-through over pkg/rules's five cmd entry points. Returns are `any`
// because internal/app must not import pkg/rules (preserves layering).
// cmd reasserts the concrete types:
//
//	allRules := app.RulesList().([]rules.Rule)
//	cfg := app.RulesLoadConfig(root).(*rules.EngineConfig)
//
// Distinct from ports.RuleRunner: the latter is the domain-neutral
// abstraction that exposes RuleDescriptor; this interface is the
// cmd-path direct pass-through (reflecting that cmd already deals with
// concrete pkg/rules types).
type RuleRunnerService interface {
	// List returns every registered Rule. Returns []rules.Rule.
	List() any

	// Get returns a specific Rule. Returns rules.Rule.
	Get(ruleID string) (any, bool)

	// LoadConfig loads rules.yaml from projectRoot and resolves
	// the cascade. Returns *rules.EngineConfig.
	LoadConfig(projectRoot string) (any, error)

	// LoadConfigBytes loads config from raw bytes. Returns
	// *rules.EngineConfig.
	LoadConfigBytes(data []byte) (any, error)

	// ResolveEffective computes the effective rule map from config +
	// projectRoot. cfg is *rules.EngineConfig — accepted as `any` and
	// re-asserted by the adapter. Returns map[string]*rules.EffectiveRule.
	ResolveEffective(cfg any, projectRoot string) (any, error)
}

// GateLogService describes the gate-judgement log use cases.
// Pass-through over pkg/gatejudgement's GlobalStore admin operations
// plus the sampled Record. Reuses the ports.GateJudgementLog interface
// to keep type safety.
type GateLogService interface {
	// Record performs a sampled write (disabled check + per-verdict
	// sampling). The caller does not need to propagate errors —
	// observability path failures are silent.
	Record(r ports.JudgementRecord)

	// RecordDirect bypasses sampling for a full record (admin
	// inject/replay use).
	RecordDirect(r ports.JudgementRecord) error

	// Query reads filtered records.
	Query(filter ports.JudgementQuery) ([]ports.JudgementRecord, error)

	// Stats aggregates verdicts / rules / FP labels.
	Stats(filter ports.JudgementQuery) (*ports.JudgementStats, error)

	// MarkFalsePositive appends an FP label.
	MarkFalsePositive(recordID string, label ports.FalsePositiveLabel) error
}

// TaskService describes the task lifecycle use cases.
// Concrete return types are exposed as `any` so that internal/app stays free
// of pkg/task imports; callers reassert the concrete types.
type TaskService interface {
	// CRUD / state transitions
	Create(title, taskType, sprint, priority, estimate, summary string, dependsOn []string) (any, error)
	Get(taskID string) (any, error)
	List(sprintPtr, statusPtr *string) (any, error)
	ListFiltered(filter any) (any, error)
	Next() (any, error)
	Start(taskID string) (any, error)
	Complete(taskID string, skipHarness bool) (any, error)
	Reopen(taskID, reason, newStatus string) (any, error)
	Update(req any) (any, error)
	DeleteWithOptions(taskID, reason string, opts any) (any, error)
	AssignSprint(taskIDs []string, sprintID string) (any, error)
	UnassignSprint(taskIDs []string) (any, error)
	Checkpoint(taskID, reason string) (any, error)

	// Helpers
	ResolvePreset(name string) any
	HeuristicCheck(summary string, cfg any) ([]string, bool)
	HasPlaceholderBody(filePath string) bool
	GetGitChangedFiles() []string
}

// SprintService describes the sprint lifecycle use cases.
type SprintService interface {
	// CRUD / state transitions
	List(status string) (any, error)
	Get(sprintID string) (any, error)
	Create(id, title, goal string, tasks []any) (any, error)
	Start(sprintID string) (any, error)
	Complete(sprintID string) (any, error)
	Discard(sprintID, reason string, returnTasks, dbOnly bool) (any, error)
	UpdateFields(id string, title, goal, status, folder *string) error

	// Aggregation / validation
	AggregateProgress(sprintID string) (any, error)
	CheckCreateGuard(id, title string) error
	AppendAutoVerifyTask(sprintID string) (any, error)
	FindSprintOnDisk(sprintID string) (any, error)
	Reconcile(sprintID string, dryRun bool) (any, error)

	// SPRINT.md field updates
	UpdateMDField(path, field, value string) error
}

// HarnessService is the high-level Harness Gate use case.
// ports.HarnessStore exposes primitives (item CRUD); this interface adds
// the domain-level aggregations (Get / Check / AutoCheckCriteria).
// Currently implemented by pkg/harness.
//
// Returns are pkg/harness concrete types passed as `any` and re-asserted
// in cmd (matching the AuditFileOps.Restore strategy). Once
// domain/harness gains neutral types they can be promoted onto the port.
type HarnessService interface {
	// Get returns the detailed Harness state for entityType+entityID.
	// Returns *harness.HarnessGetResult (re-asserted in cmd).
	Get(entityType, entityID string) (any, error)

	// Check toggles a single item (state transition). Returns
	// domain.HarnessResult as `any`.
	Check(entityType, entityID, itemID, evidence, actor string) (any, error)

	// CheckAll verifies that every required item has been checked.
	// Returns domain.HarnessResult as `any`.
	CheckAll(entityType, entityID string) (any, error)

	// AutoCheckCriteria flips every checkbox under the Task body's
	// `## Done Criteria` to [x] automatically. Returns
	// harness.CriteriaResult as `any`.
	AutoCheckCriteria(taskID string) (any, error)
}

// AuditFileOps captures file-based audit restore operations that sit
// outside the port surface. The port covers append/query/summarize;
// `cmd/audit.go restore` additionally needs a filesystem → DB
// re-injection use case. Slated for absorption into a single adapter
// later (mirrors the FrontmatterFileOps transition).
type AuditFileOps interface {
	// Restore loads an audit-backup JSON file into the DB. dryRun=true
	// returns the preview without applying any change.
	Restore(dryRun bool) (any, error)
}

// FrontmatterFileOps performs frontmatter read/update against a file
// path. The pure codec is in-memory only, but cmd carries file paths,
// so this convenience interface is added. Will be merged into a single
// adapter method later.
type FrontmatterFileOps interface {
	// ReadTask parses only the frontmatter at filePath into a map.
	ReadTask(filePath string) (map[string]any, error)

	// UpdateTask updates a subset of frontmatter fields at filePath
	// (body untouched).
	UpdateTask(filePath string, fields map[string]any) error
}

// current is the active Services that Init configured. Accessors
// (Layout, etc.) reference it.
var current *Services

// Init is called once by the composition root. A second call overrides
// the previous injection (the path tests use to swap adapters).
func Init(s *Services) {
	current = s
}

// UseAudit injects ports.AuditLog after DB initialisation. The SQLite
// adapter is not ready at init() time, so the composition root calls
// this after a successful initDB() to attach the Audit port. Calling it
// before Init() panics.
func UseAudit(a ports.AuditLog) {
	ensureInitialized()
	current.Audit = a
}

// ensureInitialized verifies that Init was called before any accessor
// runs. A missing Init is treated as a runtime mistake and panics so it
// is caught early.
func ensureInitialized() {
	if current == nil {
		panic("app.Init not called — the composition root must call app.Init(&app.Services{...}) first")
	}
}

// Layout returns the injected LayoutResolver. A nil injection is a
// configuration error.
func Layout() ports.LayoutResolver {
	ensureInitialized()
	if current.Layout == nil {
		panic("app.Services.Layout is nil — composition must inject a LayoutResolver")
	}
	return current.Layout
}

// ProjectRoot returns the absolute project root path, replacing
// fileutil.GetProjectRoot via the port.
func ProjectRoot() string { return Layout().ProjectRoot() }

// MainRepoRoot returns the main-repo path of the current git worktree,
// replacing fileutil.GetMainRepoRoot.
func MainRepoRoot() string { return Layout().MainRepoRoot() }

// RepoRelative converts an absolute path to a main-repo-relative path,
// replacing fileutil.ToRepoRelative.
func RepoRelative(absPath string) string { return Layout().RepoRelative(absPath) }

// SprintDir locates the Sprint directory for sprintID, replacing
// fileutil.FindSprintDir.
// Returns: (absolute path, location "active"|"backlog"|"completed", error).
func SprintDir(sprintID string) (string, string, error) { return Layout().SprintDir(sprintID) }

// UpdateCurrentFocus refreshes works/CURRENT-FOCUS.md whenever a Sprint
// transitions state. Routes through the LayoutResolver port so cmd no
// longer calls pkg/fileutil directly.
func UpdateCurrentFocus(activeSprintID, activeSprintTitle, status string) error {
	return Layout().UpdateCurrentFocus(activeSprintID, activeSprintTitle, status)
}

// IDAlloc returns the injected IdentifierAllocator.
func IDAlloc() ports.IdentifierAllocator {
	ensureInitialized()
	if current.IDAlloc == nil {
		panic("app.Services.IDAlloc is nil — composition must inject an IdentifierAllocator")
	}
	return current.IDAlloc
}

// NextTaskID replaces pkg/id.TaskIDNext via the port.
func NextTaskID() (string, error) { return IDAlloc().NextTaskID() }

// NextCounter replaces pkg/id.NextCounter via the port.
func NextCounter(counterKey string) (int64, error) { return IDAlloc().NextCounter(counterKey) }

// FMCodec returns the injected FrontmatterCodec.
func FMCodec() ports.FrontmatterCodec {
	ensureInitialized()
	if current.FMCodec == nil {
		panic("app.Services.FMCodec is nil — composition must inject a FrontmatterCodec")
	}
	return current.FMCodec
}

// DecodeFrontmatter decodes markdown content into (frontmatter map, body).
func DecodeFrontmatter(content []byte) (map[string]any, string, error) {
	return FMCodec().Decode(content)
}

// EncodeFrontmatter encodes (frontmatter, body) into a complete
// serialised markdown document.
func EncodeFrontmatter(fm map[string]any, body string) ([]byte, error) {
	return FMCodec().Encode(fm, body)
}

// UpdateFrontmatterField updates a single frontmatter field within content.
func UpdateFrontmatterField(content []byte, field string, value any) ([]byte, error) {
	return FMCodec().UpdateField(content, field, value)
}

// fmFile returns the injected FrontmatterFileOps.
func fmFile() FrontmatterFileOps {
	ensureInitialized()
	if current.FMFile == nil {
		panic("app.Services.FMFile is nil — composition must inject FrontmatterFileOps")
	}
	return current.FMFile
}

// ReadTaskFrontmatter parses only the frontmatter at filePath into a map.
// Replaces fileutil.ReadTaskFrontmatter via the port.
func ReadTaskFrontmatter(filePath string) (map[string]any, error) {
	return fmFile().ReadTask(filePath)
}

// UpdateTaskFrontmatter updates a subset of frontmatter fields at
// filePath. Replaces fileutil.UpdateTaskFrontmatter via the port.
func UpdateTaskFrontmatter(filePath string, fields map[string]any) error {
	return fmFile().UpdateTask(filePath, fields)
}

// KB returns the injected KBStore.
func KB() ports.KBStore {
	ensureInitialized()
	if current.KB == nil {
		panic("app.Services.KB is nil — composition must inject a KBStore")
	}
	return current.KB
}

// KBCreate creates a KB card. Replaces several pkg/kb low-level
// function calls via the port.
func KBCreate(input ports.KBCardInput) (*ports.KBCardRecord, error) {
	return KB().CreateCard(input)
}

// KBList returns KB cards matching filter.
func KBList(filter ports.KBListFilter) ([]ports.KBCardRecord, error) {
	return KB().ListCards(filter)
}

// KBRebuild rebuilds INDEX.md.
func KBRebuild() (*ports.KBRebuildResult, error) { return KB().RebuildIndex() }

// Audit returns the injected AuditLog.
func Audit() ports.AuditLog {
	ensureInitialized()
	if current.Audit == nil {
		panic("app.Services.Audit is nil — composition must inject an AuditLog")
	}
	return current.Audit
}

// AuditAppend appends an audit event. Replaces pkg/audit.LogEvent via
// the port. Keeps the 6-arg call shape so cmd substitution stays cheap.
func AuditAppend(eventType, entityType, entityID, actor string, details map[string]any, sessionID string) error {
	if actor == "" {
		actor = "claude"
	}
	return Audit().AppendAuditEvent(ports.AuditEvent{
		EventType:  eventType,
		EntityType: entityType,
		EntityID:   entityID,
		ActorID:    actor,
		SessionID:  sessionID,
		Details:    details,
	})
}

// AuditQuery returns audit events matching filter.
func AuditQuery(filter ports.AuditQueryFilter) ([]ports.AuditEvent, error) {
	return Audit().QueryAuditEvents(filter)
}

// AuditSummarize aggregates events by type for the given period.
func AuditSummarize(since, until string) (map[string]int, error) {
	return Audit().SummarizeAuditEvents(since, until)
}

// auditFile returns the injected AuditFileOps.
func auditFile() AuditFileOps {
	ensureInitialized()
	if current.AuditFile == nil {
		panic("app.Services.AuditFile is nil — composition must inject AuditFileOps")
	}
	return current.AuditFile
}

// AuditRestore restores an audit-backup JSON file. Replaces
// pkg/audit.Restore via the port.
func AuditRestore(dryRun bool) (any, error) { return auditFile().Restore(dryRun) }

// Harness returns the injected HarnessService.
func Harness() HarnessService {
	ensureInitialized()
	if current.Harness == nil {
		panic("app.Services.Harness is nil — composition must inject a HarnessService")
	}
	return current.Harness
}

// HarnessGet replaces harness.HarnessGet via the port.
// Returns *harness.HarnessGetResult, re-asserted in cmd.
func HarnessGet(entityType, entityID string) (any, error) {
	return Harness().Get(entityType, entityID)
}

// HarnessCheck replaces harness.HarnessCheck via the port.
func HarnessCheck(entityType, entityID, itemID, evidence, actor string) (any, error) {
	return Harness().Check(entityType, entityID, itemID, evidence, actor)
}

// HarnessCheckAll replaces harness.CheckHarness via the port.
func HarnessCheckAll(entityType, entityID string) (any, error) {
	return Harness().CheckAll(entityType, entityID)
}

// HarnessAutoCheckCriteria replaces harness.AutoCheckCriteria via the port.
func HarnessAutoCheckCriteria(taskID string) (any, error) {
	return Harness().AutoCheckCriteria(taskID)
}

// Sprint returns the injected SprintService.
func Sprint() SprintService {
	ensureInitialized()
	if current.Sprint == nil {
		panic("app.Services.Sprint is nil — composition must inject a SprintService")
	}
	return current.Sprint
}

// Sprint lifecycle accessors.

// SprintList replaces sprint.ListFromDB.
func SprintList(status string) (any, error) { return Sprint().List(status) }

// SprintGet replaces sprint.GetFromDB.
func SprintGet(sprintID string) (any, error) { return Sprint().Get(sprintID) }

// SprintCreate replaces sprint.Create.
func SprintCreate(id, title, goal string, tasks []any) (any, error) {
	return Sprint().Create(id, title, goal, tasks)
}

// SprintStart replaces sprint.Start.
func SprintStart(sprintID string) (any, error) { return Sprint().Start(sprintID) }

// SprintComplete replaces sprint.Complete.
func SprintComplete(sprintID string) (any, error) { return Sprint().Complete(sprintID) }

// SprintDiscard replaces sprint.Discard.
// When dbOnly=true the folder is left in place and only the DB status
// is updated (--db-only flag).
func SprintDiscard(sprintID, reason string, returnTasks, dbOnly bool) (any, error) {
	return Sprint().Discard(sprintID, reason, returnTasks, dbOnly)
}

// SprintUpdateFields replaces sprint.UpdateFields.
func SprintUpdateFields(id string, title, goal, status, folder *string) error {
	return Sprint().UpdateFields(id, title, goal, status, folder)
}

// SprintAggregateProgress replaces sprint.AggregateProgress.
func SprintAggregateProgress(sprintID string) (any, error) {
	return Sprint().AggregateProgress(sprintID)
}

// SprintCheckCreateGuard replaces sprint.CheckCreateGuard.
func SprintCheckCreateGuard(id, title string) error {
	return Sprint().CheckCreateGuard(id, title)
}

// SprintAppendAutoVerifyTask replaces sprint.AppendAutoVerifyTask.
func SprintAppendAutoVerifyTask(sprintID string) (any, error) {
	return Sprint().AppendAutoVerifyTask(sprintID)
}

// SprintFindOnDisk replaces sprint.FindSprintOnDisk.
func SprintFindOnDisk(sprintID string) (any, error) { return Sprint().FindSprintOnDisk(sprintID) }

// SprintReconcile replaces sprint.Reconcile.
func SprintReconcile(sprintID string, dryRun bool) (any, error) {
	return Sprint().Reconcile(sprintID, dryRun)
}

// SprintUpdateMDField forwards to UpdateSprintMDField.
func SprintUpdateMDField(path, field, value string) error {
	return Sprint().UpdateMDField(path, field, value)
}

// Task returns the injected TaskService.
func Task() TaskService {
	ensureInitialized()
	if current.Task == nil {
		panic("app.Services.Task is nil — composition must inject a TaskService")
	}
	return current.Task
}

// Task lifecycle accessors.

func TaskCreate(title, taskType, sprint, priority, estimate, summary string, dependsOn []string) (any, error) {
	return Task().Create(title, taskType, sprint, priority, estimate, summary, dependsOn)
}
func TaskGet(taskID string) (any, error)                 { return Task().Get(taskID) }
func TaskList(sprintPtr, statusPtr *string) (any, error) { return Task().List(sprintPtr, statusPtr) }
func TaskListFiltered(filter any) (any, error)           { return Task().ListFiltered(filter) }
func TaskNext() (any, error)                             { return Task().Next() }
func TaskStart(taskID string) (any, error)               { return Task().Start(taskID) }
func TaskComplete(taskID string, skipHarness bool) (any, error) {
	return Task().Complete(taskID, skipHarness)
}
func TaskReopen(taskID, reason, newStatus string) (any, error) {
	return Task().Reopen(taskID, reason, newStatus)
}
func TaskUpdate(req any) (any, error) { return Task().Update(req) }
func TaskDeleteWithOptions(taskID, reason string, opts any) (any, error) {
	return Task().DeleteWithOptions(taskID, reason, opts)
}
func TaskAssignSprint(taskIDs []string, sprintID string) (any, error) {
	return Task().AssignSprint(taskIDs, sprintID)
}
func TaskUnassignSprint(taskIDs []string) (any, error)  { return Task().UnassignSprint(taskIDs) }
func TaskCheckpoint(taskID, reason string) (any, error) { return Task().Checkpoint(taskID, reason) }

// Task helpers.

func TaskResolvePreset(name string) any { return Task().ResolvePreset(name) }
func TaskHeuristicCheck(summary string, cfg any) ([]string, bool) {
	return Task().HeuristicCheck(summary, cfg)
}
func TaskHasPlaceholderBody(filePath string) bool { return Task().HasPlaceholderBody(filePath) }
func TaskGetGitChangedFiles() []string            { return Task().GetGitChangedFiles() }

// Backlog returns the injected BacklogService.
func Backlog() BacklogService {
	ensureInitialized()
	if current.Backlog == nil {
		panic("app.Services.Backlog is nil — composition must inject a BacklogService")
	}
	return current.Backlog
}

// BacklogSync replaces backlog.Sync via the port.
func BacklogSync(dryRun bool) (any, error) { return Backlog().Sync(dryRun) }

// BacklogSyncWithOptions adds option granularity over BacklogSync.
func BacklogSyncWithOptions(dryRun, allowStatusRegression bool) (any, error) {
	return Backlog().SyncWithOptions(dryRun, allowStatusRegression)
}

// BacklogRebuildMD replaces backlog.RebuildMD via the port.
// opts is the concrete pkg/backlog.RebuildMDOptions value (constructed in cmd).
func BacklogRebuildMD(opts any) (any, error) { return Backlog().RebuildMD(opts) }

// GateLog returns the injected GateLogService.
func GateLog() GateLogService {
	ensureInitialized()
	if current.GateLog == nil {
		panic("app.Services.GateLog is nil — composition must inject a GateLogService")
	}
	return current.GateLog
}

// GateLog accessors.

// GateRecord replaces gatejudgement.RecordJudgement via the port.
// Sampled write — includes the disabled check + per-verdict sampling.
func GateRecord(r ports.JudgementRecord) { GateLog().Record(r) }

// GateRecordDirect replaces GlobalStore().Record via the port.
// Bypasses sampling for full record (admin inject/replay use).
func GateRecordDirect(r ports.JudgementRecord) error { return GateLog().RecordDirect(r) }

// GateQuery replaces GlobalStore().Query via the port.
func GateQuery(filter ports.JudgementQuery) ([]ports.JudgementRecord, error) {
	return GateLog().Query(filter)
}

// GateStats replaces GlobalStore().Stats via the port.
func GateStats(filter ports.JudgementQuery) (*ports.JudgementStats, error) {
	return GateLog().Stats(filter)
}

// GateMarkFalsePositive replaces GlobalStore().MarkFalsePositive via the port.
func GateMarkFalsePositive(recordID string, label ports.FalsePositiveLabel) error {
	return GateLog().MarkFalsePositive(recordID, label)
}

// Ceremony returns the injected CeremonyService.
func Ceremony() CeremonyService {
	ensureInitialized()
	if current.Ceremony == nil {
		panic("app.Services.Ceremony is nil — composition must inject a CeremonyService")
	}
	return current.Ceremony
}

// Ceremony accessors.

// CeremonyCollectSprintStart replaces ceremony.CollectSprintStart via the port.
// Returns *ceremony.SprintStartCeremony (re-asserted in cmd).
func CeremonyCollectSprintStart(sprintID string) any {
	return Ceremony().CollectSprintStart(sprintID)
}

// CeremonyCollectSprintComplete replaces ceremony.CollectSprintComplete via the port.
// Returns *ceremony.SprintCompleteCeremony.
func CeremonyCollectSprintComplete(sprintID string) any {
	return Ceremony().CollectSprintComplete(sprintID)
}

// CeremonyCollectTaskStart replaces ceremony.CollectTaskStart via the port.
// Returns *ceremony.TaskStartCeremony.
func CeremonyCollectTaskStart(taskID string) any {
	return Ceremony().CollectTaskStart(taskID)
}

// CeremonyCollectTaskComplete replaces ceremony.CollectTaskComplete via the port.
// Returns *ceremony.TaskCompleteCeremony.
func CeremonyCollectTaskComplete(taskID string) any {
	return Ceremony().CollectTaskComplete(taskID)
}

// Config returns the injected ConfigService.
func Config() ConfigService {
	ensureInitialized()
	if current.Config == nil {
		panic("app.Services.Config is nil — composition must inject a ConfigService")
	}
	return current.Config
}

// Config accessors.

// ConfigDetectProblems replaces config.DetectConfigProblems via the port.
func ConfigDetectProblems(yamlPath string) (any, error) {
	return Config().DetectProblems(yamlPath)
}

// ConfigApplyFixes replaces config.ApplyConfigFixes via the port.
// report is the concrete *config.ProblemReport value.
func ConfigApplyFixes(report any, dryRun bool) (any, error) {
	return Config().ApplyFixes(report, dryRun)
}

// ConfigRenderDiff replaces config.RenderDiff via the port.
func ConfigRenderDiff(original, migrated []byte) string {
	return Config().RenderDiff(original, migrated)
}

// ConfigCollectEffective replaces config.CollectEffective via the port.
// Returns *config.EffectiveConfig.
func ConfigCollectEffective() any { return Config().CollectEffective() }

// ConfigGetProjectKey replaces config.GetProjectKey via the port.
func ConfigGetProjectKey() string { return Config().GetProjectKey() }

// Rules returns the injected RuleRunnerService.
func Rules() RuleRunnerService {
	ensureInitialized()
	if current.Rules == nil {
		panic("app.Services.Rules is nil — composition must inject a RuleRunnerService")
	}
	return current.Rules
}

// Rules accessors.

// RulesList replaces rules.List via the port.
// Returns []rules.Rule (re-asserted in cmd).
func RulesList() any { return Rules().List() }

// RulesGet replaces rules.Get via the port.
// Returns rules.Rule (re-asserted in cmd).
func RulesGet(ruleID string) (any, bool) { return Rules().Get(ruleID) }

// RulesLoadConfig replaces rules.LoadConfig via the port.
// Returns *rules.EngineConfig (re-asserted in cmd).
func RulesLoadConfig(projectRoot string) (any, error) { return Rules().LoadConfig(projectRoot) }

// RulesLoadConfigBytes replaces rules.LoadConfigBytes via the port.
// Returns *rules.EngineConfig (re-asserted in cmd).
func RulesLoadConfigBytes(data []byte) (any, error) { return Rules().LoadConfigBytes(data) }

// RulesResolveEffective replaces rules.ResolveEffective via the port.
// cfg is the concrete *rules.EngineConfig (passed as `any`).
// Returns map[string]*rules.EffectiveRule (re-asserted in cmd).
func RulesResolveEffective(cfg any, projectRoot string) (any, error) {
	return Rules().ResolveEffective(cfg, projectRoot)
}
