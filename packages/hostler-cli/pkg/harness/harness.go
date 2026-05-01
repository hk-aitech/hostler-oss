// Package harness owns the pre-Task/Sprint-completion checklist
// enforcement logic. It looks up incomplete mandatory items in the
// harness_items table and dynamically evaluates conditional items.
// trac: HAR-CM021,HAR-CM022,HAR-CM023,HAR-CM024,HAR-EV001,HAR-EV002
// trac: HAR-EV003,HAR-EV004,HAR-EV005,HAR-EV006,HAR-EV007,HAR-EV008,HAR-EV009
// trac: HAR-EV010,HAR-EV011,HAR-EV012,HAR-FT001,HAR-FT002,HAR-FT003,HAR-FT004
// trac: HAR-FT005,HAR-FT006,HAR-FT007,HAR-FT008,HAR-FT009,HAR-FT010,HAR-FT011
// trac: HAR-FT012,HAR-FT013,HAR-FT014,HAR-FT015,HAR-QR002,HAR-QR003,HAR-QR004
// trac: HAR-WF001,HAR-WF002,HAR-WF003,HAR-WF004
package harness

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/domain"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/ports"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/apperr"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/audit"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/envalias"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/fileutil"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/gatejudgement"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/store"
)

// --------------------------------------------------------------------------
// shared types
// --------------------------------------------------------------------------

// --------------------------------------------------------------------------
// ResolveTemplateKey
// --------------------------------------------------------------------------

// ResolveTemplateKey returns the template_key for the given
// entity_type/entity_id.
// task -> DB tasks.type lookup -> "task:{type}";
// sprint -> "sprint:default".
func ResolveTemplateKey(entityType, entityID string) (string, error) {
	if entityType == "sprint" {
		return "sprint:default", nil
	}
	if entityType == "expedition" {
		return "expedition:default", nil
	}

	// task: look up the type via the store.
	gs := store.Get()
	if gs == nil {
		return "task:feature", fmt.Errorf("DB is not initialised")
	}

	taskType, err := gs.GetTaskType(entityID)
	if err != nil {
		// On lookup failure, default to feature.
		return "task:feature", nil
	}
	return "task:" + taskType, nil
}

// --------------------------------------------------------------------------
// EnsureHarnessItems
// --------------------------------------------------------------------------

// EnsureHarnessItems initialises harness_items from the template when
// rows are missing.
func EnsureHarnessItems(entityType, entityID string) ([]domain.HarnessItem, error) {
	gs := store.Get()
	if gs == nil {
		return nil, fmt.Errorf("DB is not initialised")
	}

	// Even when existing records are present, the template may have
	// added new items; EnsureHarnessItems(store) attempts a top-up.
	// Existing records are protected by the UNIQUE constraint and
	// remain untouched.

	// Initialise from the template.
	templateKey, err := ResolveTemplateKey(entityType, entityID)
	if err != nil && templateKey == "" {
		templateKey = "task:feature"
	}
	templateItems, err := loadTemplateItems(templateKey)
	if err != nil || len(templateItems) == 0 {
		return []domain.HarnessItem{}, nil
	}

	var harnessTemplates []ports.HarnessItemTemplate
	for _, item := range templateItems {
		itemID, _ := item["id"].(string)
		itemName, _ := item["name"].(string)
		conditional, _ := item["conditional"].(string)

		var isRequired bool
		if conditional != "" {
			isRequired = evaluateConditional(conditional, entityType, entityID)
		} else {
			req, ok := item["required"].(bool)
			if ok {
				isRequired = req
			} else {
				isRequired = true
			}
		}

		harnessTemplates = append(harnessTemplates, ports.HarnessItemTemplate{
			ID:          itemID,
			Description: itemName,
			Required:    isRequired,
		})
	}

	if err := gs.EnsureHarnessItems(entityType, entityID, harnessTemplates); err != nil {
		return nil, fmt.Errorf("harness_items initialisation failed: %w", err)
	}

	return fetchHarnessItems(entityType, entityID)
}

// fetchHarnessItems looks up the harness items for a particular
// entity via the store.
func fetchHarnessItems(entityType, entityID string) ([]domain.HarnessItem, error) {
	gs := store.Get()
	if gs == nil {
		return nil, fmt.Errorf("DB is not initialised")
	}

	portItems, err := gs.GetHarnessItems(entityType, entityID)
	if err != nil {
		return nil, fmt.Errorf("harness_items lookup failed: %w", err)
	}

	items := make([]domain.HarnessItem, 0, len(portItems))
	for _, pi := range portItems {
		items = append(items, domain.HarnessItem{
			ID:        pi.ID,
			Name:      pi.Description,
			Required:  pi.Required,
			Done:      pi.Done,
			Evidence:  pi.Evidence,
			CheckedAt: pi.CheckedAt,
			CheckedBy: pi.CheckedBy,
		})
	}
	return items, nil
}

// loadTemplateItems loads the template items from harness_templates
// via the store.
func loadTemplateItems(templateKey string) ([]map[string]any, error) {
	gs := store.Get()
	if gs == nil {
		return nil, fmt.Errorf("DB is not initialised")
	}
	return gs.GetHarnessTemplate(templateKey)
}

// --------------------------------------------------------------------------
// CheckHarness
// --------------------------------------------------------------------------

// CheckHarness returns the incomplete mandatory items.
func CheckHarness(entityType, entityID string) (domain.HarnessResult, error) {
	allItems, err := EnsureHarnessItems(entityType, entityID)
	if err != nil {
		return domain.HarnessResult{}, err
	}

	var uncheckedRequired []domain.HarnessItem
	requiredTotal := 0
	for _, item := range allItems {
		if item.Required {
			requiredTotal++
			if !item.Done {
				uncheckedRequired = append(uncheckedRequired, item)
			}
		}
	}

	if uncheckedRequired == nil {
		uncheckedRequired = []domain.HarnessItem{}
	}

	requiredDone := requiredTotal - len(uncheckedRequired)
	result := domain.HarnessResult{
		UncheckedRequired: uncheckedRequired,
		AllItems:          allItems,
		Blocked:           len(uncheckedRequired) > 0,
		RequiredTotal:     requiredTotal,
		RequiredDone:      requiredDone,
	}

	// Record the gate verdict in GateJudgementLog. Only BLOCK / PASS
	// are emitted; WARN / HARD_BLOCK are decided by the upper caller's
	// policy.
	recordHarnessJudgement(entityType, entityID, result)

	return result, nil
}

// recordHarnessJudgement records the CheckHarness result to the
// gate-log. Errors are silent — observability-path failures must not
// block the actual verification.
func recordHarnessJudgement(entityType, entityID string, result domain.HarnessResult) {
	verdict := ports.VerdictPass
	uncheckedSummary := ""
	if result.Blocked {
		verdict = ports.VerdictBlock
		items := make([]string, 0, len(result.UncheckedRequired))
		for _, it := range result.UncheckedRequired {
			items = append(items, it.ID)
		}
		uncheckedSummary = strings.Join(items, ",")
	}
	gatejudgement.RecordJudgement(ports.JudgementRecord{
		GateType:     "harness",
		ItemID:       entityID,
		Verdict:      verdict,
		RuleID:       entityType + ".harness",
		Expected:     "all_required_items_done",
		Actual:       uncheckedSummary,
		InputSnippet: uncheckedSummary,
		Metadata: map[string]string{
			"entity_type":    entityType,
			"required_total": strconvItoa(result.RequiredTotal),
			"required_done":  strconvItoa(result.RequiredDone),
		},
	})
}

// strconvItoa is a tiny helper used to avoid importing strconv
// directly.
func strconvItoa(n int) string {
	return fmt.Sprintf("%d", n)
}

// --------------------------------------------------------------------------
// evaluateConditional
// --------------------------------------------------------------------------

// evaluateConditional evaluates the conditional field. Returns false
// on failure (safe default).
func evaluateConditional(conditionExpr, entityType, entityID string) bool {
	defer func() { recover() }() //nolint:errcheck

	switch conditionExpr {
	case "has_design_changes":
		return evalHasDesignChanges(entityType, entityID)
	case "is_prod_sprint":
		return evalIsProdSprint(entityType, entityID)
	case "is_go_project":
		// True when go.mod exists at the repo root. Acts as a guard so
		// Go-only items (lint_passed etc.) are not enforced on
		// non-Go projects.
		return evalIsGoProject()
	case "has_go_source_changes":
		// True when git diff contains any .go file (build / tests are
		// then mandatory). When a refactor Task modifies only .md /
		// .yaml, build / tests / lint are skipped automatically.
		return evalHasGoSourceChanges()
	case "is_ci":
		// CI environment detection — true when one of the standard
		// CI env vars is set (CI=true / GITHUB_ACTIONS / GITLAB_CI).
		// Used by CI-only gate items.
		return evalIsCI()
	case "is_localdev":
		// localdev environment = not CI. Used by localdev-only gates.
		return !evalIsCI()
	default:
		return false
	}
}

// evalIsCI detects the standard CI environment variables.
func evalIsCI() bool {
	for _, v := range []string{"CI", "GITHUB_ACTIONS", "GITLAB_CI", "HOSTLER_CI"} {
		if val := os.Getenv(v); val != "" && val != "0" && val != "false" {
			return true
		}
	}
	return false
}

func evalHasDesignChanges(entityType, entityID string) bool {
	if entityType != "sprint" {
		return false
	}
	root := fileutil.GetProjectRoot()
	cmd := exec.Command("git", "log", "--oneline", "--", "docs/03-design/")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) != ""
}

// evalIsGoProject reports whether go.mod exists at the repo root or
// under cli/ · packages/*/. Recognises both the legacy cli/ layout
// and the monorepo's packages/hostler-cli/.
func evalIsGoProject() bool {
	root := fileutil.GetProjectRoot()
	candidates := []string{
		filepath.Join(root, "go.mod"),
		filepath.Join(root, "cli", "go.mod"),
		filepath.Join(root, "packages", "hostler-cli", "go.mod"),
		filepath.Join(root, "packages", "go", "go.mod"),
	}
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}
	return false
}

// evalHasGoSourceChanges reports whether git diff contains any .go
// file. Inspects both staged and unstaged changes so a refactor Task
// that touched only documentation is detected. On git command
// failure, returns true (safe default: apply the full harness).
func evalHasGoSourceChanges() bool {
	root := fileutil.GetProjectRoot()
	goSuffix := ".go"

	// Inspect unstaged changes.
	cmdUnstaged := exec.Command("git", "diff", "--name-only")
	cmdUnstaged.Dir = root
	if hasGoFiles(cmdUnstaged, goSuffix) {
		return true
	}

	// Inspect staged changes.
	cmdStaged := exec.Command("git", "diff", "--cached", "--name-only")
	cmdStaged.Dir = root
	if hasGoFiles(cmdStaged, goSuffix) {
		return true
	}

	// Inspect untracked .go files.
	cmdUntracked := exec.Command("git", "ls-files", "--others", "--exclude-standard")
	cmdUntracked.Dir = root
	return hasGoFiles(cmdUntracked, goSuffix)
}

// hasGoFiles is a helper that checks whether the git command output
// contains any .go file. Returns true on command failure (safe
// default: apply the full harness).
func hasGoFiles(cmd *exec.Cmd, suffix string) bool {
	out, err := cmd.Output()
	if err != nil {
		return true // safe default
	}
	for _, line := range strings.Split(string(out), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && strings.HasSuffix(trimmed, suffix) {
			return true
		}
	}
	return false
}

func evalIsProdSprint(entityType, entityID string) bool {
	if entityType != "sprint" {
		return false
	}
	gs := store.Get()
	if gs == nil {
		return false
	}

	sprint, err := gs.GetSprint(entityID)
	if err != nil || sprint == nil {
		return false
	}

	text := strings.ToLower(sprint.Title + " " + sprint.Goal)
	return strings.Contains(text, "prod") ||
		strings.Contains(text, "production") ||
		strings.Contains(text, "release")
}

// --------------------------------------------------------------------------
// HarnessGet
// --------------------------------------------------------------------------

// HarnessGet returns the current harness state.
func HarnessGet(entityType, entityID string) (domain.HarnessGetResult, error) {
	result, err := CheckHarness(entityType, entityID)
	if err != nil {
		return domain.HarnessGetResult{}, err
	}
	templateKey, _ := ResolveTemplateKey(entityType, entityID)

	items := make([]domain.HarnessItemView, 0, len(result.AllItems))
	for _, item := range result.AllItems {
		items = append(items, domain.HarnessItemView(item))
	}

	return domain.HarnessGetResult{
		EntityType:    entityType,
		EntityID:      entityID,
		TemplateKey:   templateKey,
		Items:         items,
		RequiredTotal: result.RequiredTotal,
		RequiredDone:  result.RequiredDone,
		Blocked:       result.Blocked,
	}, nil
}

// --------------------------------------------------------------------------
// HarnessCheck
// --------------------------------------------------------------------------

// ContextAckTTLEnv is the env var name that holds the context-ack
// TTL in minutes. When the value is <= 0 or unset, no expiry is
// applied (preserves the prior behaviour).
const ContextAckTTLEnv = "HSTL_CONTEXT_ACK_TTL_MIN"

// readContextAckTTL reads the TTL minutes from the env var.
// Returns 0 (no expiry) on parse failure, negative values, or unset.
func readContextAckTTL() int {
	v := strings.TrimSpace(envalias.Lookup("CONTEXT_ACK_TTL_MIN"))
	if v == "" {
		return 0
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return 0
	}
	return n
}

// verifyContextAckEvidence verifies the evidence for a
// context_acknowledged item. The evidence is "sha256:<hex>" or the
// raw hex; the task's work_ticket + hash pair must exist in the
// context_acknowledgments table.
// When HSTL_CONTEXT_ACK_TTL_MIN is positive, the ack creation time
// must be within TTL minutes of now. Returns a stale message on
// expiry.
func verifyContextAckEvidence(gs ports.GraphStore, taskID, evidence string) error {
	if evidence == "" {
		return fmt.Errorf("context_acknowledged: evidence missing. Call 'hstl context --ticket <work_ticket>' and pass the output hash as evidence")
	}
	hash := strings.TrimPrefix(evidence, "sha256:")
	hash = strings.TrimSpace(hash)

	ticket, err := gs.GetTaskWorkTicket(taskID)
	if err != nil {
		return fmt.Errorf("work_ticket lookup failed: %w", err)
	}
	if ticket == "" {
		return fmt.Errorf("Task %s has no work_ticket. Run 'hstl task start %s' first", taskID, taskID)
	}

	ttl := readContextAckTTL()
	ok, err := gs.HasContextAckFresh(ticket, hash, ttl)
	if err != nil {
		return fmt.Errorf("ack lookup failed: %w", err)
	}
	if !ok {
		// Distinguish stale vs not-found: when the ack exists but
		// TTL has expired, emit a different message.
		if ttl > 0 {
			existsAny, err2 := gs.HasContextAck(ticket, hash)
			if err2 == nil && existsAny {
				return fmt.Errorf("context ack stale: the ack for ticket=%s / hash=%s exceeded the TTL of %d minutes. Re-run 'hstl context --ticket %s' to obtain a fresh hash", ticket, hash, ttl, ticket)
			}
		}
		return fmt.Errorf("context ack mismatch: no record for ticket=%s with hash=%s. Run 'hstl context --ticket %s' and submit the printed hash as evidence again", ticket, hash, ticket)
	}
	return nil
}

// HarnessCheck marks a specific item as complete. Return type is
// any: domain.HarnessCheckResult on success, HarnessCheckErrorResult
// when the item is not found.
func HarnessCheck(entityType, entityID, itemID, evidence, actor string) (any, error) {
	if actor == "" {
		actor = "claude"
	}

	// Auto-initialise items when missing.
	if _, err := EnsureHarnessItems(entityType, entityID); err != nil {
		return nil, err
	}

	gs := store.Get()
	if gs == nil {
		return nil, fmt.Errorf("DB is not initialised")
	}

	// Confirm the item exists.
	_, err := gs.GetHarnessItemID(entityType, entityID, itemID)
	if err != nil {
		// Include the available item_id list in recovery_hint to
		// streamline the recovery UX.
		available := listAvailableItemIDs(entityType, entityID)
		hint := "use harness_get to inspect the list of available items."
		if len(available) > 0 {
			templateKey, _ := ResolveTemplateKey(entityType, entityID)
			hint = fmt.Sprintf(
				"valid item_ids for %s '%s' are one of: %s (template=%s). "+
					"recommended preceding call: harness_get(entity_type='%s', entity_id='%s')",
				entityType, entityID, strings.Join(available, ", "), templateKey,
				entityType, entityID,
			)
		}
		return nil, &apperr.NotFoundError{
			EntityType:   entityType,
			EntityID:     entityID,
			Message:      fmt.Sprintf("item not found: %s", itemID),
			RecoveryHint: hint,
		}
	}

	// context_acknowledged: only allow the check when the submitted
	// evidence hash matches an ack recorded for the task's
	// work_ticket in the DB.
	if itemID == "context_acknowledged" && entityType == "task" {
		if err := verifyContextAckEvidence(gs, entityID, evidence); err != nil {
			return nil, err
		}
	}

	now := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	if err = gs.CheckHarnessItem(entityType, entityID, itemID, evidence, actor); err != nil {
		return nil, fmt.Errorf("harness_items update failed: %w", err)
	}

	// Audit log.
	_ = audit.LogEvent("harness.checked", entityType, entityID, actor, map[string]any{
		"item_id":    itemID,
		"evidence":   evidence,
		"checked_at": now,
	}, "")

	// CEREMONY.md checkbox update logic removed — file deprecated.
	// The harness_items DB has already been updated via
	// gs.CheckHarnessItem (SSOT).

	// Look up the remaining incomplete mandatory items.
	result, err := CheckHarness(entityType, entityID)
	if err != nil {
		return nil, err
	}

	remaining := make([]domain.HarnessItemView, 0, len(result.UncheckedRequired))
	for _, item := range result.UncheckedRequired {
		remaining = append(remaining, domain.HarnessItemView{
			ID:       item.ID,
			Name:     item.Name,
			Required: item.Required,
		})
	}

	ret := domain.HarnessCheckResult{
		Checked:    true,
		ItemID:     itemID,
		EntityType: entityType,
		EntityID:   entityID,
		Remaining:  remaining,
		Blocked:    result.Blocked,
	}

	if len(remaining) > 0 {
		nextItemID := remaining[0].ID
		ret.SuggestedNextAction = fmt.Sprintf(
			"remaining %d items → harness_check(entity_type='%s', entity_id='%s', item_id='%s')",
			len(remaining), entityType, entityID, nextItemID,
		)
	} else {
		completeTool := "task_complete"
		if entityType == "sprint" {
			completeTool = "sprint_complete"
		}
		ret.SuggestedNextAction = fmt.Sprintf("all checks passed -> ready to call %s('%s')", completeTool, entityID)
	}

	return ret, nil
}

// listAvailableItemIDs returns the list of available item_ids for a
// given entity. When harness_items is empty, initialises it from the
// template first and then queries. Helper used to populate accurate
// item_id lists in harness_check error recovery_hints.
func listAvailableItemIDs(entityType, entityID string) []string {
	items, err := EnsureHarnessItems(entityType, entityID)
	if err != nil || len(items) == 0 {
		return nil
	}
	ids := make([]string, 0, len(items))
	for _, it := range items {
		ids = append(ids, it.ID)
	}
	return ids
}

// --------------------------------------------------------------------------
// HarnessTemplateGet
// --------------------------------------------------------------------------

// RequiresGitDiff reports whether the Task type requires the
// result-section file-path git diff verification. Returns true when
// any of {build_passed, tests_passed, code_review} is present in the
// harness_defaults.json template items.
// JSON-SSOT-based helper introduced after the legacy harness/rules.go
// was removed.
func RequiresGitDiff(taskType string) bool {
	items, err := loadTemplateItems("task:" + taskType)
	if err != nil || len(items) == 0 {
		return false
	}
	for _, item := range items {
		if id, _ := item["id"].(string); id != "" {
			switch id {
			case "build_passed", "tests_passed", "code_review":
				return true
			}
		}
	}
	return false
}

// HarnessTemplateGet returns the template for the given template_key.
func HarnessTemplateGet(templateKey string) ([]map[string]any, error) {
	items, err := loadTemplateItems(templateKey)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []map[string]any{}
	}
	return items, nil
}

// --------------------------------------------------------------------------
// AutoCheckCriteria
// --------------------------------------------------------------------------

// AutoCheckCriteria parses the checkboxes in the Task file's
// "## Done Criteria" section.
func AutoCheckCriteria(taskID string) (domain.CriteriaResult, error) {
	gs := store.Get()
	if gs == nil {
		return defaultCriteriaResult(), nil
	}

	filePath, err := gs.GetTaskFilePath(taskID)
	if err != nil {
		return defaultCriteriaResult(), nil
	}

	root := fileutil.GetProjectRoot()
	absPath := filepath.Join(root, filePath)
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return defaultCriteriaResult(), nil
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return defaultCriteriaResult(), nil
	}

	return parseCompletionCriteria(string(data)), nil
}

func defaultCriteriaResult() domain.CriteriaResult {
	return domain.CriteriaResult{
		Total:      0,
		Checked:    0,
		AllChecked: true,
		Unchecked:  []string{},
	}
}

// parseCompletionCriteria parses the checkboxes in the markdown
// content's "## Done Criteria" section.
func parseCompletionCriteria(content string) domain.CriteriaResult {
	lines := strings.Split(content, "\n")

	var criteriaLines []string
	inCriteria := false
	reCriteriaHeader := regexp.MustCompile(`(?i)^##\s+Done\s*Criteria`)
	reH2 := regexp.MustCompile(`^##\s+`)

	for _, line := range lines {
		if reCriteriaHeader.MatchString(line) {
			inCriteria = true
			continue
		}
		if inCriteria && reH2.MatchString(line) {
			break
		}
		if inCriteria {
			criteriaLines = append(criteriaLines, line)
		}
	}

	criteriaSection := strings.Join(criteriaLines, "\n")

	reChecked := regexp.MustCompile(`(?i)^\s*-\s+\[x\]\s+(.+)$`)
	reUnchecked := regexp.MustCompile(`^\s*-\s+\[\s+\]\s+(.+)$`)

	var checkedItems, uncheckedItems []string
	sectionToParse := criteriaSection
	if strings.TrimSpace(sectionToParse) == "" {
		sectionToParse = content
	}

	for _, line := range strings.Split(sectionToParse, "\n") {
		if m := reChecked.FindStringSubmatch(line); m != nil {
			checkedItems = append(checkedItems, strings.TrimSpace(m[1]))
		} else if m := reUnchecked.FindStringSubmatch(line); m != nil {
			uncheckedItems = append(uncheckedItems, strings.TrimSpace(m[1]))
		}
	}

	total := len(checkedItems) + len(uncheckedItems)
	checked := len(checkedItems)

	if uncheckedItems == nil {
		uncheckedItems = []string{}
	}

	return domain.CriteriaResult{
		Total:      total,
		Checked:    checked,
		AllChecked: total == 0 || checked == total,
		Unchecked:  uncheckedItems,
	}
}
