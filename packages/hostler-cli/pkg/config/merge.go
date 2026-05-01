// Package config — I07 Profile Composition deep-merge engine (merge.go).
//
// Deep-merge engine across YAML layers. Merges in the order
// extends-chain presets -> final YAML to produce the effective configuration.
//
// Merge rules (modelled after Renovate / ESLint flat v10 / Nx):
//
// 1. **Map**: deep merge — when both sides have the key, recurse if both are
// maps; otherwise override (later wins) for primitives.
// 2. **Array**: concat by default — appends src to dst. No de-duplication
// (the user manages it when needed).
// 3. **Scalar**: override — later value wins (ESLint v10 "order matters").
// 4. **nil vs value**: nil is ignored, value is preserved.
//
// Array fields that need de-duplication will be handled via a future
// `!override` tag or explicit rules. supports plain concat only.
package config

// mergeMaps deep-merges src into dst. dst is mutated in place.
//
// src is the later value, so on a scalar conflict src wins.
// For arrays, the result is dst's existing array concatenated with src.
// nil values are ignored (dst is preserved).
func mergeMaps(dst, src map[string]any) map[string]any {
	if dst == nil {
		dst = make(map[string]any)
	}
	for key, srcVal := range src {
		dstVal, exists := dst[key]
		if !exists || dstVal == nil {
			dst[key] = srcVal
			continue
		}
		// Both sides exist — type-driven merge.
		dstMap, dstIsMap := dstVal.(map[string]any)
		srcMap, srcIsMap := srcVal.(map[string]any)
		if dstIsMap && srcIsMap {
			dst[key] = mergeMaps(dstMap, srcMap)
			continue
		}
		dstSlice, dstIsSlice := dstVal.([]any)
		srcSlice, srcIsSlice := srcVal.([]any)
		if dstIsSlice && srcIsSlice {
			// concat: append src after dst.
			merged := make([]any, 0, len(dstSlice)+len(srcSlice))
			merged = append(merged, dstSlice...)
			merged = append(merged, srcSlice...)
			dst[key] = merged
			continue
		}
		// scalar / type mismatch -> src wins (later wins).
		dst[key] = srcVal
	}
	return dst
}
