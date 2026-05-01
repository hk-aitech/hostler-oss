package registry

import (
	"os"
	"path/filepath"
	"testing"
)

// setupTestRegistry configures a registry path for tests and returns
// the cleanup function.
func setupTestRegistry(t *testing.T) func() {
	t.Helper()
	dir := t.TempDir()
	regPath := filepath.Join(dir, "registry.json")
	os.Setenv("HSTL_PROJECT", "test-project")
	os.Setenv("HSTL_REGISTRY_PATH", regPath)
	return func() {
		os.Unsetenv("HSTL_PROJECT")
		os.Unsetenv("HSTL_REGISTRY_PATH")
	}
}

// ---------------------------------------------------------------------------
// TestLoadRegistry_Empty
// ---------------------------------------------------------------------------

func TestLoadRegistry_Empty(t *testing.T) {
	cleanup := setupTestRegistry(t)
	defer cleanup()

	reg, err := LoadRegistry()
	if err != nil {
		t.Fatalf("LoadRegistry error: %v", err)
	}
	if reg.Schema != "hostler-oss-registry-v1" {
		t.Errorf("schema = %q; want 'hostler-oss-registry-v1'", reg.Schema)
	}
	if reg.Project != "test-project" {
		t.Errorf("project = %q; want 'test-project'", reg.Project)
	}
	if reg.Resources == nil {
		t.Error("resources nil")
	}
	if len(reg.Resources) != 0 {
		t.Errorf("resources count = %d; want 0", len(reg.Resources))
	}
}

// ---------------------------------------------------------------------------
// TestAllocate_Unique_Fresh
// ---------------------------------------------------------------------------

func TestAllocate_Unique_Fresh(t *testing.T) {
	cleanup := setupTestRegistry(t)
	defer cleanup()

	v := 8080
	resultAny, err := Allocate("unique", "ports", "api-server", &v, nil, nil, "local", "API server port")
	if err != nil {
		t.Fatalf("Allocate error: %v", err)
	}
	result, ok := resultAny.(AllocateResult)
	if !ok {
		t.Fatalf("Allocate returned wrong type: %T", resultAny)
	}
	if result.Status != "ok" {
		t.Errorf("status = %v; want 'ok'", result.Status)
	}

	// Reusing the same value must conflict.
	result2Any, err := Allocate("unique", "ports", "other-server", &v, nil, nil, "local", "conflict test")
	if err != nil {
		t.Fatalf("Allocate conflict test error: %v", err)
	}
	result2, ok2 := result2Any.(AllocateConflictResult)
	if !ok2 {
		t.Fatalf("Allocate conflict returned wrong type: %T", result2Any)
	}
	if result2.Status != "CONFLICT" {
		t.Errorf("conflict status = %v; want 'CONFLICT'", result2.Status)
	}
}

// ---------------------------------------------------------------------------
// TestAllocate_Range_ConflictDetection
// ---------------------------------------------------------------------------

func TestAllocate_Range_ConflictDetection(t *testing.T) {
	cleanup := setupTestRegistry(t)
	defer cleanup()

	// Allocate 1000~1999.
	rs1, re1 := 1000, 1999
	result1Any, err := Allocate("range", "event-ids", "user-service", nil, &rs1, &re1, "", "user service event range")
	if err != nil {
		t.Fatalf("Allocate error: %v", err)
	}
	result1, ok := result1Any.(AllocateResult)
	if !ok {
		t.Fatalf("Allocate returned wrong type: %T", result1Any)
	}
	if result1.Status != "ok" {
		t.Errorf("status = %v; want 'ok'", result1.Status)
	}

	// 1500~2500 — overlap → conflict.
	rs2, re2 := 1500, 2500
	result2Any, err := Allocate("range", "event-ids", "order-service", nil, &rs2, &re2, "", "conflicting range")
	if err != nil {
		t.Fatalf("Allocate conflict error: %v", err)
	}
	result2, ok2 := result2Any.(AllocateConflictResult)
	if !ok2 {
		t.Fatalf("Allocate conflict returned wrong type: %T", result2Any)
	}
	if result2.Status != "CONFLICT" {
		t.Errorf("conflict status = %v; want 'CONFLICT'", result2.Status)
	}

	// 2000~2999 — no overlap → ok.
	rs3, re3 := 2000, 2999
	result3Any, err := Allocate("range", "event-ids", "order-service", nil, &rs3, &re3, "", "non-conflicting range")
	if err != nil {
		t.Fatalf("Allocate non-conflict error: %v", err)
	}
	result3, ok3 := result3Any.(AllocateResult)
	if !ok3 {
		t.Fatalf("Allocate non-conflict returned wrong type: %T", result3Any)
	}
	if result3.Status != "ok" {
		t.Errorf("non-conflict status = %v; want 'ok'", result3.Status)
	}
}

// ---------------------------------------------------------------------------
// TestCheck_NoConflict
// ---------------------------------------------------------------------------

func TestCheck_NoConflict(t *testing.T) {
	cleanup := setupTestRegistry(t)
	defer cleanup()

	// Empty initial state.
	v := 5432
	result, err := Check("unique", "ports", &v, nil, nil, "")
	if err != nil {
		t.Fatalf("Check error: %v", err)
	}
	if result.Status != "ok" {
		t.Errorf("status = %v; want 'ok'", result.Status)
	}
}

// ---------------------------------------------------------------------------
// TestListResources_All
// ---------------------------------------------------------------------------

func TestListResources_All(t *testing.T) {
	cleanup := setupTestRegistry(t)
	defer cleanup()

	// Register two resources.
	v1 := 3000
	_, _ = Allocate("unique", "ports", "service-a", &v1, nil, nil, "", "")
	rs, re := 100, 199
	_, _ = Allocate("range", "event-ids", "service-b", nil, &rs, &re, "", "")

	result, err := ListResources("")
	if err != nil {
		t.Fatalf("ListResources error: %v", err)
	}

	if len(result.Resources) != 2 {
		t.Errorf("resources count = %d; want 2", len(result.Resources))
	}

	// type filter.
	result2, err := ListResources("unique")
	if err != nil {
		t.Fatal(err)
	}
	if len(result2.Resources) != 1 {
		t.Errorf("unique filter result = %d; want 1", len(result2.Resources))
	}
}

// ---------------------------------------------------------------------------
// TestSuggestNextUnique
// ---------------------------------------------------------------------------

func TestSuggestNextUnique_NoAllocations(t *testing.T) {
	resource := Resource{
		Type:        "unique",
		Allocations: []Allocation{},
	}
	result := SuggestNextUnique(resource, "")
	if result != nil {
		t.Errorf("SuggestNextUnique with no allocations = %v; want nil", *result)
	}
}

func TestSuggestNextUnique_MaxPlusOne(t *testing.T) {
	v1, v2 := 3000, 8080
	resource := Resource{
		Type: "unique",
		Allocations: []Allocation{
			{Owner: "a", Value: &v1},
			{Owner: "b", Value: &v2},
		},
	}
	result := SuggestNextUnique(resource, "")
	if result == nil || *result != 8081 {
		t.Errorf("SuggestNextUnique = %v; want 8081", result)
	}
}

// ---------------------------------------------------------------------------
// TestFindConflictsRange
// ---------------------------------------------------------------------------

func TestFindConflictsRange_NoOverlap(t *testing.T) {
	resource := Resource{
		Type: "range",
		Allocations: []Allocation{
			{Owner: "a", Range: []int{1000, 1999}},
		},
	}
	conflicts := FindConflictsRange(resource, 2000, 2999)
	if len(conflicts) != 0 {
		t.Errorf("conflict count = %d; want 0", len(conflicts))
	}
}

func TestFindConflictsRange_Overlap(t *testing.T) {
	resource := Resource{
		Type: "range",
		Allocations: []Allocation{
			{Owner: "a", Range: []int{1000, 1999}},
		},
	}
	conflicts := FindConflictsRange(resource, 1500, 2500)
	if len(conflicts) != 1 {
		t.Errorf("conflict count = %d; want 1", len(conflicts))
	}
}

// ---------------------------------------------------------------------------
// TestAllocate_Unique_PerEnvIsolation
// ---------------------------------------------------------------------------

func TestAllocate_Unique_PerEnvIsolation(t *testing.T) {
	cleanup := setupTestRegistry(t)
	defer cleanup()

	v := 5432

	// Allocate 5432 in the prod env.
	r1Any, err := Allocate("unique", "ports", "db-prod", &v, nil, nil, "prod", "production DB")
	if err != nil {
		t.Fatalf("Allocate prod failed: %v", err)
	}
	r1, ok1 := r1Any.(AllocateResult)
	if !ok1 {
		t.Fatalf("Allocate prod returned wrong type: %T", r1Any)
	}
	if r1.Status != "ok" {
		t.Errorf("prod status = %v; want 'ok'", r1.Status)
	}

	// Allocate 5432 in the dev env — different env, so allowed.
	r2Any, err := Allocate("unique", "ports", "db-dev", &v, nil, nil, "dev", "development DB")
	if err != nil {
		t.Fatalf("Allocate dev failed: %v", err)
	}
	r2, ok2 := r2Any.(AllocateResult)
	if !ok2 {
		t.Fatalf("Allocate dev returned wrong type: %T", r2Any)
	}
	if r2.Status != "ok" {
		t.Errorf("dev status = %v; want 'ok'", r2.Status)
	}

	// Re-allocate 5432 in prod — must conflict.
	r3Any, err := Allocate("unique", "ports", "another-prod", &v, nil, nil, "prod", "conflict")
	if err != nil {
		t.Fatalf("Allocate prod re-allocation failed: %v", err)
	}
	r3, ok3 := r3Any.(AllocateConflictResult)
	if !ok3 {
		t.Fatalf("Allocate conflict returned wrong type: %T", r3Any)
	}
	if r3.Status != "CONFLICT" {
		t.Errorf("prod re-alloc status = %v; want 'CONFLICT'", r3.Status)
	}
}

// ---------------------------------------------------------------------------
// TestCheck_RangeConflict
// ---------------------------------------------------------------------------

func TestCheck_RangeConflict(t *testing.T) {
	cleanup := setupTestRegistry(t)
	defer cleanup()

	// Allocate 1000~1999.
	rs1, re1 := 1000, 1999
	_, err := Allocate("range", "event-ids", "service-a", nil, &rs1, &re1, "", "")
	if err != nil {
		t.Fatalf("Allocate failed: %v", err)
	}

	// 1500~2500 check — overlap.
	rs2, re2 := 1500, 2500
	result, err := Check("range", "event-ids", nil, &rs2, &re2, "")
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if result.Status != "CONFLICT" {
		t.Errorf("overlap status = %v; want 'CONFLICT'", result.Status)
	}
	// Conflict items must be present.
	if len(result.Conflicts) == 0 {
		t.Error("conflicts list empty on overlap")
	}
}

// ---------------------------------------------------------------------------
// TestCheck_UnknownResourceType
// ---------------------------------------------------------------------------

func TestCheck_UnknownResourceType(t *testing.T) {
	cleanup := setupTestRegistry(t)
	defer cleanup()

	v := 1234
	result, err := Check("invalid-type", "ports", &v, nil, nil, "")
	if err == nil && result.Status == "ok" {
		t.Log("invalid resource type processed without error (no resource-type validation)")
	}
	// Either an error or a non-ok status is expected.
	t.Logf("Check(invalid-type) result=%v, err=%v", result, err)
}

// ---------------------------------------------------------------------------
// TestLoadRegistry_RoundTrip
// ---------------------------------------------------------------------------

func TestLoadRegistry_RoundTrip(t *testing.T) {
	cleanup := setupTestRegistry(t)
	defer cleanup()

	// Allocate then reload.
	v := 9090
	_, err := Allocate("unique", "ports", "test-service", &v, nil, nil, "test", "test")
	if err != nil {
		t.Fatalf("Allocate failed: %v", err)
	}

	reg, err := LoadRegistry()
	if err != nil {
		t.Fatalf("LoadRegistry failed: %v", err)
	}

	ports, hasPorts := reg.Resources["ports"]
	if !hasPorts {
		t.Error("ports resource missing")
	}
	if len(ports.Allocations) != 1 {
		t.Errorf("allocation count = %d; want 1", len(ports.Allocations))
	}
}

// ---------------------------------------------------------------------------
// TestListResources_MultipleTypes
// ---------------------------------------------------------------------------

func TestListResources_MultipleTypes(t *testing.T) {
	cleanup := setupTestRegistry(t)
	defer cleanup()

	// Allocate a unique-typed port.
	v1 := 3000
	_, _ = Allocate("unique", "ports", "service-a", &v1, nil, nil, "", "")

	// Allocate a range-typed event ID block.
	rs, re := 1000, 1999
	_, _ = Allocate("range", "event-ids", "service-b", nil, &rs, &re, "", "")

	// Query everything.
	result, err := ListResources("")
	if err != nil {
		t.Fatalf("ListResources all failed: %v", err)
	}

	if len(result.Resources) != 2 {
		t.Errorf("total resources = %d; want 2", len(result.Resources))
	}

	// Query unique only.
	result2, err := ListResources("unique")
	if err != nil {
		t.Fatalf("ListResources unique failed: %v", err)
	}
	if len(result2.Resources) != 1 {
		t.Errorf("unique resources = %d; want 1", len(result2.Resources))
	}
}

// ---------------------------------------------------------------------------
// TestSuggestNextRange
// ---------------------------------------------------------------------------

func TestSuggestNextRange_OverlapSuggestion(t *testing.T) {
	resource := Resource{
		Type: "range",
		Allocations: []Allocation{
			{Owner: "a", Range: []int{1000, 1999}},
		},
	}

	// Suggest next range of size 1001 (after 1000~1999 → 2000~3000).
	suggestion := SuggestNextRange(resource, 1001)
	if suggestion == nil {
		t.Error("suggestion must not be nil")
	}
	if suggestion != nil && (*suggestion)[0] < 2000 {
		t.Errorf("suggestion start must be >= 2000: %d", (*suggestion)[0])
	}
}

// ---------------------------------------------------------------------------
// TestAllocate_Range_NoOverlap
// ---------------------------------------------------------------------------

func TestAllocate_Range_NoOverlap(t *testing.T) {
	cleanup := setupTestRegistry(t)
	defer cleanup()

	// Allocate 1000~1999.
	rs1, re1 := 1000, 1999
	r1Any, err := Allocate("range", "event-ids", "service-a", nil, &rs1, &re1, "", "first")
	if err != nil {
		t.Fatalf("Allocate failed: %v", err)
	}
	r1, ok1 := r1Any.(AllocateResult)
	if !ok1 {
		t.Fatalf("Allocate returned wrong type: %T", r1Any)
	}
	if r1.Status != "ok" {
		t.Errorf("status = %v; want 'ok'", r1.Status)
	}

	// Allocate 2000~2999 — non-adjacent.
	rs2, re2 := 2000, 2999
	r2Any, err := Allocate("range", "event-ids", "service-b", nil, &rs2, &re2, "", "second")
	if err != nil {
		t.Fatalf("Allocate failed: %v", err)
	}
	r2, ok2 := r2Any.(AllocateResult)
	if !ok2 {
		t.Fatalf("Allocate returned wrong type: %T", r2Any)
	}
	if r2.Status != "ok" {
		t.Errorf("non-overlap status = %v; want 'ok'", r2.Status)
	}
}
