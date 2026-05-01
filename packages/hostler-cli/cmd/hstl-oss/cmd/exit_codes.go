package cmd

// Centralized exit code constants.
//
// Numeric os.Exit literals across the CLI command package should reference
// the constants below. Follows the "no magic numbers" principle.
//
// Code meanings:
//
//	exitSuccess  - normal completion (matches cobra default)
//	exitError    - generic error (parameter error, I/O failure, etc.)
//	exitBlocked  - Harness Gate / Sprint Ceremony block (normalized to 1 by pre-commit hook)
const (
	exitSuccess    = 0
	exitError      = 1
	exitInputError = 2 // invalid CLI args (e.g. rotate-hmac-secret constraint violations)
	exitBlocked    = 3
)
