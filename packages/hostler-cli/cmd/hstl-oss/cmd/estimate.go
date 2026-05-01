// Task estimate input case normalization plus enum validation.
//
// Background: lowercase input such as `hstl task create --estimate xs` did
// not match the uppercase canonical `_validEstimates` set in pkg/task/task.go
// and was rejected. We replicate the priority normalization pattern and
// uppercase + validate at the CLI entry point.
package cmd

import (
	"fmt"
	"strings"
)

// validEstimates holds the allowed canonical estimate values (uppercase). SSOT.
// Must match `_validEstimates` in pkg/task/task.go.
var validEstimates = map[string]struct{}{
	"XS": {},
	"S":  {},
	"M":  {},
	"L":  {},
	"XL": {},
}

// estimateAliases maps non-canonical inputs to canonical values for verbose forms.
var estimateAliases = map[string]string{
	"extra-small": "XS",
	"extrasmall":  "XS",
	"small":       "S",
	"medium":      "M",
	"large":       "L",
	"extra-large": "XL",
	"extralarge":  "XL",
}

// migrationEstimateKeywords are keywords for which we have repeatedly observed
// structural underestimation bias. After observing migrate/cutover/rename
// category Tasks consistently estimated below their actual size, these are
// flagged as warning triggers. When the title or summary contains any of
// these keywords and estimate=XS, a stderr WARN is emitted.
var migrationEstimateKeywords = []string{
	"migrate", "migration", "cutover", "rename", "move to ", "relocate",
}

// detectMigrationEstimateRisk returns a WARN reason string when the
// title/summary contains migration/cutover keywords and the estimate is XS.
// Returns the empty string in normal cases.
func detectMigrationEstimateRisk(title, summary, estimate string) string {
	if strings.ToUpper(strings.TrimSpace(estimate)) != "XS" {
		return ""
	}
	hay := strings.ToLower(title + " " + summary)
	for _, kw := range migrationEstimateKeywords {
		if strings.Contains(hay, strings.ToLower(kw)) {
			return fmt.Sprintf(
				"detected %q keyword in title/summary - migration/cutover category has confirmed structural underestimation bias (KB M002). Recommend at least S instead of estimate=XS",
				kw,
			)
		}
	}
	return ""
}

// normalizeEstimate converts an estimate input string to the canonical form.
// Trims whitespace, uppercases, applies alias mapping, and validates the enum.
// Empty input passes through.
//
// Accepted: "xs", "XS", "Small", "  m  ", "extra-large" -> "XS"/"S"/"M"/"XL"
// Rejected: "huge", "xxs", "4" -> error
func normalizeEstimate(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", nil
	}
	// Alias lookup is keyed by lowercase.
	lower := strings.ToLower(trimmed)
	if mapped, ok := estimateAliases[lower]; ok {
		return mapped, nil
	}
	upper := strings.ToUpper(trimmed)
	if _, ok := validEstimates[upper]; !ok {
		return "", fmt.Errorf(
			"estimate value %q is not allowed - must be one of XS|S|M|L|XL",
			raw,
		)
	}
	return upper, nil
}
