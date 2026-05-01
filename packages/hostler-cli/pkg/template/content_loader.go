// Package template — Content-as-Template extraction.
// Moves the prose previously hard-coded under the `reminders` /
// `sprint_ceremony` sections of `project-config.yaml` (reminder
// sentences, ceremony checklist items) into Markdown files under
// `.hstl-oss/templates/`.
// Loading policy:
// The file path is determined from the event name (e.g.
// task.start) or the ceremony section name (start/complete).
// When the file exists, its frontmatter is parsed and the bullet
// list in the body is extracted.
// When the file is missing, `ErrTemplateNotFound` is returned —
// the caller proceeds to the next fallback step (the YAML
// section or a built-in default).
// File format:
//	--
//	event: task.start
//	description: reminders printed at task-start time
//	severity: info
//	--
//	first reminder item
//	second reminder item
package template

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/brand"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/fileutil"
)

// templatesRootDirName is the template root directory relative to the
// repo root.
const templatesRootDirName = brand.ProjectDirName + "/templates"

// reminderSubDir is the sub-directory name under templatesRootDirName
// for reminder templates.
const reminderSubDir = "reminders"

// ceremonySubDir is the sub-directory name under templatesRootDirName
// for ceremony templates.
const ceremonySubDir = "ceremony"

// bulletPrefix is the prefix that identifies bullet-list items in a
// body.
const bulletPrefix = "- "

// frontmatterDelim is the YAML frontmatter delimiter character.
const frontmatterDelim = "---"

// ErrTemplateNotFound is returned when the template file does not
// exist. Callers must inspect this error and proceed to the fallback
// path.
var ErrTemplateNotFound = errors.New("template file not found")

// TemplateMeta represents the frontmatter metadata. Currently
// recorded for callers that want to inspect it.
type TemplateMeta struct {
	Event       string `yaml:"event,omitempty"`
	Section     string `yaml:"section,omitempty"`
	Description string `yaml:"description,omitempty"`
	Severity    string `yaml:"severity,omitempty"`
	AppliesTo   string `yaml:"applies-to,omitempty"`
}

// LoadReminderContent loads the reminder template for the given
// event name (e.g. task.start). Returns the body bullet list as a
// string slice.
func LoadReminderContent(eventName string) ([]string, *TemplateMeta, error) {
	if eventName == "" {
		return nil, nil, fmt.Errorf("reminder event name is empty")
	}
	path := templateFilePath(reminderSubDir, eventName)
	return loadTemplate(path)
}

// LoadCeremonyContent loads the template for the given ceremony
// section name ("start" or "complete").
func LoadCeremonyContent(section string) ([]string, *TemplateMeta, error) {
	if section == "" {
		return nil, nil, fmt.Errorf("ceremony section name is empty")
	}
	path := templateFilePath(ceremonySubDir, section)
	return loadTemplate(path)
}

// templateFilePath returns the absolute template-file path relative
// to the repo root.
func templateFilePath(subdir, name string) string {
	root := fileutil.GetProjectRoot()
	return filepath.Join(root, templatesRootDirName, subdir, name+".md")
}

// loadTemplate reads the file, parses its frontmatter, and extracts
// the bullet list from the body.
func loadTemplate(path string) ([]string, *TemplateMeta, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, fmt.Errorf("%w: %s", ErrTemplateNotFound, path)
		}
		return nil, nil, fmt.Errorf("template read failed (%s): %w", path, err)
	}

	meta, body, err := splitFrontmatter(data)
	if err != nil {
		return nil, nil, fmt.Errorf("frontmatter parse (%s): %w", path, err)
	}

	items := extractBulletItems(body)
	return items, meta, nil
}

// splitFrontmatter parses the YAML frontmatter (delimited by `---`)
// at the top of the file and returns the parsed metadata together
// with the remaining body. When no frontmatter is present, an empty
// metadata struct is returned.
func splitFrontmatter(data []byte) (*TemplateMeta, []byte, error) {
	meta := &TemplateMeta{}

	// If the file does not start with `---\n`, there is no
	// frontmatter.
	const delimLine = frontmatterDelim + "\n"
	if !bytes.HasPrefix(data, []byte(delimLine)) {
		return meta, data, nil
	}

	rest := data[len(delimLine):]
	end := bytes.Index(rest, []byte("\n"+delimLine))
	if end < 0 {
		// No closing delimiter -> treat the entire body as data.
		return meta, data, nil
	}

	frontmatterBytes := rest[:end]
	if err := yaml.Unmarshal(frontmatterBytes, meta); err != nil {
		return nil, nil, err
	}

	bodyStart := end + len("\n"+delimLine)
	if bodyStart > len(rest) {
		return meta, nil, nil
	}
	return meta, rest[bodyStart:], nil
}

// extractBulletItems extracts only the content of bullet-list items
// (lines starting with `- `) from the body and returns them as a
// slice. Blank or comment lines are ignored.
func extractBulletItems(body []byte) []string {
	var items []string
	scanner := bufio.NewScanner(bytes.NewReader(body))
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), " \t")
		trimmed := strings.TrimLeft(line, " \t")
		if !strings.HasPrefix(trimmed, bulletPrefix) {
			continue
		}
		content := strings.TrimSpace(trimmed[len(bulletPrefix):])
		if content == "" {
			continue
		}
		items = append(items, content)
	}
	return items
}
