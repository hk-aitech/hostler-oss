package domain

// WorkTicket — a Task's work-ticket identifier plus HMAC signature data.
type WorkTicket struct {
	TaskID    TaskID
	TicketRef string
	CreatedAt string
}

// Signature — HMAC signature value object.
type Signature struct {
	Algorithm string // "sha256"
	Value     string // hex-encoded
}

// IsValid reports whether the signature is non-empty (a value-only check, not a cryptographic verification).
func (s Signature) IsValid() bool {
	return s.Algorithm != "" && s.Value != ""
}

// String returns the "sha256:abcdef..." representation.
func (s Signature) String() string {
	if s.Algorithm == "" {
		return s.Value
	}
	return s.Algorithm + ":" + s.Value
}
