// Package portpreflight measures port-conversion scope.
//
// Sprint planning frequently misjudged the size of refactor Tasks; this
// package automates the count of where (and how often) cmd/ directly
// invokes the public functions of a pkg/X package.
//
// Usage:
//
//	hstl sprint preflight-scope --pkg mailbox
//
// Output: list of called public functions + per-function call counts +
// per-file call counts. Replaces the manual grep that was Step 1
// (Scope measurement) of the playbook.
package portpreflight

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Excluded files — composition.go is the intentional single entry point
// for adapter wiring. Test files are not in scope for measurement.
const (
	excludeCompositionFile = "composition.go"
	testFileSuffix         = "_test.go"
)

// FunctionStat is the call statistics for a single public function.
type FunctionStat struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// FileStat is the call statistics for a single file.
type FileStat struct {
	Path  string `json:"path"`
	Count int    `json:"count"`
}

// Result is the preflight analysis result.
type Result struct {
	Package        string         `json:"package"`
	CmdDir         string         `json:"cmd_dir"`
	FilesScanned   int            `json:"files_scanned"`
	FilesWithCalls int            `json:"files_with_calls"`
	CallsTotal     int            `json:"calls_total"`
	FunctionsCount int            `json:"functions_count"`
	Functions      []FunctionStat `json:"functions"`
	Files          []FileStat     `json:"files"`
}

// Analyze searches *.go files under cmdDir (excluding _test.go and
// composition.go) for the pattern `pkgName.FuncName(` and returns the
// statistics.
func Analyze(pkgName, cmdDir string) (*Result, error) {
	if pkgName == "" {
		return nil, fmt.Errorf("pkg name is empty")
	}
	if cmdDir == "" {
		return nil, fmt.Errorf("cmd directory path is empty")
	}
	info, err := os.Stat(cmdDir)
	if err != nil {
		return nil, fmt.Errorf("access cmd directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("cmd path is not a directory: %s", cmdDir)
	}

	// Escape pkgName for use in the regex.
	// Pattern: \b<pkgName>\.([A-Z][a-zA-Z0-9_]*)\(
	// The leading \b excludes other names (e.g. myMailbox.X).
	pattern := fmt.Sprintf(`\b%s\.([A-Z][a-zA-Z0-9_]*)\(`, regexp.QuoteMeta(pkgName))
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("compile regex: %w", err)
	}

	entries, err := os.ReadDir(cmdDir)
	if err != nil {
		return nil, fmt.Errorf("read cmd directory: %w", err)
	}

	funcCounts := make(map[string]int)
	fileCounts := make(map[string]int)
	filesScanned := 0

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".go") {
			continue
		}
		if strings.HasSuffix(name, testFileSuffix) {
			continue
		}
		if name == excludeCompositionFile {
			continue
		}
		filesScanned++

		fullPath := filepath.Join(cmdDir, name)
		data, err := os.ReadFile(fullPath)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", fullPath, err)
		}
		matches := re.FindAllStringSubmatch(string(data), -1)
		if len(matches) == 0 {
			continue
		}
		for _, m := range matches {
			funcCounts[m[1]]++
		}
		fileCounts[name] = len(matches)
	}

	// Sort results — call count desc, ties broken by name asc.
	funcs := make([]FunctionStat, 0, len(funcCounts))
	for k, v := range funcCounts {
		funcs = append(funcs, FunctionStat{Name: k, Count: v})
	}
	sort.Slice(funcs, func(i, j int) bool {
		if funcs[i].Count != funcs[j].Count {
			return funcs[i].Count > funcs[j].Count
		}
		return funcs[i].Name < funcs[j].Name
	})

	files := make([]FileStat, 0, len(fileCounts))
	for k, v := range fileCounts {
		files = append(files, FileStat{Path: k, Count: v})
	}
	sort.Slice(files, func(i, j int) bool {
		if files[i].Count != files[j].Count {
			return files[i].Count > files[j].Count
		}
		return files[i].Path < files[j].Path
	})

	totalCalls := 0
	for _, v := range funcCounts {
		totalCalls += v
	}

	return &Result{
		Package:        pkgName,
		CmdDir:         cmdDir,
		FilesScanned:   filesScanned,
		FilesWithCalls: len(fileCounts),
		CallsTotal:     totalCalls,
		FunctionsCount: len(funcCounts),
		Functions:      funcs,
		Files:          files,
	}, nil
}
