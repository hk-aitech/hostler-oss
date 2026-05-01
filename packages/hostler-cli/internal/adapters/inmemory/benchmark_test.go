// Package inmemory_test — benchmark.
//
// Same scenario as the SQLite adapter — measures InMemoryStore CRUD /
// query performance. Compare against sqlite/benchmark_test.go.
//
// Run:
//
//	cd cli && go test -bench=. -benchmem ./internal/adapters/inmemory/...
//	                                 ./internal/adapters/sqlite/...
package inmemory_test

import (
	"fmt"
	"testing"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/adapters/inmemory"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/ports"
)

// benchmarkSeedCount is the number of seed Tasks — matches the SQLite benchmark.
const benchmarkSeedCount = 100

// seedTasks creates `count` Tasks in the store.
func seedTasks(b *testing.B, s *inmemory.InMemoryStore, sprint string, count int) {
	b.Helper()
	for i := 0; i < count; i++ {
		rec := &ports.TaskRecord{
			TaskID:   fmt.Sprintf("B%04d", i),
			Title:    "bench task",
			Type:     "chore",
			Status:   "todo",
			Priority: "p3",
			Estimate: "XS",
			Sprint:   sprint,
			FilePath: fmt.Sprintf("works/tasks/B%04d.md", i),
		}
		if err := s.CreateTask(rec); err != nil {
			b.Fatalf("seed CreateTask: %v", err)
		}
	}
}

// BenchmarkInMemory_CreateTask — repeatedly create a single Task on an empty store.
func BenchmarkInMemory_CreateTask(b *testing.B) {
	s := inmemory.NewForTest()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := &ports.TaskRecord{
			TaskID:   fmt.Sprintf("T%08d", i),
			Title:    "bench",
			Type:     "chore",
			Status:   "todo",
			Priority: "p3",
			Estimate: "XS",
			FilePath: fmt.Sprintf("t%d.md", i),
		}
		if err := s.CreateTask(rec); err != nil {
			b.Fatalf("CreateTask: %v", err)
		}
	}
}

// BenchmarkInMemory_ListTasks — seed 100 Tasks then call ListTasks repeatedly.
func BenchmarkInMemory_ListTasks(b *testing.B) {
	s := inmemory.NewForTest()
	seedTasks(b, s, "sprint-bench", benchmarkSeedCount)
	sprintFilter := "sprint-bench"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := s.ListTasks(&sprintFilter, nil); err != nil {
			b.Fatalf("ListTasks: %v", err)
		}
	}
}

// BenchmarkInMemory_AggregateSprintProgress — seed 1 Sprint + 100
// Tasks, then loop AggregateSprintProgress.
func BenchmarkInMemory_AggregateSprintProgress(b *testing.B) {
	s := inmemory.NewForTest()
	if err := s.SaveSprint(&ports.SprintRecord{
		SprintID: "sprint-bench",
		Title:    "bench sprint",
		Status:   "active",
	}); err != nil {
		b.Fatalf("SaveSprint: %v", err)
	}
	seedTasks(b, s, "sprint-bench", benchmarkSeedCount)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := s.AggregateSprintProgress("sprint-bench"); err != nil {
			b.Fatalf("AggregateSprintProgress: %v", err)
		}
	}
}
