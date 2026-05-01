// Package cmd - the `hstl hooks install` subcommand.
//
// The CLI assumes `hstl hooks install` exists, and warning.go references the
// command on the recovery path, but for a while no hooks subcommand was
// actually registered, leaving users unable to install the pre-commit hook
// (external projects continued to bypass HMAC verification). This file
// restores the command so it matches the recovery path emitted by warning.go.
//
// Install behavior:
//   - Writes the embedded wrapper scripts to `.git/hooks/pre-commit` and
//     `.git/hooks/commit-msg` under the current cwd, plus chmod +x.
//   - If the target file exists, back it up with the `.bak` suffix and
//     replace (no --force needed).
//   - When the existing file already matches the wrapper, this is a no-op.
//   - Works inside external projects as long as the hstl binary is on PATH
//     (self-contained).
//
// trac: OPS-CM002
package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/brand"
)

// hooksCmd is the root of the hooks subcommand.
var hooksCmd = &cobra.Command{
	Use:   "hooks",
	Short: "Manage git hooks (install / status)",
}

// ── install constants ──────────────────────────────────────────────────

const (
	// preCommitRelPath is the relative path to the target repo's pre-commit hook.
	preCommitRelPath = ".git/hooks/pre-commit"

	// commitMsgRelPath is the relative path to the target repo's commit-msg hook.
	commitMsgRelPath = ".git/hooks/commit-msg"

	// hookFileMode is the file permission mode for hooks (rwxr-xr-x).
	hookFileMode = 0o755

	// hookBackupSuffix is the suffix used when preserving the original file.
	hookBackupSuffix = ".bak"
)

// preCommitHookScript is the wrapper installed at `.git/hooks/pre-commit`.
// A minimal wrapper for external projects - works whenever the brand binary
// is on PATH. The plugin's own repo can continue to use the richer pre-commit
// from `scripts/git-hooks/install.sh` (overwriting with this wrapper still
// dispatches to the same Rule Engine, so behavior is equivalent).
var preCommitHookScript = fmt.Sprintf(`#!/usr/bin/env bash
# %s embedded pre-commit hook
# Generated automatically from the "%s hooks install" recovery path.
set -euo pipefail
exec %s rules run --phase precommit "$@"
`, brand.ProductName, brand.ShortName, brand.ShortName)

// commitMsgHookScript is the wrapper installed at `.git/hooks/commit-msg`.
var commitMsgHookScript = fmt.Sprintf(`#!/usr/bin/env bash
# %s embedded commit-msg hook
# git passes the commit-msg path as $1, so we forward it via --msg-file.
set -euo pipefail
exec %s rules run --phase commit-msg --msg-file "$1"
`, brand.ShortName, brand.ShortName)

// ── hooks install ──────────────────────────────────────────────────────

var (
	hooksInstallDryRun bool
	// Preserve the existing hook and append only the brand wrapper line.
	// In multi-tool chain hook patterns observed downstream (no-bakje +
	// other tools), avoid the chain-loss risk that comes from forced
	// replacement (.bak backup).
	hooksInstallAppend bool
)

var hooksInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install the git pre-commit + commit-msg hooks",
	Long: fmt.Sprintf(`Install the embedded %s wrapper scripts under .git/hooks/ in cwd.

Targets:
  - .git/hooks/pre-commit  - invokes %s rules run --phase precommit
  - .git/hooks/commit-msg  - invokes %s rules run --phase commit-msg

If a target file exists, it is backed up with the .bak suffix and replaced.
When the file already matches, this is a no-op.
Use --dry-run to preview the planned changes without modifying files.`, brand.ShortName, brand.ShortName, brand.ShortName),
	Example: brand.Examplef(
		`hooks install`,
		`hooks install --dry-run -o json`,
	),
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := os.Getwd()
		if err != nil {
			Out.Error("failed to detect cwd: "+err.Error(), "", "")
			os.Exit(exitError)
		}

		// Confirm .git exists - if the target is not a git repo, abort.
		gitDir := filepath.Join(root, ".git")
		if info, sErr := os.Stat(gitDir); sErr != nil || !info.IsDir() {
			Out.Error(
				fmt.Sprintf("not a git repository: %s (no .git directory)", root),
				"NOT_A_REPO",
				"",
			)
			os.Exit(exitError)
		}

		preCommitAbs := filepath.Join(root, preCommitRelPath)
		commitMsgAbs := filepath.Join(root, commitMsgRelPath)

		mode := "replace"
		if hooksInstallAppend {
			mode = "append"
		}

		preResult, err := installHookFile(preCommitAbs, preCommitHookScript, hooksInstallDryRun, mode)
		if err != nil {
			Out.Error("failed to install pre-commit: "+err.Error(), "", "")
			os.Exit(exitError)
		}
		msgResult, err := installHookFile(commitMsgAbs, commitMsgHookScript, hooksInstallDryRun, mode)
		if err != nil {
			Out.Error("failed to install commit-msg: "+err.Error(), "", "")
			os.Exit(exitError)
		}

		result := map[string]any{
			"project_root": root,
			"dry_run":      hooksInstallDryRun,
			"mode":         mode,
			"pre_commit":   preResult,
			"commit_msg":   msgResult,
		}

		var msg string
		if hooksInstallDryRun {
			msg = "[dry-run] hooks would be installed - no changes applied"
		} else {
			msg = "hooks installed - pre-commit + commit-msg"
		}
		Out.Success(msg, result)
		return nil
	},
}

// installHookFile installs content at targetPath.
//
// Mode behavior:
//   - "replace" (default; backward compatible): if the existing file
//     differs, back it up with .bak and replace it.
//   - "append": preserve the existing file and append the brand wrapper
//     line if it is not already present. If the brand line already exists,
//     this is a no-op.
//
// Common rules:
//   - If the existing file matches content exactly, returns {"action": "noop"}.
//   - If the file is missing, create it regardless of mode ({"action": "created"}).
//   - With dryRun=true, the action becomes "would_*" and no IO occurs.
func installHookFile(targetPath, content string, dryRun bool, mode string) (map[string]any, error) {
	result := map[string]any{
		"path": targetPath,
	}

	existing, readErr := os.ReadFile(targetPath)
	switch {
	case readErr == nil && bytes.Equal(existing, []byte(content)):
		result["action"] = "noop"
		return result, nil

	case readErr == nil && mode == "append":
		// append mode - branch on whether the brand line is present.
		brandLine := []byte(brand.ShortName + " rules run")
		if bytes.Contains(existing, brandLine) {
			result["action"] = "append_noop"
			result["detail"] = "brand line already present - chain preserved"
			return result, nil
		}
		appendBlock := buildAppendBlock(content)
		if dryRun {
			result["action"] = "would_append"
			result["preview"] = appendBlock
			return result, nil
		}
		f, openErr := os.OpenFile(filepath.Clean(targetPath), os.O_APPEND|os.O_WRONLY, hookFileMode)
		if openErr != nil {
			return nil, fmt.Errorf("failed to open file for append: %w", openErr)
		}
		defer f.Close()
		if _, wErr := f.WriteString(appendBlock); wErr != nil {
			return nil, fmt.Errorf("failed to append: %w", wErr)
		}
		if err := os.Chmod(targetPath, hookFileMode); err != nil {
			return nil, fmt.Errorf("failed to set permissions: %w", err)
		}
		result["action"] = "appended"
		return result, nil

	case readErr == nil:
		// replace mode (default) - back up to .bak and replace.
		backupPath := targetPath + hookBackupSuffix
		if dryRun {
			result["action"] = "would_replace"
			result["backup"] = backupPath
			return result, nil
		}
		if rnErr := os.Rename(targetPath, backupPath); rnErr != nil {
			return nil, fmt.Errorf("failed to back up existing file: %w", rnErr)
		}
		result["action"] = "replaced"
		result["backup"] = backupPath

	case os.IsNotExist(readErr):
		if dryRun {
			result["action"] = "would_create"
			return result, nil
		}
		result["action"] = "created"

	default:
		return nil, fmt.Errorf("failed to read file: %w", readErr)
	}

	if err := os.WriteFile(targetPath, []byte(content), hookFileMode); err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}
	// WriteFile does not overwrite the existing file's mode, so chmod explicitly.
	if err := os.Chmod(targetPath, hookFileMode); err != nil {
		return nil, fmt.Errorf("failed to set permissions: %w", err)
	}
	return result, nil
}

// buildAppendBlock constructs the brand call block appended at the end of an
// existing hook in append mode. Tries to preserve the chain by extracting
// only the last exec line from content. For simplicity, the standard
// wrapper's core invocation line is used as-is.
func buildAppendBlock(content string) string {
	header := fmt.Sprintf("\n# %s wrapper appended (chain-preserving mode)\n", brand.ProductName)
	// Pull the line starting with exec from content (fall back to full content).
	lines := bytes.Split([]byte(content), []byte("\n"))
	for _, ln := range lines {
		if bytes.HasPrefix(bytes.TrimSpace(ln), []byte("exec ")) {
			return header + string(ln) + "\n"
		}
	}
	return header + content + "\n"
}

// ── hooks status ───────────────────────────────────────────────────────
//
// A self-diagnostic command that lets external projects confirm hook health
// at a glance. install alone could not answer "is the install healthy right
// now?", which forced an inefficient mailbox-report -> verify -> reject loop.

var hooksStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Summarize git hook install state in one line",
	Long: fmt.Sprintf(`Summarize whether the %s wrapper is installed under .git/hooks/ in cwd.
Checks both file presence and the wrapper line (%s rules run).`,
		brand.ShortName, brand.ShortName),
	Example: brand.Examplef(`hooks status`, `hooks status -o json`),
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := os.Getwd()
		if err != nil {
			Out.Error("failed to detect cwd: "+err.Error(), "", "")
			os.Exit(exitError)
		}

		preStatus := checkHookFile(filepath.Join(root, preCommitRelPath), preCommitHookScript)
		msgStatus := checkHookFile(filepath.Join(root, commitMsgRelPath), commitMsgHookScript)

		gitDir := filepath.Join(root, ".git")
		gitOk := false
		if info, sErr := os.Stat(gitDir); sErr == nil && info.IsDir() {
			gitOk = true
		}

		summary := summaryStatus(gitOk, preStatus, msgStatus)
		result := map[string]any{
			"project_root": root,
			"git_repo":     gitOk,
			"pre_commit":   preStatus,
			"commit_msg":   msgStatus,
			"summary":      summary,
		}
		Out.Success(summary, result)
		return nil
	},
}

// hookFileStatus is the inspection result for a single hook file.
type hookFileStatus struct {
	Path           string `json:"path"`
	Exists         bool   `json:"exists"`
	WrapperMatches bool   `json:"wrapper_matches"`
	Detail         string `json:"detail"`
}

// checkHookFile reports whether targetPath matches the expected wrapper.
//   - File missing -> Exists=false
//   - Present but different content -> WrapperMatches=false plus a diff hint
//   - Match -> both true
func checkHookFile(targetPath, expected string) hookFileStatus {
	st := hookFileStatus{Path: targetPath}
	data, err := os.ReadFile(filepath.Clean(targetPath))
	if err != nil {
		if os.IsNotExist(err) {
			st.Detail = "missing"
			return st
		}
		st.Detail = "read_error: " + err.Error()
		return st
	}
	st.Exists = true
	if bytes.Equal(data, []byte(expected)) {
		st.WrapperMatches = true
		st.Detail = "ok"
		return st
	}
	if bytes.Contains(data, []byte(brand.ShortName+" rules run")) {
		st.WrapperMatches = true
		st.Detail = "wrapper_present_custom_script"
		return st
	}
	st.Detail = "wrapper_mismatch - recommend running " + brand.ShortName + " hooks install"
	return st
}

// summaryStatus produces a one-line summary.
func summaryStatus(gitOk bool, pre, msg hookFileStatus) string {
	if !gitOk {
		return "not a git repository - hooks not applied (hostler hooks install must run inside a git repo)"
	}
	preMark := "X"
	if pre.Exists && pre.WrapperMatches {
		preMark = "OK"
	}
	msgMark := "X"
	if msg.Exists && msg.WrapperMatches {
		msgMark = "OK"
	}
	return fmt.Sprintf("hooks status - pre-commit %s / commit-msg %s", preMark, msgMark)
}

// ── hooks doctor ───────────────────────────────────────────────────────
//
// 5-item comprehensive diagnostic.
//  1) hook files present (pre-commit + commit-msg)
//  2) wrapper line ($brand.ShortName rules run) present
//  3) .hostler/project-config.yaml precommit section shape (BranchSyncConfig)
//  4) hstl rules effective list contains precommit.* rules
//  5) hstl binary version

var hooksDoctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Comprehensive hooks diagnostic (5 items)",
	Long: fmt.Sprintf(`Run a comprehensive 5-item diagnostic on the %s hooks environment.

Checks:
  (1) hook files present (pre-commit / commit-msg)
  (2) wrapper line matches (%s rules run)
  (3) .hostler/project-config.yaml precommit section shape
  (4) precommit.* registered in effective rules
  (5) %s binary version`,
		brand.ProductName, brand.ShortName, brand.ShortName),
	Example: brand.Examplef(`hooks doctor`, `hooks doctor -o json`),
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := os.Getwd()
		if err != nil {
			Out.Error("failed to detect cwd: "+err.Error(), "", "")
			os.Exit(exitError)
		}

		checks := runHooksDoctorChecks(root)
		passed := 0
		for _, c := range checks {
			if c.Status == "ok" {
				passed++
			}
		}

		result := map[string]any{
			"project_root": root,
			"checks":       checks,
			"summary":      fmt.Sprintf("%d/%d ok", passed, len(checks)),
		}

		msg := fmt.Sprintf("hooks doctor - %d/%d items passing", passed, len(checks))
		if passed == len(checks) {
			Out.Success(msg, result)
			return nil
		}
		// Partial success - report via Success but include failing items in detail.
		Out.Success(msg, result)
		return nil
	},
}

// doctorCheck is a single diagnostic item.
type doctorCheck struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"` // "ok" | "warn" | "fail" | "skip"
	Detail string `json:"detail"`
}

// runHooksDoctorChecks executes the five checks in order and returns the result slice.
func runHooksDoctorChecks(root string) []doctorCheck {
	checks := make([]doctorCheck, 0, 5)
	checks = append(checks, doctorCheckHookFiles(root))
	checks = append(checks, doctorCheckWrapperLines(root))
	checks = append(checks, doctorCheckConfigPrecommit(root))
	checks = append(checks, doctorCheckRulesEffective(root))
	checks = append(checks, doctorCheckBinaryVersion())
	return checks
}

// doctorCheckHookFiles - presence of the two hook files.
func doctorCheckHookFiles(root string) doctorCheck {
	c := doctorCheck{ID: "hook_files", Name: "hook files present"}
	gitDir := filepath.Join(root, ".git")
	if info, sErr := os.Stat(gitDir); sErr != nil || !info.IsDir() {
		c.Status = "skip"
		c.Detail = "not a git repository - skipping hook check"
		return c
	}
	preExists := fileExists(filepath.Join(root, preCommitRelPath))
	msgExists := fileExists(filepath.Join(root, commitMsgRelPath))
	switch {
	case preExists && msgExists:
		c.Status = "ok"
		c.Detail = "pre-commit and commit-msg both present"
	case preExists != msgExists:
		c.Status = "warn"
		c.Detail = fmt.Sprintf("partial - pre-commit=%t commit-msg=%t (recovery: %s hooks install)", preExists, msgExists, brand.ShortName)
	default:
		c.Status = "fail"
		c.Detail = fmt.Sprintf("both missing - recovery: %s hooks install", brand.ShortName)
	}
	return c
}

// doctorCheckWrapperLines - whether the wrapper script's brand call line matches.
func doctorCheckWrapperLines(root string) doctorCheck {
	c := doctorCheck{ID: "wrapper_lines", Name: "wrapper line match"}
	preStatus := checkHookFile(filepath.Join(root, preCommitRelPath), preCommitHookScript)
	msgStatus := checkHookFile(filepath.Join(root, commitMsgRelPath), commitMsgHookScript)
	if !preStatus.Exists && !msgStatus.Exists {
		c.Status = "skip"
		c.Detail = "hook files missing - skipping wrapper check"
		return c
	}
	if preStatus.WrapperMatches && msgStatus.WrapperMatches {
		c.Status = "ok"
		c.Detail = fmt.Sprintf("wrapper match - %s rules run invocation confirmed", brand.ShortName)
		return c
	}
	c.Status = "warn"
	c.Detail = fmt.Sprintf("wrapper mismatch - pre=%s msg=%s (recovery: %s hooks install)", preStatus.Detail, msgStatus.Detail, brand.ShortName)
	return c
}

// doctorCheckConfigPrecommit - shape check for the precommit section in .hostler/project-config.yaml.
//   - File missing -> skip (private fork case)
//   - precommit section absent -> ok (default values assumed)
//   - branch_sync shape valid -> ok
//   - shape error -> warn
func doctorCheckConfigPrecommit(root string) doctorCheck {
	c := doctorCheck{ID: "config_precommit", Name: "config precommit section shape"}
	cfg, err := loadProjectConfigSilent(root)
	if err != nil {
		c.Status = "warn"
		c.Detail = "failed to load config: " + err.Error()
		return c
	}
	if cfg == nil {
		c.Status = "skip"
		c.Detail = ".hostler/project-config.yaml missing - using defaults"
		return c
	}
	if cfg.Precommit == nil {
		c.Status = "ok"
		c.Detail = "precommit section unset - using defaults"
		return c
	}
	if cfg.Precommit.BranchSync != nil {
		bs := cfg.Precommit.BranchSync
		threshold := "default"
		if bs.Threshold != nil {
			threshold = fmt.Sprintf("%d", *bs.Threshold)
		}
		warnRatio := "default"
		if bs.WarnRatio != nil {
			warnRatio = fmt.Sprintf("%d", *bs.WarnRatio)
		}
		c.Status = "ok"
		c.Detail = fmt.Sprintf("BranchSync configured - threshold=%s warn_ratio=%s", threshold, warnRatio)
		return c
	}
	c.Status = "ok"
	c.Detail = "precommit section present - branch_sync unset"
	return c
}

// doctorCheckRulesEffective - whether at least one precommit.* rule is registered.
func doctorCheckRulesEffective(root string) doctorCheck {
	c := doctorCheck{ID: "rules_effective", Name: "effective rules contain precommit.*"}
	count, err := countPrecommitRules(root)
	if err != nil {
		c.Status = "warn"
		c.Detail = "failed to load rules: " + err.Error()
		return c
	}
	if count == 0 {
		c.Status = "fail"
		c.Detail = "0 precommit.* rules - recovery: " + brand.ShortName + " rules effective"
		return c
	}
	c.Status = "ok"
	c.Detail = fmt.Sprintf("%d precommit.* rules registered", count)
	return c
}

// doctorCheckBinaryVersion shows the hstl binary version.
func doctorCheckBinaryVersion() doctorCheck {
	c := doctorCheck{ID: "binary_version", Name: "binary version"}
	c.Status = "ok"
	c.Detail = fmt.Sprintf("%s %s (commit=%s built=%s)", brand.ShortName, Version, Commit, Date)
	if Version == "dev" {
		c.Status = "warn"
		c.Detail += " - dev build (not a release)"
	}
	return c
}

// fileExists reports whether path exists as a regular file.
func fileExists(path string) bool {
	info, err := os.Stat(filepath.Clean(path))
	return err == nil && !info.IsDir()
}

// ── hooks check ────────────────────────────────────────────────────────
//
// Multi-tool chain hook pattern aware. Verifies only the brand wrapper line
// ($brand.ShortName rules run) inside hooks - no impact on the rest of the
// chain. exit 0/1 lets external tools (CI / Makefile) drive follow-up.

var hooksCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Verify the brand wrapper line inside hooks (multi-tool chain aware)",
	Long: fmt.Sprintf(`Verify only that the %s wrapper invocation line is present in
.git/hooks/pre-commit + commit-msg under cwd (without affecting any other chain).

Rules:
  - both hooks contain the brand line -> exit 0
  - either is missing -> exit 1 (recovery: %s hooks install --append)`,
		brand.ShortName, brand.ShortName),
	Example: brand.Examplef(`hooks check`, `hooks check -o json`),
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := os.Getwd()
		if err != nil {
			Out.Error("failed to detect cwd: "+err.Error(), "", "")
			os.Exit(exitError)
		}

		preOk, preDetail := containsBrandLine(filepath.Join(root, preCommitRelPath))
		msgOk, msgDetail := containsBrandLine(filepath.Join(root, commitMsgRelPath))

		result := map[string]any{
			"project_root":  root,
			"pre_commit_ok": preOk,
			"pre_commit":    preDetail,
			"commit_msg_ok": msgOk,
			"commit_msg":    msgDetail,
		}

		if preOk && msgOk {
			Out.Success(fmt.Sprintf("hooks check OK - %s wrapper line present in both hooks", brand.ShortName), result)
			return nil
		}
		Out.Error(
			fmt.Sprintf("hooks check FAIL - pre=%t msg=%t (recovery: %s hooks install --append)",
				preOk, msgOk, brand.ShortName),
			"BRAND_LINE_MISSING",
			fmt.Sprintf("Run %s hooks install --append to preserve the chain", brand.ShortName),
		)
		os.Exit(exitError)
		return nil
	},
}

// containsBrandLine reports whether the file at path contains the brand
// wrapper invocation line. File-missing or read failures both return false
// plus the reason.
func containsBrandLine(path string) (bool, string) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		if os.IsNotExist(err) {
			return false, "missing"
		}
		return false, "read_error: " + err.Error()
	}
	pattern := []byte(brand.ShortName + " rules run")
	if bytes.Contains(data, pattern) {
		return true, "ok (" + string(pattern) + ")"
	}
	return false, "brand line missing - recommend install --append"
}

// ── init ───────────────────────────────────────────────────────────────

func init() {
	hooksInstallCmd.Flags().BoolVar(&hooksInstallDryRun, "dry-run", false, "Print the planned install without modifying files")
	hooksInstallCmd.Flags().BoolVar(&hooksInstallAppend, "append", false, "Preserve the existing hook and append the brand wrapper line only (multi-tool chain)")
	hooksCmd.AddCommand(hooksInstallCmd)
	hooksCmd.AddCommand(hooksStatusCmd)
	hooksCmd.AddCommand(hooksDoctorCmd)
	hooksCmd.AddCommand(hooksCheckCmd)
	rootCmd.AddCommand(hooksCmd)
}
