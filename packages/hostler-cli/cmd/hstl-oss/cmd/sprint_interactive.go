package cmd

// Interactive prompt for sprint complete (MVP).
// Collects a Phase 5 retro from the TTY when missing and saves RETRO.md.
// Interactive resolution for the remaining phases (1/2/4/6/9) is follow-up.
//
// Minimal implementation using only bufio.Scanner, no external deps - reuses
// the same pattern as task complete --interactive.

import (
	"bufio"
	"fmt"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/output"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// retroMinBulletsPerSection is the minimum bullet count per K/P/T section (MVP policy).
const retroMinBulletsPerSection = 1

// collectRetroSection guides the user to enter bullets for one section.
// Terminate input with a blank line; each bullet is one line.
func collectRetroSection(reader *bufio.Reader, sectionName, example string) []string {
	fmt.Fprintf(output.Stderr(), "\n[%s section] - at least %d entry, terminate with blank line\n",
		sectionName, retroMinBulletsPerSection)
	fmt.Fprintf(output.Stderr(), "  example: %s\n", example)

	bullets := []string{}
	for {
		fmt.Fprintf(output.Stderr(), "  %s %d> ", sectionName, len(bullets)+1)
		line, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintf(output.Stderr(), "[--interactive] input failure: %v - continuing with collected entries\n", err)
			break
		}
		line = strings.TrimSpace(line)
		if line == "" {
			if len(bullets) >= retroMinBulletsPerSection {
				break
			}
			fmt.Fprintf(output.Stderr(), "  at least %d entry required - keep entering\n", retroMinBulletsPerSection)
			continue
		}
		bullets = append(bullets, line)
	}
	return bullets
}

// runInteractiveSprintRetro prompts for each K/P/T section when a TTY is
// detected and writes RETRO.md. If the file already exists, returns without
// overwriting (idempotent). Returns (saved bool, err error); saved=false with
// err=nil means non-TTY so we skipped.
func runInteractiveSprintRetro(sprintID, folderPath string) (bool, error) {
	if !isInteractiveTTY() {
		fmt.Fprintln(output.Stderr(), "[--interactive] not a TTY - skipping interactive retro")
		return false, nil
	}

	retroPath := filepath.Join(folderPath, "RETRO.md")
	if _, err := os.Stat(retroPath); err == nil {
		fmt.Fprintf(output.Stderr(), "[--interactive] RETRO.md already exists (%s) - skipping\n", retroPath)
		return false, nil
	}

	fmt.Fprintf(output.Stderr(), "\n%s RETRO.md interactive capture - enter the three KPT sections\n", sprintID)

	reader := bufio.NewReader(os.Stdin)
	keep := collectRetroSection(reader, "Keep", "split sprint-only branches to avoid polluting dev")
	problem := collectRetroSection(reader, "Problem", "missed registering follow-up Tasks in Phase 9")
	try := collectRetroSection(reader, "Try", "always pre-split L-estimated Tasks")

	content := buildRetroContent(sprintID, keep, problem, try)
	if err := os.WriteFile(retroPath, []byte(content), 0o644); err != nil {
		return false, fmt.Errorf("failed to write RETRO.md: %w", err)
	}
	fmt.Fprintf(output.Stderr(), "[--interactive] RETRO.md saved: %s (K=%d / P=%d / T=%d)\n",
		retroPath, len(keep), len(problem), len(try))
	return true, nil
}

// buildRetroContent serializes collected K/P/T bullets into the standard RETRO.md format.
func buildRetroContent(sprintID string, keep, problem, try []string) string {
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("sprint: %s\n", sprintID))
	sb.WriteString(fmt.Sprintf("created: %s\n", time.Now().UTC().Format("2006-01-02")))
	sb.WriteString("mode: interactive\n")
	sb.WriteString("---\n\n")
	sb.WriteString(fmt.Sprintf("# %s Retrospective (KPT - interactive)\n\n", sprintID))

	writeSection := func(title string, bullets []string) {
		sb.WriteString(fmt.Sprintf("## %s\n\n", title))
		for i, b := range bullets {
			sb.WriteString(fmt.Sprintf("- **%c%d.** %s\n", title[0], i+1, b))
		}
		sb.WriteString("\n")
	}
	writeSection("Keep - what to preserve", keep)
	writeSection("Problem - what went wrong", problem)
	writeSection("Try - what to try next", try)

	sb.WriteString("> Captured by `hstl sprint complete --interactive`.\n")
	return sb.String()
}
