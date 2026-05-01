// Package consistency provides consistency warnings to attach to MCP tool responses.
// QuickCheck returns warnings such as residual placeholder bodies, with a TTL cache.
package consistency

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/task"
)

const (
	// cacheTTL prevents repeated checks within the same session.
	cacheTTL = 30 * time.Second
)

var (
	mu          sync.Mutex
	cachedWarns []string
	cachedAt    time.Time
)

// QuickCheck returns the list of consistency warnings.
// Returns the cached result when called within the TTL.
func QuickCheck() []string {
	mu.Lock()
	defer mu.Unlock()

	if time.Since(cachedAt) < cacheTTL {
		return cachedWarns
	}

	var warnings []string

	// 1. Tasks with residual placeholder bodies.
	if placeholders := task.FindPlaceholderTasks(""); len(placeholders) > 0 {
		warnings = append(warnings, fmt.Sprintf(
			"%d Tasks with placeholder bodies: %s",
			len(placeholders), strings.Join(placeholders, ", "),
		))
	}

	// Future extensions:
	// 2. file/DB mismatch checks.
	// 3. uncommitted change checks.

	cachedWarns = warnings
	cachedAt = time.Now()
	return warnings
}

// InvalidateCache invalidates the cache (used in tests).
func InvalidateCache() {
	mu.Lock()
	defer mu.Unlock()
	cachedAt = time.Time{}
}
