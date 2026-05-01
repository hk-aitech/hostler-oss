package domain

import "time"

// MailboxItem — domain type for a mailbox entry.
type MailboxItem struct {
	ID          string
	Subject     string
	Body        string
	Source      string
	SubmittedAt time.Time
	ProcessedAt *time.Time
	Status      string // "pending" | "accepted" | "rejected"
}

// IsPending reports whether the item is unprocessed.
func (m MailboxItem) IsPending() bool {
	return m.Status == "pending"
}

// IsProcessed reports whether the item has been processed.
func (m MailboxItem) IsProcessed() bool {
	return m.ProcessedAt != nil
}
