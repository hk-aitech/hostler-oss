package domain

// DomainSeverity — rule-violation severity (mirrors pkg/rules.Severity to avoid an import cycle).
type DomainSeverity int

const (
	DomainSeverityOff       DomainSeverity = 0
	DomainSeverityAdvisory  DomainSeverity = 1
	DomainSeverityWarn      DomainSeverity = 2
	DomainSeverityBlock     DomainSeverity = 3
	DomainSeverityHardBlock DomainSeverity = 4
)

// DomainPhase — rule-execution lifecycle phase (mirrors pkg/rules.Phase).
type DomainPhase string

const (
	DomainPhaseTaskStart      DomainPhase = "task:start"
	DomainPhaseTaskComplete   DomainPhase = "task:complete"
	DomainPhaseSprintStart    DomainPhase = "sprint:start"
	DomainPhaseSprintComplete DomainPhase = "sprint:complete"
	DomainPhasePreCommit      DomainPhase = "precommit"
)

// DomainRule — domain-level rule interface (uses pure types only).
type DomainRule interface {
	ID() string
	Description() string
	DefaultSeverity() DomainSeverity
}
