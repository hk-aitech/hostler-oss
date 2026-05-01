package cmd

import "path/filepath"

// defaultFilepathMatch delegates to stdlib filepath.Match. Referenced from rules.go for DI.
func defaultFilepathMatch(pattern, name string) (bool, error) {
	return filepath.Match(pattern, name)
}
