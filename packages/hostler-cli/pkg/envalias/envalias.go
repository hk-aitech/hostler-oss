// Package envalias — thin wrapper around os.Getenv that prepends the
// HSTL_ prefix.
//
// Centralising the lookup keeps call sites uniform across the codebase
// and makes it trivial to grep for HSTL_-prefixed environment usage.
//
// Usage:
//
//	root := envalias.Lookup("PROJECT_ROOT")  // reads HSTL_PROJECT_ROOT
//	if v := envalias.Lookup("AUDIT_QUERY_LIMIT"); v != "" { ... }
package envalias

import "os"

const prefixHSTL = "HSTL_"

// Lookup returns the value of HSTL_<key>. Returns an empty string when
// the variable is not set.
//
//	key is the prefix-less suffix (e.g. "PROJECT_ROOT" / "AUDIT_QUERY_LIMIT").
func Lookup(key string) string {
	return os.Getenv(prefixHSTL + key)
}

// LookupOK is the (value, found) variant of Lookup.
//
//	Returns ("", false) when the variable is not set; ("", true) when it
//	is set to the empty string.
func LookupOK(key string) (string, bool) {
	return os.LookupEnv(prefixHSTL + key)
}
