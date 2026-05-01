// Package brand keeps the product identity constants in a single file.
//
// Forks should only need to edit this file to complete a rename pass.
package brand

import "strings"

// ProductName is the canonical lowercase product name (used in logs and messages).
const ProductName = "hostler-oss"

// ProductNameUC is the uppercase product name, used as an environment-variable prefix.
const ProductNameUC = "HOSTLER_OSS"

// BinaryName is the primary CLI binary name.
const BinaryName = "hstl-oss"

// ShortName is the short command name shown in help and examples.
const ShortName = "hstl-oss"

// EnvPrefix is the environment-variable prefix (trailing underscore included).
const EnvPrefix = ProductNameUC + "_"

// LogPrefix is the structured-log prefix.
const LogPrefix = "[" + BinaryName + "]"

// ProjectDirName is the hidden project configuration directory.
const ProjectDirName = ".hstl-oss"

// LegacyProjectDirNames lists older project-configuration directory names that
// discovery should fall back to (read-only). Writes always target ProjectDirName.
var LegacyProjectDirNames = []string{".hstl", ".hostler"}

// RegistrySchemaVersion identifies the registry format on disk.
const RegistrySchemaVersion = "hostler-oss-registry-v1"

// GitRepoURL is the source repository URL.
const GitRepoURL = "https://github.com/hk-aitech/hostler-oss"

// CmdLine returns a single-line cobra Example entry: two-space indent followed
// by ShortName and the supplied argument string.
func CmdLine(args string) string {
	return "  " + ShortName + " " + args
}

// Examplef joins multiple lines into a cobra Example block. Each line is
// prefixed with ShortName and a two-space indent.
func Examplef(lines ...string) string {
	parts := make([]string, len(lines))
	for i, l := range lines {
		parts[i] = CmdLine(l)
	}
	return strings.Join(parts, "\n")
}
