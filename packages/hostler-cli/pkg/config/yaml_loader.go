// Package config — project-config.yaml loader.
//
// Responsibilities:
//  1. Locate {brand.ProjectDirName}/project-config.yaml.
//  2. Parse YAML into the ProjectConfig struct.
//  3. Process-wide cache (ResetConfigCache provided for test isolation).
//  4. Return nil when missing — callers fall back to built-in defaults.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	"gopkg.in/yaml.v3"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/brand"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/fileutil"
)

// projectConfigDirs is the lookup order for project-config.yaml directories.
// brand.ProjectDirName is the canonical primary location; any legacy
// alternates listed in brand.LegacyProjectDirNames are searched as
// read-only fallbacks.
var projectConfigDirs = append([]string{brand.ProjectDirName}, brand.LegacyProjectDirNames...)

// Cache implementation simplification.
//
// The previous version juggled three variables: RWMutex + double-check + a
// `cachedConfigSet` flag. Replacing that with the standard `sync.Once`
// gives:
//   - initialisation runs exactly once (race-free, language-level guarantee)
//   - the flag variable becomes unnecessary
//   - the manual double-check disappears
//
// To support ResetConfigCache (tests only), the Once itself lives inside an
// atomic.Pointer so it can be swapped out. In production, after one load,
// every subsequent call hits the cache.
var (
	cachedConfig    atomic.Pointer[ProjectConfig] // immutable from the moment of first successful load
	cachedConfigErr atomic.Pointer[error]         // stores parse/IO error (nil = no error)
	configOnce      atomic.Pointer[sync.Once]     // resettable Once
)

func init() {
	configOnce.Store(&sync.Once{})
}

// findProjectConfigYAML returns the path to project-config.yaml.
// Searches brand.ProjectDirName first, then any legacy alternates.
// Returns an empty string when no file is found.
//
// The .yml extension is also accepted (yaml takes priority).
func findProjectConfigYAML() string {
	root := fileutil.GetProjectRoot()
	for _, dir := range projectConfigDirs {
		for _, name := range []string{"project-config.yaml", "project-config.yml"} {
			path := filepath.Join(root, dir, name)
			if _, err := os.Stat(path); err == nil {
				return path
			}
		}
	}
	return ""
}

// LoadProjectConfig parses project-config.yaml and returns ProjectConfig.
//
// Behaviour:
//   - missing file: returns (nil, nil) — callers use built-in defaults.
//   - parse error: returns (nil, error).
//   - success: returns (*ProjectConfig, nil) and caches the result.
//
// Cache: sync.Once performs the actual load only once. Subsequent calls
// read directly from the atomic.Pointer. ResetConfigCache swaps the Once
// instance to force re-initialisation.
func LoadProjectConfig() (*ProjectConfig, error) {
	configOnce.Load().Do(loadProjectConfigOnce)
	cfg := cachedConfig.Load()
	if errPtr := cachedConfigErr.Load(); errPtr != nil {
		return cfg, *errPtr
	}
	return cfg, nil
}

// loadProjectConfigOnce reads the YAML file once and stores the result in
// the global cache. Only invoked from sync.Once.Do.
//
// Order:
//  1. Parse YAML into the ProjectConfig struct.
//  2. Resolve extends and deep-merge presets (later YAML wins).
//  3. Apply AutoMigrate for in-memory v1 -> v2 conversion.
//  4. Major fail-fast: error when version major differs from CurrentSchemaMajor.
//  5. Store in the cache.
//
// Failures propagate at warning severity; cfg is cached either way (fail-safe).
func loadProjectConfigOnce() {
	path := findProjectConfigYAML()
	if path == "" {
		// no file -> (nil, nil) — callers fall back to defaults.
		return
	}

	data, err := os.ReadFile(path)
	if err != nil {
		cachedConfigErr.Store(&err)
		return
	}

	// When extends is present, load the presets and merge them. Parse the
	// YAML to a generic map first to handle extends, then unmarshal into
	// ProjectConfig.
	merged, mergeErr := applyExtendsAndMerge(data)
	if mergeErr != nil {
		cachedConfigErr.Store(&mergeErr)
		return
	}
	finalYAML, remarshErr := yaml.Marshal(merged)
	if remarshErr != nil {
		cachedConfigErr.Store(&remarshErr)
		return
	}

	var cfg ProjectConfig
	if err := yaml.Unmarshal(finalYAML, &cfg); err != nil {
		cachedConfigErr.Store(&err)
		return
	}

	// Migration Runner removed — v2 only.
	if vErr := checkVersionMajor(&cfg); vErr != nil {
		cachedConfigErr.Store(&vErr)
		return
	}

	cachedConfig.Store(&cfg)
}

// checkVersionMajor verifies that the top-level version field's major
// matches CurrentSchemaMajor. On a mismatch, returns an error that points
// to migrate.
//
// Rules:
//   - empty Version -> skip (treated as pre-versioning).
//   - ParseSchemaVersion failure -> warning-level error (format mismatch).
//   - Compatibility Incompatible -> fail-fast error + migrate guidance.
//   - otherwise (Identical/Patch/Minor) -> pass.
func checkVersionMajor(cfg *ProjectConfig) error {
	if cfg == nil || cfg.Version == "" {
		return nil
	}
	ver, err := ParseSchemaVersion(cfg.Version)
	if err != nil {
		return fmt.Errorf("version format error %q: %w (run hstl config migrate --apply to fix)",
			cfg.Version, err)
	}
	current := SchemaVersion{Major: CurrentSchemaMajor, Minor: 0, Patch: 0}
	if ver.Compatible(current) == CompatibilityIncompatible {
		return fmt.Errorf(
			"version major mismatch — file %q, currently supported major %d. "+
				"Run hstl config migrate --apply to migrate the file to the latest format",
			cfg.Version, CurrentSchemaMajor)
	}
	return nil
}

// applyExtendsAndMerge resolves the extends field in the YAML bytes and
// returns the deep-merged result as map[string]any.
//
// When extends is absent or empty, parses the original YAML into a map and
// returns it unchanged. Errors during preset loading bubble up so the
// caller can propagate them as loader errors.
//
// Merge rules:
//  1. Merge the presets in extends order (later wins) -> base.
//  2. Merge the project YAML on top of base (project wins) -> result.
func applyExtendsAndMerge(data []byte) (map[string]any, error) {
	var root map[string]any
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, err
	}
	if root == nil {
		return nil, nil
	}
	extRaw, hasExt := root["extends"]
	if !hasExt {
		return root, nil
	}
	extList, ok := toStringSlice(extRaw)
	if !ok || len(extList) == 0 {
		delete(root, "extends")
		return root, nil
	}
	presetMerged, err := ResolveExtends(extList, 0, nil)
	if err != nil {
		return nil, err
	}
	// Drop the extends field from the project YAML before cloning (the
	// struct still keeps it).
	projectMap := make(map[string]any, len(root))
	for k, v := range root {
		if k == "extends" {
			continue
		}
		projectMap[k] = v
	}
	// presetMerged + projectMap (projectMap is later, so it wins).
	result := mergeMaps(presetMerged, projectMap)
	// Keep the extends field for struct-level traceability (the validator can inspect it).
	result["extends"] = extList
	return result, nil
}

// ResetConfigCache invalidates the cache. Test isolation only.
// Swapping the sync.Once with a fresh instance ensures the next
// LoadProjectConfig call re-initialises.
func ResetConfigCache() {
	configOnce.Store(&sync.Once{})
	cachedConfig.Store(nil)
	cachedConfigErr.Store(nil)
}
