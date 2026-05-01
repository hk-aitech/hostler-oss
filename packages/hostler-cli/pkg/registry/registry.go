// Package registry provides the resource registry service.
// Project-finite resources (ports, event ranges, ...) are
// managed as a JSON file.
//
// Two-layer storage:
//
//	~/{brand.ProjectDirName}/data/{project_key}/registry.json  ← runtime SSOT
//	{project_root}/{brand.ProjectDirName}/data/registry.json   ← git snapshot
//
// Path resolution: brand.ProjectDirName plus a ConfigProvider priority chain.
//  1. HSTL_REGISTRY_PATH env (official override)
//  2. registry.path in project-config.yaml
//  3. Default (brand-derived)
package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/brand"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/apperr"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/envalias"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/fileutil"
)

// ---------------------------------------------------------------------------
// Constants.
// ---------------------------------------------------------------------------

const (
	// defaultRangeSizeConst is the default range size for range-typed resources.
	defaultRangeSizeConst = 100
	// defaultRangeStartConst is the default start value when range suggestion fails.
	defaultRangeStartConst = 1
)

// getDefaultRangeSize returns the default range size for the range type.
// Honours the HSTL_REGISTRY_RANGE_SIZE environment variable when set.
func getDefaultRangeSize() int {
	if raw := strings.TrimSpace(envalias.Lookup("REGISTRY_RANGE_SIZE")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			return n
		}
	}
	return defaultRangeSizeConst
}

// getDefaultRangeStart returns the fallback start value for range suggestion.
// Honours the HSTL_REGISTRY_RANGE_START environment variable when set.
func getDefaultRangeStart() int {
	if raw := strings.TrimSpace(envalias.Lookup("REGISTRY_RANGE_START")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n >= 0 {
			return n
		}
	}
	return defaultRangeStartConst
}

// ---------------------------------------------------------------------------
// Global mutex (Python's fcntl.flock → Go sync.Mutex).
// ---------------------------------------------------------------------------

var mu sync.Mutex

// ---------------------------------------------------------------------------
// Type definitions.
// ---------------------------------------------------------------------------

// Registry is the top-level structure of registry JSON.
type Registry struct {
	Schema    string              `json:"$schema"`
	Project   string              `json:"project"`
	UpdatedAt string              `json:"updated_at"`
	Resources map[string]Resource `json:"resources"`
}

// Resource defines a single resource.
type Resource struct {
	Type        string       `json:"type"` // "unique" | "range"
	Description string       `json:"description"`
	Allocations []Allocation `json:"allocations"`
}

// Allocation is a single allocation entry.
type Allocation struct {
	Owner string `json:"owner"`
	Value *int   `json:"value,omitempty"` // unique type
	Range []int  `json:"range,omitempty"` // range type [start, end]
	Env   string `json:"env,omitempty"`
	Note  string `json:"note,omitempty"`
}

// ---------------------------------------------------------------------------
// Path helpers.
// ---------------------------------------------------------------------------

// registryConfigReader is a function pointer used to break the circular
// import with pkg/config. The config package registers itself via
// init(). It returns an empty string before registration or in test
// environments.
var registryConfigReader = func(field string) string { return "" }

// RegisterConfigReader allows pkg/config to register itself with the
// registry package.
// field argument: "runtime" | "docs".
func RegisterConfigReader(reader func(field string) string) {
	if reader != nil {
		registryConfigReader = reader
	}
}

// GetRuntimeRegistryPath returns the runtime registry path.
//
// Priority:
//  1. HSTL_REGISTRY_PATH environment variable (official override).
//  2. registry.path field in project-config.yaml.
//  3. Default: ~/{brand.ProjectDirName}/data/{project_key}/registry.json.
func GetRuntimeRegistryPath() (string, error) {
	// 1. Environment variable wins.
	if override := strings.TrimSpace(envalias.Lookup("REGISTRY_PATH")); override != "" {
		return override, nil
	}
	// 2. YAML override.
	if yamlPath := registryConfigReader("runtime"); yamlPath != "" {
		return yamlPath, nil
	}
	// 3. Default path (brand.ProjectDirName-derived).
	projectKey := strings.TrimSpace(envalias.Lookup("PROJECT"))
	if projectKey == "" {
		return "", &apperr.InvalidStateError{
			EntityType:   "Registry",
			Message:      "HSTL_PROJECT environment variable is not set",
			RecoveryHint: "Set the HSTL_PROJECT environment variable to the project key.",
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", &apperr.InvalidStateError{
			EntityType:   "Registry",
			Message:      fmt.Sprintf("failed to detect home directory: %v", err),
			RecoveryHint: "Verify that the home directory is configured correctly.",
		}
	}
	return filepath.Join(home, brand.ProjectDirName, "data", projectKey, "registry.json"), nil
}

// GetDocsRegistryPath returns the git-managed snapshot path.
//
// Priority:
//  1. HSTL_REGISTRY_DOCS_PATH environment variable (official override).
//  2. registry.docs_path field in project-config.yaml.
//  3. Default: {project_root}/{brand.ProjectDirName}/data/registry.json.
func GetDocsRegistryPath(projectRoot string) string {
	if override := strings.TrimSpace(envalias.Lookup("REGISTRY_DOCS_PATH")); override != "" {
		return override
	}
	if yamlPath := registryConfigReader("docs"); yamlPath != "" {
		return yamlPath
	}
	return filepath.Join(projectRoot, brand.ProjectDirName, "data", "registry.json")
}

// ---------------------------------------------------------------------------
// Initial structure.
// ---------------------------------------------------------------------------

// emptyRegistry returns the initial structure for an empty registry.
func emptyRegistry() Registry {
	projectKey := strings.TrimSpace(envalias.Lookup("PROJECT"))
	return Registry{
		Schema:    brand.RegistrySchemaVersion,
		Project:   projectKey,
		UpdatedAt: nowUTC(),
		Resources: make(map[string]Resource),
	}
}

// ---------------------------------------------------------------------------
// Read / write.
// ---------------------------------------------------------------------------

// LoadRegistry reads the runtime registry JSON.
// If the file is missing, it tries to bootstrap from docs/registry.json
// before falling back to an empty structure.
func LoadRegistry() (Registry, error) {
	runtimePath, err := GetRuntimeRegistryPath()
	if err != nil {
		return Registry{}, err
	}

	if data, err := os.ReadFile(runtimePath); err == nil {
		var reg Registry
		if jsonErr := json.Unmarshal(data, &reg); jsonErr == nil {
			return reg, nil
		}
	}

	// Bootstrap: docs/registry.json → runtime.
	projectRoot := fileutil.GetProjectRoot()
	docsPath := GetDocsRegistryPath(projectRoot)
	if data, err := os.ReadFile(docsPath); err == nil {
		var reg Registry
		if jsonErr := json.Unmarshal(data, &reg); jsonErr == nil {
			// Copy into the runtime path.
			if mkErr := os.MkdirAll(filepath.Dir(runtimePath), 0o755); mkErr == nil {
				_ = os.WriteFile(runtimePath, data, 0o644)
			}
			return reg, nil
		}
	}

	return emptyRegistry(), nil
}

// SaveRegistry writes the runtime registry JSON.
// Refreshes the updated_at timestamp to the current time.
// Call this directly only when the caller does not already hold mu.
// Allocate/Check internals must use saveRegistryLocked instead.
func SaveRegistry(reg Registry) error {
	mu.Lock()
	defer mu.Unlock()
	return saveRegistryLocked(reg)
}

// saveRegistryLocked writes the registry while mu is already held.
func saveRegistryLocked(reg Registry) error {
	runtimePath, err := GetRuntimeRegistryPath()
	if err != nil {
		return fmt.Errorf("saveRegistryLocked: failed to resolve registry path: %w", err)
	}

	reg.UpdatedAt = nowUTC()

	data, err := json.MarshalIndent(reg, "", "  ")
	if err != nil {
		return &apperr.InvalidStateError{
			EntityType:   "Registry",
			Message:      fmt.Sprintf("failed to serialise registry JSON: %v", err),
			RecoveryHint: "Inspect the registry data structure.",
		}
	}

	if err := os.MkdirAll(filepath.Dir(runtimePath), 0o755); err != nil {
		return &apperr.InvalidStateError{
			EntityType:   "Registry",
			Message:      fmt.Sprintf("failed to create registry directory: %v", err),
			RecoveryHint: "Check write permissions on the ~/.hstl-oss/data/ directory.",
		}
	}

	if err := os.WriteFile(runtimePath, data, 0o644); err != nil {
		return &apperr.InvalidStateError{
			EntityType:   "Registry",
			Message:      fmt.Sprintf("failed to save registry: %v", err),
			RecoveryHint: "Check write permissions on the registry file path.",
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Conflict checks.
// ---------------------------------------------------------------------------

// FindConflictsUnique looks for value collisions in unique-typed resources.
// When env is set, only allocations within the same env are inspected.
func FindConflictsUnique(resource Resource, value int, env string) []Allocation {
	var conflicts []Allocation
	for _, alloc := range resource.Allocations {
		if alloc.Value == nil || *alloc.Value != value {
			continue
		}
		// env filter: matched only when both sides have a non-empty env.
		if env != "" && alloc.Env != "" && alloc.Env != env {
			continue
		}
		conflicts = append(conflicts, alloc)
	}
	return conflicts
}

// FindConflictsRange looks for overlaps in range-typed resources.
// A conflict is reported when [start, end] ∩ [alloc_start, alloc_end] ≠ ∅.
func FindConflictsRange(resource Resource, start, end int) []Allocation {
	var conflicts []Allocation
	for _, alloc := range resource.Allocations {
		if len(alloc.Range) < 2 {
			continue
		}
		allocStart, allocEnd := alloc.Range[0], alloc.Range[1]
		// Overlap condition: NOT (end < allocStart OR start > allocEnd).
		if !(end < allocStart || start > allocEnd) {
			conflicts = append(conflicts, alloc)
		}
	}
	return conflicts
}

// ---------------------------------------------------------------------------
// Available-value suggestions.
// ---------------------------------------------------------------------------

// SuggestNextUnique suggests the next available integer for a
// unique-typed resource. Returns max(allocated) + 1, or nil if there
// are no allocations.
func SuggestNextUnique(resource Resource, env string) *int {
	var values []int
	for _, alloc := range resource.Allocations {
		v := alloc.Value
		if v == nil {
			continue
		}
		if env == "" {
			values = append(values, *v)
		} else {
			if alloc.Env == "" || alloc.Env == env {
				values = append(values, *v)
			}
		}
	}
	if len(values) == 0 {
		return nil
	}
	maxV := values[0]
	for _, v := range values[1:] {
		if v > maxV {
			maxV = v
		}
	}
	result := maxV + 1
	return &result
}

// SuggestNextRange suggests a non-overlapping range for a range-typed
// resource. Returns [maxEnd+1, maxEnd+size] or nil when no allocations
// exist yet.
func SuggestNextRange(resource Resource, size int) *[2]int {
	var ends []int
	for _, alloc := range resource.Allocations {
		if len(alloc.Range) >= 2 {
			ends = append(ends, alloc.Range[1])
		}
	}
	if len(ends) == 0 {
		return nil
	}
	maxEnd := ends[0]
	for _, e := range ends[1:] {
		if e > maxEnd {
			maxEnd = e
		}
	}
	nextStart := maxEnd + 1
	result := [2]int{nextStart, nextStart + size - 1}
	return &result
}

// ---------------------------------------------------------------------------
// Allocate / Check / List.
// ---------------------------------------------------------------------------

// Allocate adds a new allocation to a resource.
//
// resourceType: "unique" | "range".
// name: resource name (key in resources).
// owner: allocation owner.
// value: unique-typed value (auto-suggested if nil).
// rangeStart, rangeEnd: range-typed bounds (auto-suggested if nil).
// env, note: optional metadata.
func Allocate(resourceType, name, owner string, value *int, rangeStart, rangeEnd *int, env, note string) (any, error) {
	mu.Lock()
	defer mu.Unlock()

	reg, err := LoadRegistry()
	if err != nil {
		return nil, fmt.Errorf("Allocate: failed to load registry: %w", err)
	}

	resource, exists := reg.Resources[name]
	if !exists {
		resource = Resource{
			Type:        resourceType,
			Description: name,
			Allocations: []Allocation{},
		}
	}

	alloc := Allocation{
		Owner: owner,
		Env:   env,
		Note:  note,
	}

	var conflicts []Allocation

	switch resourceType {
	case "unique":
		// Auto-suggest a value if absent.
		if value == nil {
			value = SuggestNextUnique(resource, env)
			if value == nil {
				v := 1
				value = &v
			}
		}
		conflicts = FindConflictsUnique(resource, *value, env)
		alloc.Value = value

	case "range":
		if rangeStart == nil || rangeEnd == nil {
			suggested := SuggestNextRange(resource, getDefaultRangeSize())
			if suggested != nil {
				rs, re := suggested[0], suggested[1]
				rangeStart = &rs
				rangeEnd = &re
			} else {
				rs, re := getDefaultRangeStart(), getDefaultRangeStart()+getDefaultRangeSize()-1
				rangeStart = &rs
				rangeEnd = &re
			}
		}
		conflicts = FindConflictsRange(resource, *rangeStart, *rangeEnd)
		alloc.Range = []int{*rangeStart, *rangeEnd}

	default:
		return nil, &apperr.RejectedError{
			Reason:       fmt.Sprintf("unsupported resourceType: %q", resourceType),
			RecoveryHint: "Only the unique and range types are supported.",
		}
	}

	if len(conflicts) > 0 {
		return AllocateConflictResult{
			Status:    "CONFLICT",
			Conflicts: conflicts,
			Message:   fmt.Sprintf("conflicts detected: %d", len(conflicts)),
		}, nil
	}

	resource.Allocations = append(resource.Allocations, alloc)
	if reg.Resources == nil {
		reg.Resources = make(map[string]Resource)
	}
	reg.Resources[name] = resource

	if err := saveRegistryLocked(reg); err != nil {
		return AllocateResult{}, err
	}

	return AllocateResult{
		Status:    "ok",
		Resource:  name,
		Owner:     owner,
		Allocated: alloc,
	}, nil
}

// Check inspects whether a candidate allocation would conflict.
func Check(resourceType, name string, value *int, rangeStart, rangeEnd *int, env string) (CheckResult, error) {
	mu.Lock()
	defer mu.Unlock()

	reg, err := LoadRegistry()
	if err != nil {
		return CheckResult{}, err
	}

	resource, exists := reg.Resources[name]
	if !exists {
		return CheckResult{
			Status:    "ok",
			Conflicts: []Allocation{},
			Message:   "resource missing — no conflict",
		}, nil
	}

	var conflicts []Allocation
	switch resourceType {
	case "unique":
		if value == nil {
			return CheckResult{
				Status:     "ok",
				Conflicts:  []Allocation{},
				Suggestion: SuggestNextUnique(resource, env),
			}, nil
		}
		conflicts = FindConflictsUnique(resource, *value, env)

	case "range":
		if rangeStart == nil || rangeEnd == nil {
			return CheckResult{
				Status:     "ok",
				Conflicts:  []Allocation{},
				Suggestion: SuggestNextRange(resource, getDefaultRangeSize()),
			}, nil
		}
		conflicts = FindConflictsRange(resource, *rangeStart, *rangeEnd)

	default:
		return CheckResult{}, &apperr.RejectedError{
			Reason:       fmt.Sprintf("unsupported resourceType: %q", resourceType),
			RecoveryHint: "Only the unique and range types are supported.",
		}
	}

	if len(conflicts) > 0 {
		return CheckResult{
			Status:    "CONFLICT",
			Conflicts: conflicts,
		}, nil
	}
	return CheckResult{
		Status:    "ok",
		Conflicts: []Allocation{},
	}, nil
}

// ListResources returns the registry's resource list.
// An empty resourceType returns every resource.
func ListResources(resourceType string) (ListResult, error) {
	reg, err := LoadRegistry()
	if err != nil {
		return ListResult{}, err
	}

	var resources []ResourceSummary
	for name, resource := range reg.Resources {
		if resourceType != "" && resource.Type != resourceType {
			continue
		}
		resources = append(resources, ResourceSummary{
			Name:            name,
			Type:            resource.Type,
			Description:     resource.Description,
			AllocationCount: len(resource.Allocations),
			Allocations:     resource.Allocations,
		})
	}

	if resources == nil {
		resources = []ResourceSummary{}
	}

	return ListResult{
		Resources: resources,
		Project:   reg.Project,
		UpdatedAt: reg.UpdatedAt,
	}, nil
}

// SyncToProject copies the runtime registry to docs/registry.json.
func SyncToProject(projectRoot string) (SyncResult, error) {
	runtimePath, err := GetRuntimeRegistryPath()
	if err != nil {
		return SyncResult{}, err
	}
	docsPath := GetDocsRegistryPath(projectRoot)

	if _, err := os.Stat(runtimePath); os.IsNotExist(err) {
		return SyncResult{
			Synced:    false,
			DiffCount: 0,
			DocsPath:  docsPath,
		}, nil
	}

	runtimeData, err := os.ReadFile(runtimePath)
	if err != nil {
		return SyncResult{
			Synced:    false,
			DiffCount: 0,
			Error:     err.Error(),
		}, nil
	}

	var runtimeReg Registry
	if err := json.Unmarshal(runtimeData, &runtimeReg); err != nil {
		return SyncResult{
			Synced:    false,
			DiffCount: 0,
			Error:     err.Error(),
		}, nil
	}

	// Compute diff (allocation-count basis).
	diffCount := 0
	if docsData, err := os.ReadFile(docsPath); err == nil {
		var docsReg Registry
		if jsonErr := json.Unmarshal(docsData, &docsReg); jsonErr == nil {
			runtimeAllocs := countAllocations(runtimeReg)
			docsAllocs := countAllocations(docsReg)
			if runtimeAllocs > docsAllocs {
				diffCount = runtimeAllocs - docsAllocs
			} else {
				diffCount = docsAllocs - runtimeAllocs
			}
		} else {
			diffCount = -1
		}
	}

	if err := os.MkdirAll(filepath.Dir(docsPath), 0o755); err != nil {
		return SyncResult{}, &apperr.InvalidStateError{
			EntityType:   "Registry",
			Message:      fmt.Sprintf("failed to create docs directory: %v", err),
			RecoveryHint: "Check write permissions on .hstl-oss/data/ inside the project root.",
		}
	}
	if err := os.WriteFile(docsPath, runtimeData, 0o644); err != nil {
		return SyncResult{}, &apperr.InvalidStateError{
			EntityType:   "Registry",
			Message:      fmt.Sprintf("failed to write docs/registry.json: %v", err),
			RecoveryHint: "Check write permissions on the docs/registry.json file path.",
		}
	}

	return SyncResult{
		Synced:    true,
		DiffCount: diffCount,
		DocsPath:  docsPath,
	}, nil
}

// ---------------------------------------------------------------------------
// Internal utilities.
// ---------------------------------------------------------------------------

// nowUTC returns an ISO 8601 timestamp in UTC.
func nowUTC() string {
	return time.Now().UTC().Format("2006-01-02T15:04:05Z")
}

// countAllocations returns the total number of allocations in the registry.
func countAllocations(reg Registry) int {
	total := 0
	for _, r := range reg.Resources {
		total += len(r.Allocations)
	}
	return total
}
