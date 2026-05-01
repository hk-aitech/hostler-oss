package db

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
)

// Work ticket issuance plus context acknowledgment persistence.

const (
	workTicketRandomBytes = 4 // 8 hex chars
	workTicketPrefix      = "WT-"
)

// GenerateWorkTicket issues a work ticket keyed by Task ID.
// Format: WT-{taskID}-{8 hex chars}
// Example: WT-T542-a1b2c3d4
func GenerateWorkTicket(taskID string) (string, error) {
	buf := make([]byte, workTicketRandomBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return fmt.Sprintf("%s%s-%s", workTicketPrefix, taskID, hex.EncodeToString(buf)), nil
}

// SetTaskWorkTicket updates the tasks.work_ticket column.
func SetTaskWorkTicket(database *sql.DB, taskID, ticket string) error {
	_, err := database.Exec(
		"UPDATE tasks SET work_ticket = ?, updated_at = ? WHERE task_id = ?",
		ticket, NowUTC(), taskID,
	)
	return err
}

// GetTaskWorkTicket returns the tasks.work_ticket value (empty string if absent).
func GetTaskWorkTicket(database *sql.DB, taskID string) (string, error) {
	var ticket sql.NullString
	err := database.QueryRow(
		"SELECT work_ticket FROM tasks WHERE task_id = ?",
		taskID,
	).Scan(&ticket)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if !ticket.Valid {
		return "", nil
	}
	return ticket.String, nil
}

// InsertContextAck records a single ack into context_acknowledgments
// (idempotent).
func InsertContextAck(database *sql.DB, ticket, hash string, size int) error {
	_, err := database.Exec(
		`INSERT OR IGNORE INTO context_acknowledgments (ticket, hash, content_size)
		 VALUES (?, ?, ?)`,
		ticket, hash, size,
	)
	return err
}

// HasContextAck reports whether the given ticket + hash pair has been recorded.
func HasContextAck(database *sql.DB, ticket, hash string) (bool, error) {
	var cnt int
	err := database.QueryRow(
		"SELECT COUNT(*) FROM context_acknowledgments WHERE ticket = ? AND hash = ?",
		ticket, hash,
	).Scan(&cnt)
	if err != nil {
		return false, err
	}
	return cnt > 0, nil
}

// CountContextAcks returns the ack count for a given ticket (for debugging
// and auditing).
func CountContextAcks(database *sql.DB, ticket string) (int, error) {
	var cnt int
	err := database.QueryRow(
		"SELECT COUNT(*) FROM context_acknowledgments WHERE ticket = ?",
		ticket,
	).Scan(&cnt)
	return cnt, err
}
