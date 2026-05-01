package projectinfo

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Constants — minimum/maximum values controlling the parser.
const (
	// sprintSummaryMaxLines is the maximum number of summary lines extracted from the SPRINT.md body.
	sprintSummaryMaxLines = 8
	// sprintSummaryMinMeaningfulLen is the minimum length (whitespace excluded) for a "meaningful" line.
	sprintSummaryMinMeaningfulLen = 3
)

// SprintSummary holds summary information extracted from a SPRINT.md file.
// Includes goal + multi-line summary so users can grasp the entire Sprint at a glance.
type SprintSummary struct {
	// Goal is the goal field from the frontmatter (1 line).
	Goal string `json:"goal,omitempty"`
	// Summary contains meaningful lines extracted from the `## Goal` or
	// `## Overview` section (>= 3 lines recommended). Enough to convey the
	// Sprint character at a glance.
	Summary []string `json:"summary,omitempty"`
	// TaskCount is the number of Tasks in the `## Task list` table (summary).
	TaskCount int `json:"task_count"`
}

// LoadSprintSummary reads SPRINT.md from a sprint folder path and extracts
// the summary. On failure returns an empty SprintSummary (callers can use
// the value directly without a nil check).
//
// Extraction rules:
//   - frontmatter.goal -> Goal
//   - `## Goal` section body -> Summary (first sprintSummaryMaxLines meaningful lines)
//   - if `## Overview` exists, append after `## Goal`
//   - `## Task list` table row count -> TaskCount
func LoadSprintSummary(sprintFolder string) SprintSummary {
	if sprintFolder == "" {
		return SprintSummary{}
	}
	path := filepath.Join(sprintFolder, "SPRINT.md")
	f, err := os.Open(path)
	if err != nil {
		return SprintSummary{}
	}
	defer func() { _ = f.Close() }()

	var (
		summary         SprintSummary
		inFrontmatter   bool
		inGoalSection   bool
		inTasksSection  bool
		lines           []string
		frontmatterDone bool
	)

	scanner := bufio.NewScanner(f)
	// SPRINT.md may include long Task titles, so expand the buffer.
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		raw := scanner.Text()
		trimmed := strings.TrimSpace(raw)

		// Handle frontmatter boundaries.
		if !frontmatterDone {
			if trimmed == "---" {
				if !inFrontmatter {
					inFrontmatter = true
					continue
				}
				inFrontmatter = false
				frontmatterDone = true
				continue
			}
			if inFrontmatter {
				if k, v, ok := parseFrontmatterGoal(trimmed); ok && k == "goal" {
					summary.Goal = v
				}
				continue
			}
		}

		// Section heading transition.
		if strings.HasPrefix(trimmed, "## ") {
			header := strings.TrimPrefix(trimmed, "## ")
			inGoalSection = false
			inTasksSection = false
			switch {
			case strings.HasPrefix(header, "Goal") || strings.HasPrefix(header, "Overview"):
				inGoalSection = true
			case strings.HasPrefix(header, "Task list") || strings.HasPrefix(header, "Tasks"):
				inTasksSection = true
			}
			continue
		}

		// Collect section body.
		if inGoalSection {
			if len(lines) >= sprintSummaryMaxLines {
				continue
			}
			compact := strings.TrimSpace(raw)
			if len(compact) < sprintSummaryMinMeaningfulLen {
				continue
			}
			// Light link/image cleanup (asterisks/backticks preserved).
			lines = append(lines, compact)
			continue
		}

		if inTasksSection {
			// Table row count: starts with "| T" (excludes header/divider).
			if strings.HasPrefix(trimmed, "| T") {
				summary.TaskCount++
			}
			continue
		}
	}

	// Drop lines from Summary that match Goal exactly (no duplicate display).
	if summary.Goal != "" {
		filtered := lines[:0]
		for _, l := range lines {
			if l != summary.Goal {
				filtered = append(filtered, l)
			}
		}
		lines = filtered
	}

	// When Summary is empty, fall back to a Task-count line.
	if len(lines) == 0 && summary.TaskCount > 0 {
		lines = append(lines, fmt.Sprintf("%d Tasks scheduled", summary.TaskCount))
	}

	summary.Summary = lines
	return summary
}

// parseFrontmatterGoal extracts key/value from a YAML single-value line
// like `goal: "..."`. Simple parsing — complex YAML must use
// yaml.Unmarshal, but goal is conventionally a single-line string so this
// is sufficient.
func parseFrontmatterGoal(line string) (key, value string, ok bool) {
	idx := strings.Index(line, ":")
	if idx <= 0 {
		return "", "", false
	}
	key = strings.TrimSpace(line[:idx])
	value = strings.TrimSpace(line[idx+1:])
	// Strip matching surrounding quotes.
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') ||
			(value[0] == '\'' && value[len(value)-1] == '\'') {
			value = value[1 : len(value)-1]
		}
	}
	if value == "" {
		return "", "", false
	}
	return key, value, true
}
