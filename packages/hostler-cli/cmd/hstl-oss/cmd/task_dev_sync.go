// Package cmd - dev sync pre-check that runs at task start.
//
// Background: in parallel-worktree environments, while you work on a
// sprint-NN branch, origin/dev may already have advanced from another
// sprint's merge. Without knowing this, you can hit conflicts during the
// sprint complete -> dev merge or skip the reverse-merge ceremony entirely.
//
// Fix: just before task start, check ahead/behind against origin/dev. When
// behind > 0, emit a stderr WARN. Opt out with the env var
// HSTL_TASK_START_DEV_SYNC=off. Any git failure is a silent skip.
package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const devSyncEnvKey = "HSTL_TASK_START_DEV_SYNC"

// checkDevSyncWarn checks the sync state between origin/dev and the current
// branch and emits a stderr WARN when dev is ahead.
//
// Steps:
//  1. If the opt-out env is "off", skip immediately.
//  2. git fetch origin dev (5s timeout, silent).
//  3. Identify the current branch with git rev-parse --abbrev-ref HEAD.
//  4. Skip when on dev or main (only sprint-* / hotfix-* branches matter).
//  5. git rev-list --count HEAD..origin/dev -> behind count.
//  6. If behind > 0, emit a stderr WARN.
//
// Graceful: any git / fetch / branch detection failure becomes a silent skip.
func checkDevSyncWarn() {
	if strings.EqualFold(os.Getenv(devSyncEnvKey), "off") {
		return
	}

	// 5-second timeout fetch
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if !isGitRepo() {
		return
	}

	branch, err := currentBranch()
	if err != nil || branch == "" {
		return
	}

	// Skip the check on dev/main - only sprint-* / hotfix-* branches matter.
	if branch == "dev" || branch == "main" {
		return
	}

	// fetch (silent, continue on failure)
	fetchCmd := exec.CommandContext(ctx, "git", "fetch", "origin", "dev")
	fetchCmd.Stdout = nil
	fetchCmd.Stderr = nil
	_ = fetchCmd.Run()

	behind, err := countBehindOriginDev()
	if err != nil {
		return
	}
	if behind == 0 {
		return
	}

	fmt.Fprintf(os.Stderr,
		"[task start] origin/dev is %d commit(s) ahead - recommend syncing before work: git merge origin/dev (opt-out: %s=off)\n",
		behind, devSyncEnvKey,
	)
}

// isGitRepo reports whether cwd is a git repository.
func isGitRepo() bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Stderr = nil
	return cmd.Run() == nil
}

// currentBranch returns the current branch name.
func currentBranch() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// countBehindOriginDev returns the commit count for HEAD..origin/dev (how far
// behind dev the current branch is).
func countBehindOriginDev() (int, error) {
	out, err := exec.Command("git", "rev-list", "--count", "HEAD..origin/dev").Output()
	if err != nil {
		return 0, err
	}
	n, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		return 0, err
	}
	return n, nil
}
