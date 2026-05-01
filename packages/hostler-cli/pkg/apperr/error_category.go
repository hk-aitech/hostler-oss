// Package apperr — error category constants.
//
// Canonical values for the JSON response `error_category` field. When an
// AI agent receives response.status="error", it uses this field as the
// branching key to decide automatic responses (task_reopen / harness
// check / write Task body, etc.).
//
// Naming: SCREAMING_SNAKE_CASE, three levels of entity / cause / detail.
// e.g. PLACEHOLDER_BODY, RESULT_STRICT_MISSING_FILES, HARNESS_BLOCKED.
//
// Detailed catalog (cause / response / bypass conditions):
// docs/04-guides/cli-error-category-catalog.md
// ADR-003 follow-up — two layers: (audit observability) +
// (JSON response schema).
package apperr

// ErrorCategory is a canonical constant for the JSON response
// error_category field. Any value outside this set in a response is
// treated as missing documentation or drift.
type ErrorCategory string

// Standard error categories — used across all CLI paths.
const (
	// Base 4 (existed before , made constants in).
	CategoryNotFound ErrorCategory = "NOT_FOUND"
	CategoryBlocked ErrorCategory = "BLOCKED"
	CategoryRejected ErrorCategory = "REJECTED"
	CategoryInvalidState ErrorCategory = "INVALID_STATE"
	CategoryInternalError ErrorCategory = "INTERNAL_ERROR"

	// detail — BLOCKED cause variants.
	CategoryHarnessBlocked ErrorCategory = "HARNESS_BLOCKED"
	CategoryPlaceholderBody ErrorCategory = "PLACEHOLDER_BODY"
	CategoryCeremonySectionMissing ErrorCategory = "CEREMONY_SECTION_MISSING"
	CategoryResultStrictMissingFiles ErrorCategory = "RESULT_STRICT_MISSING_FILES"
	CategoryKBLinkMissing ErrorCategory = "KB_LINK_MISSING"
	CategoryAutoCheckParserWarn ErrorCategory = "AUTO_CHECK_PARSER_WARN"

	// detail — REJECTED cause variants.
	CategoryAssignFailed ErrorCategory = "ASSIGN_FAILED"
	CategorySummaryValidation ErrorCategory = "SUMMARY_VALIDATION"
	CategoryAlreadyInState ErrorCategory = "ALREADY_IN_STATE"
	CategoryDependencyNotReady ErrorCategory = "DEPENDENCY_NOT_READY"

	// detail — INVALID_STATE cause variants.
	CategoryStateTransitionBlocked ErrorCategory = "STATE_TRANSITION_BLOCKED"

	// hostler-specific (introduced in , unified into
	// ErrorCategory in ).
	CategoryInvalidInput ErrorCategory = "INVALID_INPUT"
	CategoryConflict ErrorCategory = "CONFLICT"
	CategoryConfigInvalid ErrorCategory = "CONFIG_INVALID"
	CategoryHMACVerifyFailed ErrorCategory = "HMAC_VERIFY_FAILED"
	CategoryGPGVerifyFailed ErrorCategory = "GPG_VERIFY_FAILED"
	CategoryDBError ErrorCategory = "DB_ERROR"
	CategoryInternal ErrorCategory = "INTERNAL"

	// Monorepo-specific ( sprint-status rollback guard).
	CategorySprintStatusRegression ErrorCategory = "SPRINT_STATUS_REGRESSION"
)

// String is invoked automatically by fmt — compatible with existing
// string-comparison code.
func (c ErrorCategory) String() string { return string(c) }

// AllCategories returns the full list of categories for use in catalog
// docs / validation / audit. Must stay aligned with
// docs/04-guides/cli-error-category-catalog.md (drift prevention).
func AllCategories() []ErrorCategory {
	return []ErrorCategory{
		CategoryNotFound,
		CategoryBlocked,
		CategoryRejected,
		CategoryInvalidState,
		CategoryInternalError,
		CategoryHarnessBlocked,
		CategoryPlaceholderBody,
		CategoryCeremonySectionMissing,
		CategoryResultStrictMissingFiles,
		CategoryKBLinkMissing,
		CategoryAutoCheckParserWarn,
		CategoryAssignFailed,
		CategorySummaryValidation,
		CategoryAlreadyInState,
		CategoryDependencyNotReady,
		CategoryStateTransitionBlocked,
		CategoryInvalidInput,
		CategoryConflict,
		CategoryConfigInvalid,
		CategoryHMACVerifyFailed,
		CategoryGPGVerifyFailed,
		CategoryDBError,
		CategoryInternal,
		CategorySprintStatusRegression,
	}
}
