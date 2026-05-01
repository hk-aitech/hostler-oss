package ceremony_test

import (
	"errors"
	"testing"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/ceremony"
)

type fakeLookup struct {
	kbs       map[string]bool
	tasks     map[string]bool
	kbErr     error
	taskErr   error
}

func (f *fakeLookup) KBExists(id string) (bool, error) {
	if f.kbErr != nil {
		return false, f.kbErr
	}
	return f.kbs[id], nil
}

func (f *fakeLookup) TaskExists(id string) (bool, error) {
	if f.taskErr != nil {
		return false, f.taskErr
	}
	return f.tasks[id], nil
}

func TestVerifyRegistration_AllPassed(t *testing.T) {
	lookup := &fakeLookup{
		kbs:   map[string]bool{"M001": true, "A001": true},
		tasks: map[string]bool{"T100": true, "T200": true},
	}
	res, err := ceremony.VerifyRegistration([]string{"M001", "A001"}, []string{"T100", "T200"}, lookup)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Passed {
		t.Errorf("expected passed=true, got false. missing_kbs=%v missing_tasks=%v", res.MissingKBs, res.MissingTasks)
	}
	if res.CheckedKBs != 2 || res.CheckedTasks != 2 {
		t.Errorf("expected 2/2 checks, got %d/%d", res.CheckedKBs, res.CheckedTasks)
	}
}

func TestVerifyRegistration_MissingKB(t *testing.T) {
	lookup := &fakeLookup{
		kbs:   map[string]bool{"M001": true},
		tasks: map[string]bool{"T100": true},
	}
	res, err := ceremony.VerifyRegistration([]string{"M001", "M002"}, []string{"T100"}, lookup)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Passed {
		t.Errorf("expected passed=false (M002 missing)")
	}
	if len(res.MissingKBs) != 1 || res.MissingKBs[0] != "M002" {
		t.Errorf("expected MissingKBs=[M002], got %v", res.MissingKBs)
	}
	if len(res.MissingTasks) != 0 {
		t.Errorf("expected no missing tasks, got %v", res.MissingTasks)
	}
}

func TestVerifyRegistration_MissingTask(t *testing.T) {
	lookup := &fakeLookup{
		kbs:   map[string]bool{},
		tasks: map[string]bool{"T100": true},
	}
	res, err := ceremony.VerifyRegistration([]string{}, []string{"T100", "T200", "T300"}, lookup)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Passed {
		t.Errorf("expected passed=false")
	}
	if len(res.MissingTasks) != 2 {
		t.Errorf("expected 2 missing tasks (T200, T300), got %v", res.MissingTasks)
	}
}

func TestVerifyRegistration_MissingBoth(t *testing.T) {
	lookup := &fakeLookup{
		kbs:   map[string]bool{},
		tasks: map[string]bool{},
	}
	res, err := ceremony.VerifyRegistration([]string{"M001"}, []string{"T100"}, lookup)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Passed {
		t.Errorf("expected passed=false")
	}
	if len(res.MissingKBs) != 1 || len(res.MissingTasks) != 1 {
		t.Errorf("expected 1+1 missing, got KB=%d Task=%d", len(res.MissingKBs), len(res.MissingTasks))
	}
}

func TestVerifyRegistration_EmptyInput(t *testing.T) {
	lookup := &fakeLookup{kbs: map[string]bool{}, tasks: map[string]bool{}}
	res, err := ceremony.VerifyRegistration(nil, nil, lookup)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Passed {
		t.Errorf("expected passed=true for empty input")
	}
	if res.CheckedKBs != 0 || res.CheckedTasks != 0 {
		t.Errorf("expected 0/0 checks, got %d/%d", res.CheckedKBs, res.CheckedTasks)
	}
}

func TestVerifyRegistration_LookupError(t *testing.T) {
	lookup := &fakeLookup{
		kbs:   map[string]bool{},
		kbErr: errors.New("DB connection failed"),
	}
	_, err := ceremony.VerifyRegistration([]string{"M001"}, nil, lookup)
	if err == nil {
		t.Errorf("expected lookup error to propagate")
	}
}

func TestVerifyRegistration_SkipEmptyIDs(t *testing.T) {
	lookup := &fakeLookup{
		kbs:   map[string]bool{"M001": true},
		tasks: map[string]bool{"T100": true},
	}
	res, err := ceremony.VerifyRegistration([]string{"", "M001", ""}, []string{"T100", ""}, lookup)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Passed {
		t.Errorf("expected passed=true (empty IDs skipped)")
	}
}
