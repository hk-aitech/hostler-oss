// path_self_heal.go — task lifecycle file SSOT self-heal.
// Background: the sprint package auto-reconciles file_path drift during the
// sprint lifecycle via an fs-SSOT scan. However, immediately after task
// complete (the moment the file is moved, or when another worktree has
// moved the same task file), a regression was observed twice during the
// surrounding ceremony where DB tasks.file_path was left stale.
// This file is invoked right after transitionTask in the task complete
// flow and reconciles every task file_path under the sprint directory via
// an fs scan.
// Design: same algorithm as pkg/sprint/sprint.go::reconcileTaskPathsFromFS,
// but a separate implementation lives here to avoid the sprint -> task
// import cycle. The DRY violation can later be unified into an
// internal/sync helper package.
// Safety:
// Missing folder / missing frontmatter / DB failure all silent-skip.
// Call cost: O(N) frontmatter parses, with N typically < 10.
package task

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/ports"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/fileutil"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/store"
)

// reconcileTaskPathsForSprint scans the sprint's active tasks/ directory
// and uses each file's frontmatter id as the SSOT to force-update DB
// tasks.sprint + file_path.
// No-op when the sprint argument is empty. Self-heal entry point.
func reconcileTaskPathsForSprint(sprintID string) {
	if strings.TrimSpace(sprintID) == "" {
		return
	}
	gs := store.Get()
	if gs == nil {
		return
	}
	root := fileutil.GetProjectRoot()
	tasksDir := filepath.Join(root, "works", "sprints", "active", sprintID, "tasks")
	entries, err := os.ReadDir(tasksDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		filePath := filepath.Join(tasksDir, e.Name())
		fm, ferr := fileutil.ReadTaskFrontmatter(filePath)
		if ferr != nil || fm == nil {
			continue
		}
		taskIDRaw, ok := fm["id"]
		if !ok {
			continue
		}
		taskID := strings.TrimSpace(fmt.Sprintf("%v", taskIDRaw))
		if taskID == "" || !strings.HasPrefix(taskID, "T") {
			continue
		}
		relPath := fileutil.ToRepoRelative(filePath)
		sp := sprintID
		_ = gs.UpdateTaskDirect(taskID, ports.TaskUpdateFields{
			Sprint:   &sp,
			FilePath: &relPath,
		})
	}
}
