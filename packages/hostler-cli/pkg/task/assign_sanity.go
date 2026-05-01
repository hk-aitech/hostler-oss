// Package task — assign sanity check.
// Background: hstl task assign trusted DB tasks.file_path unconditionally
// and ran `os.Rename` into the sprint folder. When file_path pointed at
// archive residue (an old same-ID file), the archive file was wrongly
// moved into the sprint folder and the frontmatter id ended up mismatched
// with the task_id.
// Fix: pre-assign sanity check — archive-path fast-fail + frontmatter id
// verification.
// The generalised helper now lives in
// pkg/fileutil/frontmatter_sanity.go; this function is a wrapper.
package task

import (
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/fileutil"
)

// assignSanityCheck verifies the source file's integrity right before a
// task assign.
// Check order (delegated to fileutil.CheckFrontmatterIDSanity):
// 1. archive/ fast-fail — archive files are never move targets (the
// fastest block).
// 2. Extract frontmatter id and compare with task_id — BLOCK on mismatch.
// Returns a violation reason string (passed to the caller as
// FailedItem.Reason). An empty string indicates pass.
func assignSanityCheck(taskID, resolvedPath string) string {
	r := fileutil.CheckFrontmatterIDSanity(taskID, resolvedPath, "task")
	return r.Reason
}

// extractFrontmatterID — backward-compat shim that delegates to the
// generalised helper.
func extractFrontmatterID(path string) (string, error) {
	return fileutil.ExtractFrontmatterID(path)
}
